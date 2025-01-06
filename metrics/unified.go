package metrics

import (
	"fmt"
	"log"
	"strings"

	"github.com/containerd/cgroups/v3/cgroup2"
)

type unified struct {
	root string
}

func (u *unified) GetGroupsWithPIDs() map[string]map[uint64]bool {

	var pids = make(map[string]map[uint64]bool)

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
			groupPids = make(map[uint64]bool)
		}
		groupPids[p] = true

		pids[group] = groupPids
	}

	return pids
}

func (u *unified) CGroupInfo(cg string) (cgroupInfo, error) {
	var info cgroupInfo

	manager, err := cgroup2.Load(cg)
	if err != nil {
		return info, fmt.Errorf("could not load cgroup '%s': %s", cg, err)
	}

	stat, err := manager.Stat()
	if err != nil {
		return info, fmt.Errorf("could not load stats from cgroup '%s': %s", cg, err)
	}

	if stat.CPU != nil {
		info.cpuUsage = float64(stat.CPU.UsageUsec) / USPerS
	}

	if stat.Memory != nil {
		info.memoryUsage = stat.Memory.Usage
	}

	username, err := lookupUsername(cg)
	if err != nil {
		return info, fmt.Errorf("could not lookup username for cgroup '%s': %s", cg, err)
	}

	info.username = username
	return info, nil
}
