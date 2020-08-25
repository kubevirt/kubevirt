# Conformance testing

KubeVirt releases a `kubevirt/conformance:<release>` binary starting from
`v0.33.0`. It can be executed as a [Sonobuoy](https://sonobuoy.io/) plugin to
verify if the underlying cluster meets the basic needs to run KubeVirt.

By default tests with a `[Conformance]` tag will be executed and tests with a
`[Disruptive]` tag will be skipped. Therefore the tests will not destroy or
modify anything outside of the explicitly created test namespaces by the
binary.

## Executing conformance tests for a specific release

To execute the conformance tests for a released conformance test suite, run

```bash
$ sonobuoy gen plugin --name kubevirt-conformance --cmd /usr/bin/conformance --image kubevirt/conformance:<tag> -f junit > kubevirt.yaml
```
Replace `<tag>` with the kubevirt version which you have installed.

Then launch the suite:

```bash
$ sonobuoy run --plugin kubevirt.yaml
```

## Executing conformace tests in the development environment

Here a full flow in the development environment (note the `registry:500` prefix and the `:devel` tag:

```bash
$ sonobuoy gen plugin --name kubevirt-conformance --cmd /usr/bin/conformance --image registry:5000/kubevirt/conformance:devel -f junit > conformance.yaml
sonobuoy gen plugin --name kubevirt-conformance --cmd /usr/bin/conformance --image registry:5000/kubevirt/conformance:devel -f junit > conformance.yaml
$ sonobuoy retrieve
202008201609_sonobuoy_8f8d0b0e-1d37-485a-b61d-bf7185198fbf.tar.gz
$ sonobuoy results 202008201609_sonobuoy_8f8d0b0e-1d37-485a-b61d-bf7185198fbf.tar.gz
Plugin: kubevirt-conformance
Status: passed
Total: 580
Passed: 1
Failed: 0
Skipped: 579
```
