package metrics

import (
	"fmt"
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
	metricLock = sync.RWMutex{}
	cacheLock  = sync.RWMutex{}
)

func MetricsHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		collector := NewCollector(root)
		registry.MustRegister(collector)
		gatherers := prometheus.Gatherers{registry}
		gatherers = append(gatherers, prometheus.DefaultGatherer)
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
	CreateMetric(cgroup string, pids pidSet) *Metric
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
			if metric != nil {
				metricLock.Lock()
				metrics = append(metrics, *metric)
				metricLock.Unlock()
			}
		}(group, pids)
	}

	wg.Wait()

	for group := range groupCache {
		if _, found := groupPIDs[group]; !found {
			delete(groupPIDs, group)
		}
	}

	return metrics
}

type Process struct {
	cpu    float64
	memory uint64
	count  uint64
}

type PIDCacheEntry struct {
	cpu    float64
	memory uint64
}

type PIDCache map[uint64]PIDCacheEntry

type CommandCacheEntry struct {
	inactiveCPU float64
	inactiveMem uint64
	activePIDs  PIDCache
}

type CommandCache map[string]CommandCacheEntry

var groupCache = make(map[string]CommandCache)

// Given a set of PIDs, aggregate process count, memory, and
// CPU usage on the process name associated with the PIDs.
// Returns a map of process names -> aggregate usage.
func ProcInfo(pids map[uint64]bool, cgroup string) map[string]Process {
	processes := make(map[string]Process)

	fs, err := procfs.NewDefaultFS()
	if err != nil {
		log.Printf("could not mount procfs: %s\n", err)
		return processes
	}

	cacheLock.Lock()
	commandCache, found := groupCache[cgroup]
	cacheLock.Unlock()

	if !found {
		commandCache = make(CommandCache)
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

		pce := PIDCacheEntry{cpu: stat.CPUTime(), memory: uint64(stat.ResidentMemory())}

		commandCacheEntry, found := commandCache[comm]
		if !found {
			commandCacheEntry = CommandCacheEntry{inactiveCPU: 0, inactiveMem: 0, activePIDs: make(PIDCache)}
		}

		commandCacheEntry.activePIDs[pid] = pce
		commandCache[comm] = commandCacheEntry
	}

	for command, commandEntry := range commandCache {
		var cpu float64
		var mem uint64
		for pid, pidEntry := range commandEntry.activePIDs {
			if _, found := pids[pid]; !found {
				commandEntry.inactiveCPU += pidEntry.cpu
				commandEntry.inactiveMem += pidEntry.memory
				delete(commandEntry.activePIDs, pid)
			} else {
				cpu += pidEntry.cpu
				mem += pidEntry.memory
			}
		}
		p := Process{cpu: cpu + commandEntry.inactiveCPU, memory: mem + commandEntry.inactiveMem, count: uint64(len(commandEntry.activePIDs))}
		processes[command] = p
		commandCache[command] = commandEntry
	}

	cacheLock.Lock()
	groupCache[cgroup] = commandCache
	cacheLock.Unlock()

	return processes
}

var uidRe = regexp.MustCompile(`user-(\d+)\.slice`)

// Looks up the username associated with a user slice cgroup.
// Slice of the form 'user-1000.slice' or '/user.slice/user-1234.slice'
// Must be compiled with CGO_ENABLED if used over NFS.
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
