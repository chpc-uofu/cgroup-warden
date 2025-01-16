package main

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/chpc-uofu/cgroup-warden/control"
	"github.com/chpc-uofu/cgroup-warden/metrics"
	"github.com/containerd/cgroups/v3"
)

type wardenConfig struct {
	cgroup   string
	listen   string
	cert     string
	key      string
	bearer   string
	insecure bool
	proc     bool
	meta     bool
	level    string
}

func authorize(next http.Handler, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			slog.Warn("unauthorized request", "address", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func stringEnvRequired(flag string) string {
	s, set := os.LookupEnv(flag)
	if !set {
		slog.Error("flag is required", "flag", flag)
		os.Exit(1)
	}
	return s
}

func stringEnvWithDefault(flag string, value string) string {
	s, set := os.LookupEnv(flag)
	if !set {
		return value
	} else {
		return s
	}
}

func boolEnvWithDefault(flag string, value bool) bool {
	s, set := os.LookupEnv(flag)
	if !set {
		return value
	} else {
		b, err := strconv.ParseBool(s)
		if err != nil {
			slog.Error("invalid value", flag, value, "type", "bool")
		}
		return b
	}
}

func updateLogLevel(level string) {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warning":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}
	slog.SetLogLoggerLevel(slogLevel)
}

func readConfigFromEnvironment() *wardenConfig {
	conf := &wardenConfig{}

	conf.listen = stringEnvWithDefault("CGROUP_WARDEN_LISTEN_ADDRESS", ":2112")
	conf.cgroup = stringEnvWithDefault("CGROUP_WARDEN_ROOT_CGROUP", "/user.slice")
	conf.insecure = boolEnvWithDefault("CGROUP_WARDEN_INSECURE_MODE", false)
	conf.proc = boolEnvWithDefault("CGROUP_WARDEN_COLLECT_PROCESS_INFO", true)
	conf.meta = boolEnvWithDefault("CGROUP_WARDEN_META_METRICS", true)
	conf.level = stringEnvWithDefault("CGROUP_WARDEN_LOG_LEVEL", "info")

	if !conf.insecure {
		conf.cert = stringEnvRequired("CGROUP_WARDEN_CERTIFICATE")
		conf.key = stringEnvRequired("CGROUP_WARDEN_PRIVATE_KEY")
		conf.bearer = stringEnvRequired("CGROUP_WARDEN_BEARER_TOKEN")
	}

	return conf
}

func main() {
	conf := readConfigFromEnvironment()

	updateLogLevel(conf.level)
	slog.Info("set log level", "level", conf.level)

	mode := cgroups.Mode()

	switch mode {
	case cgroups.Hybrid:
		slog.Info("using hybrid cgroups")
	case cgroups.Legacy:
		slog.Info("using legacy cgroups")
	case cgroups.Unified:
		slog.Info("using unified cgroups")
	case cgroups.Unavailable:
		slog.Error("cgroups unavailable")
		os.Exit(1)
	default:
		slog.Error("unknown cgroups mode", "mode", mode)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.MetricsHandler(conf.cgroup, conf.meta))
	mux.Handle("/", http.NotFoundHandler())

	if conf.insecure {
		mux.Handle("/control", control.ControlHandler)
		slog.Info("running in insecure mode")
		slog.Error("server error", "err", http.ListenAndServe(conf.listen, mux))
		os.Exit(1)

	} else {
		mux.Handle("/control", authorize(control.ControlHandler, conf.bearer))
		slog.Info("running in secure mode")
		slog.Error("server error", "err", http.ListenAndServeTLS(conf.listen, conf.cert, conf.key, mux))
		os.Exit(1)
	}
}
