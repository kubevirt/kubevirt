This Directory contains E2E Performance tests

## virt-handler start-pool measurement

Cold starts used to hold one of the 10 virt-handler VM reconcile workers
across the whole `SyncVMI` gRPC. The start pool in `pkg/virt-handler`
moves that work off the worker.

Three places keep that honest:

* Unit tests next to the code: `pkg/virt-handler/start_pool_test.go` and
  the "should not hold the reconcile worker for the whole VM start" spec
  in `pkg/virt-handler/vm_test.go`. These run in ordinary `go test` / Prow.
* This directory's density suite (`density.go`) still measures
  creation-to-running for 100 container-disk VMIs on the periodic. Set
  `KUBEVIRT_PERF_VMI_COUNT` to run a smaller batch locally.
* `hack/virt-handler-start-perf.sh` is the A/B harness, not a Prow job.
  It cluster-syncs two SHAs on one kubevirtci node, creates 1/10/30 cirros
  VMIs with kubectl, and scrapes Prometheus. `make perftest` AfterSuite
  dies on TLS handshake timeout against a nested kubevirtci API; the
  script does not use ginkgo.

The metric that showed the pool working is
`kubevirt_workqueue_work_duration_seconds` for `name="virt-handler-vm"`.
Periodic density should grow to record that P95 so the number does not
live only in a local TSV. This PR does not change the 45s/60s density
gates.

```bash
export KUBEVIRT_PROVIDER=k8s-1.37
export KUBEVIRT_DEPLOY_PROMETHEUS=true
# 32G: 16G fills at ~21 virt-launchers (455Mi request each)
hack/virt-handler-start-perf.sh --fresh
```

Results go to `virt-handler-start-perf-results/<id>/results.tsv` in the
worktree. Run the script from a worktree of this branch; it refuses the
primary clone.

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