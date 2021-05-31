#!/bin/bash -xe

kubectl=./cluster-up/kubectl.sh

function wait-for-daemonset() {
    retries=10
    while [[ $retries -ge 0 ]]; do
        sleep 3
        ready=$($kubectl -n $1 get daemonset $2 -o jsonpath="{.status.numberReady}")
        required=$($kubectl -n $1 get daemonset $2 -o jsonpath="{.status.desiredNumberScheduled}")
        if [[ $ready -eq $required ]]; then
            #echo "Succeeded"
            break
        fi
        ((retries--))
    done
}

BAZEL=bazelisk PUSH_TARGETS="virt-controller virt-handler virt-launcher" ./hack/bazel-push-images.sh

registry_port=$(./cluster-up/cli.sh ports registry)
registry_url=localhost:$registry_port

handler_sha256=$(skopeo inspect docker://$registry_url/kubevirt/virt-handler:latest --tls-verify=false | jq ".Digest" -r)
$kubectl set env deployment -n kubevirt virt-operator VIRT_HANDLER_SHASUM=$handler_sha256

launcher_sha256=$(skopeo inspect docker://$registry_url/kubevirt/virt-launcher:latest --tls-verify=false | jq ".Digest" -r)
$kubectl set env deployment -n kubevirt virt-operator VIRT_LAUNCHER_SHASUM=$launcher_sha256

controller_sha256=$(skopeo inspect docker://$registry_url/kubevirt/virt-controller:latest --tls-verify=false | jq ".Digest" -r)
$kubectl set env deployment -n kubevirt virt-operator VIRT_CONTROLLER_SHASUM=$controller_sha256

# Force virt-handler restart to ensure is taking virt-launcher changes
$kubectl delete ds -n kubevirt virt-handler

while ! $kubectl get ds -n kubevirt virt-handler -o yaml | grep $handler_sha256; do
    sleep 5
done

while ! $kubectl get deployment -n kubevirt virt-controller -o yaml | grep $controller_sha256; do
    sleep 5
done

wait-for-daemonset kubevirt virt-handler

$kubectl wait -n kubevirt deployment virt-controller --for=condition=available --timeout=120s
