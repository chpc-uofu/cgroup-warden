package metrics

import (
	"log/slog"
	"strings"

	"github.com/containerd/cgroups/v3/cgroup2"
)

type unified struct {
	root string
}

func (u *unified) GetGroupsWithPIDs() (map[string]map[uint64]bool, error) {

	var pids = make(map[string]map[uint64]bool)

	manager, err := cgroup2.Load(u.root)
	if err != nil {
		return nil, err
	}

	procs, err := manager.Procs(true)
	if err != nil {
		return nil, err
	}

	for _, p := range procs {
		path, err := cgroup2.PidGroupPath(int(p))
		if err != nil {
			slog.Info("could not determine cgroup of pid", "pid", p, "err", err)
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

	return pids, nil
}

func (u *unified) CGroupInfo(cg string) (cgroupInfo, error) {
	var info cgroupInfo

	manager, err := cgroup2.Load(cg)
	if err != nil {
		return info, err
	}

	stat, err := manager.Stat()
	if err != nil {
		return info, err
	}

	if stat.CPU != nil {
		info.cpuUsage = float64(stat.CPU.UsageUsec) / USPerS
	}

	if stat.Memory != nil {
		info.memoryRSS = stat.Memory.Usage
	}

	username, err := lookupUsername(cg)
	if err != nil {
		return info, err
	}

	info.username = username
	return info, nil
}
