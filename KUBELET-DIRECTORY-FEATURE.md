# Alternative Kubelet Directories Feature - Implementation Summary

## Overview

This feature allows KubeVirt to be configured to work with Kubernetes distributions that use non-standard kubelet directories, addressing GitHub issue [#5913](https://github.com/kubevirt/kubevirt/issues/5913).

## Problem Statement

KubeVirt previously hardcoded the kubelet directory to `/var/lib/kubelet`, which is the standard location for most Kubernetes distributions. However, some distributions use alternative paths:
- **k3s**: `/var/lib/rancher/k3s/agent/kubelet`
- **k0s**: `/var/lib/k0s/kubelet`
- **MicroK8s** (older versions): `/var/snap/microk8s/common/var/lib/kubelet`

This prevented KubeVirt from working properly on these distributions without manual workarounds (like creating symlinks).

## Solution

Added a new `kubeletRootDir` configuration field to the KubeVirt CR that allows users to specify the kubelet root directory. When configured, KubeVirt automatically:
1. Mounts the correct kubelet directory in virt-handler pods
2. Passes the correct paths to virt-handler via command-line flags
3. Configures seccomp profile installation correctly

## Changes Made

### 1. API Changes (`staging/src/kubevirt.io/api/core/v1/types.go`)

Added `KubeletRootDir` field to `KubeVirtConfiguration`:

```go
// KubeletRootDir specifies the root directory for the kubelet on the node.
// Defaults to "/var/lib/kubelet" if not specified.
// This is useful for Kubernetes distributions that use non-standard kubelet directories
// (e.g., k3s uses "/var/lib/rancher/k3s/agent/kubelet", k0s uses "/var/lib/k0s/kubelet").
// +optional
KubeletRootDir            string               `json:"kubeletRootDir,omitempty"`
```

### 2. Operator Config (`pkg/virt-operator/util/config.go`)

Added:
- `GetKubeletRootDir()` method to return the configured kubelet root directory (defaults to `/var/lib/kubelet`)
- Extraction of `KubeletRootDir` from KubeVirt CR in `GetTargetConfigFromKVWithEnvVarManager()`

### 3. Daemonset Generation (`pkg/virt-operator/resource/generate/components/daemonsets.go`)

Modified `NewHandlerDaemonSet()` to:
- Accept `kubeletRootDir` parameter
- Use the parameter for volume mounts instead of hardcoded `util.KubeletRoot`
- Pass `--kubelet-root` and `--kubelet-pods-dir` flags to virt-handler

### 4. Strategy Integration (`pkg/virt-operator/resource/generate/install/strategy.go`)

Updated the call to `NewHandlerDaemonSet()` to pass `config.GetKubeletRootDir()`

### 5. Test Updates

Updated all test files that call `NewHandlerDaemonSet()`:
- `tools/util/marshaller_test.go`
- `pkg/virt-operator/resource/apply/apps_test.go`
- `pkg/virt-operator/kubevirt_test.go`

### 6. Documentation

Created:
- `docs/alternative-kubelet-directories.md` - Comprehensive documentation for the feature
- `examples/kubevirt-cr-with-custom-kubelet-root.yaml` - Example configurations for different distributions

## Usage

### For k3s:

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    kubeletRootDir: "/var/lib/rancher/k3s/agent/kubelet"
```

### For k0s:

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    kubeletRootDir: "/var/lib/k0s/kubelet"
```

## Verification

After applying the configuration:

1. Check virt-handler daemonset has the correct arguments:
   ```bash
   kubectl get daemonset virt-handler -n kubevirt -o yaml | grep -A 2 "kubelet-root"
   ```

2. Verify volume mounts:
   ```bash
   kubectl get daemonset virt-handler -n kubevirt -o yaml | grep -A 5 "volumeMounts:" | grep kubelet
   ```

## Backward Compatibility

The feature is fully backward compatible:
- If `kubeletRootDir` is not specified, it defaults to `/var/lib/kubelet`
- Existing KubeVirt deployments continue to work without any changes
- No breaking changes to the API

## Testing

All existing unit tests have been updated to work with the new parameter. The changes are covered by:
- Operator unit tests
- Component generation tests
- Integration tests (via existing test suite)

## Related Files

- **API**: `staging/src/kubevirt.io/api/core/v1/types.go`
- **Operator**: `pkg/virt-operator/util/config.go`
- **Components**: `pkg/virt-operator/resource/generate/components/daemonsets.go`
- **Strategy**: `pkg/virt-operator/resource/generate/install/strategy.go`
- **Tests**: `pkg/virt-operator/kubevirt_test.go`, `pkg/virt-operator/resource/apply/apps_test.go`, `tools/util/marshaller_test.go`
- **Docs**: `docs/alternative-kubelet-directories.md`
- **Examples**: `examples/kubevirt-cr-with-custom-kubelet-root.yaml`

## Migration from Workarounds

Users who previously worked around this limitation (e.g., by creating symlinks) can now:
1. Update their KubeVirt CR with the appropriate `kubeletRootDir` configuration
2. Wait for the virt-handler daemonset to be updated
3. Remove their workarounds (optional but recommended)

## Future Considerations

This implementation focuses on the virt-handler component. If other KubeVirt components need access to the kubelet directory in the future, they can use the same configuration pattern.

## References

- GitHub Issue: https://github.com/kubevirt/kubevirt/issues/5913
- Related Issue: https://github.com/kubevirt/kubevirt/issues/5069
- k0s Discussion: https://github.com/k0sproject/k0s/issues/428

