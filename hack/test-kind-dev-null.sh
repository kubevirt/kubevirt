#!/bin/bash

set -xe

export KUBEVIRT_PROVIDER=kind-k8s-1.17.0-ipv6
make cluster-up
cat <<'EOF' > pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: virt-launcher-dev-null
spec:
  containers:
  - name: virt-launcher-dev-null
    image: kubevirt/virt-launcher:v0.30.2
    command: ["/bin/bash"]
    args: ["-c", "sleep infinity"]
EOF
./cluster-up/kubectl.sh delete --ignore-not-found -f pod.yaml
./cluster-up/kubectl.sh apply -f pod.yaml
./cluster-up/kubectl.sh wait pod virt-launcher-dev-null --for condition=Ready
./cluster-up/kubectl.sh exec virt-launcher-dev-null -- bash -xe -c "ls -la /dev/null && echo foo > /dev/null"
