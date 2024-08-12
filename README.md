# cgroup-warden

[![License: GPL v2](https://img.shields.io/badge/License-GPL_v2-blue.svg)](https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html)


cgroup-warden is a daemon that monitors [cgroups](https://man7.org/linux/man-pages/man7/cgroups.7.html), and provides a way to set resource limits on those cgroups. Created to support CHPC's [Arbiter](https://github.com/chpc-uofu/arbiter), but it may also run stand-alone. 

## Metrics

cgroup-warden exposes cgroup metrics in the [OpenMetrics](https://openmetrics.io/) format through the `/metrics` endpoint. These metrics are
provided by the [cgroup_exporter](https://github.com/treydock/cgroup_exporter), with cgroup-warden just wrapping the collector. 

## Control

cgroup-warden allows requests to be made to modify resource limits per cgroup through the `/control` endpoint. The resources are limited by modifying [systemd](https://systemd.io) properties. An example JSON request is below. 

```json
{
        property: "CPUQuotaPerSecUSec",
        value: "1000000000"
        runtime: "false"
}
```

## Install
Download the [latest release](https://github.com/CHPC-UofU/releases)

## Build
```bash
go get github.com/CHPC-UofU/cgroup-warden
```

## Contribute
Contributions are welcomed. To contribute, fork this repository on GitHub and submit a pull request with your proposed changes. Bug reports and feature requests are also appreciated, and can be made via GitHub Issues. 

