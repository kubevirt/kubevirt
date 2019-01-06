#!/bin/bash -e

source ./cluster/gocli.sh
source ./hack/build/config.sh

num_nodes=${KUBEVIRT_NUM_NODES:-1}
mem_size=${KUBEVIRT_MEMORY_SIZE:-5120M}

re='^-?[0-9]+$'
if ! [[ $num_nodes =~ $re ]] || [[ $num_nodes -lt 1 ]] ; then
    num_nodes=1
fi

image=$(getClusterType)
echo "Image:${image}"
if [[ $image == $KUBERNETES_IMAGE ]]; then
    $gocli run --random-ports --nodes ${num_nodes} --memory ${mem_size} --background kubevirtci/${image}
    cluster_port=$($gocli ports k8s | tr -d '\r')
    $gocli scp /usr/bin/kubectl - > ./cluster/.kubectl
    chmod u+x ./cluster/.kubectl
    $gocli scp /etc/kubernetes/admin.conf - > ./cluster/.kubeconfig
    export KUBECONFIG=./cluster/.kubeconfig
    ./cluster/.kubectl config set-cluster kubernetes --server=https://127.0.0.1:$cluster_port
    ./cluster/.kubectl config set-cluster kubernetes --insecure-skip-tls-verify=true

elif [[ $image == $OPENSHIFT_IMAGE ]]; then

    # If on a developer setup, expose ocp on 8443, so that the openshift web console can be used (the port is important because of auth redirects)
    if [ -z "${JOB_NAME}" ]; then
        KUBEVIRT_PROVIDER_EXTRA_ARGS="${KUBEVIRT_PROVIDER_EXTRA_ARGS} --ocp-port 8443"
    fi

    $gocli run --random-ports --reverse --nodes ${num_nodes} --memory ${mem_size} --background kubevirtci/${image} ${KUBEVIRT_PROVIDER_EXTRA_ARGS}
    cluster_port=$($gocli ports ocp | tr -d '\r')
    $gocli scp /etc/origin/master/admin.kubeconfig - > ./cluster/.kubeconfig
    $gocli ssh node01 -- sudo cp /etc/origin/master/admin.kubeconfig ~vagrant/
    $gocli ssh node01 -- sudo chown vagrant:vagrant ~vagrant/admin.kubeconfig

    # Copy oc tool and configuration file
    $gocli scp /usr/bin/oc - >./cluster/.kubectl
    chmod u+x ./cluster/.kubectl
    $gocli scp /etc/origin/master/admin.kubeconfig - > ./cluster/.kubeconfig
    # Update Kube config to support unsecured connection
    export KUBECONFIG=./cluster/.kubeconfig
    ./cluster/.kubectl config set-cluster node01:8443 --server=https://127.0.0.1:$cluster_port
    ./cluster/.kubectl config set-cluster node01:8443 --insecure-skip-tls-verify=true
fi

echo 'Wait until all nodes are ready'
until [[ $(./cluster/kubectl.sh get nodes --no-headers | wc -l) -eq $(./cluster/kubectl.sh get nodes --no-headers | grep " Ready" | wc -l) ]]; do
    sleep 1
done

