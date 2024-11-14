module github.com/chpc-uofu/cgroup-warden

go 1.22

toolchain go1.22.5

require (
	github.com/containerd/cgroups/v3 v3.0.3
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/godbus/dbus/v5 v5.1.0
	github.com/prometheus/client_golang v1.20.3
	github.com/prometheus/procfs v0.15.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cilium/ebpf v0.11.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/exp v0.0.0-20230224173230-c95f2b4c22f2 // indirect
	golang.org/x/sys v0.22.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/containerd/cgroups/v3 => github.com/jay-mckay/cgroups/v3 v3.0.3
