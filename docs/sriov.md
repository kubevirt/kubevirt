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

If all goes well, after reboot you should be able to enable SR-IOV VFs for
capable NICs.

To enable VFs, you should do the following:
```
$ find /sys -name *vfs*
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.0/sriov_numvfs
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.0/sriov_totalvfs
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.1/sriov_numvfs
/sys/devices/pci0000:00/0000:00:09.0/0000:05:00.1/sriov_totalvfs
$ cat /sys/devices/pci0000:00/0000:00:09.0/0000:05:00.0/sriov_totalvfs
7
$ echo 7 > /sys/devices/pci0000:00/0000:00:09.0/0000:05:00.0/sriov_numvfs
```

If all goes well you should see VFs in lspci output:

```
$ lspci
...
05:10.0 Ethernet controller: Intel Corporation 82576 Virtual Function (rev 01)
05:10.1 Ethernet controller: Intel Corporation 82576 Virtual Function (rev 01)
05:10.2 Ethernet controller: Intel Corporation 82576 Virtual Function (rev 01)
05:10.3 Ethernet controller: Intel Corporation 82576 Virtual Function (rev 01)
05:10.4 Ethernet controller: Intel Corporation 82576 Virtual Function (rev 01)
05:10.5 Ethernet controller: Intel Corporation 82576 Virtual Function (rev 01)
05:10.6 Ethernet controller: Intel Corporation 82576 Virtual Function (rev 01)
05:10.7 Ethernet controller: Intel Corporation 82576 Virtual Function (rev 01)
...
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

Also, make sure that vfio devices are accessible to kvm group by configuring udev rules:

```
$ cat /etc/udev/rules.d/10-qemu-hw-users.rules
SUBSYSTEM=="vfio", OWNER="root", GROUP="root", MODE="0666"
KERNEL=="vfio", SUBSYSTEM=="misc", OWNER="root", GROUP="root", MODE="0666"
KERNEL=="kvm", GROUP="root", MODE="0666"
$ udevadm control --reload-rules && udevadm trigger
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
$ ./cluster/kubectl.sh create -f $GOPATH/src/github.com/intel/sriov-network-device-plugin/images/sriovdp-daemonset.yaml
```

Deploy SR-IOV CNI plugin.

```
$ go get -u -d github.com/intel/sriov-cni/
$ ./cluster/kubectl.sh create -f $GOPATH/src/github.com/intel/sriov-cni/images/sriov-cni-daemonset.yaml
```

Finally, create a new SR-IOV network CRD that will use SR-IOV device plugin to allocate devices.

```
./cluster/kubectl.sh  create -f $GOPATH/src/github.com/intel/sriov-network-device-plugin/deployments/sriov-crd.yaml
```

# Install kubevirt services

The SR-IOV feature is gated, so you would need to enable the `SRIOV` gate
feature using `kubevirt-config` map before deploying Kubevirt. For example,

```
cat <<EOF | ./cluster/kubectl.sh create -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubevirt-config
  namespace: kubevirt
  labels:
    kubevirt.io: ""
data:
  feature-gates: "SRIOV"
EOF
```

After that, you are ready to deploy Kubevirt. As you can see, this particular
step is not specific to SR-IOV.

```
make cluster-sync
```

If all goes well, you should be able to post a VMI spec referring to the SR-IOV
multus network and get a PCI device allocated to virt-launcher and passed
through into qemu. Please consult cluster/examples/vmi-sriov.yaml for example.
