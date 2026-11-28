# Alternative launcher image updates

Admission integrations that replace the `compute` container image in a
`virt-launcher` pod can set this label on the same pod:

```yaml
metadata:
  labels:
    kubevirt.io/skip-launcher-image-check: "true"
```

Only the exact value `"true"` enables the exemption. Set the label on every
launcher pod receiving the alternative image, including migration targets.
This is a pod label; setting it only on a VM or VMI does not enable the
exemption.

KubeVirt continues recording the actual launcher image in
`VMI.status.launcherContainerImageVersion`, but does not mark the VMI or pod
with `kubevirt.io/outdatedLauncherImage` or initiate workload migration or
eviction because that image differs from the stock launcher. Other workloads
retain their configured automatic update behavior. Migrations required for
live updates, such as memory hotplug or volume updates, remain enabled.

The alternative launcher provider owns image lifecycle and compatibility,
including during KubeVirt upgrades. This exemption does not establish that an
alternative launcher supports migration or a particular control-plane version.
Removing the label restores the stock image comparison; an alternative image
will then be considered outdated. Apply the label at pod creation before an automatic update can be scheduled;
it is not a cancellation mechanism for updates already in progress.
