# DRA Hostpath Test Driver

A Kubernetes Dynamic Resource Allocation (DRA) kubelet plugin for testing. Allocates a directory on the host (`/var/run/kubevirt/cdi/<claim>/<request>`) and exposes it through CDI to the container holding the claim. The directory path is provided via the `KUBEVIRT_HOSTPATH_PATH` environment variable and mounted into the container.

**Driver Name:** `test.kubevirt.io`

## Example Usage

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: resource.k8s.io/v1
kind: DeviceClass
metadata:
  name: test.kubevirt.io
spec:
  selectors:
    - cel:
        expression: device.driver == "test.kubevirt.io"
---
apiVersion: v1
kind: Pod
metadata:
  name: dra-hostpath-test
spec:
  resourceClaims:
    - name: hostpath-device
      resourceClaimTemplateName: hostpath-claim-template
  containers:
    - name: test
      image: busybox
      command: ["sh", "-c", "env | sort && ls -la $KUBEVIRT_HOSTPATH_PATH && sleep 3600"]
      resources:
        claims:
          - name: hostpath-device
            request: hostpath-req
---
apiVersion: resource.k8s.io/v1
kind: ResourceClaimTemplate
metadata:
  name: hostpath-claim-template
spec:
  spec:
    devices:
      requests:
        - name: hostpath-req
          exactly:
            count: 1
            deviceClassName: test.kubevirt.io
EOF
```

The pod will receive environment variables:
- `KUBEVIRT_HOSTPATH_DEVICE` - allocated device name
- `KUBEVIRT_HOSTPATH_PATH` - path to mounted directory
- `KUBEVIRT_HOSTPATH_REQUEST` - request name from the claim
