# Additional Virt-Handlers Demo

This demo shows how to configure additional virt-handler DaemonSets to serve
heterogeneous node pools with custom virt-handler and virt-launcher images.

## Prerequisites

- A KubeVirt cluster with multiple nodes
- The `AdditionalVirtHandlers` feature gate enabled
- Alternative virt-handler and virt-launcher images available

## Environment

This demo uses a local kubevirtci cluster with 3 nodes:

```
$ kubectl get nodes
NAME     STATUS   ROLES           AGE   VERSION
node01   Ready    control-plane   45m   v1.34.2
node02   Ready    worker          44m   v1.34.2
node03   Ready    worker          44m   v1.34.2
```

Available images:
- `registry:5000/kubevirt/virt-handler:devel` (default)
- `registry:5000/kubevirt/virt-handler:devel_alt` (alternative)
- `registry:5000/kubevirt/virt-launcher:devel` (default)
- `registry:5000/kubevirt/virt-launcher:devel_alt` (alternative)

## Step 1: Enable Feature Gate and Configure Additional Handler

Patch the KubeVirt CR to enable the feature gate and add an additional handler
configuration for an "alt-pool" targeting nodes with `kubevirt.io/alt-pool=true`:

```bash
kubectl patch kubevirt kubevirt -n kubevirt --type=merge -p '
{
  "spec": {
    "configuration": {
      "developerConfiguration": {
        "featureGates": ["AdditionalVirtHandlers"]
      }
    },
    "additionalVirtHandlers": [
      {
        "name": "alt-pool",
        "virtHandlerImage": "registry:5000/kubevirt/virt-handler:devel_alt",
        "virtLauncherImage": "registry:5000/kubevirt/virt-launcher:devel_alt",
        "nodeSelector": {
          "kubevirt.io/alt-pool": "true"
        }
      }
    ]
  }
}'
```

## Step 2: Verify Additional DaemonSet Created

After a few seconds, the additional DaemonSet should be created:

```
$ kubectl get ds -n kubevirt
NAME                    DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR                                      AGE
disks-images-provider   3         3         3       3            3           <none>                                             7m
virt-handler            3         3         3       3            3           kubernetes.io/os=linux                             6m
virt-handler-alt-pool   0         0         0       0            0           kubernetes.io/os=linux,kubevirt.io/alt-pool=true   1m
```

Note that `virt-handler-alt-pool` has DESIRED=0 because no nodes have the
`kubevirt.io/alt-pool=true` label yet.

## Step 3: Label a Node for the Alt-Pool

Label node03 to be part of the alt-pool:

```bash
kubectl label node node03 kubevirt.io/alt-pool=true
```

## Step 4: Verify Handler Distribution

After labeling the node, verify that:
- The primary virt-handler stops running on node03 (due to anti-affinity)
- The alt-pool virt-handler starts running on node03

```
$ kubectl get ds -n kubevirt
NAME                    DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR                                      AGE
disks-images-provider   3         3         3       3            3           <none>                                             8m
virt-handler            2         2         2       2            2           kubernetes.io/os=linux                             7m
virt-handler-alt-pool   1         1         1       1            1           kubernetes.io/os=linux,kubevirt.io/alt-pool=true   2m
```

Verify the pods and their images:

```
$ kubectl get pods -n kubevirt -l kubevirt.io=virt-handler -o custom-columns=NAME:.metadata.name,NODE:.spec.nodeName,IMAGE:.spec.containers[0].image
NAME                          NODE     IMAGE
virt-handler-xxxxx            node01   registry:5000/kubevirt/virt-handler:devel
virt-handler-yyyyy            node02   registry:5000/kubevirt/virt-handler:devel
virt-handler-alt-pool-zzzzz   node03   registry:5000/kubevirt/virt-handler:devel_alt
```

## Step 5: Verify Anti-Affinity on Primary Handler

The primary virt-handler DaemonSet should have anti-affinity rules to avoid
nodes targeted by additional handlers:

```
$ kubectl get ds virt-handler -n kubevirt -o jsonpath='{.spec.template.spec.affinity.nodeAffinity}'
{
  "requiredDuringSchedulingIgnoredDuringExecution": {
    "nodeSelectorTerms": [{
      "matchExpressions": [{
        "key": "kubevirt.io/alt-pool",
        "operator": "NotIn",
        "values": ["true"]
      }]
    }]
  }
}
```

## Step 6: Launch a VMI Targeting the Alt-Pool

Create a VMI with a node selector matching the alt-pool:

```bash
kubectl apply -f - <<'EOF'
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: test-vmi-alt-pool
  namespace: default
spec:
  nodeSelector:
    kubevirt.io/alt-pool: "true"
  domain:
    resources:
      requests:
        memory: 128Mi
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

## Step 7: Verify VMI Uses Custom Virt-Launcher Image

Check that the VMI is running on node03 and using the custom virt-launcher image:

```
$ kubectl get vmi -n default
NAME                AGE   PHASE     IP             NODENAME   READY
test-vmi-alt-pool   10s   Running   10.244.32.18   node03     True
```

Verify the virt-launcher pod uses the `devel_alt` image:

```
$ kubectl get pod -n default -l kubevirt.io=virt-launcher -o custom-columns=NAME:.metadata.name,IMAGE:.spec.containers[0].image
NAME                                    IMAGE
virt-launcher-test-vmi-alt-pool-xxxxx   registry:5000/kubevirt/virt-launcher:devel_alt
```

## Summary

| Component | Node | Image |
|-----------|------|-------|
| virt-handler (primary) | node01 | virt-handler:devel |
| virt-handler (primary) | node02 | virt-handler:devel |
| virt-handler-alt-pool | node03 | virt-handler:devel_alt |
| virt-launcher (VMI on alt-pool) | node03 | virt-launcher:devel_alt |

The Additional Virt-Handlers feature enables:

1. **Custom virt-handler images** per node pool for specialized hardware support
2. **Custom virt-launcher images** automatically selected for VMIs targeting those pools
3. **Automatic anti-affinity** to ensure each node runs only one virt-handler

## Cleanup

```bash
# Delete the test VMI
kubectl delete vmi test-vmi-alt-pool -n default

# Remove the node label
kubectl label node node03 kubevirt.io/alt-pool-

# Remove the additional handler configuration
kubectl patch kubevirt kubevirt -n kubevirt --type=json -p='[{"op": "remove", "path": "/spec/additionalVirtHandlers"}]'

# Disable the feature gate (optional)
kubectl patch kubevirt kubevirt -n kubevirt --type=json -p='[{"op": "remove", "path": "/spec/configuration/developerConfiguration/featureGates"}]'
```
