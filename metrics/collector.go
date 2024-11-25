package metrics

import (
	"log"
	"net/http"
	"os/user"
	"regexp"
	"sync"

	"github.com/containerd/cgroups/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/procfs"
)

const (
	USPerS     = 1000000    // million
	NSPerS     = 1000000000 // billion
	cgroupRoot = "/sys/fs/cgroup"
)

var (
	namespace  = "cgroup_warden"
	labels     = []string{"cgroup", "username"}
	procLabels = []string{"cgroup", "username", "proc"}
	lock       = sync.RWMutex{}
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

type Metric struct {
	cgroup      string
	username    string
	memoryUsage uint64
	cpuUsage    float64
	processes   map[string]Process
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.memoryUsage
	ch <- c.cpuUsage
	ch <- c.procCPU
	ch <- c.procMemory
	ch <- c.procCount
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	stats := c.CollectMetrics()
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

// set of process IDs
type pidSet map[uint64]bool

// map from a cgroup name to a set of proceess IDs
type groupPIDMap map[string]pidSet

// legacy or unified cgroup hierarchy
type hierarchy interface {

	// returns a map of all cgroups underneath the root with their respective PIDs
	GetGroupsWithPIDs() groupPIDMap

	// creates a metric for the cgroup with the PID information
	CreateMetric(cgroup string, pids pidSet) Metric
}

func (c *Collector) CollectMetrics() []Metric {

	var h hierarchy
	if c.mode == cgroups.Unified {
		h = &unified{root: c.root}
	} else {
		h = &legacy{root: c.root}
	}

	groupPIDs := h.GetGroupsWithPIDs()

	wg := &sync.WaitGroup{}
	wg.Add(len(groupPIDs))

	var metrics []Metric
	for group, pids := range groupPIDs {
		go func(group string, procs map[uint64]bool) {
			defer wg.Done()
			metric := h.CreateMetric(group, pids)
			lock.Lock()
			metrics = append(metrics, metric)
			lock.Unlock()
		}(group, pids)
	}

	wg.Wait()

	return metrics
}

type Process struct {
	cpu    float64
	memory uint64
	count  uint64
}

// Given a set of PIDs, aggregate process count, memory, and
// CPU usage on the process name associated with the PIDs.
// Returns a map of process names -> aggregate usage.
func ProcInfo(pids map[uint64]bool) map[string]Process {
	processes := make(map[string]Process)

	fs, err := procfs.NewDefaultFS()
	if err != nil {
		log.Printf("could not mount procfs: %s\n", err)
		return processes
	}

	for pid := range pids {
		proc, err := fs.Proc(int(pid))
		if err != nil {
			continue
		}

		comm, err := proc.Comm()
		if err != nil {
			continue
		}

		stat, err := proc.Stat()
		if err != nil {
			continue
		}

		process, ok := processes[comm]
		if !ok {
			process = Process{cpu: 0, memory: 0, count: 0}
		}

		process.cpu = process.cpu + stat.CPUTime()
		process.memory = process.memory + uint64(stat.RSS)
		process.count = process.count + 1

		processes[comm] = process
	}

	return processes
}

var userSliceRe = regexp.MustCompile(`user-(\d+)\.slice`)

// Looks up the username associated with a user slice cgroup.
// Slice of the form 'user-1000.slice' or '/user.slice/user-1234.slice'
// Must be compiled with CGO_ENABLED if used over NFS.
func lookupUsername(slice string) string {
	match := userSliceRe.FindStringSubmatch(slice)

	if len(match) < 2 {
		return "unknown user"
	}

	user, err := user.LookupId(match[1])
	if err != nil {
		return "unknown user"
	}

	return user.Username
}
