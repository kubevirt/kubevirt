# VEP #200: Worker Pools Demo

This demo shows the WorkerPool CRD lifecycle on a 2-node kubevirtci cluster,
with pool DaemonSets running custom virt-handler images, anti-affinity
ensuring one handler per node, and label-based VMI matching selecting a
custom virt-launcher image.

## Prerequisites

```bash
KUBEVIRT_NUM_NODES=2 make cluster-up cluster-sync
```

Enable the feature gate:

```bash
kubectl patch kubevirt kubevirt -n kubevirt --type merge \
  -p '{"spec":{"configuration":{"developerConfiguration":{"featureGates":["WorkerPools"]}}}}'
kubectl wait --for=jsonpath='{.status.phase}'=Deployed kubevirt/kubevirt -n kubevirt --timeout=120s
```

Verify the cluster — two nodes, default virt-handler running on both:

```
$ kubectl get pods -n kubevirt -l kubevirt.io=virt-handler \
    -o custom-columns='NAME:.metadata.name,NODE:.spec.nodeName,IMAGE:.spec.containers[0].image'
NAME                 NODE     IMAGE
virt-handler-mzt2l   node01   registry:5000/kubevirt/virt-handler:devel
virt-handler-trfhh   node02   registry:5000/kubevirt/virt-handler:devel
```

## 1. Create a GPU pool with a custom image on node02

Label `node02` as a GPU node and create a WorkerPool using the `devel_alt`
handler and launcher images. This pool uses a **VM label selector** — any VMI
with the label `workload: gpu` will be matched to this pool and receive the
custom virt-launcher image:

```bash
kubectl label node node02 node-role.kubernetes.io/gpu=true

kubectl apply -f - <<'EOF'
apiVersion: worker.kubevirt.io/v1alpha1
kind: WorkerPool
metadata:
  name: gpu-pool
spec:
  virtHandlerImage: registry:5000/kubevirt/virt-handler:devel_alt
  virtLauncherImage: registry:5000/kubevirt/virt-launcher:devel_alt
  nodeSelector:
    node-role.kubernetes.io/gpu: "true"
  selector:
    vmLabels:
      matchLabels:
        workload: gpu
EOF
```

The operator creates `virt-handler-gpu-pool` on `node02` and applies
anti-affinity to the primary handler, removing it from `node02`. Each node
now runs exactly **one** virt-handler:

```
$ kubectl get daemonsets -n kubevirt -l kubevirt.io=virt-handler
NAME                    DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR                                             AGE
virt-handler            1         1         1       1            1           kubernetes.io/os=linux                                    3m
virt-handler-gpu-pool   1         1         1       1            1           kubernetes.io/os=linux,node-role.kubernetes.io/gpu=true   31s

$ kubectl get pods -n kubevirt -l kubevirt.io=virt-handler \
    -o custom-columns='NAME:.metadata.name,READY:.status.containerStatuses[0].ready,NODE:.spec.nodeName,IMAGE:.spec.containers[0].image'
NAME                          READY   NODE     IMAGE
virt-handler-gpu-pool-swfsd   true    node02   registry:5000/kubevirt/virt-handler:devel_alt
virt-handler-mzt2l            true    node01   registry:5000/kubevirt/virt-handler:devel
```

Note: the primary handler's `DESIRED` dropped from 2 to 1 — anti-affinity
excludes it from `node02` (the GPU pool node).

## 2. Launch a VMI that matches the pool

Create a VMI with the `workload: gpu` label. The virt-controller matches it
to `gpu-pool` and automatically uses the pool's custom `virt-launcher` image:

```bash
kubectl apply -f - <<'EOF'
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: gpu-vm
  labels:
    workload: gpu
spec:
  domain:
    resources:
      requests:
        memory: 64Mi
    devices:
      disks:
        - name: containerdisk
          disk:
            bus: virtio
  volumes:
    - name: containerdisk
      containerDisk:
        image: registry:5000/kubevirt/alpine-container-disk-demo:devel
EOF
```

The VMI starts on `node02` using the pool's `devel_alt` launcher image:

```
$ kubectl get vmi gpu-vm
NAME     AGE   PHASE     IP             NODENAME   READY
gpu-vm   10s   Running   10.244.16.49   node02     True

$ kubectl get pods -l kubevirt.io=virt-launcher \
    -o custom-columns='NAME:.metadata.name,NODE:.spec.nodeName,IMAGE:.spec.containers[0].image'
NAME                         NODE     IMAGE
virt-launcher-gpu-vm-7pcl5   node02   registry:5000/kubevirt/virt-launcher:devel_alt
```

The virt-launcher pod is annotated with the pool name:

```
$ kubectl get pods -l kubevirt.io=virt-launcher \
    -o jsonpath='{.items[0].metadata.annotations.kubevirt\.io/worker-pool}'
gpu-pool
```

## 3. Delete the pool — primary handler returns

```bash
kubectl delete vmi gpu-vm
kubectl delete workerpool gpu-pool
```

The pool DaemonSet is removed from `node02` and the primary handler returns
to serve it with the default image.

## 4. Cleanup

```bash
kubectl delete workerpool --all
kubectl label node node02 node-role.kubernetes.io/gpu-
```
