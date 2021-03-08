# Conformance testing

KubeVirt releases a `kubevirt/conformance:<release>` image and
`conformance.yaml` manifest starting from `v0.33.0`. It can be executed as a
[Sonobuoy](https://sonobuoy.io/) plugin to verify if the underlying cluster
meets the basic needs to run KubeVirt.

By default tests with a `[Conformance]` tag will be executed and tests with a
`[Disruptive]` tag will be skipped. These tests will not destroy or modify
anything outside of the explicitly created test namespaces by the binary.

## Executing conformance tests for a specific release

To execute the conformance tests for a released conformance test suite, run:

```bash
VERSION=v0.33.0
sonobuoy run --plugin https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/conformance.yaml
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

## Executing conformace tests in the development environment

```bash
make conformance
```

To run without outside connectivity tests add the argument:

```bash
make conformance SKIP_OUTSIDE_CONN_TESTS=true
```

## Generate manifests

Conformance tests plugin manifest template under
`manifests/release/conformance.yaml` was generated with following commands:

```bash
sonobuoy gen plugin --name kubevirt-conformance --cmd /usr/bin/conformance --image IMAGE -f junit > manifests/release/conformance.yaml.in
sed -i 's#IMAGE#{{.DockerPrefix}}/conformance:{{.DockerTag}}#' manifests/release/conformance.yaml.in
```
