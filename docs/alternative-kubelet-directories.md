# Alternative Kubelet Directories

## Overview

KubeVirt requires access to the kubelet directory to function properly. By default, KubeVirt assumes the kubelet directory is located at `/var/lib/kubelet`, which is the standard location for most Kubernetes distributions.

However, some Kubernetes distributions use non-standard kubelet directories:
- **k3s**: `/var/lib/rancher/k3s/agent/kubelet`
- **k0s**: `/var/lib/k0s/kubelet`
- **MicroK8s** (older versions): `/var/snap/microk8s/common/var/lib/kubelet`

Starting with KubeVirt version X.XX, you can configure the kubelet root directory to support these alternative distributions.

## Configuration

To configure an alternative kubelet directory, set the `KUBELET_ROOT_DIR` environment variable on the `virt-operator` deployment:

```bash
kubectl set env deployment/virt-operator -n kubevirt KUBELET_ROOT_DIR="/var/lib/rancher/k3s/agent/kubelet"
```

Alternatively, you can patch the deployment directly:

```bash
kubectl patch deployment virt-operator -n kubevirt --type='json' -p='[{
  "op": "add",
  "path": "/spec/template/spec/containers/0/env/-",
  "value": {
    "name": "KUBELET_ROOT_DIR",
    "value": "/var/lib/rancher/k3s/agent/kubelet"
  }
}]'
```

## Examples

### k3s

Set the environment variable for k3s installations:

```bash
kubectl set env deployment/virt-operator -n kubevirt KUBELET_ROOT_DIR="/var/lib/rancher/k3s/agent/kubelet"
```

### k0s

Set the environment variable for k0s installations:

```bash
kubectl set env deployment/virt-operator -n kubevirt KUBELET_ROOT_DIR="/var/lib/k0s/kubelet"
```

### Installation Time Configuration

If you're installing KubeVirt from manifests, you can set the environment variable before deploying:

1. Download the KubeVirt operator manifest:
   ```bash
   export VERSION=$(curl -s https://api.github.com/repos/kubevirt/kubevirt/releases/latest | jq -r .tag_name)
   kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml
   ```

2. Before creating the KubeVirt CR, set the environment variable:
   ```bash
   kubectl set env deployment/virt-operator -n kubevirt KUBELET_ROOT_DIR="/var/lib/rancher/k3s/agent/kubelet"
   ```

3. Create the KubeVirt CR:
   ```bash
   kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml
   ```

## Verification

After setting the `KUBELET_ROOT_DIR` environment variable, you can verify that the virt-handler DaemonSet has been configured correctly:

1. Wait for the virt-operator to restart and reconcile:
   ```bash
   kubectl rollout status deployment/virt-operator -n kubevirt
   ```

2. Check that the virt-handler pods are running:
   ```bash
   kubectl get pods -n kubevirt -l kubevirt.io=virt-handler
   ```

3. Verify the kubelet directory configuration in the virt-handler container:
   ```bash
   kubectl get DaemonSet virt-handler -n kubevirt -o yaml | grep -A 2 "kubelet-root"
   ```

   You should see arguments like:
   ```yaml
   - --kubelet-root
   - /var/lib/rancher/k3s/agent/kubelet
   - --kubelet-pods-dir
   - /var/lib/rancher/k3s/agent/kubelet/pods
   ```

4. Check the volume mounts:
   ```bash
   kubectl get DaemonSet virt-handler -n kubevirt -o yaml | grep -A 5 "volumeMounts:" | grep kubelet
   ```

## Troubleshooting

### Issue: virt-handler pods are not starting

If the virt-handler pods fail to start after configuring a custom kubelet directory, check:

1. **Incorrect kubelet directory**: Ensure the path specified in `KUBELET_ROOT_DIR` matches the actual kubelet directory on your nodes.

2. **Directory permissions**: The kubelet directory must be accessible from the virt-handler pod.

3. **Check virt-handler logs**:
   ```bash
   kubectl logs -n kubevirt -l kubevirt.io=virt-handler
   ```

### Issue: VMIs fail to start

If VMIs fail to start after changing the kubelet directory:

1. **Verify the configuration was applied**: Check that the virt-handler DaemonSet has been updated with the new configuration.

2. **Check the virt-operator environment**: Ensure the environment variable was set correctly:
   ```bash
   kubectl get deployment virt-operator -n kubevirt -o yaml | grep -A 2 KUBELET_ROOT_DIR
   ```

3. **Check for stale configurations**: If you previously installed KubeVirt without the custom kubelet directory, you may need to restart the virt-handler pods:
   ```bash
   kubectl delete pods -n kubevirt -l kubevirt.io=virt-handler
   ```

## Migration from Previous Workarounds

If you were previously using a workaround such as creating a symlink from `/var/lib/kubelet` to your distribution's kubelet directory, you can now remove that workaround:

1. Set the `KUBELET_ROOT_DIR` environment variable on the virt-operator deployment.
2. Wait for the virt-handler DaemonSet to be updated.
3. Remove the symlink from your nodes (optional, but recommended for cleanliness).

## Related Issues

This feature addresses the following GitHub issues:
- [#5913](https://github.com/kubevirt/kubevirt/issues/5913): Allow configuration of alternative kubelet directories
- [#5069](https://github.com/kubevirt/kubevirt/issues/5069): Related issue discussing kubelet path problems

## Technical Details

When you configure `KUBELET_ROOT_DIR`, KubeVirt automatically:
- Reads the environment variable from the virt-operator deployment
- Updates the virt-handler DaemonSet to mount the correct kubelet directory
- Passes the `--kubelet-root` and `--kubelet-pods-dir` flags to the virt-handler container
- Configures the seccomp profile installation path correctly

The configuration is read from the environment variable by the operator, which then generates the appropriate DaemonSet configuration for virt-handler.

