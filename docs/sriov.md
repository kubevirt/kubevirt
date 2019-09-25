# Enable SR-IOV on the host

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

If all goes well, after reboot your SR-IOV capable NICs should be ready to be used.


Check for sr-iov devices by doing the following:
```
$ find /sys -name *vfs*
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.0/sriov_numvfs
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.0/sriov_totalvfs
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.1/sriov_numvfs
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.1/sriov_totalvfs
```

Kubevirt will use `vfio` userspace driver to pass through PCI devices into
`qemu`. For this to work, load the following driver:

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
as you deploy on bare metal.

Current recommendation is to use ```kubevirt-ansible``` to deploy the cluster.
Ansible playbooks will also deploy all the relevant SR-IOV components
for you. See [here](https://github.com/kubevirt/kubevirt-ansible/).

You may still want to deploy software using `local` provider if you'd like to
deploy from Kubevirt sources though.

In the following example, we configure the cluster using `local` provider which
is part of kubevirt/kubevirt repo. Please consult cluster/local/README.md for
general information on setting up a host using the `local` provider.

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

First, deploy latest Multus with default Flannel backend. We will need to use
the latest code from their tree, hence using `snapshot` image tag instead of
`latest`. The `snapshot` image adds support for reading IDs of devices
allocated by device plugins from "checkpoint" files, which is needed to make
the whole setup work.

```
$ go get -u -d github.com/intel/multus-cni
$ cd $GOPATH/src/github.com/intel/multus-cni/
$ vi images/multus-daemonset.yml # change to refer to nfvpe/multus:snapshot
$ mkdir -p /etc/cni/net.d
$ cp images/70-multus.conf /etc/cni/net.d/
$ ./cluster/kubectl.sh create -f $GOPATH/src/github.com/intel/multus-cni/images/multus-daemonset.yml
$ ./cluster/kubectl.sh create -f $GOPATH/src/github.com/intel/multus-cni/images/flannel-daemonset.yml
```

The best way to configure your SR-IOV devices is by using the [OpenShift SR-IOV operator](https://github.com/openshift/sriov-network-operator).

It will setup the devices according to the configuration provided and it will deploy the components needed to make SR-IOV work in your cluster (in particular, the [SR-IOV CNI plugin](https://github.com/intel/sriov-cni) and the [SR-IOV device plugin](https://github.com/intel/sriov-network-device-plugin)).

The OpenShift SR-IOV quickstart guide can be found [here](https://github.com/openshift/sriov-network-operator/blob/master/doc/quickstart.md). They provide instructions to run the operator both on Kubernetes and OpenShift.

Just remember to set the `SriovNetworkNodePolicy` to use `deviceType: vfio-pci`.

# Install kubevirt services

This particular step is not specific to SR-IOV.

```
make cluster-sync
```

If all goes well, you should be able to post a VMI spec referring to the SR-IOV
multus network and get a PCI device allocated to virt-launcher and passed
through into qemu. Please consult cluster/examples/vmi-sriov.yaml for example.
