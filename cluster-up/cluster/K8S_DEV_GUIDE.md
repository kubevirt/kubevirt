# kubevirtci K8s provider dev guide.

Note: in the following scenarios we are using `${KUBEVIRT_PROVIDER_VERSION}` as pointer to the `major.minor` k8s version we are using

This then can map to any of these folders:
* `cluster-provision/k8s/${KUBEVIRT_PROVIDER_VERSION}`
* `cluster-up/cluster/k8s-${KUBEVIRT_PROVIDER_VERSION}`

## Creating or updating a provider

The purpose of kubevirtci is to create pre-provisioned K8s clusters as container images,
allowing people to easily run a K8s cluster.

The target audience is developers of kubevirtci, who want to create a new provider, or to update an existing one.

Please refer first to the following documents on how to run k8s-1.x:\
[k8s-1.x cluster-up](./K8S.md)

In this doc, we will go on what kubevirtci provider image consists of, what its inner architecture is,
flow of starting a pre-provisioned cluster, flow of creating a new provider, and how to create a new provider.

A provider includes all the images (K8s base image, nodes OS image) and the scripts that allows it to start a
cluster offline, without downloading / installing / compiling new resources.
Deploying a cluster will create containers, which communicate with each other, in order to act as a K8s cluster.
It's a bit different from running bare-metal cluster where the nodes are physical machines or when the nodes are virtual machines on the host itself,
It gives us isolation advantage and state freezing of the needed components, allowing offline deploy, agnostic of the host OS, and installed packages.

# Project structure
* cluster-provision folder - creating preprovisioned clusters.
* cluster-up folder - spinning up preprovisioned clusters.
* gocli - gocli is a binary that assist in provisioning and spinning up a cluster. sources of gocli are at cluster-provision/gocli.

# K8s Deployment
Running `make cluster-up` will deploy a pre-provisioned cluster.
Upon finishing deployment of a K8s deploy, we will have 3 containers:
* k8s-${KUBEVIRT_PROVIDER_VERSION} vm container - a container that runs a qemu VM, which is the K8s node, in which the pods will run.
* Registry container - a shared image registry.
* k8s-${KUBEVIRT_PROVIDER_VERSION} dnsmasq container - a container that run dnsmasq, which gives dns and dhcp services.

The containers are running and look like this:
```
[root@modi01 1.21.0]# docker ps
CONTAINER ID        IMAGE                   COMMAND                  CREATED             STATUS              PORTS                                                                                                                          NAMES
3589e85efc7d        kubevirtci/k8s-1.21.0   "/bin/bash -c '/vm.s…"   About an hour ago   Up About an hour                                                                                                                                   k8s-1.21.0-node01
4742dc02add2        registry:2.7.1          "/entrypoint.sh /etc…"   About an hour ago   Up About an hour                                                                                                                                   k8s-1.21.0-registry
13787e7d4ac9        kubevirtci/k8s-1.21.0   "/bin/bash -c /dnsma…"   About an hour ago   Up About an hour    127.0.0.1:8443->8443/tcp, 0.0.0.0:32794->2201/tcp, 0.0.0.0:32793->5000/tcp, 0.0.0.0:32792->5901/tcp, 0.0.0.0:32791->6443/tcp   k8s-1.21.0-dnsmasq
```

Nodes:
```
[root@modi01 kubevirtci]# oc get nodes
NAME     STATUS   ROLES    AGE   VERSION
node01   Ready    master   83m   v1.21.0
```

# Inner look of a deployed cluster
We can connect to the node of the cluster by:
```
./cluster-up/ssh.sh node01
```

List the pods
```
[vagrant@node01 ~]$ sudo crictl pods
POD ID              CREATED             STATE               NAME                             NAMESPACE           ATTEMPT
403513878c8b7       10 minutes ago      Ready               coredns-6955765f44-m6ckl         kube-system         4
0c3e25e58b9d0       10 minutes ago      Ready               local-volume-provisioner-fkzgk   default             4
e6d96770770f4       10 minutes ago      Ready               coredns-6955765f44-mhfgg         kube-system         4
19ad529c78acc       10 minutes ago      Ready               kube-flannel-ds-amd64-mq5cx      kube-system         0
47acef4276900       10 minutes ago      Ready               kube-proxy-vtj59                 kube-system         0
df5863c55a52f       11 minutes ago      Ready               kube-scheduler-node01            kube-system         0
ca0637d5ac82f       11 minutes ago      Ready               kube-apiserver-node01            kube-system         0
f0d90506ce3b8       11 minutes ago      Ready               kube-controller-manager-node01   kube-system         0
f873785341215       11 minutes ago      Ready               etcd-node01                      kube-system         0
```

Check kubelet service status
```
[vagrant@node01 ~]$ systemctl status kubelet
● kubelet.service - kubelet: The Kubernetes Node Agent
   Loaded: loaded (/usr/lib/systemd/system/kubelet.service; enabled; vendor preset: disabled)
  Drop-In: /usr/lib/systemd/system/kubelet.service.d
           └─10-kubeadm.conf
   Active: active (running) since Wed 2020-01-15 13:39:54 UTC; 11min ago
     Docs: https://kubernetes.io/docs/
 Main PID: 4294 (kubelet)
   CGroup: /system.slice/kubelet.service
           ‣ 4294 /usr/bin/kubelet --bootstrap-kubeconfig=/etc/kubernetes/boo...
```

Connect to the container that runs the vm:
```
CONTAINER=$(docker ps | grep vm | awk '{print $1}')
docker exec -it $CONTAINER bash
```

From within the container we can see there is a process of qemu which runs the node as a virtual machine.
```
[root@855de8c8310f /]# ps -ef | grep qemu
root         1     0 36 13:39 ?        00:05:22 qemu-system-x86_64 -enable-kvm -drive format=qcow2,file=/var/run/disk/disk.qcow2,if=virtio,cache=unsafe -device virtio-net-pci,netdev=network0,mac=52:55:00:d1:55:01 -netdev tap,id=network0,ifname=tap01,script=no,downscript=no -device virtio-rng-pci -vnc :01 -cpu host -m 5120M -smp 5 -serial pty
```

# Flow of K8s provisioning ${KUBEVIRT_PROVIDER_VERSION}
`cluster-provision/k8s/${KUBEVIRT_PROVIDER_VERSION}/provision.sh`
* Runs the common cluster-provision/k8s/provision.sh.
    * Runs cluster-provision/cli/cli (bash script).
        * Creates a container for dnsmasq and runs dnsmasq.sh in it.
        * Create a container, and runs vm.sh in it.
            * Creates a vm using qemu, and checks its ready (according ssh).
            * Runs cluster-provision/k8s/scripts/provision.sh in the container.
                * Update docker trusted registries.
                * Start kubelet service and K8s cluster.
                * Enable ip routing.
                * Apply additional manifests, such as flannel.
                * Wait for pods to become ready.
                * Pull needed images such as Ceph CSI, fluentd logger.
                * Create local volume directories.
            * Shutdown the vm and commit its container.

# Flow of K8s cluster-up ${KUBEVIRT_PROVIDER_VERSION}
Run
```
export KUBEVIRT_PROVIDER=k8s-${KUBEVIRT_PROVIDER_VERSION}
make cluster-up
```
* Runs cluster-up/up.sh which sources the following:
    * cluster-up/cluster/k8s-${KUBEVIRT_PROVIDER_VERSION}/provider.sh (selected according $KUBEVIRT_PROVIDER), which sources:
        * cluster-up/cluster/k8s-provider-common.sh
* Runs `up` (which appears at cluster-up/cluster/k8s-provider-common.sh).
It Triggers `gocli run` - (cluster-provision/gocli/cmd/run.go) which create the following containers:
    * Cluster container (that one with the vm from the provisioning, vm.sh is used with parameters here that starts an already created vm).
    * Registry.
    * Container for dnsmasq (provides dns, dhcp services).

# Creating new K8s provider
Clone folders of k8s, folder name should be x/y as in the provider name x-y (ie. k8s-${KUBEVIRT_PROVIDER_VERSION}.0) and includes:
* cluster-provision/k8s/${KUBEVIRT_PROVIDER_VERSION}/provision.sh  # used to create a new provider
* cluster-provision/k8s/${KUBEVIRT_PROVIDER_VERSION}/publish.sh  # used to publish new provider
* cluster-up/cluster/k8s-${KUBEVIRT_PROVIDER_VERSION}/provider.sh  # used by cluster-up
* cluster-up/cluster/k8s-${KUBEVIRT_PROVIDER_VERSION}/README.md

# Example - Adding a new manifest to K8s
* First add the file at cluster-provision/manifests, this folder would be copied to /tmp in the container,
by cluster-provision/cli/cli as part of provision.
* Add this snippet at cluster-provision/k8s/scripts/provision.sh, before "Wait at least for 7 pods" line.
```
custom_manifest="/tmp/custom_manifest.yaml"
kubectl --kubeconfig=/etc/kubernetes/admin.conf create -f "$custom_manifest"
```
* Run ./cluster-provision/k8s/${KUBEVIRT_PROVIDER_VERSION}/provision.sh, it will create a new provision and test it.

# Manual steps for publishing a new provider

The steps to create, test and integrate a new KubeVirtCI provider are [mostly automated](./K8S_AUTOMATION.md), but just in case you need to do it manually:

* Run `./cluster-provision/k8s/${KUBEVIRT_PROVIDER_DIR}/publish.sh`, it will publish the new created image to quay.io
* Create a PR with the following files:
    * The new manifest.
    * Updated `cluster-provision/k8s/scripts/provision.sh`
    * Updated `cluster-up/cluster/images.sh`.
