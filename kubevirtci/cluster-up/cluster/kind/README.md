# K8S in a Kind cluster

This folder serves as base to spin a k8s cluster up using [kind](https://github.com/kubernetes-sigs/kind) The cluster is completely ephemeral and is recreated on every cluster restart. 
The KubeVirt containers are built on the local machine and are then pushed to a registry which is exposed at
`localhost:5000`.

A kind cluster must specify:
* KIND_NODE_IMAGE referring the kind node image as one among those listed [here](https://hub.docker.com/r/kindest/node/tags) (please be aware that there might be compatibility issues between the kind executable and the node version)
* CLUSTER_NAME representing the cluster name 

The provider is supposed to copy a valid `kind.yaml` file under `${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml`

Check [kind-k8s-1.19](../kind-k8s-1.19) or [kind-1.22-sriov](kind-1.22-sriov) as examples on how to implement a kind cluster provider.
