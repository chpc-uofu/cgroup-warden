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

type CacheEntry struct {
	inactiveCPU float64
	activePIDS  map[float64]bool
}

type ProcessCache struct {
	cache map[string]CacheEntry
	lock  sync.Mutex
}

func NewProcessCache() *ProcessCache {
	return &ProcessCache{
		cache: make(map[string]CacheEntry),
		lock:  sync.Mutex{},
	}
}

func ProcessInfo(pids map[uint64]bool) map[string]Process {
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

	return processes
}
