# KubeVirt

## Hacking

### Setup

First make sure you have [govendor](https://github.com/kardianos/govendor)
installed.

To install govendor in your `$GOPATH/bin` simply run

```bash
go get -u github.com/kardianos/govendor
```

If you don't have the `$GOPATH/bin` folder on your path, do

```bash
export PATH=$PATH:$GOPATH/bin
```

### Building

To build the whole project, type

```bash
make
```

To build all docker images type

```bash
make docker
```

It is also possible to target only specific modules. For instance to build only the `virt-launcher`, type

```bash
make build WAHT=virt-launcher
```

### Testing

Type

```bash
make test
```

to run all tests.
