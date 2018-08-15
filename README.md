# go-hello-serf

## Usage

A simple program to show how to integrate
[serf](https://www.serf.io).

### Invocation

```console
go-hello-serf
```

## Demonstrate

In terminal #1

```console
export GOPATH="${HOME}/go"
export PATH="${PATH}:${GOPATH}/bin:/usr/local/go/bin"
export PROJECT_DIR="${GOPATH}/src/github.com/docktermj"
export REPOSITORY_DIR="${PROJECT_DIR}/go-hello-serf"

cd ${REPOSITORY_DIR}
make dependencies

cd ${REPOSITORY_DIR}
make build-demo
```

Verify `docker` network is 172.17.0.1.
If gateway is not 172.17.0.1, the following `docker` statements need to be modified before being run.

```console
$ docker network inspect bridge | grep Gateway
                   "Gateway": "172.17.0.1"
```

In terminal #2

```console
docker run -e ADVERTISE_ADDR=172.17.0.2 -p 8080:8080 local/go-hello-serf-demo
```

In terminal #3

```console
docker run -e ADVERTISE_ADDR=172.17.0.3 -e CLUSTER_ADDR=172.17.0.2 -p 8081:8080 local/go-hello-serf-demo
```

In terminal #4

```console
docker run -e ADVERTISE_ADDR=172.17.0.4 -e CLUSTER_ADDR=172.17.0.3 -p 8082:8080 local/go-hello-serf-demo
```

In terminal #5

```console
serf agent -join 172.17.0.2
```

In terminal #6, try these commands

```console
serf members
serf query time
serf event bob

curl -v http://localhost:8080/get | jq
curl -v http://localhost:8082/set/7
curl -v http://localhost:8080/get | jq
```

## Development

### Dependencies

#### Set environment variables

```console
export GOPATH="${HOME}/go"
export PATH="${PATH}:${GOPATH}/bin:/usr/local/go/bin"
export PROJECT_DIR="${GOPATH}/src/github.com/docktermj"
export REPOSITORY_DIR="${PROJECT_DIR}/go-hello-serf"
```

#### Download project

```console
mkdir -p ${PROJECT_DIR}
cd ${PROJECT_DIR}
git clone git@github.com:docktermj/go-hello-serf.git
```

#### Download dependencies

```console
cd ${REPOSITORY_DIR}
make dependencies
```

### Build

#### Local build

```console
cd ${REPOSITORY_DIR}
make
```

The results will be in the `${GOPATH}/bin` directory.

#### Docker build

Create `rpm` and `deb` installation packages.

```console
cd ${REPOSITORY_DIR}
make build
```

The results will be in the `${REPOSITORY_DIR}/target` directory.

### Test

```console
cd ${REPOSITORY_DIR}
make test-local
```

### Install

#### RPM-based

Example distributions: openSUSE, Fedora, CentOS, Mandrake

##### RPM Install

Example:

```console
sudo rpm -ivh go-hello-serf-M.m.P-I.x86_64.rpm
```

##### RPM Update

Example:

```console
sudo rpm -Uvh go-hello-serf-M.m.P-I.x86_64.rpm
```

#### Debian

Example distributions: Ubuntu

##### Debian Install / Update

Example:

```console
sudo dpkg -i go-hello-serf_M.m.P-I_amd64.deb
```

### Cleanup

```console
cd ${REPOSITORY_DIR}
make clean
```

### References

1. [Building a simple, distributed one-value database with Hashicorp Serf](https://jacobmartins.com/2017/01/29/practical-golang-building-a-simple-distributed-one-value-database-with-hashicorp-serf/)
