package metrics

import (
	"sync"

	"github.com/prometheus/procfs"
)

type process struct {
	cpuSeconds  float64
	memoryBytes uint64
	command     string
	current     bool
}

type ProcessAggregation struct {
	cpuSecondsTotal  float64
	memoryBytesTotal uint64
	count            uint64
}

type processCache struct {
	data  map[string]*entry
	mutex sync.Mutex
}

func newProcessCache() *processCache {
	return &processCache{
		data:  make(map[string]*entry),
		mutex: sync.Mutex{},
	}
}

func (pc *processCache) get(cgroup string) *entry {
	defer pc.mutex.Unlock()
	pc.mutex.Lock()
	value, ok := pc.data[cgroup]
	if !ok {
		value = newEntry()
	}
	return value
}

func (pc *processCache) put(cgroup string, processes *entry) {
	defer pc.mutex.Unlock()
	pc.mutex.Lock()
	pc.data[cgroup] = processes
}

func (pc *processCache) clean(active map[string]bool) {
	defer pc.mutex.Unlock()
	pc.mutex.Lock()
	for cgroup := range pc.data {
		if _, ok := active[cgroup]; !ok {
			delete(pc.data, cgroup)
		}
	}
}

func CleanProcessCache(active map[string]bool) {
	cache.clean(active)
}

type entry struct {
	data  map[uint64]process
	mutex sync.Mutex
}

func newEntry() *entry {
	return &entry{
		data:  make(map[uint64]process),
		mutex: sync.Mutex{},
	}
}

func (e *entry) update(processes map[uint64]process) {
	defer e.mutex.Unlock()
	e.mutex.Lock()
	for pid, process := range processes {
		e.data[pid] = process
	}
}

func (e *entry) clean(active map[string]bool) {
	defer e.mutex.Unlock()
	e.mutex.Lock()
	for pid, process := range e.data {
		if _, ok := active[process.command]; !ok {
			delete(e.data, pid)
		}
	}
}

func (e *entry) aggregate() map[string]ProcessAggregation {
	results := make(map[string]ProcessAggregation)
	defer e.mutex.Unlock()
	e.mutex.Lock()
	for pid, process := range e.data {
		r := results[process.command]
		r.cpuSecondsTotal += process.cpuSeconds
		if process.current {
			r.memoryBytesTotal += process.memoryBytes
			r.count += 1
		}
		results[process.command] = r
		process.current = false
		e.data[pid] = process
	}

	return results
}

var cache = newProcessCache()

func ProcessInfo(cg string, pids map[uint64]bool) (map[string]ProcessAggregation, error) {
	fs, err := procfs.NewDefaultFS()
	if err != nil {
		return nil, err
	}

	active := make(map[string]bool)
	processes := make(map[uint64]process)

	for pid := range pids {

		proc, err := fs.Proc(int(pid))
		if err != nil {
			continue
		}

		command, err := proc.Comm()
		if err != nil {
			continue
		}

		stat, err := proc.Stat()
		if err != nil {
			continue
		}

		process := process{
			cpuSeconds:  stat.CPUTime(),
			memoryBytes: uint64(stat.ResidentMemory()),
			command:     command,
			current:     true,
		}

		active[command] = true
		processes[pid] = process
	}

	e := cache.get(cg)
	e.update(processes)
	e.clean(active)
	results := e.aggregate()
	cache.put(cg, e)
	return results, nil
}
