package main

import (
	"log"
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/log"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
)

func authorize(next http.Handler, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func newServer(paths []string, logger kitlog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler(paths, logger))
	mux.Handle("/control", controlHandler(logger))
	mux.Handle("/", http.NotFoundHandler())
	var handler http.Handler = mux
	return handler
}

func main() {
	var (
		app    = kingpin.New("cgroup-warden", "cgroup monitoring and resource control daemon")
		paths  = app.Flag("paths", "list of cgroup root paths to monitor").Default("/user.slice").Strings()
		listen = app.Flag("listen", "address to listen on for telemetery").Default(":2112").String()
		tls    = app.Flag("tls", "ahether to use tls for telemetry.").Default("false").Bool()
		cert   = app.Flag("tls.cert", "certificate file to use for TLS verification").Envar("CGROUP_WARDEN_TLS_CERT_FILE").String()
		key    = app.Flag("tls.key", "key file to use for TLS verification").Envar("CGROUP_WARDEN_TLS_KEY_FILE").String()
		bearer = app.Flag("bearer", "bearer token to authorize /control requests").Envar("CGROUP_WARDEN_BEARER_TOKEN").String()
	)

	promlogConfig := &promlog.Config{}
	flag.AddFlags(app, promlogConfig)
	app.DefaultEnvars()
	kingpin.MustParse(app.Parse(os.Args[1:]))
	logger := promlog.New(promlogConfig)
	server := newServer(*paths, logger)

	var err error
	if *tls {
		if *cert == "" {
			app.FatalUsage("certificate required for use with TLS")
		}
		if *key == "" {
			app.FatalUsage("key required for use with TLS")
		}
		if *bearer != "" {
			server = authorize(server, *bearer)
		}
		err = http.ListenAndServeTLS(*listen, *cert, *key, server)
	} else {
		err = http.ListenAndServe(*listen, server)
	}

	if err != nil {
		log.Fatal(err)
	}
}
