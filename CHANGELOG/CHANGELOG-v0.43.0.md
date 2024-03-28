KubeVirt v0.43.0
================

This release follows v0.42.1 and consists of 370 changes, contributed by 41 people, leading to 569 files changed, 17418 insertions(+), 24973 deletions(-).
v0.43.0 is a promotion of release candidate v0.43.0-rc.1 which was originally published 2021-07-08
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.43.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.43.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #5952][mhenriks] Use CDI beta API. CDI v1.20.0 is now the minimum requirement for kubevirt.
- [PR #5846][rmohr] Add "spec.cpu.numaTopologyPassthrough" which allows emulating a host-alligned virtual numa topology for high performance
- [PR #5894][rmohr] Add `spec.migrations.disableTLS` to the KubeVirt CR to allow disabling encrypted migrations. They stay secure by default.
- [PR #5649][awels] Enhancement: remove one attachment pod per disk limit (behavior on upgrade with running VM with hotplugged disks is undefined)
- [PR #5742][rmohr] VMIs which choose evictionStrategy `LifeMigrate` and request the `invtsc` cpuflag are now live-migrateable
- [PR #5911][dhiller] Bumps kubevirtci, also suppresses kubectl.sh output to avoid confusing checks
- [PR #5863][xpivarc] Fix: ioerrors don't cause crash-looping of notify server
- [PR #5867][mlsorensen] New build target added to export virt-* images as a tar archive.
- [PR #5766][davidvossel] Addition of kubevirt_vmi_phase_transition_seconds_since_creation to monitor how long it takes to transition a VMI to a specific phase from creation time.
- [PR #5823][dhiller] Change default branch to `main` for `kubevirt/kubevirt` repository
- [PR #5763][nunnatsa] Fix bug 1945589: Prevent migration of VMIs that uses virtiofs
- [PR #5827][mlsorensen] Auto-provisioned disk images on empty PVCs now leave 128KiB unused to avoid edge cases that run the volume out of space.
- [PR #5849][davidvossel] Fixes event recording causing a segfault in virt-controller
- [PR #5797][rhrazdil] Add serviceAccountDisk automatically when Istio is enabled in VMI annotations
- [PR #5723][ashleyschuett] Allow virtctl to stop VM and ignore the graceful shutdown period
- [PR #5806][mlsorensen] configmap, secret, and cloud-init raw disks now work when underlying node storage has 4k blocks.
- [PR #5623][iholder-redhat] [bugfix]: Allow migration of VMs with host-model CPU to migrate only for compatible nodes
- [PR #5716][rhrazdil] Fix issue with virt-launcher becoming `NotReady` after migration when Istio is used.
- [PR #5778][ashleyschuett] Update ca-bundle if it is unable to be parsed
- [PR #5787][acardace] migrated references of authorization/v1beta1 to authorization/v1
- [PR #5461][rhrazdil] Add support for Istio proxy when no explicit ports are specified on masquerade interface
- [PR #5751][acardace] EFI VMIs with secureboot disabled can now be booted even when only OVMF_CODE.secboot.fd and OVMF_VARS.fd are present in the virt-launcher image
- [PR #5629][andreyod] Support starting Virtual Machine with its guest CPU paused using `virtctl start --paused`
- [PR #5725][dhiller] Generate REST API coverage report after functional tests
- [PR #5758][davidvossel] Fixes kubevirt_vmi_phase_count to include all phases, even those that occur before handler hand off.
- [PR #5745][ashleyschuett] Alert with resource usage exceeds resource requests
- [PR #5759][mhenriks] Update CDI to 1.34.1
- [PR #5038][kwiesmueller] Add exec command to VM liveness and readinessProbe executed through the qemu-guest-agent.
- [PR #5431][alonSadan] Add NFT and IPTables rules to allow port-forward to non-declared ports on the VMI. Declaring ports on VMI will limit

Contributors
------------
41 people contributed to this release:

```
47	Roman Mohr <rmohr@redhat.com>
22	Kevin Wiesmueller <kwiesmul@redhat.com>
20	Daniel Hiller <dhiller@redhat.com>
18	David Vossel <dvossel@redhat.com>
14	Miguel Duarte Barroso <mdbarroso@redhat.com>
12	Alexander Wels <awels@redhat.com>
12	Ashley Schuett <aschuett@redhat.com>
10	Radim Hrazdil <rhrazdil@redhat.com>
9	Alona Kaplan <alkaplan@redhat.com>
9	Itamar Holder <iholder@redhat.com>
8	Vasiliy Ulyanov <vulyanov@suse.de>
6	Andrey Odarenko <andreyo@il.ibm.com>
6	Marcus Sorensen <marcus_sorensen@apple.com>
5	Zvi Cahana <zvic@il.ibm.com>
5	alonsadan <asadan@redhat.com>
4	Antonio Cardace <acardace@redhat.com>
4	Federico Gimenez <fgimenez@redhat.com>
4	L. Pivarc <lpivarc@redhat.com>
4	Quique Llorente <ellorent@redhat.com>
4	Shelly Kagan <skagan@redhat.com>
3	Andrea Bolognani <abologna@redhat.com>
3	Howard Zhang <howard.zhang@arm.com>
3	Igor Bezukh <ibezukh@redhat.com>
3	Shirly Radco <sradco@redhat.com>
2	Dan Kenigsberg <danken@redhat.com>
2	Edward Haas <edwardh@redhat.com>
2	Maya Rashish <mrashish@redhat.com>
2	Michael Henriksen <mhenriks@redhat.com>
2	Vatsal Parekh <vparekh@redhat.com>
2	Zhou Hao <zhouhao@fujitsu.com>
1	Andrew DeMaria <ademaria@cloudflare.com>
1	Daniel Hiller <daniel.hiller.1972@gmail.com>
1	Jed Lejosne <jed@redhat.com>
1	Kedar Bidarkar <kbidarka@redhat.com>
1	Marcin Franczyk <marcin0franczyk@gmail.com>
1	Marcus Sorensen <mls@apple.com>
1	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
1	Ram Lavi <ralavi@redhat.com>
1	ansijain <ansi.jain@india.nec.com>
```

Additional Resources
--------------------

- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]

[contributing]: https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/master/LICENSE
