package metrics

import (
	"log"
	"strings"

	"github.com/containerd/cgroups/v3/cgroup2"
)

type unified struct {
	root string
}

func (u *unified) GetGroupsWithPIDs() groupPIDMap {

	var pids = make(groupPIDMap)

	manager, err := cgroup2.Load(u.root)
	if err != nil {
		log.Printf("could not load cgroup '%s': %s\n", u.root, err.Error())
		return pids
	}

	procs, err := manager.Procs(true)
	if err != nil {
		log.Printf("could not load cgroup '%s' processes: %s\n", u.root, err.Error())
		return pids
	}

	for _, p := range procs {
		path, err := cgroup2.PidGroupPath(int(p))
		if err != nil {
			continue
		}
		dirs := strings.Split(path, "/")
		group := strings.Join(dirs[0:3], "/")

		groupPids, ok := pids[group]
		if !ok {
			groupPids = make(pidSet)
		}
		groupPids[p] = true

		pids[group] = groupPids
	}

	return pids
}

func (u *unified) CreateMetric(group string, pids pidSet) *Metric {
	var metric Metric

	manager, err := cgroup2.Load(group)
	if err != nil {
		log.Printf("could not load cgroup '%s': %s\n", group, err)
		return nil
	}

	stat, err := manager.Stat()
	if err != nil || stat == nil {
		log.Printf("could not get stats from cgroup '%s': %s\n", group, err)
		return nil
	}

	if stat.CPU != nil {
		metric.cpuUsage = float64(stat.CPU.UsageUsec) / USPerS
	}

	if stat.Memory != nil {
		metric.memoryUsage = stat.Memory.Usage
	}

	metric.processes = ProcInfo(pids)

	metric.cgroup = group

	username, err := lookupUsername(group)
	if err != nil {
		log.Println(err)
		return &metric
	}
	metric.username = username

	return &metric
}
