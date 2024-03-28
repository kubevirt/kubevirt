KubeVirt v0.50.0
================

This release follows v0.49.0 and consists of 286 changes, contributed by 38 people, leading to 2776 files changed, 223758 insertions(+), 107570 deletions(-)
warning: inexact rename detection was skipped due to too many files.
warning: you may want to set your diff.renameLimit variable to at least 1099 and retry the command..
v0.50.0 is a promotion of release candidate v0.50.0-rc.0 which was originally published 2022-02-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.50.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.50.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #7056][fossedihelm] Update k8s dependencies to 0.23.1
- [PR #7135][davidvossel] Switch from reflects.DeepEquals to equality.Semantic.DeepEquals() across the entire project
- [PR #7052][sradco] Updated recording rule "kubevirt_vm_container_free_memory_bytes"
- [PR #7000][iholder-redhat] Adds a possibility to override default libvirt log filters though VMI annotations
- [PR #7064][davidvossel] Fixes issue associated with blocked uninstalls when VMIs exist during removal
- [PR #7097][iholder-redhat] [Bug fix] VMI with kernel boot stuck on "Terminating" status if more disks are defined
- [PR #6700][VirrageS] Simplify replacing `time.Ticker` in agent poller and fix default values for `qemu-*-interval` flags
- [PR #6581][ormergi] SRIOV network interfaces are now hot-plugged when disconnected manually or due to aborted migrations.
- [PR #6924][EdDev] Support for legacy GPU definition is removed. Please see https://kubevirt.io/user-guide/virtual_machines/host-devices on how to define host-devices.
- [PR #6735][uril] The command `migrate_cancel` was added to virtctl. It cancels an active VM migration.
- [PR #6883][rthallisey] Add instance-type to cloud-init metadata
- [PR #6999][maya-r] When expanding disk images, take the minimum between the request and the capacity - avoid using the full underlying file system on storage like NFS, local.
- [PR #6946][vladikr] Numa information of an assigned device will be presented in the devices metadata
- [PR #6042][iholder-redhat] Fully support cgroups v2, include a new cohesive package and perform major refactoring.
- [PR #6968][vladikr] Added Writeback disk cache support
- [PR #6995][sradco] Alert OrphanedVirtualMachineImages name was changed to OrphanedVirtualMachineInstances.
- [PR #6923][rhrazdil] Fix issue with ssh being unreachable on VMIs with Istio proxy
- [PR #6821][jean-edouard] Migrating VMIs that contain dedicated CPUs will now have properly dedicated CPUs on target
- [PR #6793][oshoval] Add infoSource field to vmi.status.interfaces.

Contributors
------------
38 people contributed to this release:

```
37	Edward Haas <edwardh@redhat.com>
22	Itamar Holder <iholder@redhat.com>
14	Orel Misan <omisan@redhat.com>
14	Shelly Kagan <skagan@redhat.com>
10	Ryan Hallisey <rhallisey@nvidia.com>
9	Jed Lejosne <jed@redhat.com>
8	Dan Kenigsberg <danken@redhat.com>
8	Vladik Romanovsky <vromanso@redhat.com>
7	Marcelo Amaral <marcelo.amaral1@ibm.com>
6	David Vossel <dvossel@redhat.com>
6	L. Pivarc <lpivarc@redhat.com>
5	Or Shoval <oshoval@redhat.com>
5	Uri Lublin <uril@redhat.com>
4	Alice Frosi <afrosi@redhat.com>
4	Omer Yahud <oyahud@redhat.com>
4	Or Mergi <ormergi@redhat.com>
4	Roman Mohr <rmohr@redhat.com>
3	Igor Bezukh <ibezukh@redhat.com>
2	Barak Mordehai <bmordeha@redhat.com>
2	Ben Ukhanov <ben1zuk321@gmail.com>
2	Daniel Hiller <dhiller@redhat.com>
2	Maya Rashish <mrashish@redhat.com>
2	Radim Hrazdil <rhrazdil@redhat.com>
2	Shirly Radco <sradco@redhat.com>
2	Vasiliy Ulyanov <vulyanov@suse.de>
2	Xiaoli Ai <xlai@suse.com`>
2	fossedihelm <fossedihelm@gmail.com>
1	Alex Kalenyuk <akalenyu@redhat.com>
1	Bartosz Rybacki <brybacki@redhat.com>
1	Cedric Hauber <hauber.c@gmail.com>
1	Denys Shchedrivyi <dshchedr@redhat.com>
1	Erkan Erol <eerol@redhat.com>
1	Janusz Marcinkiewicz <januszm@nvidia.com>
1	Miguel Duarte Barroso <mdbarroso@redhat.com>
1	Mor Cohen <mocohen@redhat.com>
1	prnaraya <prnaraya@redhat.com>
```

Additional Resources
--------------------

- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]

[contributing]: https://github.com/kubevirt/kubevirt/blob/main/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/main/LICENSE
