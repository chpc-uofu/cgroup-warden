// Copyright (C) 2024 Center for High Performance Computing <helpdesk@chpc.utah.edu>

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

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/control", controlHandler())
	mux.Handle("/", http.NotFoundHandler())
	var handler http.Handler = mux
	return handler
}

func main() {
	var (
		listenAddr  string
		certFile    string
		keyFile     string
		bearerToken string
		insecure    bool
	)

	flag.StringVar(&listenAddr, "listenAddr", ":2112", "address to listen on for telemetry")
	flag.StringVar(&certFile, "certFile", "", "file containing certificate to use for tls")
	flag.StringVar(&keyFile, "keyFile", "", "file containing key to use for tls")
	flag.StringVar(&bearerToken, "bearerToken", "", "bearer token to use for authentication")
	flag.BoolVar(&insecure, "insecure", false, "disable tls and bearer token authentication")
	flag.Parse()

	if !insecure {
		if certFile == "" {
			log.Fatal("certificate required for use with tls")
		}
		if keyFile == "" {
			log.Fatal("key required for use with tls")
		}
		if bearerToken == "" || len(bearerToken) < 16 {
			log.Fatal("token of length > 16 required for authentication")
		}

		handler := authorize(newHandler(), bearerToken)
		log.Fatal(http.ListenAndServeTLS(listenAddr, certFile, keyFile, handler))

	} else {
		handler := newHandler()
		log.Fatal(http.ListenAndServe(listenAddr, handler))
	}
}
