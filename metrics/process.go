package metrics

import (
	"log"
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
	data  map[string]map[uint64]process
	mutex sync.Mutex
}

func newProcessCache() *processCache {
	return &processCache{
		data:  make(map[string]map[uint64]process),
		mutex: sync.Mutex{},
	}
}

func (pc *processCache) get(cgroup string) map[uint64]process {
	defer pc.mutex.Unlock()
	pc.mutex.Lock()
	value, ok := pc.data[cgroup]
	if !ok {
		value = make(map[uint64]process)
	}
	return value
}

func (pc *processCache) put(cgroup string, processes map[uint64]process) {
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

var cache = newProcessCache()

func ProcessInfo(cg string, pids map[uint64]bool) map[string]ProcessAggregation {
	results := make(map[string]ProcessAggregation)

	fs, err := procfs.NewDefaultFS()
	if err != nil {
		log.Printf("could not mount procfs: %s\n", err)
		return results
	}

	activeCommands := make(map[string]bool)

	cacheEntry := cache.get(cg)

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

		activeCommands[command] = true

		cacheEntry[pid] = process

	}

	for pid, process := range cacheEntry {

		if _, ok := activeCommands[process.command]; !ok {
			delete(cacheEntry, pid)
			continue
		}

		agg := results[process.command]
		agg.cpuSecondsTotal += process.cpuSeconds
		if process.current {
			agg.memoryBytesTotal += process.memoryBytes
			agg.count += 1
		}
		results[process.command] = agg

		process.current = false
		cacheEntry[pid] = process
	}

	cache.put(cg, cacheEntry)

	return results
}
