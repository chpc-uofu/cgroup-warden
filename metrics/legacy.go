package metrics

import (
	"log"
	"strings"

	"github.com/containerd/cgroups/v3/cgroup1"
)

type legacy struct {
	root string
}

func (l *legacy) GetGroupsWithPIDs() groupPIDMap {

	var pids = make(groupPIDMap)

	manager, err := cgroup1.Load(cgroup1.StaticPath(l.root), cgroup1.WithHierarchy(subsystem))
	if err != nil {
		log.Printf("could not load cgroup '%s': %s\n", l.root, err.Error())
		return pids
	}

	procs, err := manager.Processes(cgroup1.Cpuacct, true)
	if err != nil {
		log.Printf("could not load cgroup '%s' processes: %s\n", l.root, err.Error())
		return pids
	}

	for _, p := range procs {
		dirs := strings.Split(p.Path, "/")
		group := "/" + strings.Join(dirs[5:7], "/")

		groupPids, ok := pids[group]
		if !ok {
			groupPids = make(pidSet)
		}
		groupPids[uint64(p.Pid)] = true

		pids[group] = groupPids
	}

	return pids
}

func (l *legacy) CreateMetric(group string, pids pidSet) Metric {
	var metric Metric

	metric.cgroup = group

	manager, err := cgroup1.Load(cgroup1.StaticPath(group), cgroup1.WithHierarchy(subsystem))
	if err != nil {
		log.Printf("could not load cgroup '%s': %s\n", group, err)
		return metric
	}

	stat, err := manager.Stat(cgroup1.IgnoreNotExist)
	if err != nil || stat == nil {
		log.Printf("could not get stats from cgroup '%s': %s\n", group, err)
		return metric
	}

	if stat.CPU != nil {
		metric.cpuUsage = float64(stat.CPU.Usage.Total) / NSPerS
	}

	if stat.Memory != nil {
		metric.memoryUsage = stat.Memory.TotalRSS
	}

	metric.processes = ProcInfo(pids)

	return metric
}

func subsystem() ([]cgroup1.Subsystem, error) {
	s := []cgroup1.Subsystem{
		cgroup1.NewCpuacct(cgroupRoot),
		cgroup1.NewMemory(cgroupRoot),
	}
	return s, nil
}
