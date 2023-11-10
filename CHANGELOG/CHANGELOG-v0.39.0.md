KubeVirt v0.39.0
================

This release follows v0.38.1 and consists of 227 changes, contributed by 38 people, leading to 480 files changed, 16457 insertions(+), 16077 deletions(-).
v0.39.0 is a promotion of release candidate v0.39.0-rc.0 which was originally published 2021-03-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.39.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.39.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #5010][jean-edouard] Migrated VMs stay persistent and can therefore survive S3, among other things.
- [PR #4952][ashleyschuett] Create warning NodeUnresponsive event if a node is running a VMI pod but not a virt-handler pod
- [PR #4686][davidvossel] Automated workload updates via new KubeVirt WorkloadUpdateStrategy API
- [PR #4886][awels] Hotplug support for WFFC datavolumes.
- [PR #5026][AlonaKaplan] virt-launcher, masquerade binding - prefer nft over iptables.
- [PR #4921][borod108] Added support for Sysprep in the API. A user can now add a answer file through a ConfigMap or a Secret. The User Guide is updated accordingly. /kind feature
- [PR #4874][ormergi] Add new feature-gate SRIOVLiveMigration,
- [PR #4917][iholder-redhat] Now it is possible to enable QEMU SeaBios debug logs setting virt-launcher log verbosity to be greater than 5.
- [PR #4966][arnongilboa] Solve virtctl "Error when closing file ... file already closed" that shows after successful image upload
- [PR #4489][salanki] Fix a bug where a disk.img file was created on filesystems mounted via Virtio-FS
- [PR #4982][xpivarc] Fixing handling of transient domain
- [PR #4984][ashleyschuett] Change customizeComponents.patches such that '*' resourceName or resourceType matches all, all fields of a patch (type, patch, resourceName, resourceType) are now required.
- [PR #4972][vladikr] allow disabling pvspinlock to support older guest kernels
- [PR #4927][yuhaohaoyu] Fix of XML and JSON marshalling/unmarshalling for user defined device alias names which can make migrations fail.
- [PR #4552][rthallisey] VMs using bridged networking will survive a kubelet restart by having kubevirt create a dummy interface on the virt-launcher pods, so that some Kubernetes CNIs, that have implemented the `CHECK` RPC call, will not cause VMI pods to enter a failed state.
- [PR #4883][iholder-redhat] Bug fixed: Enabling libvirt debug logs only if debugLogs label value is "true", disabling otherwise.
- [PR #4840][alicefr] Generate k8s events on IO errors
- [PR #4940][vladikr] permittedHostDevices will support both upper and lowercase letters in the device ID

Contributors
------------
38 people contributed to this release:

```
37	David Vossel <dvossel@redhat.com>
14	Miguel Duarte Barroso <mdbarroso@redhat.com>
13	Roman Mohr <rmohr@redhat.com>
10	Or Mergi <ormergi@redhat.com>
9	Vladik Romanovsky <vromanso@redhat.com>
8	Ashley Schuett <aschuett@redhat.com>
7	Dan Kenigsberg <danken@redhat.com>
7	Daniel Hiller <dhiller@redhat.com>
7	Edward Haas <edwardh@redhat.com>
7	iholder <iholder@redhat.com>
6	Alexander Wels <awels@redhat.com>
4	Jakub Guzik <jakubmguzik@gmail.com>
4	Jed Lejosne <jed@redhat.com>
3	Alice Frosi <afrosi@redhat.com>
3	Itamar Holder <iholder@redhat.com>
3	Or Shoval <oshoval@redhat.com>
2	Federico Gimenez <fgimenez@redhat.com>
2	Michael Henriksen <mhenriks@redhat.com>
2	Peter Salanki <peter@salanki.st>
2	Ryan Hallisey <rhallisey@nvidia.com>
2	Yuval Turgeman <yturgema@redhat.com>
1	Alona Kaplan <alkaplan@redhat.com>
1	Andrew DeMaria <ademaria@cloudflare.com>
1	Arnon Gilboa <agilboa@redhat.com>
1	Cole Robinson <crobinso@redhat.com>
1	Fabian Deutsch <fabiand@redhat.com>
1	Hao Yu <yuh@us.ibm.com>
1	L. Pivarc <lpivarc@redhat.com>
1	Marcin Franczyk <marcin0franczyk@gmail.com>
1	Petr Horacek <phoracek@redhat.com>
1	Quique Llorente <ellorent@redhat.com>
1	Radim Hrazdil <rhrazdil@redhat.com>
1	Stu Gott <sgott@redhat.com>
1	Yan Zhu <yanzhu@alauda.io>
1	borod108 <bodnopoz@redhat.com>
1	zouyu <zouy.fnst@cn.fujitsu.com>
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
