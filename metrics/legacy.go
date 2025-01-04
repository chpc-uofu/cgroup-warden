package metrics

import (
	"fmt"
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

func (l *legacy) CGroupInfo(cg string) (cgroupInfo, error) {
	var info cgroupInfo

	manager, err := cgroup1.Load(cgroup1.StaticPath(cg), cgroup1.WithHierarchy(subsystem))
	if err != nil {
		return info, fmt.Errorf("could not load cgroup '%s': %s", cg, err)
	}

	stat, err := manager.Stat(cgroup1.IgnoreNotExist)
	if err != nil || stat == nil {
		return info, fmt.Errorf("could not get stats from cgroup '%s': %s", cg, err)
	}

	if stat.CPU != nil {
		info.cpuUsage = float64(stat.CPU.Usage.Total) / NSPerS
	}

	if stat.Memory != nil {
		info.memoryUsage = stat.Memory.TotalRSS
	}

	username, err := lookupUsername(cg)
	if err != nil {
		return info, fmt.Errorf("could not lookup username for cgroup '%s': %s", cg, err)
	}

	info.username = username
	return info, nil
}

func subsystem() ([]cgroup1.Subsystem, error) {
	s := []cgroup1.Subsystem{
		cgroup1.NewCpuacct(cgroupRoot),
		cgroup1.NewMemory(cgroupRoot),
	}
	return s, nil
}
