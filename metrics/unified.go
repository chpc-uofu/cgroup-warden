package metrics

import (
	"log"
	"strings"
	"sync"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/prometheus/procfs"
)

var lock = sync.RWMutex{}

const USPerS = 1000000 //million

func UnifiedMetrics(root string) []Metric {
	var metrics []Metric

	groupProcs := getUnifiedPids(root)
	wg := &sync.WaitGroup{}
	wg.Add(len(groupProcs))

	for group, pids := range groupProcs {
		go func(group string, procs map[uint64]bool) {
			defer wg.Done()
			stat := getUnifiedStatistics(group, pids)
			lock.Lock()
			metrics = append(metrics, stat)
			lock.Unlock()
		}(group, pids)
	}

	wg.Wait()

	return metrics
}

func getUnifiedPids(cgroup string) map[string]map[uint64]bool {

	var procs = make(map[string]map[uint64]bool)

	manager, err := cgroup2.Load(cgroup)
	if err != nil {
		log.Printf("could not load cgroup '%s': %s\n", cgroup, err.Error())
		return procs
	}

	pids, err := manager.Procs(true)
	if err != nil {
		log.Printf("could not load cgroup '%s' processes: %s\n", cgroup, err.Error())
		return procs
	}

	for _, pid := range pids {
		path, err := cgroup2.PidGroupPath(int(pid))
		if err != nil {
			continue
		}
		dirs := strings.Split(path, "/")
		group := strings.Join(dirs[0:3], "/")

		groupPids, ok := procs[group]
		if !ok {
			groupPids = make(map[uint64]bool)
		}
		groupPids[pid] = true

		procs[group] = groupPids
	}

	return procs
}

func getUnifiedStatistics(group string, pids map[uint64]bool) Metric {
	var metric Metric

	manager, err := cgroup2.Load(group)
	if err != nil {
		log.Printf("could not load cgroup '%s': %s\n", group, err)
		return metric
	}

	stat, err := manager.Stat()
	if err != nil || stat == nil {
		log.Printf("could not get stats from cgroup '%s': %s\n", group, err)
		return metric
	}

	if stat.CPU != nil {
		metric.cpuUsage = float64(stat.CPU.UsageUsec) / USPerS
	}

	if stat.Memory != nil {
		metric.memoryUsage = stat.Memory.Usage
	}

	fs, err := procfs.NewDefaultFS()
	if err != nil {
		log.Printf("could not mount procfs: %s\n", err)
	}

	processes := make(map[string]Process)

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
	metric.processes = processes
	return metric
}
