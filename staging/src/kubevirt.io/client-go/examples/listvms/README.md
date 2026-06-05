# listvms

List all VMs and VMIs objects in KubeVirt

## How to build
```
$ export GO111MODULE=on
$ go build
```
## How to test this example
first you need to export an environment variable `KUBECONFIG` pointing to your kubernetes config, where KubeVirt is installed.

```
$ export KUBECONFIG=/home/<user>/.kubeconfig
$ ./listvms
Type                       Name          Namespace     Status
VirtualMachine             vm-cirros     default       true
VirtualMachineInstance     vm-cirros     default       Running

```