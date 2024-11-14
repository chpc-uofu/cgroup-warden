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
	root        string
	mode        cgroups.CGMode
	memoryUsage *prometheus.Desc
	cpuUsage    *prometheus.Desc
	procCPU     *prometheus.Desc
	procMemory  *prometheus.Desc
	procCount   *prometheus.Desc
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.memoryUsage
	ch <- c.cpuUsage
	ch <- c.procCPU
	ch <- c.procMemory
	ch <- c.procCount
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	var stats []Metric
	if c.mode == cgroups.Unified {
		stats = UnifiedMetrics(c.root)
	} else if c.mode == cgroups.Legacy {
		stats = LegacyMetrics(c.root)
	} else if c.mode == cgroups.Hybrid {
		stats = LegacyMetrics(c.root)
	} else {
		log.Println("Could not determine cgroup mode")
	}

	for _, s := range stats {
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
	cgroup      string
	username    string
	memoryUsage uint64
	cpuUsage    float64
	processes   map[string]Process
}
