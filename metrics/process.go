package metrics

import (
	"log"
	"sync"

	"github.com/prometheus/procfs"
)

type ProcInfo struct {
	cpu    float64
	memory uint64
	count  uint64
}

// ProcessInfo provides the aggregates the count, CPU, and memory usage of a
// process running in a cgroup. cg is the name of the cgroup, and pids is a
// set of PIDs running in that cgroup. This function will look up each pid
// in /procfs, and combine the results based on the process name. Returns
// a map of process name to a ProcInfo.
func ProcessInfo(cg string, pids map[uint64]bool) map[string]ProcInfo {
	processes := make(map[string]ProcInfo)

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
			p = ProcInfo{}
		}

		p.cpu += stat.CPUTime()
		p.memory += uint64(stat.ResidentMemory())
		p.count += 1

		processes[comm] = p
	}

	reconcileCPU(cg, processes)

	return processes
}

// An entry in the processCache.
type cacheEntry struct {
	lastSeen map[string]float64
}

// creats a new cacheEntry.
func newCacheEntry() cacheEntry {
	return cacheEntry{
		lastSeen: make(map[string]float64),
	}
}

// Cache used to store process CPU information. Mutex used for conurrent safety.
type processCache struct {
	cache map[string]cacheEntry
	lock  *sync.Mutex
}

// Retrieves the cacheEntry mapped to the key cg from this processCache.
// If no element exists, return the default cacheEntry. Concurrent safe.
func (pc processCache) Get(cg string) cacheEntry {
	defer pc.lock.Unlock()
	pc.lock.Lock()
	entry, ok := pc.cache[cg]
	if !ok {
		entry = newCacheEntry()
	}
	return entry
}

// Updates this processCache. Concurrent safe.
func (pc processCache) Put(cg string, entry cacheEntry) {
	defer pc.lock.Unlock()
	pc.lock.Lock()
	pc.cache[cg] = entry
}

// Cleans up this processCache, removing all entries not contained
// in currentGroupNames.
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

// Creates a new processCache struct.
func newProcessCache() processCache {
	return processCache{
		cache: make(map[string]cacheEntry),
		lock:  &sync.Mutex{},
	}
}

// The global cache used to ensure the CPU process counter performs as one.
var cache = newProcessCache()

// Tidy cleans up the global process cache by removing all data
// that refers to inactive cgroups. currentCGNames is a set of
// cgroup names that are currently active.
func Tidy(currentCGNames map[string]bool) {
	cache.tidy(currentCGNames)
}

// reconcileCPU ensures that the aggregate CPU usage per process
// is not lower than the last collected aggregate. This can happen
// if multiple instances of the same process are running and one
// stops. The process CPU metric is a counter, so if this happens
// we retain the last value.
//
// cg is the name of the cgroup, used for indexing into the cache,
// and processes is a map of process names to a ProcIfno. If the CPU
// value needs to be updated, this change is reflected in process.
//
// A global cache is maintained, and must
// be cleaned up with Tidy.
func reconcileCPU(cg string, processes map[string]ProcInfo) {
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

	cache.Put(cg, entry)
}
