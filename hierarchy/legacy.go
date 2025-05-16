package hierarchy

import (
	"log/slog"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type Legacy struct {
	Root string
}

func (l *Legacy) SetMemorySwap(unit string, limit int64) (int64, error) {
	cgroup := path.Join(l.Root, unit)

	manager, err := cgroup1.Load(cgroup1.StaticPath(cgroup), cgroup1.WithHierarchy(subsystem))
	if err != nil {
		return -1, err
	}

	stat, err := manager.Stat(cgroup1.IgnoreNotExist)
	if err != nil || stat == nil || stat.Memory == nil {
		return  -1, err
	}
	
	newLimit := max(limit, int64(stat.Memory.TotalRSS) + 4096*1000)

	resources := &specs.LinuxResources{
		Memory: &specs.LinuxMemory{
			Swap: &newLimit,
			Limit: &newLimit,
		},
	}

	err = manager.Update(resources)
	return newLimit, err
}

func (l *Legacy) GetGroupsWithPIDs() (map[string]map[uint64]bool, error) {

	var pids = make(map[string]map[uint64]bool)

	manager, err := cgroup1.Load(cgroup1.StaticPath(l.Root), cgroup1.WithHierarchy(subsystem))
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

func (l *Legacy) CGroupInfo(cg string) (CGroupInfo, error) {
	var info CGroupInfo

	manager, err := cgroup1.Load(cgroup1.StaticPath(cg), cgroup1.WithHierarchy(subsystem))
	if err != nil {
		return info, err
	}

	stat, err := manager.Stat(cgroup1.IgnoreNotExist)
	if err != nil || stat == nil {
		return info, err
	}

	if stat.CPU != nil {
		info.CPUUsage = float64(stat.CPU.Usage.Total) / NSPerS
		info.CPUQuota = readCPUQuotaLegacy(cg)
	}

	if stat.Memory != nil {
		info.MemoryUsage = stat.Memory.TotalRSS
		info.MemoryMax = stat.Memory.Usage.Limit
	}

	username, err := lookupUsername(cg)
	if err != nil {
		return info, err
	}

	info.Username = username
	return info, nil
}

func subsystem() ([]cgroup1.Subsystem, error) {
	s := []cgroup1.Subsystem{
		cgroup1.NewCpuacct(cgroupRoot),
		cgroup1.NewMemory(cgroupRoot),
	}
	return s, nil
}

func readCPUQuotaLegacy(cg string) int64 {
	cgroupPath := path.Join("/sys/fs/cgroup/cpu", cg)
	pathQuota := path.Join(cgroupPath, "cpu.cfs_quota_us")
	pathPeriod := path.Join(cgroupPath, "cpu.cfs_period_us")

	quotaBuffer, err := os.ReadFile(pathQuota)
	if err != nil {
		slog.Error("unable to read cpu quota", "err", err)
		return 0
	}

	quota, err := strconv.ParseInt(strings.TrimSpace(string(quotaBuffer)), 10, 64)
	if err != nil {
		slog.Error("unable to read cpu quota", "err", err)
		return 0
	}

	periodBuffer, err := os.ReadFile(pathPeriod)
	if err != nil {
		slog.Error("unable to read cpu quota", "err", err)
		return 0
	}

	period, err := strconv.ParseUint(strings.TrimSpace(string(periodBuffer)), 10, 64)
	if err != nil {
		slog.Error("unable to read cpu quota", "err", err)
		return 0
	}

	if period < 0 {
		slog.Error("unable to parse cpu.cfs_period_us", "err", "period is less than 0")
		return 0

	}
	cpuQuotaPerSecUSec := uint64(math.MaxUint64)
	if quota > 0 {
		cpuQuotaPerSecUSec = uint64(quota*USPerS) / period
	}

	return int64(cpuQuotaPerSecUSec)
}
