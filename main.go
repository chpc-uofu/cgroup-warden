package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/chpc-uofu/cgroup-warden/control"
	"github.com/chpc-uofu/cgroup-warden/metrics"
)

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

func updateLogLevel(level string) {
	var slogLevel slog.Level = slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warning":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	}
	slog.SetLogLoggerLevel(slogLevel)
}

func main() {

	conf, err := NewConfig()
	if err != nil {
		slog.Error("Unable to parse configuration", "err", err)
		os.Exit(1)
	}
	updateLogLevel(conf.LogLevel)

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.MetricsHandler(conf.RootCGroup, conf.MetaMetrics))
	mux.Handle("/", http.NotFoundHandler())

	if conf.InsecureMode {
		mux.Handle("/control", control.ControlHandler)
		slog.Info("Starting server")
		slog.Error("server error", "err", http.ListenAndServe(conf.ListenAddress, mux))
		os.Exit(1)

	} else {
		mux.Handle("/control", authorize(control.ControlHandler, conf.BearerToken))
		slog.Info("Starting server")
		slog.Error("server error", "err", http.ListenAndServeTLS(conf.ListenAddress, conf.Certificate, conf.PrivateKey, mux))
		os.Exit(1)
	}
}
