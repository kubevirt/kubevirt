This Directory contains E2E Performance tests

## KWOK Performance tests

KWOK (Kubernetes WithOut Kubelet) is a high-performance, lightweight Kubernetes simulation and testing tool. It allows you to simulate a Kubernetes cluster environment without the need to run the kubelet or other node components. This makes it an efficient tool for quickly deploying and testing clusters, especially for scenarios that don't require full-fledged nodes.

### Workflow of the KWOK E2E test
* Create fake nodes
* Create fake VMIs
* Delete fake VMIs
* Collect metrics
* Create fake VMs
* Delete fake VMs
* Collect metrics
* Delete fake nodes

### Running tests locally

To run tests locally, you will first need to set up a Kubernetes cluster using KWOK and then deploy KubeVirt on the cluster
```bash
export KUBEVIRT_DEPLOY_KWOK = "true"
export KUBEVIRT_DEPLOY_PROMETHEUS = "true"

make cluster-up
```
Once the cluster is up, deploy KubeVirt by running the following command:
```bash
make cluster-sync
```
Open port-forward tunnel to connect to prometheus
```bash
kubectl port-forward -n monitoring svc/prometheus-nodeport 30007:9090
```
You can customize the number of nodes and virtual machines (VMs) you want to run the tests against. If not specified, the following default values will be used, NODE_COUNT = 100 and VM_COUNT = 1000.

```bash
export KWOK_NODE_COUNT = "10"
export VM_COUNT = "100"
```
With the environment set up, run the performance tests using the following command:
```bash
make kwok-perftest
```