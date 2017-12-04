# Local development environment

This document explains how to setup a local development environment
for kubernetes and kubevirt. In this setup everything runs in the
OS you are developing on, in constrast to the Vagrant environment
which spins up a separate virtual machine. This local env is useful
if you are already developing from inside a virtual machine, in
which case Vagrant would be forced to use a QEMU emulated env with
no KVM acceleration.

## Getting the source

```bash
mkdir -p $HOME/src/k8s/{src,bin,pkg}
echo "export GOPATH=$HOME/src/k8s" >> ~/.bashrc
echo "export PATH=\$GOPATH/bin:\$GOPATH/src/k8s.io/kubernetes/_output/bin:\$PATH" >> ~/.bashrc
source ~/.bashrc
```

If you have a working Go environment, run

```bash
go get -d k8s.io/kubernetes
cd $GOPATH/src/k8s.io/kubernetes
```

to fetch the latest kubernetes sources.

## Running kubernetes

The first step is to get kubernetes itself up & running on the local
machine. This setup provides just a single compute node running
locally.

Assuming a machine with a hostname of `kubdev` and IP address
of `192.168.122.13`, then from the root of a k8s checkout

```bash
export API_HOST_IP=192.168.122.13
export KUBELET_HOST=192.168.122.13
export HOSTNAME_OVERRIDE=kubdev
export ALLOW_PRIVILEGED=1
export ALLOW_SECURITY_CONTEXT=1
export KUBE_DNS_DOMAIN="cluster.local"
export KUBE_DNS_SERVER_IP="10.0.0.10"
export KUBE_ENABLE_CLUSTER_DNS=true
./hack/local-up-cluster.sh
```

Once k8s has been launched once, you can skip the slow compilation
step using

```bash
./hack/local-up-cluster.sh -o _output/local/bin/linux/amd64/
```


## Building kubevirt

First configure kubevirt with site specific parameters. As above
we need the IP address of the host.

```bash
cat > hack/config-local.sh <<EOF
  master_ip=192.168.122.13
  docker_tag=latest
EOF
make manifests docker
```

## Running kubevirt

Simply load all the manifests into k8s, and then wait for all
pods to change to running state:

```bash
for i in manifests/*.yaml
   do
     kubectl create -f $i
   done
kubectl get pods
```


## Launching a VM

```bash
kubectl create -f cluster/vm.yaml
```

## Running functional tests

```bash
make functest
```
