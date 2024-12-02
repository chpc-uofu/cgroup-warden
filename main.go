package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/chpc-uofu/cgroup-warden/metrics"
)

type wardenConfig struct {
	pattern  string
	listen   string
	cert     string
	key      string
	bearer   string
	insecure bool
	proc     bool
}

func authorize(next http.Handler, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func stringEnvRequired(flag string) string {
	s, set := os.LookupEnv(flag)
	if !set {
		log.Fatalf("%s is required", flag)
	}
	return s
}

func stringEnvWithDefault(flag string, value string) string {
	s, set := os.LookupEnv(flag)
	if !set {
		log.Printf("%s not set, using default value of '%s'", flag, value)
		return value
	} else {
		return s
	}
}

func boolEnvWithDefault(flag string, value bool) bool {
	s, set := os.LookupEnv(flag)
	if !set {
		log.Printf("%s not set, using default value of '%t'", flag, value)
		return value
	} else {
		b, err := strconv.ParseBool(s)
		if err != nil {
			log.Fatalf("%s set to invalid value of '%s'", flag, s)
		}
		return b
	}
}

func readConfigFromEnvironment() *wardenConfig {
	conf := &wardenConfig{}

	conf.listen = stringEnvWithDefault("CGROUP_WARDEN_LISTEN_ADDRESS", ":2112")
	conf.pattern = stringEnvWithDefault("CGROUP_WARDEN_UNIT_PATTERN", "/user.slice")
	conf.insecure = boolEnvWithDefault("CGROUP_WARDEN_INSECURE_MODE", false)
	conf.proc = boolEnvWithDefault("CGROUP_WARDEN_COLLECT_PROCESS_INFO", true)

	if !conf.insecure {
		conf.cert = stringEnvRequired("CGROUP_WARDEN_CERTIFICATE")
		conf.key = stringEnvRequired("CGROUP_WARDEN_PRIVATE_KEY")
		conf.bearer = stringEnvRequired("CGROUP_WARDEN_BEARER_TOKEN")
	}

	return conf
}

func main() {
	conf := readConfigFromEnvironment()

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.MetricsHandler(conf.pattern))
	mux.Handle("/", http.NotFoundHandler())

	if conf.insecure {
		mux.Handle("/control", ControlHandler)
		log.Println("running in insecure mode")
		log.Fatal(http.ListenAndServe(conf.listen, mux))

	} else {
		mux.Handle("/control", authorize(ControlHandler, conf.bearer))
		log.Println("running in secure mode")
		log.Fatal(http.ListenAndServeTLS(conf.listen, conf.cert, conf.key, mux))
	}
}
