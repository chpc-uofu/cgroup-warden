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
VERSION=0.0.5
PREFIX="/opt/"
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
