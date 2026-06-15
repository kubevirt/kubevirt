# Pre-Migration Target-Side Hooks

## Overview

During live migration, the domain XML that describes the VM must be adjusted
to match the target node's environment. Historically, these modifications were
applied on the source virt-launcher pod before the XML was sent to the
target. This is problematic because during rolling upgrades, only the target
pod runs the new virt-launcher image while the source pod still runs the old
one. If a bug fix is applied to the XML modification logic, it will not take
effect until the source pod is also updated, which may not happen until the
second migration or restart.

To address this, VEP-141 introduces **target-side pre-migration hooks**,
built on top of [libvirt's QEMU hook mechanism](https://libvirt.org/hooks.html).
Libvirt invokes the `/etc/libvirt/hooks/qemu` executable on the destination host
with the `migrate` operation at the beginning of incoming migration, passing
the domain XML via stdin and reading the modified XML from stdout. KubeVirt
places its hook client binary at that path, which forwards the domain XML to
a hook server running on the target virt-launcher pod over a unix socket.
The hook server applies the registered modifications and returns the updated
XML before the VM starts on the target node.
With target-side hooks, target-based XML modifications are applied by
the target pod itself, which always runs the latest virt-launcher image during upgrade.

## Architecture

The pre-migration hook system consists of:

1. **Hook Server** (`pkg/virt-launcher/premigration-hook-server/hook_server.go`):
   Runs on the target virt-launcher pod. Listens on a unix socket, receives
   the domain XML, applies all registered hooks, and returns the modified XML.

2. **Hook Functions**: Each hook implements the `HookFunc` signature:
   ```go
   type HookFunc func(c *ConverterContext, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error
   ```
   Hooks modify the `domain` in-place. They are called in registration order.
   The `domain` is the source XML from libvirt. The `vmi` is passed
   from virt-handler on the target pod, and the `ConverterContext` is
   generated locally on the target pod.

3. **Hook Client** (`cmd/virt-launcher/libvirt-hook-client/main.go`):
   Triggered by libvirt's `qemu` hook during the `migrate` operation
   on the destination host. The hook client connects to the hook server's
   unix socket, sends the domain XML for modification, and returns the
   modified XML back to libvirt.

## Migration Flow

```
Source virt-launcher                          Target virt-launcher
─────────────────                          ──────────────────────
                                            Start hook server
                                            (listen on unix socket)
                                                   │
Prepare domain XML ──── send XML ──────────────▶  │
                                            Receive XML
                                            Apply hooks:
                                              1. disk source path
                                              2. CPU
                                              3. vGPU
                                              4. network naming
                                            Return modified XML
         ◀──────────── receive XML ────────────  │
Continue migration
with modified XML
```

## Adding a New Hook

1. Create a new package under `pkg/virt-launcher/premigration-hook-server/`.
2. Implement a function matching the `HookFunc` signature.
3. Register the hook in the virt-launcher startup
   (`cmd/virt-launcher/virt-launcher.go`) by passing it to
   `NewPreMigrationHookServer()`.

## References

- [VEP-141: Target-Side Pre-Migration Hooks](https://github.com/kubevirt/enhancements/issues/141)
