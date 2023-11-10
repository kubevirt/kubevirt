KubeVirt v0.38.0
================

This release follows v0.37.2 and consists of 283 changes, contributed by 34 people, leading to 1794 files changed, 110324 insertions(+), 36909 deletions(-).
v0.38.0 is a promotion of release candidate v0.38.0-rc.0 which was originally published 2021-02-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.38.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.38.0`.

Pre-built containers are published on Docker Hub and can be viewed at: <https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- [PR #4870][qinqon] Bump k8s deps to 0.20.2
- [PR #4571][yuvalturg] Added os, workflow and flavor labels to the kubevirt_vmi_phase_count metric
- [PR #4659][salanki] Fixed an issue where non-root users inside a guest could not write to a Virtio-FS mount.
- [PR #4844][xpivarc] Fixed limits/requests to accept int again
- [PR #4850][rmohr] virtio-scsi now respects the useTransitionalVirtio flag instead of assigning a virtio version depending on the machine layout
- [PR #4672][vladikr] allow increasing logging verbosity of infra components in KubeVirt CR
- [PR #4838][rmohr] Fix an issue where it may not be able to update the KubeVirt CR after creation for up to minutes due to certificate propagation delays
- [PR #4806][rmohr] Make the mutating webhooks for VMIs and VMs  required to avoid letting entities into the cluster which are not properly defaulted
- [PR #4779][brybacki] Error messsge on virtctl image-upload to WaitForFirstConsumer DV
- [PR #4749][davidvossel] KUBEVIRT_CLIENT_GO_SCHEME_REGISTRATION_VERSION env var for specifying exactly what client-go scheme version is registered
- [PR #4772][jean-edouard] Faster VMI phase transitions thanks to an increased number of VMI watch threads in virt-controller
- [PR #4730][rmohr] Add spec.domain.devices.useVirtioTransitional boolean to support virtio-transitional for old guests

Contributors
------------
34 people contributed to this release:

```
74	Roman Mohr <rmohr@redhat.com>
31	Zhou Hao <zhouhao@cn.fujitsu.com>
19	Dan Kenigsberg <danken@redhat.com>
11	Vladik Romanovsky <vromanso@redhat.com>
9	Ezra Silvera <ezra@il.ibm.com>
8	Alona Kaplan <alkaplan@redhat.com>
8	iholder <iholder@redhat.com>
6	David Vossel <dvossel@redhat.com>
6	Miguel Duarte Barroso <mdbarroso@redhat.com>
6	Vasiliy Ulyanov <vulyanov@suse.de>
5	Bartosz Rybacki <brybacki@redhat.com>
5	Federico Gimenez <fgimenez@redhat.com>
4	Andrey Odarenko <andreyo@il.ibm.com>
3	Alexander Wels <awels@redhat.com>
3	L. Pivarc <lpivarc@redhat.com>
2	Andrea Bolognani <abologna@redhat.com>
2	Edward Haas <edwardh@redhat.com>
2	Jed Lejosne <jed@redhat.com>
2	Quique Llorente <ellorent@redhat.com>
1	Ashley Schuett <aschuett@redhat.com>
1	Daniel Hiller <dhiller@redhat.com>
1	Hu Shuai <hus.fnst@cn.fujitsu.com>
1	Kedar Bidarkar <kbidarka@redhat.com>
1	Or Shoval <oshoval@redhat.com>
1	Peter Salanki <peter@salanki.st>
1	Ryan Hallisey <rhallisey@nvidia.com>
1	Shaul Garbourg <sgarbour@redhat.com>
1	Yoshiki Fujikane <ffjlabo@gmail.com>
1	Yuval Turgeman <yturgema@redhat.com>
1	alonSadan <asadan@redhat.com>
1	ansijain <ansi.jain@india.nec.com>
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
