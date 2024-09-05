package main

import (
	"context"
	"log"
	"net/http"
	"os/user"
	"regexp"

	systemd "github.com/coreos/go-systemd/v22/dbus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var MetricsHandler = http.HandlerFunc(metricsHandler)

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()
	collector := NewCollector("user-*.slice")
	registry.MustRegister(collector)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

var namespace = "systemd_unit"
var labels = []string{"unit", "username"}

type Collector struct {
	pattern          string
	memoryAccounting *prometheus.Desc
	memoryMax        *prometheus.Desc
	memoryMin        *prometheus.Desc
	memoryHigh       *prometheus.Desc
	memoryLow        *prometheus.Desc
	memoryCurrent    *prometheus.Desc
	cpuAccounting    *prometheus.Desc
	cpuUsage         *prometheus.Desc
	cpuQuota         *prometheus.Desc
}

type Metric struct {
	memoryAccounting bool
	memoryMax        uint64
	memoryMin        uint64
	memoryHigh       uint64
	memoryLow        uint64
	memoryCurrent    uint64
	cpuAccounting    bool
	cpuUsage         uint64
	cpuQuota         uint64
	unit             string
	username         string
}

func NewCollector(pattern string) *Collector {
	return &Collector{
		pattern: pattern,
		memoryAccounting: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "accounting"),
			"Whether memory accounting is enabled", labels, nil),
		memoryMax: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "max_bytes"),
			"Memory maximum limit", labels, nil),
		memoryMin: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "min_bytes"),
			"Memory minimum limit", labels, nil),
		memoryHigh: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "high_bytes"),
			"Memory high limit", labels, nil),
		memoryLow: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "low_bytes"),
			"Memory low limit", labels, nil),
		memoryCurrent: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "current_bytes"),
			"Resident shared size memory usage", labels, nil),
		cpuAccounting: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "accounting"),
			"Whether CPU accounting is enabled", labels, nil),
		cpuUsage: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "user_seconds"),
			"Total CPU usage", labels, nil),
		cpuQuota: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "quota_seconds_per_second"),
			"CPU Quota", labels, nil),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.memoryAccounting
	ch <- c.memoryMax
	ch <- c.memoryMin
	ch <- c.memoryHigh
	ch <- c.memoryLow
	ch <- c.memoryCurrent
	ch <- c.cpuAccounting
	ch <- c.cpuUsage
	ch <- c.cpuQuota
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	metrics := c.collectMetrics()
	for _, m := range metrics {
		ch <- prometheus.MustNewConstMetric(c.memoryAccounting, prometheus.GaugeValue, b2f(m.memoryAccounting), m.unit, m.username)
		ch <- prometheus.MustNewConstMetric(c.memoryMax, prometheus.GaugeValue, float64(m.memoryMax), m.unit, m.username)
		ch <- prometheus.MustNewConstMetric(c.memoryMin, prometheus.GaugeValue, float64(m.memoryMin), m.unit, m.username)
		ch <- prometheus.MustNewConstMetric(c.memoryHigh, prometheus.GaugeValue, float64(m.memoryHigh), m.unit, m.username)
		ch <- prometheus.MustNewConstMetric(c.memoryLow, prometheus.GaugeValue, float64(m.memoryLow), m.unit, m.username)
		ch <- prometheus.MustNewConstMetric(c.memoryCurrent, prometheus.GaugeValue, float64(m.memoryCurrent), m.unit, m.username)
		ch <- prometheus.MustNewConstMetric(c.cpuAccounting, prometheus.GaugeValue, b2f(m.cpuAccounting), m.unit, m.username)
		ch <- prometheus.MustNewConstMetric(c.cpuUsage, prometheus.CounterValue, float64(m.cpuUsage), m.unit, m.username)
		ch <- prometheus.MustNewConstMetric(c.cpuQuota, prometheus.CounterValue, float64(m.cpuQuota), m.unit, m.username)
	}
}

func (c *Collector) collectMetrics() []Metric {

	var metrics []Metric
	ctx := context.Background()
	conn, err := systemd.NewSystemConnectionContext(ctx)
	if err != nil {
		log.Println(err)
		return metrics
	}
	defer conn.Close()

	units, err := conn.ListUnitsByPatternsContext(ctx, []string{}, []string{c.pattern})
	if err != nil {
		log.Println(err)
		return metrics
	}

	for _, unit := range units {
		props, err := conn.GetUnitTypePropertiesContext(ctx, unit.Name, "Slice")
		if err != nil {
			log.Println(err)
			continue
		}
		metric := Metric{
			memoryAccounting: props["MemoryAccounting"].(bool),
			memoryMax:        props["MemoryMax"].(uint64),
			memoryMin:        props["MemoryMin"].(uint64),
			memoryHigh:       props["MemoryHigh"].(uint64),
			memoryLow:        props["MemoryLow"].(uint64),
			memoryCurrent:    props["MemoryCurrent"].(uint64),
			cpuAccounting:    props["CPUAccounting"].(bool),
			cpuUsage:         props["CPUUsageNSec"].(uint64),
			cpuQuota:         props["CPUQuotaPerSecUSec"].(uint64),
			unit:             unit.Name,
			username:         lookupUsername(unit),
		}
		metrics = append(metrics, metric)
	}
	return metrics
}

func lookupUsername(unit systemd.UnitStatus) string {
	pattern := `^user-(\d+)\.slice$`
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(unit.Name)

	if len(match) < 1 {
		return "unknown user"
	}

	user, err := user.LookupId(match[1])
	if err != nil {
		return "unknown user"
	}

	return user.Username
}

func b2f(b bool) float64 {
	if !b {
		return -1.0
	}
	return 1.0
}
