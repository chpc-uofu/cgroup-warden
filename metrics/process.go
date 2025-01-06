package metrics

import (
	"log"
	"sync"

	"github.com/prometheus/procfs"
)

type Process struct {
	cpu    float64
	memory uint64
	count  uint64
}

type cacheEntry struct {
	lastSeen map[string]float64
}

func newCacheEntry() cacheEntry {
	return cacheEntry{
		lastSeen: make(map[string]float64),
	}
}

type processCache struct {
	cache map[string]cacheEntry
	lock  *sync.Mutex
}

func (pc processCache) Get(cg string) cacheEntry {
	defer pc.lock.Unlock()
	pc.lock.Lock()
	entry, ok := pc.cache[cg]
	if !ok {
		entry = newCacheEntry()
	}
	return entry
}

func (pc processCache) tidy(currentGroupNames map[string]bool) {
	defer pc.lock.Unlock()
	pc.lock.Lock()

	for name := range pc.cache {
		_, found := currentGroupNames[name]
		if !found {
			delete(pc.cache, name)
		}
	}
}

func newProcessCache() processCache {
	return processCache{
		cache: make(map[string]cacheEntry),
		lock:  &sync.Mutex{},
	}
}

func ProcessInfo(cg string, pids map[uint64]bool) map[string]Process {
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

		p, ok := processes[comm]
		if !ok {
			p = Process{}
		}

		p.cpu += stat.CPUTime()
		p.memory += uint64(stat.ResidentMemory())
		p.count += 1

		processes[comm] = p
	}

	reconcileCPU(cg, processes)

	return processes
}

var cache = newProcessCache()

func Tidy(currentCGNames map[string]bool) {
	cache.tidy(currentCGNames)
}

func reconcileCPU(cg string, processes map[string]Process) {
	entry := cache.Get(cg)
	for name, process := range processes {

		last := entry.lastSeen[name]

		if process.cpu < last {
			process.cpu = last
			processes[name] = process
		}

		entry.lastSeen[name] = process.cpu
	}

	for name := range entry.lastSeen {
		_, found := processes[name]
		if !found {
			delete(entry.lastSeen, name)
		}
	}
}
