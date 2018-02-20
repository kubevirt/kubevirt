# libmacouflage

libmacouflage is a Golang library for setting the MAC address for network
interfaces on Linux-based operating systems.

The reason for implementing these functions as a library is to make it easy
to develop command-line and GUI applications for managing MAC addresses.
We at Subgraph will use this library in our own macouflage command-line/GUI
application as well as in other places. But others are free to use the library
as they wish.

It supports different modes of operation such as those already available in
[GNU Mac Changer](https://github.com/alobbs/macchanger). It also supports a
"popular" mode, similar to what is available in the
[macchiato](https://github.com/EtiennePerot/macchiato) project.

The data for the "popular" mode is in fact derived from the macchiato project.
libmacouflage uses a JSON database that is generated based off the data in
macchiato. Scripts to generate the database are hosted in the
[ouiner](https://github.com/mckinney-subgraph/ouiner) project.

The ouiner database ships with libmacouflage so that is available to client
programs based on libmacouflage. It is embedded in the libmacouflage object
binary using go-bindata.

## Testing

libmacouflage includes unit tests. Most functions will pass the existing tests
without any special caveats. However, setting a MAC address on an interface has
a few caveats:

1. Only the root user can set a MAC address
2. The target network interface must be down to set the MAC address
3. Setting a completely random MAC address fails sometimes on certain ranges,
which is why it is better to enable the "burned-in address" setting to avoid
these ranges, however, specific tests that do not set "burned-in address" may
fail as a result

### How to test with a specific network interface

Take down the network interface (eth0 is used as an example):
```
$ sudo ip link set eth0 down
```

Then run the tests (interface is specified via the TEST_INTERFACE environment
variable):
```
$ sudo TEST_INTERFACE=eth0 GOPATH=<your_gopath> go test
```

NOTE: GOPATH should point to the GOPATH for your regular user account or
dependencies such as the testify test-suite will not be found

### How to test with "any" network interface (test suite just chooses the first found)

For each network interface:
```
$ sudo ip link set <interface> down
```

Then run the tests:
```
$ sudo GOPATH=<your_gopath> go test
```


