set -ex pipefail

source hack/common.sh
source cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

function prefetch-images::find_node_names() {
    if [[ $KUBEVIRT_PROVIDER == "external" ]] || [[ $KUBEVIRT_PROVIDER =~ kind.* ]] || [[ $KUBEVIRT_PROVIDER == "local" ]]; then
        echo "" # in case of external provider / kind we have no control over the nodes
    else
        local nodes=()
        nodes+=($(_kubectl get nodes -o name | sed "s#node/##g"))
        echo "${nodes[@]}"
    fi
}

# Given a list of images, find nodes, SSH into each node and execute a command to pull image
function prefetch-images::pull_on_nodes() {
    local -r containers_to_pull=$@
    local -r nodes=$(prefetch-images::find_node_names)
    local -r max_retry=10

    for node in ${nodes[@]}; do
        count=0
        until ${KUBEVIRT_PATH}cluster-up/ssh.sh ${node} "echo \"${containers_to_pull}\" | xargs \-\-max-args=1 sudo docker pull"; do
            count=$((count + 1))
            if [ $count -eq max_retry ]; then
                echo "Failed to 'docker pull' in ${node}" >&2
                exit 1
            fi
            # increase the sleep time with each retry to give it a bit more time in case of repeated failures
            sleep $count
        done
    done
}

# Given a list of images and tags, find nodes, SSH into each node and execute a command to tag image
function prefetch-images::tag_on_nodes() {
    local -r container_alias=$@
    local -r nodes=$(prefetch-images::find_node_names)
    local -r max_retry=10

    for node in ${nodes[@]}; do
        count=0
        until ${KUBEVIRT_PATH}cluster-up/ssh.sh ${node} "echo \"${container_alias}\" | xargs \-\-max-args=2 sudo docker tag"; do
            count=$((count + 1))
            if [ $count -eq max_retry ]; then
                echo "Failed to 'docker tag' in ${node}" >&2
                exit 1
            fi
            # increase the sleep time with each retry to give it a bit more time in case of repeated failures
            sleep $count
        done
    done
}
