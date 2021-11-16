# Architecture

There are multiple components involved to provide SR-IOV support in KubeVirt
environment.

* [SR-IOV device plugin](https://github.com/intel/sriov-network-device-plugin)
  discovers available SR-IOV resources, advertises them to Kubernetes resource
  manager, and allocates them as needed to pods and VMIs. This component
  doesn't modify host resources, only discovers, advertises and allocates them.
  When resources are advertized, they are registered in `capacity` section of
  each node, that is then used for scheduling purposes.  For VMIs, it allocates
  `vfio` device nodes to pods; it also sets environment variables with PCI IDs
  of allocated devices. These variables are later used by KubeVirt to configure
  libvirt domain.
* [SR-IOV CNI plugin](https://github.com/intel/sriov-cni) configures allocated
  SR-IOV resources before a pod or VMI is started. This component modifies host
  resources to prepare them to be used inside a pod / VMI. Executed in root
  network namespace context. Uses netlink and other commands to configure and
  move SR-IOV VF / PF resources into pod namespaces.
* [Multus](https://github.com/intel/multus-cni) is a meta-plugin. It will call
  to SR-IOV CNI plugin when VMI is attached to an SR-IOV interface. It is
  configured through `NetworkAttachmentDefinition` CRD objects. For SR-IOV, the
  object annotations should refer to the resource name configured inside device
  plugin `config.json` configuration file. This reference is used by KubeVirt
  to automatically fill in `requests` and `limits` sections of `virt-launcher`
  pods. (The same mechanism is used by other plugins, for example, `bridge`.)
  If used alone, SR-IOV CNI plugin would need the PCI address of the device we
  want to use inside the pod (and then, inside the VM). By using it in
  combination with the device plugin and Multus, thanks to `resourceName`
  annotation, the pod will receive a device from the pool allocated by the
  device plugin, which allows to avoid manually passing the required PCI
  address.
* [OpenShift SR-IOV operator](https://github.com/openshift/sriov-network-operator)
  configures kernel for SR-IOV, drains resources and reboots the kernel as
  needed, deploys components listed above, and, based on values in policy CRD
  resources managed by admin, configures the above listed components and the
  host.
* KubeVirt, based on values of environment variables set by SR-IOV device
  plugin, configures libvirt domain for SR-IOV attached VMIs to use the right
  PCI IDs. Also, fills in `requests` and `limits` sections of `virt-launcher`
  pod spec as per attached NetworkAttachmentDefinition CRD.

# Configuration

> NOTE: steps to configure SR-IOV on a host are only applicable if you don't
> use OpenShift SR-IOV operator to deploy SR-IOV components. Despite the name,
> the operator supports installation on a plain Kubernetes cluster. The
> operator handles most of the steps discussed below, including configuring
> kernel parameters. The only step that cannot be handled by the operator is
> BIOS setup, which should be handled by other means (either manually or
> through some kind of IPMI driven automation).
>
> Documentation on how to use the operator
> [is located](https://github.com/openshift/sriov-network-operator/blob/master/README.md)
> [elsewhere](https://github.com/openshift/sriov-network-operator/blob/master/doc/quickstart.md).
> Just remember to set `SriovNetworkNodePolicy` to use `deviceType: vfio-pci`
> when configuring the operator.
>
> For local development purposes, one should be able to (re)use the same
> [kind](https://kind.sigs.k8s.io/) based provider that the official upstream
> SR-IOV CI relies on, for example:
>
> $ export KUBEVIRT_PROVIDER=kind-k8s-sriov-1.14.2
>
> $ make cluster-up
>
> Assuming you run these commands on a SR-IOV enabled host, they should bring
> up a dockerized cluster with all SR-IOV components and resources set up and
> ready to use.
>
> If you know what you are doing and still would like to follow the manual
> configuration path, keep reading.

Before you deploy your cluster, make sure your host has a SR-IOV capable NIC
plugged in, and that it supports it. You may need to adjust BIOS settings to
make it work. For example, make sure VT-x (hardware virtualization) and Intel
VT for Directed I/O are enabled in BIOS.

You should also configure kernel to enable IOMMU. This can be achieved by
adding the following kernel parameters to kernel command line:

```
intel_iommu=on
```

Some hosts may experience problems detecting VFs. If you experience issues
configuring VFs, you may try to add one of the following parameters:

```
pci=realloc
pci=assign-busses
```

If all goes well, after reboot your SR-IOV capable NICs should be ready to use.


Check for SR-IOV devices by doing the following:
```
$ find /sys -name *vfs*
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.0/sriov_numvfs
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.0/sriov_totalvfs
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.1/sriov_numvfs
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.1/sriov_totalvfs
```

KubeVirt will use [vfio](https://www.kernel.org/doc/Documentation/vfio.txt)
userspace driver to pass through PCI devices into `qemu`. VFIO is an
established way to pass raw devices into running virtual machines, in contrast
to using the regular `netdevice` device plugin mode used to attach regular pods
to `netlink` interfaces registered with corresponding Linux kernel network
drivers. More information on VFIO can also be found
[here](https://www.linux-kvm.org/images/b/b4/2012-forum-VFIO.pdf).

For this to work, load the following driver:

```
$ modprobe vfio-pci
```

Depending on your hardware platform, the driver may need additional kernel
options. For example, if your platform does not support interrupt remapping,
you may need to configure the host as follows:

```
$ echo "options vfio_iommu_type1 allow_unsafe_interrupts=1" > /etc/modprobe.d/iommu_unsafe_interrupts.conf
```

Now you are ready to set up your cluster.

# Set up kubernetes cluster

You can use your preferred mechanism to deploy your kubernetes cluster as long
as you deploy on bare metal. Note that using virtualized environment is
problematic with SR-IOV because to use SR-IOV PFs in such environment, one
would first need to pass through PF devices from hypervisor level into the
virtual machines. That's why right now it's impossible to use regular providers
that use qemu machines for SR-IOV development.

Current recommendation is to use the official KubeVirt operator to deploy
clusters.

You may still want to deploy software using `local` provider if you'd like to
deploy from KubeVirt sources though.

In the following example, we configure the cluster using `local` provider which
is part of kubevirt/kubevirt repo. Please consult
[documentation](https://github.com/kubevirt/kubevirt/blob/main/cluster-up/cluster/local/README.md)
for general information on setting up a host using the `local` provider.

The `local` provider does not install default CNI plugins like `loopback`. So
first, install default CNI plugins:

```
$ go get -u -d github.com/containernetworking/plugins/
$ cd $GOPATH/src/github.com/containernetworking/plugins/
$ ./build.sh
$ mkdir -p /opt/cni/bin/
$ cp bin/* /opt/cni/bin/
```

Then, prepare kubernetes tree for CNI enabled deployment:

```
$ go get -u -d k8s.io/kubernetes
$ cd $GOPATH/src/k8s.io/kubernetes
$ git diff
diff --git a/hack/local-up-cluster.sh b/hack/local-up-cluster.sh
index bcf988b..9911eed 100755
--- a/hack/local-up-cluster.sh
+++ b/hack/local-up-cluster.sh
@@ -639,6 +639,8 @@ function start_controller_manager {
       --use-service-account-credentials \
       --controllers="${KUBE_CONTROLLERS}" \
       --leader-elect=false \
+      --cert-dir="$CERT_DIR" \
+      --allocate-node-cidrs=true --cluster-cidr=10.244.0.0/16 \
       --master="https://${API_HOST}:${API_SECURE_PORT}" >"${CTLRMGR_LOG}" 2>&1 &
     CTLRMGR_PID=$!
 }
export NET_PLUGIN=cni
export CNI_CONF_DIR=/etc/cni/net.d/
export CNI_BIN_DIR=/opt/cni/bin/
```

Install etcd:

```
$ ./hack/install-etcd.sh
```

Use `local` provider for kubevirt:

```
$ export KUBEVIRT_PROVIDER=local
```

Now finally, deploy kubernetes:

```
$ cd $GOPATH/src/kubevirt.io/kubevirt
$ make cluster-up
```

Once the cluster is deployed, we can move to SR-IOV specific components.

# Deploy SR-IOV services

> NOTE: as stated above, manual deployment of SR-IOV components is not
> recommended. Please consider using SR-IOV operator instead. But if you know
> what you are doing, keep reading.

First, deploy latest Multus with default Flannel backend.

```
$ go get -u -d github.com/intel/multus-cni
$ cd $GOPATH/src/github.com/intel/multus-cni/
$ mkdir -p /etc/cni/net.d
$ cp images/70-multus.conf /etc/cni/net.d/
$ ./cluster-up/kubectl.sh create -f $GOPATH/src/github.com/intel/multus-cni/images/multus-daemonset.yml
$ ./cluster-up/kubectl.sh create -f $GOPATH/src/github.com/intel/multus-cni/images/flannel-daemonset.yml
```

Now, deploy SR-IOV device plugin. Adjust config.json file for your particular
setup. More information about configuration file format:
https://github.com/intel/sriov-network-device-plugin/blob/master/README.md#configurations

```
$ go get -u -d github.com/intel/sriov-network-device-plugin/
$ cat <<EOF > /etc/pcidp/config.json
{
    "resourceList":
    [
        {
            "resourceName": "sriov",
            "rootDevices": ["05:00.0", "05:00.1"],
            "sriovMode": true,
            "deviceType": "vfio"
        }
    ]
}
EOF
$ ./cluster-up/kubectl.sh create -f $GOPATH/src/github.com/intel/sriov-network-device-plugin/images/sriovdp-daemonset.yaml
```

Deploy SR-IOV CNI plugin.

```
$ go get -u -d github.com/intel/sriov-cni/
$ ./cluster-up/kubectl.sh create -f $GOPATH/src/github.com/intel/sriov-cni/images/sriov-cni-daemonset.yaml
```

Finally, create a new SR-IOV network CRD that will use SR-IOV device plugin to allocate devices.

```
./cluster-up/kubectl.sh create -f $GOPATH/src/github.com/intel/sriov-network-device-plugin/deployments/sriov-crd.yaml
```

Just make sure that the network spec refers to the right resource name for
SR-IOV resources configured in SR-IOV device plugin configuration file
(config.json).

# Install kubevirt services

This particular step is not specific to SR-IOV.

```
make cluster-sync
```

# Usage

If all goes well, you should be able to post a VMI spec referring to the SR-IOV
multus network and get a PCI device allocated to virt-launcher and passed
through into qemu. Please consult
[the VMI spec example](https://github.com/kubevirt/kubevirt/blob/main/examples/vmi-sriov.yaml).

As long as the VMI spec `networks` section refers to the proper
`NetworkAttachmentDefinition` that describes a SR-IOV network, you should be
able to post it and get a machine attached to an SR-IOV device via VFIO. Note
that the `NetworkAttachmentDefinition` resource should also refer, in its
annotations, to a correct resource name as reported by SR-IOV device plugin for
all this to work. More details on usage can be found in
[KubeVirt](https://kubevirt.io/user-guide/#/creation/interfaces-and-networks?id=sriov)
and [SR-IOV operator](https://github.com/openshift/sriov-network-operator/blob/master/doc/quickstart.md)
user documentation.

> **NOTE:**  In cases where no VLAN is required on the VF, an explicit definition of `vlan: 0` needs to be set on the NAD.
> If the VLAN field is missing, the VF VLAN will not be set and any existing setting on it will be left untouched.
>
> For more details please address the SR-IOV-CNI issue at: 
> https://github.com/openshift/sriov-cni/issues/25#issue-816231435

# External resources

* [User guide section on SR-IOV](https://kubevirt.io/user-guide/#/creation/interfaces-and-networks?id=sriov)
* [OpenShift user docs on SR-IOV](https://docs.openshift.com/container-platform/4.2/networking/multiple-networks/configuring-sr-iov.html)
* [Doug Smith's blog post on deploying and testing SR-IOV](https://dougbtv.com/nfvpe/2019/05/15/kubevirt-sriov/)
