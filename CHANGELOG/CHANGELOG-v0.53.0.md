KubeVirt v0.53.0
================

This release follows v0.52.0 and consists of 293 changes, contributed by 40 people, leading to 642 files changed, 24571 insertions(+), 17591 deletions(-).
v0.53.0 is a promotion of release candidate v0.53.0-rc.0 which was originally published 2022-05-02
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.53.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.53.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #7533][akalenyu] Add several VM snapshot metrics
- [PR #7574][rmohr] Pull in cdi dependencies with minimized transitive dependencies to ease API adoption
- [PR #7318][iholder-redhat] Snapshot restores now support restoring to a target VM different than the source
- [PR #7474][borod108] Added the following metrics for live migration: kubevirt_migrate_vmi_data_processed_bytes, kubevirt_migrate_vmi_data_remaining_bytes, kubevirt_migrate_vmi_dirty_memory_rate_bytes
- [PR #7441][rmohr] Add `virtctl scp` to ease copying files from and to VMs and VMIs
- [PR #7265][rthallisey] Support steady-state job types in the load-generator tool
- [PR #7544][fossedihelm] Upgraded go version to 1.17.8
- [PR #7582][acardace] Fix failed reported migrations when actually they were successful.
- [PR #7546][0xFelix] Update virtio-container-disk to virtio-win version 0.1.217-1
- [PR #7530][iholder-redhat] [External Kernel Boot]: Disallow kernel args without providing custom kernel
- [PR #7493][davidvossel] Adds new EvictionStrategy "External" for blocking eviction which is handled by an external controller
- [PR #7563][akalenyu] Switch VolumeSnapshot to v1
- [PR #7406][acardace] Reject `LiveMigrate` as a workload-update strategy if the `LiveMigration` feature gate is not enabled.
- [PR #7103][jean-edouard] Non-persistent vTPM now supported. Keep in mind that the state of the TPM is wiped after each shutdown. Do not enable Bitlocker!
- [PR #7277][andreabolognani] This version of KubeVirt includes upgraded virtualization technology based on libvirt 8.0.0 and QEMU 6.2.0.
- [PR #7130][Barakmor1] Add field to kubevirtCR to set Prometheus ServiceMonitor object's namespace
- [PR #7401][iholder-redhat] virt-api deployment is now scalable - replicas are determined by the number of nodes in the cluster
- [PR #7500][awels] BugFix: Fixed RBAC for admin/edit user to allow virtualmachine/addvolume and removevolume. This allows for persistent disks
- [PR #7328][apoorvajagtap] Don't ignore --identity-file when setting --local-ssh=true on `virtctl ssh`
- [PR #7469][xpivarc] Users can now enable the NonRoot feature gate instead of NonRootExperimental
- [PR #7451][fossedihelm] Reduce virt-launcher memory usage by splitting monitoring and launcher processes

Contributors
------------
40 people contributed to this release:

```
26	Edward Haas <edwardh@redhat.com>
19	Itamar Holder <iholder@redhat.com>
14	Alex Kalenyuk <akalenyu@redhat.com>
11	Alona Kaplan <alkaplan@redhat.com>
11	Jed Lejosne <jed@redhat.com>
11	Roman Mohr <rmohr@redhat.com>
10	Andrea Bolognani <abologna@redhat.com>
10	Ryan Hallisey <rhallisey@nvidia.com>
9	Miguel Duarte Barroso <mdbarroso@redhat.com>
7	Dan Kenigsberg <danken@redhat.com>
5	Antonio Cardace <acardace@redhat.com>
5	L. Pivarc <lpivarc@redhat.com>
5	Lee Yarwood <lyarwood@redhat.com>
5	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
5	Radim Hrazdil <rhrazdil@redhat.com>
4	Barak Mordehai <bmordeha@redhat.com>
4	David Vossel <dvossel@redhat.com>
4	Vasiliy Ulyanov <vulyanov@suse.de>
3	fossedihelm <ffossemo@redhat.com>
2	Alexander Wels <awels@redhat.com>
2	Andrej Krejcir <akrejcir@redhat.com>
2	Diana Teplits <dteplits@redhat.com>
2	Felix Matouschek <fmatouschek@redhat.com>
2	Killercoda <accounts+github@killercoda.com>
2	Stu Gott <sgott@redhat.com>
2	bmordeha <bmodeha@redhat.com>
1	Andrew Burden <aburden@redhat.com>
1	Apoorva Jagtap <apoorvajagtap4@gmail.com>
1	Bartosz Rybacki <brybacki@redhat.com>
1	Caleb Crane <ccrane@suse.de>
1	Igor Bezukh <ibezukh@redhat.com>
1	Janusz Marcinkiewicz <januszm@nvidia.com>
1	Shirly Radco <sradco@redhat.com>
1	Shweta Padubidri <spadubid@redhat.com>
1	Zhe Peng <zpeng@redhat.com>
1	assaf-admi <aadmi@redhat.com>
1	borod108 <boris.od@gmail.com>
1	shewensheng <shewensheng@chinatelecom.cn>
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
