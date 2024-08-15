# cgroup-warden

[![License: GPL v2](https://img.shields.io/badge/License-GPL_v2-blue.svg)](https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html)


cgroup-warden is a daemon that provides a way to set resource limits on those cgroups. Created to support CHPC's [Arbiter](https://github.com/chpc-uofu/arbiter), but it may also run stand-alone. 


## Control

cgroup-warden allows requests to be made to modify resource limits per cgroup through the `/control` endpoint. The resources are limited by modifying [systemd](https://systemd.io) properties. An example JSON request is below. 

```
{
        property: "CPUQuotaPerSecUSec",
        value: "1000000000"
        runtime: "false"
}
```

## Install
Download the [latest release](https://github.com/CHPC-UofU/releases)

## Contribute
Contributions are welcomed. To contribute, fork this repository on GitHub and submit a pull request with your proposed changes. Bug reports and feature requests are also appreciated, and can be made via GitHub Issues. 

