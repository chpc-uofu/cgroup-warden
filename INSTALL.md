# Installation Guide

## Install
cgroup-warden can be installed in a few ways. The easiest, provided go is installed, is
```
go install github.com/chpc-uofu/cgroup-warden
```
This will install to whatever `$GOBIN` evaluates to. 


Alternatively, one can download the latest release [here](https://github.com/chpc-uofu/cgroup-warden/releases/latest).
You can then extract the download, which will contain the executable.
```bash
VERSION=0.0.5 # version desired
PREFIX="/opt/" # wherever you want to install to
mkdir $PREFIX/
cd $PREFIX/cgroup-warden
wget https://github.com/chpc-uofu/cgroup-warden/releases/download/v$VERSION/cgroup-warden-$VERSION.tar.gz/
ln -s latest cgroup-warden-$VERSION/cgroup-warden
cd latest
``` 

Finally, one can clone this repo and build the binary themselves,
```
git clone https://github.com/chpc-uofu/cgroup-warden.git
cd cgroup-warden
go build .
```

## Configure

The following flags are passed as environment variables  

`CGROUP_WARDEN_LISTEN_ADDRESS` : Address for the service to listen on. Defaults to `:2112`.  
`CGROUP_WARDEN_UNIT_PATTERN` : Unit for service to match systemd units on. Defaults to `user-*.slice`.  
`CGROUP_WARDEN_INSECURE_MODE` : Whether to run without bearer token authentication and TLS. Defaults to `false`.  
`CGROUP_WARDEN_COLLECT_PROCESS_INFO` : Whether to collect detailed process usage information. Defaults to `true`.  
`CGROUP_WARDEN_CERTIFICATE` : Path to TLS certificate. Required if running in secure mode.  
`CGROUP_WARDEN_PRIVATE_KEY`: Path to TLS private key. Required if running in secure mode.  
`CGROUP_WARDEN_BEARER_TOKEN` : Bearer token to use for authentication. Required if running in secure mode.   
