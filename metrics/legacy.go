package metrics

import (
	"strings"

	"github.com/containerd/cgroups/v3/cgroup1"
)

type legacy struct {
	root string
}

func (l *legacy) GetGroupsWithPIDs() (map[string]map[uint64]bool, error) {

	var pids = make(map[string]map[uint64]bool)

	manager, err := cgroup1.Load(cgroup1.StaticPath(l.root), cgroup1.WithHierarchy(subsystem))
	if err != nil {
		return nil, err
	}

	procs, err := manager.Processes(cgroup1.Cpuacct, true)
	if err != nil {
		return nil, err
	}

	for _, p := range procs {
		dirs := strings.Split(p.Path, "/")
		group := "/" + strings.Join(dirs[5:7], "/")

		groupPids, ok := pids[group]
		if !ok {
			groupPids = make(map[uint64]bool)
		}
		groupPids[uint64(p.Pid)] = true

		pids[group] = groupPids
	}

	return pids, nil
}

func (l *legacy) CGroupInfo(cg string) (cgroupInfo, error) {
	var info cgroupInfo

	manager, err := cgroup1.Load(cgroup1.StaticPath(cg), cgroup1.WithHierarchy(subsystem))
	if err != nil {
		return info, err
	}

	stat, err := manager.Stat(cgroup1.IgnoreNotExist)
	if err != nil || stat == nil {
		return info, err
	}

	if stat.CPU != nil {
		info.cpuUsage = float64(stat.CPU.Usage.Total) / NSPerS
	}

	if stat.Memory != nil {
		info.memoryRSS = stat.Memory.TotalRSS
	}

	username, err := lookupUsername(cg)
	if err != nil {
		return info, err
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
