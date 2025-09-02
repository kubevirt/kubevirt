# Conformance testing

The Conformance tests are a subset of our e2e test suite that cover a set of features
defining Kubevirt core functionalities. The goal of the suite is to make sure that selected features work across different Vendors, CNIs, CSIs, CRIs implementations.

## How are Conformance tests selected

By default tests with a `conformance` decorator will be executed and tests with a
`[Disruptive]` tag will be skipped. These tests will not destroy or modify
anything outside of the explicitly created test namespaces by the binary.

The tests are always part of Kubevirt e2e tests that are usually approved by individual [Sigs](https://github.com/kubevirt/community/blob/main/sig-list.md#special-interest-groups). For the Conformance tests we have higher requirements than the regular tests, most significant requirements are:
- Test is not Disruptive
- Test is not flaky and is reliable per data available from CI team
- For more concrete details see below


## Conformance tests requirements:
- The feature being tested is GA (no feature gate)
- The feature is not deprecated
- The feature is turned on by default (no feature toggle)
- The test uses only GAed Kubernetes APIs/features
- The test uses only APIs available to user
- The test doesn't use direct access to virt-launcher/libvirt
- It doesn't require any special hardware (such as gpu/pci devices,...) being available on the cluster other than the default required device nodes (kvm, virtio-net)
- It doesn't use `Skip` or any other helper that uses it
- It does not require access to public network, all images or resources should be able to be pre-pulled
- It does not rely on environment of the executor
- It has history of stability and is not flaky, and consequently doesn't have the QUARANTINE label.
- It doesn't rely on specific implementation of CRI, CSI, CNI

Exceptions are allowed but must be explicit and reasoned.

## Delivery

KubeVirt releases a `kubevirt/conformance:<release>` image and
`conformance.yaml` manifest starting from `v0.33.0`. It can be executed as a
[Sonobuoy](https://sonobuoy.io/) plugin to verify if the underlying cluster
meets the basic needs to run KubeVirt.

## Executing conformance tests for a specific release

To execute the conformance tests for a released conformance test suite, run:

```bash
KUBEVIRT_VERSION=v0.41.0
sonobuoy run --plugin https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/conformance.yaml
```

The execution can be monitored using the `status` command:

```bash
sonobuoy status
```

```
                 PLUGIN     STATUS   RESULT   COUNT
   kubevirt-conformance   complete   passed       1

Sonobuoy has completed. Use `sonobuoy retrieve` to get results.
```

Once the test run finishes, the result can be fetched:

```bash
sonobuoy retrieve
```

```
202008201609_sonobuoy_8f8d0b0e-1d37-485a-b61d-bf7185198fbf.tar.gz
```

And interpreted:

```bash
sonobuoy results 202008201609_sonobuoy_8f8d0b0e-1d37-485a-b61d-bf7185198fbf.tar.gz
```

```
Plugin: kubevirt-conformance
Status: passed
Total: 580
Passed: 1
Failed: 0
Skipped: 579
```

### Debugging failures

In case of a failure the artifacts will be collected as well.  
Find them at `plugins/kubevirt-conformance/results/global/k8s-reporter` of the tar file.

## Executing conformance tests in the development environment

```bash
make conformance
```

To run without outside connectivity tests add the following environment variable:

```bash
SKIP_OUTSIDE_CONN_TESTS=true make conformance
```

In case one does not have block storage/snapshot support in the cluster,  
It's possible to run without block storage/snapshot tests via these environment variables:

```bash
SKIP_BLOCK_STORAGE_TESTS=true SKIP_SNAPSHOT_STORAGE_TESTS=true make conformance
```

The following environment variable is only used for running the conformance tests on the Arm64 test infrastructure in KubeVirtCI.
This is necessary because some specific setups, such as IPv6, are not enabled on the Arm64 test infrastructure.
By adding the environment variable, we can skip the unsupported tests.

```bash
RUN_ON_ARM64_INFRA=true make conformance
```

To focus on specific tests pass KUBEVIRT_E2E_FOCUS environment variable:

```bash
KUBEVIRT_E2E_FOCUS=sig-network make conformance
```

To use on specific images tag on test suite override DOCKER_TAG environment variable:

```bash
DOCKER_TAG=mybuild make conformance
```

## Generate manifests

Conformance tests plugin manifest template under
`manifests/release/conformance.yaml` was generated with following commands:

```bash
sonobuoy gen plugin --name kubevirt-conformance --cmd /usr/bin/conformance --image IMAGE -f junit > manifests/release/conformance.yaml.in
sed -i 's#IMAGE#{{.DockerPrefix}}/conformance:{{.DockerTag}}#' manifests/release/conformance.yaml.in
```
