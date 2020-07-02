#!/bin/bash

set -xe

function eventually {
    timeout=30
    interval=5
    cmd=$@
    echo "Checking eventually $cmd"
    while ! $cmd; do
        sleep $interval
        timeout=$(( $timeout - $interval ))
        if [ $timeout -le 0 ]; then
            return 1
        fi
    done
}

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
eventually ./cluster-up/kubectl.sh apply -f pod.yaml
./cluster-up/kubectl.sh wait pod virt-launcher-dev-null --for condition=Ready --timeout=200s

# Run for 30 minutes writting to /dev/null every half a second
./cluster-up/kubectl.sh exec virt-launcher-dev-null -- bash -xe -c "cnt=0; while [ \$cnt -le 360 ]; do echo \$cnt; cnt=\$((\$cnt + 1)) ;sleep 0.5 && ls -la /dev/null && echo foo | tee /dev/null; done"
