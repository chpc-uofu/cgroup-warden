package main

import (
	"fmt"
	"net/http"

	"github.com/containerd/cgroups"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors/version"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/treydock/cgroup_exporter/collector"
)

func metricsHandler(paths []string, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		v2 := (cgroups.Mode() == cgroups.Unified)
		cgroupCollector := collector.NewCgroupCollector(v2, paths, logger)
		registry.MustRegister(cgroupCollector)
		registry.MustRegister(version.NewCollector(fmt.Sprintf("%s_exporter", collector.Namespace)))
		gatherers := prometheus.Gatherers{registry}
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}
