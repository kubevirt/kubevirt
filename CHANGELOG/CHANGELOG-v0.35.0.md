KubeVirt v0.35.0
================

This release follows v0.34.0 and consists of 275 changes, contributed by 31 people, leading to 254 files changed, 18061 insertions(+), 4438 deletions(-).
v0.35.0 is a promotion of release candidate v0.35.0-rc.0 which was originally published 2020-11-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.35.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.35.0`.

Pre-built containers are published on Docker Hub and can be viewed at: <https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- [PR #4409][vladikr] Increase the static memory overhead by 10Mi
- [PR #4272][maiqueb] Add `ip-family` to the `virtctl expose` command.
- [PR #4398][rmohr] VMIs reflect deleted stuck virt-launcher pods with the "PodTerminating" Reason in the ready condition. The VMIRS detects this reason and immediately creates replacement VMIs.
- [PR #4393][salanki] Disable legacy service links in `virt-launcher` Pods to speed up Pod instantiation and decrease Kubelet load in namespaces with many services.
- [PR #2935][maiqueb] Add the macvtap BindMechanism.
- [PR #4132][mstarostik] fixes a bug that prevented unique device name allocation when configuring both scsi and sata drives
- [PR #3257][xpivarc] Added support of `kubectl explain` for Kubevirt resources.
- [PR #4288][ezrasilvera] Adding DownwardAPI volumes type
- [PR #4233][maya-r] Update base image used for pods to Fedora 31.
- [PR #4192][xpivarc] We now run gosec in Kubevirt
- [PR #4328][stu-gott] Version 2.x QEMU guest agents are supported.
- [PR #4289][AlonaKaplan] Masquerade binding - set the virt-launcher pod interface MTU on the bridge.
- [PR #4300][maiqueb] Update the NetworkInterfaceMultiqueue openAPI documentation to better specify its semantics within KubeVirt.
- [PR #4277][awels] PVCs populated by DVs are now allowed as volumes.
- [PR #4265][dhiller] Fix virtctl help text when running as a plugin
- [PR #4273][dhiller] Only run Travis build for PRs against release branches

Contributors
------------
31 people contributed to this release:

```
33	Miguel Duarte Barroso <mdbarroso@redhat.com>
30	L. Pivarc <lpivarc@redhat.com>
26	Roman Mohr <rmohr@redhat.com>
12	Or Shoval <oshoval@redhat.com>
11	Alona Kaplan <alkaplan@redhat.com>
10	Ezra Silvera <ezra@il.ibm.com>
9	David Vossel <dvossel@redhat.com>
8	Edward Haas <edwardh@redhat.com>
8	alonSadan <asadan@redhat.com>
6	Igor Bezukh <ibezukh@redhat.com>
6	Or Mergi <ormergi@redhat.com>
4	Maya Rashish <mrashish@redhat.com>
4	Vladik Romanovsky <vromanso@redhat.com>
3	Quique Llorente <ellorent@redhat.com>
2	Alexander Wels <awels@redhat.com>
2	Bartosz Rybacki <brybacki@redhat.com>
2	Cole Robinson <crobinso@redhat.com>
2	Daniel Hiller <daniel.hiller.1972@gmail.com>
2	Malte Starostik <github@xodtsoq.de>
2	Michael Henriksen <mhenriks@redhat.com>
2	Petr Horacek <phoracek@redhat.com>
2	Stu Gott <sgott@redhat.com>
1	Dan Kenigsberg <danken@redhat.com>
1	Kedar Bidarkar <kbidarka@redhat.com>
1	Nelson Mimura Gonzalez <nelson@ibm.com>
1	Peter Salanki <peter@salanki.st>
1	Tomasz Baranski <tbaransk@redhat.com>
1	ipinto <ipinto@redhat.com>
1	屈骏 <qujun@tiduyun.com>
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
