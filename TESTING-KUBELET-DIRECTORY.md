# Testing Alternative Kubelet Directories Feature

This guide explains how to test the alternative kubelet directories feature for KubeVirt.

## Prerequisites

- Access to a k3s or k0s Kubernetes cluster (or any distribution with a non-standard kubelet directory)
- kubectl configured to access the cluster
- Appropriate permissions to deploy KubeVirt

## Testing on k3s

### Step 1: Set up k3s

If you don't have k3s installed, install it:

```bash
curl -sfL https://get.k3s.io | sh -
```

### Step 2: Deploy KubeVirt with custom kubelet directory

Create a KubeVirt CR with the k3s kubelet directory:

```yaml
# kubevirt-k3s.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kubevirt
---
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    kubeletRootDir: "/var/lib/rancher/k3s/agent/kubelet"
  certificateRotateStrategy: {}
  imagePullPolicy: IfNotPresent
  workloadUpdateStrategy: {}
```

Apply it:

```bash
kubectl apply -f kubevirt-k3s.yaml
```

### Step 3: Deploy the KubeVirt operator

```bash
# Download and apply the latest KubeVirt operator manifest
export RELEASE=$(curl -s https://api.github.com/repos/kubevirt/kubevirt/releases/latest | jq -r .tag_name)
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${RELEASE}/kubevirt-operator.yaml
```

Or if testing locally built images:

```bash
# Build and deploy from source
make cluster-up
make cluster-sync
```

### Step 4: Verify the configuration

1. Check that virt-handler pods are running:
   ```bash
   kubectl get pods -n kubevirt -l kubevirt.io=virt-handler
   ```

2. Verify the kubelet-root flag is set correctly:
   ```bash
   kubectl get daemonset virt-handler -n kubevirt -o yaml | grep -A 2 "kubelet-root"
   ```

   You should see:
   ```yaml
   - --kubelet-root
   - /var/lib/rancher/k3s/agent/kubelet
   ```

3. Check the volume mounts:
   ```bash
   kubectl get daemonset virt-handler -n kubevirt -o yaml | grep -A 3 "name: kubelet"
   ```

   You should see:
   ```yaml
   - mountPath: /var/lib/rancher/k3s/agent/kubelet
     name: kubelet
   ```

4. Check the logs for any errors:
   ```bash
   kubectl logs -n kubevirt -l kubevirt.io=virt-handler --tail=50
   ```

### Step 5: Test VMI Creation

Create a test VMI:

```yaml
# test-vmi.yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: testvmi-ephemeral
spec:
  domain:
    devices:
      disks:
      - disk:
          bus: virtio
        name: containerdisk
      - disk:
          bus: virtio
        name: cloudinitdisk
    resources:
      requests:
        memory: 512M
  volumes:
  - containerDisk:
      image: quay.io/kubevirt/cirros-container-disk-demo
    name: containerdisk
  - cloudInitNoCloud:
      userData: |
        #!/bin/sh
        echo 'Hello from KubeVirt!'
    name: cloudinitdisk
```

Apply and check:

```bash
kubectl apply -f test-vmi.yaml
kubectl get vmi
kubectl get vmi testvmi-ephemeral -o yaml
```

### Step 6: Clean up

```bash
kubectl delete vmi testvmi-ephemeral
kubectl delete -f kubevirt-k3s.yaml
```

## Testing on k0s

### Step 1: Set up k0s

Install k0s:

```bash
curl -sSLf https://get.k0s.sh | sudo sh
sudo k0s install controller --single
sudo k0s start
```

### Step 2: Deploy KubeVirt with custom kubelet directory

Create a KubeVirt CR with the k0s kubelet directory:

```yaml
# kubevirt-k0s.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kubevirt
---
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    kubeletRootDir: "/var/lib/k0s/kubelet"
  certificateRotateStrategy: {}
  imagePullPolicy: IfNotPresent
  workloadUpdateStrategy: {}
```

Apply it:

```bash
sudo k0s kubectl apply -f kubevirt-k0s.yaml
```

### Steps 3-6: Same as k3s

Follow the same verification and testing steps as outlined for k3s, but use `sudo k0s kubectl` instead of `kubectl`.

## Testing with Standard Kubernetes (Backward Compatibility)

To ensure backward compatibility, test that the default behavior still works:

### Step 1: Deploy without specifying kubeletRootDir

```yaml
# kubevirt-default.yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  certificateRotateStrategy: {}
  imagePullPolicy: IfNotPresent
  workloadUpdateStrategy: {}
```

### Step 2: Verify default path is used

```bash
kubectl get daemonset virt-handler -n kubevirt -o yaml | grep -A 2 "kubelet-root"
```

You should see the default path:
```yaml
- --kubelet-root
- /var/lib/kubelet
```

## Troubleshooting

### Issue: virt-handler pods are CrashLooping

**Check:**
1. Verify the kubelet directory path is correct for your distribution:
   ```bash
   # On a k3s node
   ls -la /var/lib/rancher/k3s/agent/kubelet
   
   # On a k0s node
   ls -la /var/lib/k0s/kubelet
   ```

2. Check the virt-handler logs:
   ```bash
   kubectl logs -n kubevirt -l kubevirt.io=virt-handler
   ```

3. Verify the daemonset has the correct volume mounts:
   ```bash
   kubectl get daemonset virt-handler -n kubevirt -o yaml
   ```

### Issue: VMIs fail to start

**Check:**
1. Ensure virt-handler is running:
   ```bash
   kubectl get pods -n kubevirt -l kubevirt.io=virt-handler
   ```

2. Check VMI events:
   ```bash
   kubectl describe vmi <vmi-name>
   ```

3. Verify the kubelet directory contains the expected subdirectories:
   ```bash
   ls -la <kubelet-root-dir>/pods
   ```

### Issue: Seccomp profile errors

The seccomp profile installation depends on the kubelet root directory. If you see errors related to seccomp:

1. Check if the seccomp directory exists:
   ```bash
   ls -la <kubelet-root-dir>/seccomp/
   ```

2. Check virt-handler logs for seccomp-related errors:
   ```bash
   kubectl logs -n kubevirt -l kubevirt.io=virt-handler | grep seccomp
   ```

## Unit Tests

Run the existing unit tests to ensure no regressions:

```bash
# Run operator tests
make test

# Run specific component tests
go test ./pkg/virt-operator/resource/generate/components/...
go test ./pkg/virt-operator/util/...
```

## Integration Tests

Run the full integration test suite:

```bash
make cluster-up
make cluster-sync
make functest
```

## Cleanup

After testing, clean up the resources:

```bash
kubectl delete kubevirt kubevirt -n kubevirt
kubectl delete namespace kubevirt
```

## Reporting Issues

If you encounter any issues while testing:

1. Gather diagnostic information:
   ```bash
   kubectl get all -n kubevirt
   kubectl get kubevirt -n kubevirt -o yaml
   kubectl logs -n kubevirt -l kubevirt.io=virt-handler
   kubectl describe daemonset virt-handler -n kubevirt
   ```

2. Report the issue on GitHub: https://github.com/kubevirt/kubevirt/issues/5913

