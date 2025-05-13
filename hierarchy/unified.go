package hierarchy

import (
	"errors"
	"log/slog"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/containerd/cgroups/v3/cgroup2"
)

type Unified struct {
	Root string
}

func (u *Unified) GetGroupsWithPIDs() (map[string]map[uint64]bool, error) {

	var pids = make(map[string]map[uint64]bool)

	manager, err := cgroup2.Load(u.Root)
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

func (u *Unified) CGroupInfo(cg string) (CGroupInfo, error) {
	var info CGroupInfo

	manager, err := cgroup2.Load(cg)
	if err != nil {
		return info, err
	}

	stat, err := manager.Stat()
	if err != nil {
		return info, err
	}

	if stat.CPU != nil {
		info.CPUUsage = float64(stat.CPU.UsageUsec) / USPerS
		info.CPUQuota = readCPUQuotaUnified(cg)
	}

	if stat.Memory != nil {
		info.MemoryUsage = stat.Memory.Usage
		info.MemoryMax = stat.Memory.UsageLimit
	}

	username, err := lookupUsername(cg)
	if err != nil {
		return info, err
	}

	info.Username = username
	return info, nil
}

func (u *Unified) SetMemorySwap(unit string, limit int64) (int64, error) {
	return -1, errors.New("unsupported operation")
}

func readCPUQuotaUnified(cg string) int64 {
	cgroupPath := path.Join("/sys/fs/cgroup", cg)
	p := path.Join(cgroupPath, "cpu.max")
	buf, err := os.ReadFile(p)
	if err != nil {
		slog.Error("unable to read cpu quota", "err", err)
		return 0
	}
	values := strings.Split(strings.TrimSpace(string(buf)), " ")

	var quota int64
	var period uint64

	if values[0] == "max" {
		return -1
	}

	quota, err = strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		slog.Error("unable to parse cpu.max quota", "err", err)
		return 0
	}

	period, err = strconv.ParseUint(values[1], 10, 64)
	if err != nil {
		slog.Error("unable to parse cpu.max period", "err", err)
		return 0
	}

	if period < 0 {
		slog.Error("unable to parse cpu.max", "err", "period is less than 0")
		return 0
	}

	cpuQuotaPerSecUSec := uint64(math.MaxUint64)
	if quota > 0 {
		cpuQuotaPerSecUSec = uint64(quota*USPerS) / period
	}
	return int64(cpuQuotaPerSecUSec)
}
