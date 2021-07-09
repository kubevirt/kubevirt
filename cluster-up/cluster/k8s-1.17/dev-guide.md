# kubevirtci K8s provider dev guide.

The purpose of kubevirtci is to create pre-provisioned K8s clusters as container images,
allowing people to easily run a K8s cluster.

The target audience is developers of kubevirtci, who want to create a new provider, or to update an existing one.

Please refer first to the following documents on how to run k8s-1.17:\
[k8s-1.17 cluster-up](https://github.com/kubevirt/kubevirtci/blob/master/cluster-up/cluster/k8s-1.17/README.md)

In this doc, we will go on what kubevirtci provider image consist of, what its inner architecture,
flow of start a pre-provisioned cluster, flow of creating a new provider, and how to create a new provider.

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
* k8s-1.17 vm container - a container that runs a qemu VM, which is the K8s node, in which the pods will run.
* Registry container - a shared image registry.
* k8s-1.17 dnsmasq container - a container that run dnsmasq, which gives dns and dhcp services.

The containers are running and looks like this:
```
[root@modi01 1.17]# docker ps
CONTAINER ID   IMAGE                                            COMMAND                  CREATED              STATUS              PORTS                                                                                                                                                                         NAMES
8ddefc88cdd2   quay.io/kubevirtci/k8s-1.17:2103240101-142f745   "/bin/bash -c '/vm.s…"   About a minute ago   Up About a minute                                                                                                                                                                                 k8s-1.17-node01
1e10735ba935   registry:2.7.1                                   "/entrypoint.sh /etc…"   About a minute ago   Up About a minute                                                                                                                                                                                 k8s-1.17-registry
930002ada03f   quay.io/kubevirtci/k8s-1.17:2103240101-142f745   "/bin/bash -c /dnsma…"   About a minute ago   Up About a minute   127.0.0.1:8443->8443/tcp, 0.0.0.0:49189->80/tcp, 0.0.0.0:49188->443/tcp, 0.0.0.0:49187->2201/tcp, 0.0.0.0:49186->5000/tcp, 0.0.0.0:49185->5901/tcp, 0.0.0.0:49184->6443/tcp   k8s-1.17-dnsmasq
```

Nodes:
```
[root@modi01 kubevirtci]# oc get nodes
NAME     STATUS   ROLES           AGE   VERSION
node01   Ready    master,worker   58s   v1.17.16-rc.0
```

# Inner look of a deployed cluster
We can connect to the node of the cluster by:
```
[ -z "$KUBEVIRTCI_TAG" ] && export KUBEVIRTCI_TAG=$(curl -L -Ss https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest)
./cluster-up/ssh.sh node01
```

List the pods
```
[vagrant@node01 ~]$ sudo crictl pods
POD ID              CREATED              STATE               NAME                                       NAMESPACE           ATTEMPT
6c890a5d57157       About a minute ago   Ready               coredns-76655995b5-2px4q                   kube-system         1
1d845e00e764b       About a minute ago   Ready               coredns-76655995b5-pc5ps                   kube-system         1
725f7ff152aec       About a minute ago   Ready               local-volume-provisioner-89658             default             1
db5569450da4c       About a minute ago   Ready               calico-kube-controllers-54f8c7fccd-2gfq5   kube-system         1
7c97735359c97       About a minute ago   Ready               calico-node-nlnbx                          kube-system         0
66f0f1cad7d14       About a minute ago   Ready               kube-proxy-dd7rn                           kube-system         0
97d6164314dfa       About a minute ago   Ready               kube-scheduler-node01                      kube-system         0
480f67ee94f93       About a minute ago   Ready               kube-controller-manager-node01             kube-system         0
fe045104e56a8       About a minute ago   Ready               kube-apiserver-node01                      kube-system         0
6dc9bd9868ea2       About a minute ago   Ready               etcd-node01                                kube-system         0
```

Check kubelet service status
```
[vagrant@node01 ~]$ systemctl status kubelet
● kubelet.service - kubelet: The Kubernetes Node Agent
   Loaded: loaded (/usr/lib/systemd/system/kubelet.service; enabled; vendor pre>
  Drop-In: /usr/lib/systemd/system/kubelet.service.d
           └─10-kubeadm.conf
   Active: active (running) since Wed 2021-03-24 10:27:13 UTC; 2min 3s ago
     Docs: https://kubernetes.io/docs/
 Main PID: 2928 (kubelet)
    Tasks: 0 (limit: 31372)
   Memory: 35.2M
   CGroup: /system.slice/kubelet.service
           ‣ 2928 /usr/bin/kubelet --bootstrap-kubeconfig=/etc/kubernetes/boots>
```

Connect to the container that runs the vm:
```
CONTAINER=$(docker ps | grep vm | awk '{print $1}')
docker exec -it $CONTAINER bash
```

From within the container we can see there is a process of qemu which runs the node as a virtual machine.
```
[root@930002ada03f /]# ps -ef | grep qemu | grep -v grep
root           1       0 63 10:26 ?        00:02:29 qemu-system-x86_64 -enable-kvm -drive format=qcow2,file=/var/run/disk/disk.qcow2,if=virtio,cache=unsafe -device virtio-net-pci,netdev=network0,mac=52:55:00:d1:55:01 -netdev tap,id=network0,ifname=tap01,script=no,downscript=no -device virtio-rng-pci -vnc :01 -cpu host -m 5120M -smp 6 -serial pty -serial pty -M q35,accel=kvm,kernel_irqchip=split -device intel-iommu,intremap=on,caching-mode=on -soundhw hda
```

# Flow of K8s provisioning (1.17 for example)
`cluster-provision/k8s/1.17/provision.sh`
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

# Flow of K8s cluster-up (1.17 for example)
Run
```
export KUBEVIRT_PROVIDER=k8s-1.17
make cluster-up
```
* Runs cluster-up/up.sh which sources the following:
    * cluster-up/cluster/k8s-1.17/provider.sh (selected according $KUBEVIRT_PROVIDER), which sources:
        * cluster-up/cluster/k8s-provider-common.sh
* Runs `up` (which appears at cluster-up/cluster/k8s-provider-common.sh).
It Triggers `gocli run` - (cluster-provision/gocli/cmd/run.go) which create the following containers:
    * Cluster container (that one with the vm from the provisioning, vm.sh is used with parameters here that starts an already created vm).
    * Registry.
    * Container for dnsmasq (provides dns, dhcp services).

# Creating new K8s provider
Clone folders of k8s, folder name should be x/y as in the provider name x-y (ie. k8s-1.17) and includes:
* cluster-provision/k8s/1.17/provision.sh  # used to create a new provider
* cluster-provision/k8s/1.17/publish.sh  # used to publish new provider
* cluster-up/cluster/k8s-1.17/provider.sh  # used by cluster-up
* cluster-up/cluster/k8s-1.17/README.md

# Example - Adding a new manifest to K8s 1.17
* First add the file at cluster-provision/manifests, this folder would be copied to /tmp in the container,
by cluster-provision/cli/cli as part of provision.
* Add this snippet at cluster-provision/k8s/scripts/provision.sh, before "Wait at least for 7 pods" line.
```
custom_manifest="/tmp/custom_manifest.yaml"
kubectl --kubeconfig=/etc/kubernetes/admin.conf create -f "$custom_manifest" 
```
* Run ./cluster-provision/k8s/1.17/provision.sh, it will create a new provision and test it.
* Run ./cluster-provision/k8s/1.17/publish.sh, it will publish the new created image to docker.io
* Update k8s-1.17 image line at cluster-up/cluster/images.sh, to point on the newly published image.
* Create a PR with the following files:
    * The new manifest.
    * Updated cluster-provision/k8s/scripts/provision.sh
    * Updated cluster-up/cluster/images.sh.
