package metrics

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os/user"
	"regexp"
	"sync"

	"github.com/containerd/cgroups/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	USPerS               = 1000000    // million
	NSPerS               = 1000000000 // billion
	MaxCGroupMemoryLimit = 9223372036854771712
	cgroupRoot           = "/sys/fs/cgroup"
)

var (
	namespace  = "cgroup_warden"
	labels     = []string{"cgroup", "username"}
	procLabels = []string{"cgroup", "username", "proc"}
)

func MetricsHandler(root string, meta bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		collector := NewCollector(root)
		registry.MustRegister(collector)
		gatherers := prometheus.Gatherers{registry}
		if meta {
			gatherers = append(gatherers, prometheus.DefaultGatherer)
		}
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
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
	memoryMax   *prometheus.Desc
	cpuQuota    *prometheus.Desc
}

type cgroupInfo struct {
	username    string
	memoryUsage uint64
	cpuUsage    float64
	memoryMax   uint64
	cpuQuota    int64
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.memoryUsage
	ch <- c.cpuUsage
	ch <- c.procCPU
	ch <- c.procMemory
	ch <- c.procCount
	ch <- c.memoryMax
	ch <- c.cpuQuota
}

func (c *Collector) newHierarchy() hierarchy {
	var h hierarchy

	if c.mode == cgroups.Unified {
		h = &unified{root: c.root}
	} else {
		h = &legacy{root: c.root}
	}

	return h
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	h := c.newHierarchy()

	groups, err := h.GetGroupsWithPIDs()
	if err != nil {
		slog.Error("could not collect cgroups with pids", "err", err)
		return
	}

	wg := sync.WaitGroup{}
	active := make(map[string]bool)
	for cg, pids := range groups {
		active[cg] = true
		wg.Add(1)
		go func() {
			defer wg.Done()

			info, err := h.CGroupInfo(cg)
			if err != nil {
				slog.Warn("unable to collect group info", "cgroup", cg, "err", err)
				return
			}

			ch <- prometheus.MustNewConstMetric(c.memoryUsage, prometheus.GaugeValue, float64(info.memoryUsage), cg, info.username)
			ch <- prometheus.MustNewConstMetric(c.cpuUsage, prometheus.CounterValue, info.cpuUsage, cg, info.username)
			ch <- prometheus.MustNewConstMetric(c.memoryMax, prometheus.GaugeValue, negativeOneIfMax(info.memoryMax), cg, info.username)
			ch <- prometheus.MustNewConstMetric(c.cpuQuota, prometheus.CounterValue, float64(info.cpuQuota), cg, info.username)

			procs, err := ProcessInfo(cg, pids)
			if err != nil {
				slog.Warn("unable to collect process info", "cgroup", cg, "err", err)
				return
			}

			for name, p := range procs {
				ch <- prometheus.MustNewConstMetric(c.procCPU, prometheus.CounterValue, float64(p.cpuSecondsTotal), cg, info.username, name)
				ch <- prometheus.MustNewConstMetric(c.procMemory, prometheus.GaugeValue, float64(p.memoryBytesTotal), cg, info.username, name)
				ch <- prometheus.MustNewConstMetric(c.procCount, prometheus.GaugeValue, float64(p.count), cg, info.username, name)
			}
		}()
	}
	wg.Wait()
	CleanProcessCache(active)
}

func NewCollector(root string) *Collector {
	mode := cgroups.Mode()
	return &Collector{
		root: root,
		mode: mode,
		memoryUsage: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "usage_bytes"),
			"Total memory usage in bytes", labels, nil),
		cpuUsage: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "usage_seconds"),
			"Total CPU usage in seconds", labels, nil),
		procCPU: prometheus.NewDesc(prometheus.BuildFQName(namespace, "proc", "cpu_usage_seconds"),
			"Aggregate CPU usage for this process in seconds", procLabels, nil),
		procMemory: prometheus.NewDesc(prometheus.BuildFQName(namespace, "proc", "memory_usage_bytes"),
			"Aggregate memory usage for this process", procLabels, nil),
		procCount: prometheus.NewDesc(prometheus.BuildFQName(namespace, "proc", "count"),
			"Instance count of this process", procLabels, nil),
		memoryMax: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "max"),
			"Maximum memory limit of this unit in bytes.", labels, nil),
		cpuQuota: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "quota"),
			"Maximum CPU quota of this unit in micro seconds per second", labels, nil),
	}
}

type hierarchy interface {
	GetGroupsWithPIDs() (map[string]map[uint64]bool, error)
	CGroupInfo(cg string) (cgroupInfo, error)
}

var uidRe = regexp.MustCompile(`user-(\d+)\.slice`)

// lookupUsername looks up a username given the systemd user slice name.
// If compiled with CGO, this function will call the C function getpwuid_r
// from the standard C library; This is necessary when user identities are
// provided by services like sss and ldap.
func lookupUsername(slice string) (string, error) {
	match := uidRe.FindStringSubmatch(slice)

	if len(match) < 2 {
		return "", fmt.Errorf("cannot determine uid from '%s'", slice)
	}

	user, err := user.LookupId(match[1])
	if err != nil {
		return "", fmt.Errorf("unable to lookup user with id '%s'", match[1])
	}

	return user.Username, nil
}

// max memory value is a maxint64 rounded down to the nearest page number
func negativeOneIfMax(value uint64) float64 {
	if value == MaxCGroupMemoryLimit || value == math.MaxUint64 {
		return -1
	}
	return float64(value)
}
