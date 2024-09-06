package main

import (
	"flag"
	"log"
	"net/http"
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

func main() {
	var (
		pattern     string
		listenAddr  string
		certFile    string
		keyFile     string
		bearerToken string
		insecure    bool
		collectProc bool
	)

	flag.StringVar(&pattern, "pattern", "user-*.slice", "unit pattern to match units on")
	flag.StringVar(&listenAddr, "listenAddr", ":2112", "address to listen on for telemetry")
	flag.StringVar(&certFile, "certFile", "", "file containing certificate to use for tls")
	flag.StringVar(&keyFile, "keyFile", "", "file containing key to use for tls")
	flag.StringVar(&bearerToken, "bearerToken", "", "bearer token to use for authentication")
	flag.BoolVar(&insecure, "insecure", false, "disable tls and bearer token authentication")
	flag.BoolVar(&collectProc, "collectProc", false, "enable the collection of process metrics")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/metrics", MetricsHandler(pattern, collectProc))
	mux.Handle("/", http.NotFoundHandler())

	if !insecure {
		mux.Handle("/control", authorize(ControlHandler, bearerToken))
		if certFile == "" {
			log.Fatal("certificate required for use with tls")
		}
		if keyFile == "" {
			log.Fatal("key required for use with tls")
		}
		if bearerToken == "" || len(bearerToken) < 16 {
			log.Fatal("token of length > 16 required for authentication")
		}

		log.Fatal(http.ListenAndServeTLS(listenAddr, certFile, keyFile, mux))

	} else {
		mux.Handle("/control", ControlHandler)
		log.Println("running in insecure mode")
		log.Fatal(http.ListenAndServe(listenAddr, mux))
	}
}
