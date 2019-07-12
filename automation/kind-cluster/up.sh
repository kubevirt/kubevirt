#!/bin/bash -e

wget https://github.com/kubernetes-sigs/kind/releases/download/v0.3.0/kind-linux-amd64 -O /usr/local/bin/kind
chmod +x /usr/local/bin/kind

function wait_kind_up {
    echo "Waiting for kind to be ready ..."  
    while [ -z "$(docker exec --privileged ${CLUSTER_CONTROL_PLANE} kubectl --kubeconfig=/etc/kubernetes/admin.conf get nodes --selector=node-role.kubernetes.io/master -o=jsonpath='{.items..status.conditions[-1:].status}' | grep True)" ]; do
        echo "Waiting for kind to be ready ..."        
        sleep 10
    done
    echo "Waiting for dns to be ready ..."        
    kubectl wait -n kube-system --timeout=12m --for=condition=Ready -l k8s-app=kube-dns pods
}

function wait_containers_ready {
    echo "Waiting for all containers to become ready ..."
    kubectl wait --for=condition=Ready pod --all -n kube-system --timeout 12m
}

kind --loglevel debug create cluster --retain --name=${CLUSTER_NAME} --config=${MANIFESTS_DIR}/kind.yaml
kubectl create -f $MANIFESTS_DIR/kube-flannel.yaml

wait_kind_up
kind get kubeconfig-path --name=${CLUSTER_NAME} #needed not for the env variable but to override the file with the current port
kubectl cluster-info

function configure-insecure-registry-and-reload() {
    local cmd_context="${1}" # context to run command e.g. sudo, docker exec
    ${cmd_context} "$(insecure-registry-config-cmd)"
    ${cmd_context} "$(reload-docker-daemon-cmd)"
}

function reload-docker-daemon-cmd() {
    echo "kill -s SIGHUP \$(pgrep dockerd)"
}

function insecure-registry-config-cmd() {
    echo "cat <<EOF > /etc/docker/daemon.json
{
    \"insecure-registries\": [\"${CONTAINER_REGISTRY_HOST}\"]
}
EOF
"
}

until kubectl get nodes --no-headers
do
    echo "Waiting for all nodes to become ready ..."
    sleep 10
done

# wait until k8s pods are running
while [ -n "$(kubectl get pods --all-namespaces --no-headers | grep -v Running)" ]; do
    echo "Waiting for all pods to enter the Running state ..."
    kubectl get pods --all-namespaces --no-headers | >&2 grep -v Running || true
    sleep 10
done

# wait until all containers are ready
wait_containers_ready

# Start local registry
configure-insecure-registry-and-reload "${CLUSTER_CMD} bash -c"

until [ -z "$(docker ps -a | grep registry)" ]; do
    docker stop registry || true
    docker rm registry || true
    sleep 5
done
docker run -d -p 5000:5000 --restart=always --name registry registry:2
${CLUSTER_CMD} socat TCP-LISTEN:5000,fork TCP:$(docker inspect --format '{{.NetworkSettings.IPAddress }}' registry):5000
