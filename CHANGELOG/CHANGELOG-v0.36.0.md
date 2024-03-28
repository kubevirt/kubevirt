KubeVirt v0.36.0
================

This release follows v0.35.0 and consists of 314 changes, contributed by 35 people, leading to 308 files changed, 55480 insertions(+), 4880 deletions(-).
v0.36.0 is a promotion of release candidate v0.36.0-rc.1 which was originally published 2020-12-14
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.36.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.36.0`.

Pre-built containers are published on Docker Hub and can be viewed at: <https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- [PR #4667][kubevirt-bot] Update libvirt base container to be based of packages in rhel-av 8.3
- [PR #4634][kubevirt-bot] Failure detection and handling for VM with EFI Insecure Boot in KubeVirt environments where EFI Insecure Boot is not supported by design.
- [PR #4647][kubevirt-bot] Re-introduce the CAP_NET_ADMIN, to allow migration of VMs already having it.
- [PR #4627][kubevirt-bot] Fix guest agent reporting.
- [PR #4458][awels] It is now possible to hotplug DataVolume and PVC volumes into a running Virtual Machine.
- [PR #4025][brybacki] Adds a special handling for DataVolumes in WaitForFirstConsumer state to support CDI's delayed binding mode.
- [PR #4217][mfranczy] Set only an IP address for interfaces reported by qemu-guest-agent. Previously that was CIDR.
- [PR #4195][davidvossel] AccessCredentials API for dynamic user/password and ssh public key injection
- [PR #4335][oshoval] VMI status displays SRIOV interfaces with their network name only when they have originally
- [PR #4408][andreabolognani] This version of KubeVirt includes upgraded virtualization technology based on libvirt 6.6.0 and QEMU 5.1.0.
- [PR #4514][ArthurSens] `domain` label removed from metric `kubevirt_vmi_memory_unused_bytes`
- [PR #4542][danielBelenky] Fix double migration on node evacuation
- [PR #4506][maiqueb] Remove CAP_NET_ADMIN from the virt-launcher pod.
- [PR #4501][AlonaKaplan] CAP_NET_RAW removed from virt-launcher.
- [PR #4488][salanki] Disable Virtio-FS metadata cache to prevent OOM conditions on the host.
- [PR #3937][vladikr] Generalize host devices assignment. Provides an interface between kubevirt and external device plugins. Provides a mechanism for whitelisting host devices.
- [PR #4443][rmohr] All kubevirt webhooks support now dry-runs.

Contributors
------------
35 people contributed to this release:

```
32	David Vossel <dvossel@redhat.com>
32	Vladik Romanovsky <vromanso@redhat.com>
23	Roman Mohr <rmohr@redhat.com>
17	Alexander Wels <awels@redhat.com>
15	Edward Haas <edwardh@redhat.com>
15	Jed Lejosne <jed@redhat.com>
13	Ezra Silvera <ezra@il.ibm.com>
13	Quique Llorente <ellorent@redhat.com>
11	Marcelo Amaral <marcelo.amaral1@ibm.com>
11	Miguel Duarte Barroso <mdbarroso@redhat.com>
9	Or Shoval <oshoval@redhat.com>
8	Bartosz Rybacki <brybacki@redhat.com>
7	Daniel Hiller <daniel.hiller.1972@gmail.com>
5	Alona Kaplan <alkaplan@redhat.com>
5	Andrea Bolognani <abologna@redhat.com>
5	Stu Gott <sgott@redhat.com>
4	Fabian Deutsch <fabiand@redhat.com>
3	Igor Bezukh <ibezukh@redhat.com>
3	L. Pivarc <lpivarc@redhat.com>
2	Daniel Belenky <dbelenky@redhat.com>
2	alonSadan <asadan@redhat.com>
2	屈骏 <qujun@tiduyun.com>
1	Adam Litke <alitke@redhat.com>
1	Christoph Stäbler <chresse1605@gmail.com>
1	Dan Kenigsberg <danken@redhat.com>
1	Federico Gimenez <federico.gimenez@gmail.com>
1	Hao Yu <yuh@us.ibm.com>
1	Marcin Franczyk <marcin0franczyk@gmail.com>
1	Or Mergi <ormergi@redhat.com>
1	Peter Salanki <peter@salanki.st>
1	Tomasz Baranski <tbaransk@redhat.com>
1	arthursens <arthursens2005@gmail.com>
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
