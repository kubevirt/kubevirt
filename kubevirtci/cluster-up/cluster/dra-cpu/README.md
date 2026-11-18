# DRA CPUs for KubeVirt e2e

Installs [dra-driver-cpu](https://github.com/kubernetes-sigs/dra-driver-cpu) on a running cluster so
KubeVirt's `CPUsWithDRA` e2e tests can run against a real DRA driver.

## Contents

| File | Purpose |
| ---- | ------- |
| `install_dra_driver_cpu.sh` | Installs the upstream `dra-driver-cpu` Helm chart and waits for it to publish its CPU topology |

## Prerequisites

| Requirement | Minimum | Status on `k8s-*` providers |
| ----------- | ------- | --------------------------- |
| Kubernetes | 1.36, so that `DRAConsumableCapacity` is beta and on by default | met by `k8s-1.36` and `k8s-1.37` |
| Container runtime | CRI-O 1.30 or containerd 2.0, for NRI and CDI | met; CRI-O enables both by default |
| Kubelet | `cpuManagerPolicy: none` | met; this is the kubelet default |

KubeVirt's claims consume `dra.cpu/cpu` capacity from the driver's `grouped` device mode, which is
what needs `DRAConsumableCapacity`. On Kubernetes 1.34 and 1.35 that gate is alpha and would have
to be enabled on the apiserver, scheduler, controller-manager and kubelet, so stick to 1.36+.

Host tools: `kubectl`, `curl`, and `helm` (downloaded on demand if missing).

## Running the tests

```bash
export KUBEVIRT_PROVIDER=k8s-1.37
export KUBEVIRT_USE_DRA_CPU=true
export KUBEVIRT_NUM_NUMA_NODES=2
export KUBEVIRT_NUM_VCPU=12

# installs the driver as part of cluster-up, because KUBEVIRT_USE_DRA_CPU is true
make cluster-up
make cluster-sync

./kubevirtci/cluster-up/kubectl.sh patch kubevirt kubevirt -n kubevirt --type merge \
  -p '{"spec":{"configuration":{"developerConfiguration":{"featureGates":["CPUsWithDRA"]}}}}'

FUNC_TEST_LABEL_FILTER="--label-filter=(DRA-CPU)" make functest
```

`KUBEVIRT_USE_DRA_CPU` is the only switch. The `up()` function in `k8s-provider-common.sh` runs
the install script when it is `true`, the same way `KUBEVIRT_USE_FAKE_VFIO` pulls in the fake VFIO
setup, so there is no separate install step. The script also checks the flag itself and does
nothing when it is unset, so invoking it by hand on a cluster that does not want the driver is
harmless.

In CI the `sig-compute-dra-cpu` lane sets the flag along with the NUMA sizing, enables the feature
gate through `add_feature_gate`, and filters on the `DRA-CPU` label. Any lane can opt in by
exporting the flag.

## Sizing the nodes

The driver's default `grouped` mode publishes one device per NUMA cell, and a KubeVirt CPU claim
asks for all of its CPUs from a *single* device. So the largest VM that can run is bounded by the
allocatable CPUs of one cell, not of the whole node:

```
per-cell allocatable = KUBEVIRT_NUM_VCPU / KUBEVIRT_NUM_NUMA_NODES - (reserved CPUs in that cell)
```

With `KUBEVIRT_NUM_VCPU=12` and `KUBEVIRT_NUM_NUMA_NODES=2` each cell holds six CPUs. The default
`DRA_CPU_RESERVED_CPUS=0-1` falls entirely in cell 0, leaving four allocatable there and six in
cell 1 - room for a four-vCPU VM with an isolated emulator thread.

Leaving `KUBEVIRT_NUM_NUMA_NODES` at its default of 1 works, but makes the default grouping
degenerate: the driver publishes one device spanning the machine, and nothing about NUMA grouping
gets exercised.

## Configuration

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `KUBEVIRT_USE_DRA_CPU` | `false` | Must be `true` for the driver to be installed during `make cluster-up`. Anything else makes the install script a no-op |
| `DRA_CPU_RESERVED_CPUS` | `0-1` | CPUs the driver never hands out, left for kubelet, the CRI and the KubeVirt infra pods |
| `DRA_CPU_GROUP_BY` | chart default (`numanode`) | How the driver groups CPUs into devices: `numanode`, `socket` or `machine` |
| `DRA_CPU_CHART` | `oci://registry.k8s.io/dra-driver-cpu/charts/dra-driver-cpu` | Helm chart to install |
| `DRA_CPU_CHART_VERSION` | `0.3.0` | Chart version |
| `DRA_CPU_DRIVER_NAMESPACE` | `dra-driver-cpu` | Namespace the driver is installed into |

## A note on the non-DRA dedicated-CPU tests

KubeVirt's dedicated-CPU tests that do *not* use DRA need a node labelled
`kubevirt.io/cpumanager=true`, which virt-handler only sets where kubelet's policy is `static`. A
`k8s-*` cluster has no such node either way, which is why those tests carry the
`requires-node-with-cpu-manager` label and are filtered out. Installing this driver does not change
that.
