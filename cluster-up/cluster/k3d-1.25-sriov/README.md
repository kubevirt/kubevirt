# K8s 1.25.x with SR-IOV in a K3d cluster

Provides a pre-deployed containerized k8s cluster with version 1.25.x that runs
using [K3d](https://github.com/k3d-io/k3d)
The cluster is completely ephemeral and is recreated on every cluster restart. The KubeVirt containers are built on the
local machine and are then pushed to a registry which is exposed at
`127.0.0.1:5000`.

This version requires to have SR-IOV enabled nics (SR-IOV Physical Function) on the current host, and will move
physical interfaces into the `K3d`'s cluster agent node(s) (agent node is a worker node on k3d terminology)
so that they can be used through multus and SR-IOV
components.

This provider also deploys [multus](https://github.com/k8snetworkplumbingwg/multus-cni)
, [sriov-cni](https://github.com/k8snetworkplumbingwg/sriov-cni)
and [sriov-device-plugin](https://github.com/k8snetworkplumbingwg/sriov-network-device-plugin).

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=k3d-1.25-sriov
export KUBECONFIG=$(realpath _ci-configs/k3d-1.25-sriov/.kubeconfig)
make cluster-up
```
```
$ kubectl get nodes
NAME                 STATUS   ROLES                  AGE   VERSION
k3d-sriov-server-0   Ready    control-plane,master   67m   v1.25.6+k3s1
k3d-sriov-agent-0    Ready    worker                 67m   v1.25.6+k3s1
k3d-sriov-agent-1    Ready    worker                 67m   v1.25.6+k3s1

$ kubectl get pods -n kube-system -l app=multus
NAME                   READY   STATUS    RESTARTS   AGE
kube-multus-ds-z9hvs   1/1     Running   0          66m
kube-multus-ds-7shgv   1/1     Running   0          66m
kube-multus-ds-l49xj   1/1     Running   0          66m

$ kubectl get pods -n sriov -l app=sriov-cni
NAME                            READY   STATUS    RESTARTS   AGE
kube-sriov-cni-ds-amd64-4pndd   1/1     Running   0          66m
kube-sriov-cni-ds-amd64-68nhh   1/1     Running   0          65m

$ kubectl get pods -n sriov -l app=sriovdp
NAME                                   READY   STATUS    RESTARTS   AGE
kube-sriov-device-plugin-amd64-qk66v   1/1     Running   0          66m
kube-sriov-device-plugin-amd64-d5r5b   1/1     Running   0          65m
```

### Conneting to a node
```bash
export KUBEVIRT_PROVIDER=k3d-1.25-sriov
./cluster-up/ssh.sh <node_name> /bin/sh
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=k3d-1.25-sriov
make cluster-down
```

This destroys the whole cluster, and gracefully moves the SR-IOV nics to the root network namespace.

Note: killing the containers / cluster without gracefully moving the nics to the root ns before it,
might result in unreachable nics for few minutes.
`find /sys/class/net/*/device/sriov_numvfs` can be used to see when the nics are reachable again.

## Using podman
Podman v4 is required.

Run:
```bash
systemctl enable --now podman.socket
ln -s /run/podman/podman.sock /var/run/docker.sock
```
The rest is as usual.
For more info see https://k3d.io/v5.4.1/usage/advanced/podman.

## Updating the provider

### Bumping K3D
Update `K3D_TAG` (see `cluster-up/cluster/k3d/common.sh` for more info)

### Bumping CNI
Update `CNI_VERSION` (see `cluster-up/cluster/k3d/common.sh` for more info)

### Bumping Multus
Download the newer manifest `https://github.com/k8snetworkplumbingwg/multus-cni/blob/master/deployments/multus-daemonset-crio.yml`
replace this file `cluster-up/cluster/$KUBEVIRT_PROVIDER/sriov-components/manifests/multus/multus.yaml`
and update the kustomization file `cluster-up/cluster/$KUBEVIRT_PROVIDER/sriov-components/manifests/multus/kustomization.yaml`
according needs.

### Bumping calico
1. Fetch new calico yaml (https://docs.tigera.io/calico/3.25/getting-started/kubernetes/k3s/quickstart)
   Enable `allow_ip_forwarding` (See https://k3d.io/v5.4.7/usage/advanced/calico)
   Or use the one that is suggested here https://k3d.io/v5.4.7/usage/advanced/calico whenever it is updated.
2. Prefix the images in the yaml with `quay.io/` unless they have it already.
3. Update `cluster-up/cluster/k3d/manifests/calico.yaml` (see `CALICO` at `cluster-up/cluster/k3d/common.sh` for more info)

Note: Make sure to follow the latest verions on the links above.