module github.com/chpc-uofu/cgroup-warden

go 1.23.0

require (
	github.com/containerd/cgroups/v3 v3.0.5
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/godbus/dbus/v5 v5.1.0
	github.com/prometheus/client_golang v1.20.5
	github.com/prometheus/procfs v0.15.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cilium/ebpf v0.17.1 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.61.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/exp v0.0.0-20250106191152-7588d65b2ba8 // indirect
	golang.org/x/sys v0.29.0 // indirect
	google.golang.org/protobuf v1.36.2 // indirect
)

replace github.com/containerd/cgroups/v3 => github.com/jay-mckay/cgroups/v3 v3.0.3
