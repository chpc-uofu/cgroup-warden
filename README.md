# cgroup-warden

[![License: GPL v2](https://img.shields.io/badge/License-GPL_v2-blue.svg)](https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html)
[![Go Reference](https://pkg.go.dev/badge/github.com/chpc-uofu/cgroup-warden.svg)](https://pkg.go.dev/github.com/chpc-uofu/cgroup-warden)


cgroup-warden is a daemon that provides a way to set resource limits on those cgroups. Created to support CHPC's [Arbiter](https://github.com/chpc-uofu/arbiter), but it may also run stand-alone. 

## Aquiring

The code can be obtained via git or a release downloaded [here](https://github.com/CHPC-UofU/cgroup-warden/releases).

```shell
VERSION=0.1.8 # See https://github.com/chpc-uofu/releases/latest

INSTALL_DIR=/opt/cgroup-warden # wherever you wish to install

cd $INSTALL_DIR
curl -OL https://github.com/chpc-uofu/cgroup-warden/releases/download/v${VERSION}/cgroup-warden-linux-amd64-${VERSION}.tar.gz
tar -xzf cgroup-warden-linux-amd64-${VERSION}.tar.gz
ln -s cgroup-warden-linux-amd64-${VERSION}/cgroup-warden latest
```

Alternatively, you can clone this repo and build the project yourself. For example,
```shell
VERSION=0.1.8 # see https://github.com/chpc-uofu/releases/latest

INSTALL_DIR=/opt/cgroup-warden # wherever you wish to install

cd $INSTALL_DIR
git clone https://github.com/chpc-uofu/cgroup-warden .
git checkout v${VERSION}
go build .
ln -s cgroup-warden latest
```

To test the installation, you can simply try to run the program
```shell
/opt/cgroup-warden/latest # path to wherever you installed it to
```

## Configure

The following flags are passed as environment variables  

`CGROUP_WARDEN_LISTEN_ADDRESS` : Address for the service to listen on. Defaults to `:2112`.  
`CGROUP_WARDEN_ROOT_CGROUP` : Monitor all cgroups underneath this one. Defaults to `/user.slice`.  
`CGROUP_WARDEN_INSECURE_MODE` : Whether to run without bearer token authentication and TLS. Defaults to `false`.  
`CGROUP_WARDEN_COLLECT_PROCESS_INFO` : Whether to collect detailed process usage information. Defaults to `true`.  
`CGROUP_WARDEN_CERTIFICATE` : Path to TLS certificate. Required if running in secure mode.  
`CGROUP_WARDEN_PRIVATE_KEY`: Path to TLS private key. Required if running in secure mode.  
`CGROUP_WARDEN_BEARER_TOKEN` : Bearer token to use for authentication. Required if running in secure mode.
`CGROUP_WARDEN_META_METRICS` : Whether to export metrics regarding the running warden itself. Defaults to `true`.

When passing these to a systemd service, you can put them into an environment file like
```shell
CGROUP_WARDEN_LISTEN_ADDRESS="0.0.0.0:2112"
CGROUP_WARDEN_BEARER_TOKEN="super-secret-bearer-token"
...
```
Make sure this file is private.

## Running as a service
The cgroup-warden is best run as a systemd service. The service must be run as root if the cgroup-warden is to set limits.
There is an example service file [here](cgroup-warden.service).

## Contribute
Contributions are welcomed. To contribute, fork this repository on GitHub and submit a pull request with your proposed changes. Bug reports and feature requests are also appreciated, and can be made via GitHub Issues. 

