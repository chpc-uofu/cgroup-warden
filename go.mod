module github.com/jay-mckay/cgroup-warden

go 1.22

toolchain go1.22.5

require (
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/containerd/cgroups v1.1.0
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/go-kit/log v0.2.1
	github.com/godbus/dbus/v5 v5.1.0
	github.com/prometheus/client_golang v1.19.1
	github.com/prometheus/common v0.53.0
	github.com/treydock/cgroup_exporter v0.9.1
)

require (
	github.com/alecthomas/units v0.0.0-20231202071711-9a357b53e9c9 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cilium/ebpf v0.15.0 // indirect
	github.com/containerd/cgroups/v3 v3.0.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/sys v0.20.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

// supports cgroups v2
replace github.com/treydock/cgroup_exporter => github.com/treydock/cgroup_exporter v1.0.0-rc.4
