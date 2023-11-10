KubeVirt v0.51.0
================

This release follows v0.50.0 and consists of 180 changes, contributed by 28 people, leading to 184 files changed, 5739 insertions(+), 4160 deletions(-).
v0.51.0 is a promotion of release candidate v0.51.0-rc.0 which was originally published 2022-03-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.51.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.51.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #7102][machadovilaca] Add Virtual Machine name label to virt-launcher pod
- [PR #7139][davidvossel] Fixes inconsistent VirtualMachinePool VM/VMI updates by using controller revisions
- [PR #6754][jean-edouard] New and resized disks are now always 1MiB-aligned
- [PR #7086][acardace] Add 'EvictionStrategy' as a cluster-wide setting in the KubeVirt CR
- [PR #7232][rmohr] Properly format the PDB scale event during migrations
- [PR #7223][Barakmor1] Add a name label to virt-operator pods
- [PR #7221][davidvossel] RunStrategy: Once - allows declaring a VM should run once to a finalized state
- [PR #7091][EdDev] SR-IOV interfaces are now reported in the VMI status even without an active guest-agent.
- [PR #7169][rmohr] Improve device plugin de-registration in virt-handler and some test stabilizations
- [PR #6604][alicefr] Add shareable option to identify if the disk is shared with other VMs
- [PR #7144][davidvossel] Garbage collect finalized migration objects only leaving the most recent 5 objects
- [PR #6110][xpivarc] [Nonroot] SRIOV is now available.

Contributors
------------
28 people contributed to this release:

```
28	Dan Kenigsberg <danken@redhat.com>
18	Roman Mohr <rmohr@redhat.com>
13	David Vossel <dvossel@redhat.com>
8	Edward Haas <edwardh@redhat.com>
7	Antonio Cardace <acardace@redhat.com>
5	Alice Frosi <afrosi@redhat.com>
4	Andrej Krejcir <akrejcir@redhat.com>
4	L. Pivarc <lpivarc@redhat.com>
3	Igor Bezukh <ibezukh@redhat.com>
3	Vasiliy Ulyanov <vulyanov@suse.de>
3	Victor Toso <victortoso@redhat.com>
3	fossedihelm <ffossemo@redhat.com>
2	Daniel Hiller <dhiller@redhat.com>
2	Jed Lejosne <jed@redhat.com>
2	João Vilaça <jvilaca@redhat.com>
2	Karel Šimon <ksimon@redhat.com>
2	Michael Henriksen <mhenriks@redhat.com>
2	Ryan Hallisey <rhallisey@nvidia.com>
2	Shelly Kagan <skagan@redhat.com>
1	Alexander Wels <awels@redhat.com>
1	Barak Mordehai <bmordeha@redhat.com>
1	Bartosz Rybacki <brybacki@redhat.com>
1	Marcelo Amaral <marcelo.amaral1@ibm.com>
1	Shirly Radco <sradco@redhat.com>
1	Simone Tiraboschi <stirabos@redhat.com>
1	jbpratt <jbpratt78@gmail.com>
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
