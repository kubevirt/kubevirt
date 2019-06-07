# Local Kubernets Provider

This provider allows developing against bleeding-edge Kubernetes code. The
k8s sources will be compiled and a single-node cluster will be started.

## Environment preparation

Since the `local` provider deploys the cluster on the host and not inside
virtual machines, you may need to adjust some settings on the node.

Specifically, you may need to make sure that your firewall of choice doesn't
block connectivity between cluster IP and service pods. If you experience
connectivity issues, consider tweaking or disabling your firewall. Also, make
sure forwarding is enabled on the host:

```bash
$ systemctl disable firewalld --now
$ iptables -P FORWARD ACCEPT
$ sysctl net.ipv4.conf.all.forwarding=1
```

## Bringing the cluster up

First get the k8s sources:

```bash
go get -u -d k8s.io/kubernetes
```

Then compile and start the cluster:

```bash
export KUBEVIRT_PROVIDER=local
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME     STATUS   ROLES    AGE     VERSION
kubdev   Ready    <none>   5m20s   v1.12.0-beta.2
```

Note: you may need to cherry-pick
[acdb1b0e9855ab671f2972f10605d20cad26284b](https://github.com/kubernetes/kubernetes/commit/acdb1b0e9855ab671f2972f10605d20cad26284b)
if it's not present in your kubernetes tree yet.

## CNI

By default, local provider deploys cluster with no CNI support. To make CNI
work, you should set the following variables before spinning up cluster:

```bash
$ export NET_PLUGIN=cni
$ export CNI_CONF_DIR=/etc/cni/net.d/
$ export CNI_BIN_DIR=/opt/cni/bin/
```

Depending on your CNI of choice (for example, Flannel), you may also need to
add the following arguments to controller-manager inside
`hack/local-cluster-up.sh`:

```bash
$ git diff
diff --git a/hack/local-up-cluster.sh b/hack/local-up-cluster.sh
index bcf988b..9911eed 100755
--- a/hack/local-up-cluster.sh
+++ b/hack/local-up-cluster.sh
@@ -639,6 +639,8 @@ function start_controller_manager {
       --use-service-account-credentials \
       --controllers="${KUBE_CONTROLLERS}" \
       --leader-elect=false \
       --cert-dir="$CERT_DIR" \
+      --allocate-node-cidrs=true --cluster-cidr=10.244.0.0/16 \
       --master="https://${API_HOST}:${API_SECURE_PORT}" >"${CTLRMGR_LOG}" 2>&1 &
     CTLRMGR_PID=$!
 }
```

Also, you will need to install [reference CNI plugins](https://github.com/containernetworking/plugins):

```bash
$ go get -u -d github.com/containernetworking/plugins/
$ cd $GOPATH/src/github.com/containernetworking/plugins/
$ ./build.sh
$ sudo mkdir -p /opt/cni/bin/
$ sudo cp bin/* /opt/cni/bin/
```

In some cases (for example, Multus), your CNI plugin may also require presence
of `/etc/kubernetes/kubelet.conf` file. In this case, you should create a
symlink pointing to the right location:

```bash
$ sudo mkdir /etc/kubernetes
$ sudo ln -s $GOPATH/src/kubevirt.io/kubevirt/cluster/local/certs/kubelet.kubeconfig /etc/kubernetes/kubelet.conf
```
