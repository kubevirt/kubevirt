# DRA vfio-gpu for KubeVirt e2e

Provides fake `vfio-pci` devices for KubeVirt DRA e2e tests on the
virtualized `k8s-*` providers. The setup uses synthetic PCI devices from the
`fake-iommu` and `fake-pci` kernel modules, but builds and loads those modules
inside the virtualized worker nodes instead of on the host.

> Inspired by the approach explored in
> [kubevirt/kubevirt#16712](https://github.com/kubevirt/kubevirt/pull/16712)
> (synthetic kernel devices for GPU e2e testing without real hardware).

> The host still needs to be able to run kubevirtci `k8s-*` providers with KVM,
> but the fake VFIO kernel modules are loaded in the provider VMs.

> **WARNING:** loading and binding the fake VFIO kernel modules requires
> `root`. If you run the host-side module setup directly, use
> `sudo -E cluster-up/cluster/vfio-gpu/setup-host-vfio-pci.sh`. The `k8s-*`
> provider path loads the fake modules inside the provider worker nodes.

## Contents

| File / dir | Purpose |
| ---------- | ------- |
| `fake-iommu/` | Kernel module that exposes fake IOMMU groups |
| `fake-pci/` | Kernel module that publishes synthetic PCI devices on bus `faca` |
| `vfio-node/setup_node_vfio.sh` | Script copied into each worker node to build/load fake modules and bind devices to `vfio-pci` |
| `setup-fake-pci-host.sh` | Shared helper used inside the worker node to load/unload modules and bind devices |
| `setup-host-vfio-pci.sh` | Host-side helper for loading fake VFIO modules when testing against your own Kind cluster |
| `config_vfio_cluster.sh` | Post-create cluster setup: fake VFIO modules, node labels, DRA driver install |
| `install_dra_example_driver.sh` | Builds, pushes, and installs the DRA example driver |
| `../k8s-*/config_vfio_cluster.sh` | Provider wrapper gated by `KUBEVIRT_USE_FAKE_VFIO=true` |

## Prerequisites

Host tools: `docker` or `podman`, `kubectl`, `helm`, `git`, `make`, and a
kubevirtci `k8s-*` provider that can run nested virtualization.

Worker node packages: `make`, `gcc`, and matching kernel headers for the
running worker-node kernel. These must be present in the provider image; the
VFIO setup validates them before building the fake modules.

## Cluster setup


```bash
export KUBEVIRT_PROVIDER=k8s-1.36
export KUBEVIRT_USE_FAKE_VFIO=true
export FAKE_PCI_DEVICES=8
export FAKE_IOMMU=true

make cluster-up
```

`make cluster-up` creates the `k8s-*` provider and, when
`KUBEVIRT_USE_FAKE_VFIO=true`, runs the provider's `config_vfio_cluster.sh`
wrapper. That wrapper delegates to this directory and configures each worker
node by:

- copying the fake VFIO sources to `/tmp/fake-vfio`;
- building and loading `fake-iommu.ko` and `fake-pci.ko` inside the worker;
- binding fake PCI devices to `vfio-pci`;
- labeling nodes with `fake-vfio-capable=true`;
- building, pushing, and installing the DRA example driver.

CPU Manager must be enabled for the KubeVirt e2e setup. The `k8s-*` worker
bootstrap configures static CPU Manager policy for supported architectures.

To rerun only the VFIO/DRA setup against an existing cluster:

```bash
bash cluster-up/cluster/k8s-1.36/config_vfio_cluster.sh
```

## Kind cluster testing

If you want to test against your own Kind cluster, load and bind the fake VFIO
kernel modules on the host first:

```bash
sudo -E cluster-up/cluster/vfio-gpu/setup-host-vfio-pci.sh
```

Create the Kind cluster with `/dev/vfio` bind-mounted into the node container,
then make the VFIO control device writable inside the node:

```yaml
nodes:
- role: control-plane
  extraMounts:
  - hostPath: /dev/vfio/
    containerPath: /dev/vfio/
```

```bash
docker exec <kind-node> mount -o remount,rw /sys
docker exec <kind-node> chmod 666 /dev/vfio/vfio
```

The `k8s-*` `config_vfio_cluster.sh` flow does not run for an existing Kind
cluster unless a Kind provider explicitly calls it.

## Configuration

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `KUBEVIRT_USE_FAKE_VFIO` | `false` | Enables fake VFIO setup during `make cluster-up` for supported `k8s-*` providers |
| `FAKE_PCI_DEVICES` | `8` | Number of synthetic PCI devices to create inside each worker node |
| `FAKE_IOMMU` | `true` | Load the fake IOMMU companion so fake devices can bind to `vfio-pci` |
| `DRA_DRIVER_PROFILE` | `vfio-gpu` | DRA example driver profile installed by Helm |
| `DRA_DRIVER_NAME` | `vfio-gpu.example.com` | DRA driver name exposed to KubeVirt |
| `DRA_DRIVER_IMAGE_NAME` | `dra-example-driver` | Image name used for the locally built DRA driver |
