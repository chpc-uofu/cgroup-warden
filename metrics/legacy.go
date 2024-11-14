package metrics

import "log"

func LegacyStats(root string) []Metric {
	var stats []Metric
	return stats
}

func legacyPids(cgroup string) []uint64 {
	var pids []uint64
	log.Println(cgroup)
	return pids
}
