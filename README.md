# cgroup-warden

[![License: GPL v2](https://img.shields.io/badge/License-GPL_v2-blue.svg)](https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html)
[![Go Reference](https://pkg.go.dev/badge/github.com/chpc-uofu/cgroup-warden.svg)](https://pkg.go.dev/github.com/chpc-uofu/cgroup-warden)


cgroup-warden is a daemon that provides a way to set resource limits on those cgroups. Created to support CHPC's [Arbiter](https://github.com/chpc-uofu/arbiter).

## Installation


### Releases
Pre-built binaries can be found on the [Releases](https://github.com/chpc-uofu/cgroup-warden/releases) page.

### Build from source
The binary can also be built from source.
```
git clone https://github.com/chpc-uofu/cgroup-warden.git
cd cgroup-warden
go build .
```

If non-local user accounts are used, then the binary must be built with `CGO_ENABLED=1` for proper username resolution.

## Configure

The following flags are passed as environment variables  

`CGROUP_WARDEN_LISTEN_ADDRESS` : Address for the service to listen on. Defaults to `:2112`.  
`CGROUP_WARDEN_ROOT_CGROUP` : Monitor all cgroups underneath this one. Defaults to `/user.slice`.  
`CGROUP_WARDEN_INSECURE_MODE` : Whether to run without bearer token authentication and TLS. Defaults to `false`.  
`CGROUP_WARDEN_CERTIFICATE` : Path to TLS certificate. Required if running in secure mode.  
`CGROUP_WARDEN_PRIVATE_KEY`: Path to TLS private key. Required if running in secure mode.  
`CGROUP_WARDEN_BEARER_TOKEN` : Bearer token to use for authentication. Required if running in secure mode.  
`CGROUP_WARDEN_META_METRICS` : Whether to export metrics regarding the running warden itself. Defaults to `true`.
`CGROUP_WARDEN_LOG_LEVEL` : Level at which to log messages. Choices are `debug`, `info`, `warning`, and `error`. Defaults to `info`.

When passing these to a systemd service, you can put them into an environment file:
```shell
CGROUP_WARDEN_LISTEN_ADDRESS=0.0.0.0:2112
CGROUP_WARDEN_BEARER_TOKEN=super-secret-bearer-token
...
```
Make sure this file is private.

## Running as a service
The cgroup-warden is best run as a systemd service. The service must be run as root if the cgroup-warden is to set limits.

## Running in secure mode
Because the cgroup-warden runs in a priveledged mode, it is highly recommended to run the program in secure mode. This means enabling HTTPS, and using bearer token authentication. The environment would contain:
```shell
...
CGROUP_WARDEN_BEARER_TOKEN=super-secret-bearer-token
CGROUP_wARDEN_CERTIFICATE=/path/to/certificate
CGROUP_WARDEN_PRIVATE_KEY=/path/to/key
CGROUP_WARDEN_INSECURE_MODE=false
...
```

## Contribute
Contributions are welcomed. To contribute, fork this repository on GitHub and submit a pull request with your proposed changes. Bug reports and feature requests are also appreciated, and can be made via GitHub Issues. 

