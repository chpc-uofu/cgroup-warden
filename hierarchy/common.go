package hierarchy

import (
	"fmt"
	"os/user"
	"regexp"

	"github.com/containerd/cgroups/v3"
)

const (
	USPerS               = 1000000    // million
	NSPerS               = 1000000000 // billion
	MaxCGroupMemoryLimit = 9223372036854771712
	cgroupRoot           = "/sys/fs/cgroup"
)

type Hierarchy interface {
	GetGroupsWithPIDs() (map[string]map[uint64]bool, error)
	CGroupInfo(cg string) (CGroupInfo, error)
	SetMemorySwap(unit string, limit int64) (int64, error)
}

func NewHierarchy(root string) Hierarchy {

	mode := cgroups.Mode()

	var h Hierarchy

	if mode == cgroups.Unified {
		h = &Unified{Root: root}
	} else {
		h = &Legacy{Root: root}
	}

	return h
}

type CGroupInfo struct {
	Username    string
	MemoryUsage uint64
	CPUUsage    float64
	MemoryMax   uint64
	CPUQuota    int64
}

var uidRe = regexp.MustCompile(`user-(\d+)\.slice`)

// lookupUsername looks up a username given the systemd user slice name.
// If compiled with CGO, this function will call the C function getpwuid_r
// from the standard C library; This is necessary when user identities are
// provided by services like sss and ldap.
func lookupUsername(slice string) (string, error) {
	match := uidRe.FindStringSubmatch(slice)

	if len(match) < 2 {
		return "", fmt.Errorf("cannot determine uid from '%s'", slice)
	}

	user, err := user.LookupId(match[1])
	if err != nil {
		return "", fmt.Errorf("unable to lookup user with id '%s'", match[1])
	}

	return user.Username, nil
}
