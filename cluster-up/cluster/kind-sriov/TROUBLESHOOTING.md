# How to troubleshoot a failing kind job

If logging and output artifacts are not enough, there is a way to connect to a running CI pod and troubleshoot directly from there.

## Pre-requisites

- A working (enabled) account on the [CI cluster](shift.ovirt.org), specifically enabled to the `kubevirt-prow-jobs` project.
- The [mkpj tool](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/mkpj) installed

## Launching a custom job

Through the `mkpj` tool, it's possible to craft a custom Prow Job that can be executed on the CI cluster. 

Just `go get` it by running `go get k8s.io/test-infra/prow/cmd/mkpj`

Then run the following command from a checkout of the [project-infra repo](https://github.com/kubevirt/project-infra):

```bash
mkpj --pull-number $KUBEVIRTPRNUMBER -job pull-kubevirt-e2e-kind-k8s-sriov-1.17.0 -job-config-path github/ci/prow/files/jobs/kubevirt/kubevirt-presubmits.yaml --config-path github/ci/prow/files/config.yaml > debugkind.yaml
```

You will end up having a ProwJob manifest in the `debugkind.yaml` file.

It's strongly recommended to replace the job's name, as it will be easier to find and debug the relative pod, by replacing `metadata.name` with something more recognizeable.

The $KUBEVIRTPRNUMBER can be an actual PR on the [kubevirt repo](https://github.com/kubevirt/kubevirt).

In case we just want to debug the cluster provided by the CI, it's recommended to override the entry point, either in the test PR we are instrumenting (a good sample can be found [here](https://github.com/kubevirt/kubevirt/pull/3022)), or by overriding the entry point directly in the prow job's manifest.

Remember that we want the cluster long living, so a long sleep must be provided as part of the entry point.

Make sure you switch to the `kubevirt-prow-jobs` project, and apply the manifest:

```bash
    kubectl apply -f debugkind.yaml
```

You will end up with a ProwJob object, and a pod with the same name you gave to the ProwJob.

Once the pod is up & running, connect to it via bash:

```bash
    kubectl exec -it debugprowjobpod bash
```

### Logistics

Once you are in the pod, you'll be able to troubleshoot what's happening in the environment CI is running its tests.

Run the follow to bring up a [kind](https://github.com/kubernetes-sigs/kind) cluster with a single node setup and the SR-IOV operator already setup to go (if it wasn't already done by the job itself).

```bash
KUBEVIRT_PROVIDER=kind-k8s-sriov-1.17.0 make cluster-up
```

The kubeconfig file will be available under `/root/.kube/kind-config-sriov`.

The `kubectl` binary is already on board and in `$PATH`.

The container acting as node is the one named `sriov-control-plane`. You can even see what's in there by running `docker exec -it sriov-control-plane bash`.
