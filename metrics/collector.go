package metrics

import (
	"log"
	"net/http"

	"github.com/containerd/cgroups/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	namespace  = "cgroup_warden"
	labels     = []string{"cgroup", "username"}
	procLabels = []string{"cgroup", "username", "proc"}
)

// TODO: add meta metrics
func MetricsHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		collector := NewCollector(root)
		registry.MustRegister(collector)
		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

type Collector struct {
	root             string
	mode             cgroups.CGMode
	memoryAccounting *prometheus.Desc
	cpuAccounting    *prometheus.Desc
	memoryQuota      *prometheus.Desc
	cpuQuota         *prometheus.Desc
	memoryUsage      *prometheus.Desc
	cpuUsage         *prometheus.Desc
	procCPU          *prometheus.Desc
	procMemory       *prometheus.Desc
	procCount        *prometheus.Desc
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.memoryAccounting
	ch <- c.cpuAccounting
	ch <- c.memoryQuota
	ch <- c.cpuQuota
	ch <- c.memoryUsage
	ch <- c.cpuUsage
	ch <- c.procCPU
	ch <- c.procMemory
	ch <- c.procCount
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	var stats []Metric
	if c.mode == cgroups.Unified {
		log.Println("running in unified mode")
		stats = UnifiedStats(c.root)
	} else if c.mode == cgroups.Legacy {
		log.Println("running in legacy mode")
		stats = LegacyStats(c.root)
	} else if c.mode == cgroups.Hybrid {
		log.Println("running in hybrid mode")
		stats = LegacyStats(c.root)
	} else {
		log.Println("Could not determine cgroup mode")
	}

	log.Println(stats)

	for _, s := range stats {
		ch <- prometheus.MustNewConstMetric(c.memoryAccounting, prometheus.GaugeValue, b2f(s.memoryAccounting), s.cgroup, s.username)
		ch <- prometheus.MustNewConstMetric(c.cpuAccounting, prometheus.GaugeValue, b2f(s.cpuAccounting), s.cgroup, s.username)
		ch <- prometheus.MustNewConstMetric(c.memoryQuota, prometheus.GaugeValue, float64(s.memoryQuota), s.cgroup, s.username)
		ch <- prometheus.MustNewConstMetric(c.cpuQuota, prometheus.GaugeValue, float64(s.cpuQuota), s.cgroup, s.username)
		ch <- prometheus.MustNewConstMetric(c.memoryUsage, prometheus.GaugeValue, float64(s.memoryUsage), s.cgroup, s.username)
		ch <- prometheus.MustNewConstMetric(c.cpuUsage, prometheus.CounterValue, s.cpuUsage, s.cgroup, s.username)
		for name, p := range s.processes {
			ch <- prometheus.MustNewConstMetric(c.procCPU, prometheus.CounterValue, float64(p.cpu), s.cgroup, s.username, name)
			ch <- prometheus.MustNewConstMetric(c.procMemory, prometheus.GaugeValue, float64(p.memory), s.cgroup, s.username, name)
			ch <- prometheus.MustNewConstMetric(c.procCount, prometheus.GaugeValue, float64(p.count), s.cgroup, s.username, name)
		}
	}
}

func NewCollector(root string) *Collector {

	mode := cgroups.Mode()
	return &Collector{
		root: root,
		mode: mode,
		memoryAccounting: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "accounting"),
			"Whether memory accounting is enabled", labels, nil),
		cpuAccounting: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "accounting"),
			"Whether CPU accounting is enabled", labels, nil),
		memoryQuota: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "quota_bytes"),
			"Memory Quota in bytes", labels, nil),
		cpuQuota: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "quota_us_per_s"),
			"CPU Quota in microseconds per second", labels, nil),
		memoryUsage: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "usage_bytes"),
			"Total memory usage in bytes", labels, nil),
		cpuUsage: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "usage_seconds"),
			"Total CPU usage in seconds", labels, nil),
		procCPU: prometheus.NewDesc(prometheus.BuildFQName(namespace, "proc", "cpu_usage_seconds"),
			"Aggregate CPU usage for this process in seconds", procLabels, nil),
		procMemory: prometheus.NewDesc(prometheus.BuildFQName(namespace, "proc", "memory_usage_bytes"),
			"Aggregate memory usage for this process", procLabels, nil),
		procCount: prometheus.NewDesc(prometheus.BuildFQName(namespace, "proc", "count"),
			"Instance count of this process", procLabels, nil),
	}
}

type Process struct {
	cpu    float64
	memory uint64
	count  uint64
}

type Metric struct {
	cgroup           string
	username         string
	memoryAccounting bool
	cpuAccounting    bool
	memoryQuota      int64
	cpuQuota         int64
	memoryUsage      uint64
	cpuUsage         float64
	processes        map[string]Process
}

func b2f(b bool) float64 {
	if !b {
		return -1.0
	}
	return 1.0
}
