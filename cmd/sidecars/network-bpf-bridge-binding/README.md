# KubeVirt Network BPF Bridge Binding Plugin

## Summary

`network-bpf-bridge-binding` is a KubeVirt sidecar binding plugin arranged to match the
`cmd/sidecars/<plugin>` layout used by the KubeVirt repository.

The sidecar:

- creates a TAP device and a helper veth pair in the virt-launcher pod network namespace;
- loads a prebuilt `tc` eBPF program and configures it with the concrete interface indexes;
- attaches ingress `tc` hooks to forward traffic between TAP and veth without a Linux bridge;
- rewrites the libvirt domain XML so the VM NIC targets the TAP device.

## Repository layout

The code is intentionally colocated the same way as the upstream sidecars under
`cmd/sidecars/network-passt-binding`:

- `main.go` contains the sidecar entrypoint;
- `server/` implements the hook gRPC server;
- `domain/` mutates the libvirt domain;
- `callback/` handles domain XML marshal/unmarshal;
- `netsetup/` creates TAP and veth wiring;
- `bpfattach/` loads and attaches the `tc` eBPF program;
- `bpf/` stores the eBPF sources used by the image build.

## How to use

Register the binding plugin with a sidecar image:

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    network:
      binding:
        bpfbridge:
          sidecarImage: quay.io/example/network-bpf-bridge-binding:devel
          domainAttachmentType: tap
```

In the VM spec, bind an interface with `binding.name: bpfbridge`.

## Notes for embedding into `3p-kubevirt`

- The Go packages and file layout now match the expected `cmd/sidecars` structure.
- Internal imports now target `kubevirt.io/kubevirt/cmd/sidecars/network-bpf-bridge-binding/...`
  directly, so the directory is ready to be moved into the KubeVirt tree.
- `BUILD.bazel` includes a Bazel image target and packages the compiled `bpf_bridge.o` into
  `/opt/network-bpf-bridge-binding/bpf_bridge.o`.
- This repository's standalone Go module is now only a staging area for the sidecar sources; the
  authoritative build context for these packages is the target KubeVirt repository.
