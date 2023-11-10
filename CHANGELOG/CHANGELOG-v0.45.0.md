KubeVirt v0.45.0
================

This release follows v0.44.1 and consists of 290 changes, contributed by 38 people, leading to 302 files changed, 13624 insertions(+), 4851 deletions(-).
v0.45.0 is a promotion of release candidate v0.45.0-rc.0 which was originally published 2021-09-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.45.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.45.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6191][marceloamaral] Addition of perfscale-load-generator to perform stress tests to evaluate the control plane
- [PR #6248][VirrageS] Reduced logging in hot paths
- [PR #6079][weihanglo] Hotplug volume can be unplugged at anytime and reattached after a VM restart.
- [PR #6101][rmohr] Make k8s client rate limits configurable
- [PR #6204][sradco] This PR adds to each alert the runbook url that points to a runbook that provides additional details on each alert and how to mitigate it.
- [PR #5974][vladikr] a list of desired mdev types can now be provided in KubeVirt CR to kubevirt to configure these devices on relevant nodes
- [PR #6147][rmohr] Fix rbac permissions for freeze/unfreeze, addvolume/removevolume, guestosinfo, filesystemlist and userlist
- [PR #6161][ashleyschuett] Remove HostDevice validation on VMI creation
- [PR #6078][zcahana] Report ErrImagePull/ImagePullBackOff VM status when image pull errors occur
- [PR #6176][kwiesmueller] Fix goroutine leak in virt-handler, potentially causing issues with a high turnover of VMIs.
- [PR #6047][ShellyKa13] Add phases to the vm snapshot api, specifically a failure phase
- [PR #6138][ansijain] NA

Contributors
------------
38 people contributed to this release:

```
23	Roman Mohr <rmohr@redhat.com>
20	Shelly Kagan <skagan@redhat.com>
15	David Vossel <dvossel@redhat.com>
15	Vladik Romanovsky <vromanso@redhat.com>
13	Miguel Duarte Barroso <mdbarroso@redhat.com>
13	Or Shoval <oshoval@redhat.com>
13	Zvi Cahana <zvic@il.ibm.com>
11	Weihang Lo <weihang.lo@suse.com>
8	Marcelo Amaral <marcelo.amaral1@ibm.com>
7	L. Pivarc <lpivarc@redhat.com>
6	Radim Hrazdil <rhrazdil@redhat.com>
5	Edward Haas <edwardh@redhat.com>
5	Quique Llorente <ellorent@redhat.com>
4	Federico Gimenez <fgimenez@redhat.com>
4	Igor Bezukh <ibezukh@redhat.com>
3	Alexander Wels <awels@redhat.com>
3	Ashley Schuett <aschuett@redhat.com>
3	Israel Pinto <ipinto@redhat.com>
3	Janusz Marcinkiewicz <januszm@nvidia.com>
3	Jed Lejosne <jed@redhat.com>
3	Vatsal Parekh <vparekh@redhat.com>
2	Dan Kenigsberg <danken@redhat.com>
2	Kedar Bidarkar <kbidarka@redhat.com>
2	Or Mergi <ormergi@redhat.com>
2	alonsadan <asadan@redhat.com>
1	Alex Kalenyuk <akalenyu@redhat.com>
1	Alona Kaplan <alkaplan@redhat.com>
1	Hao Yu <yuh@us.ibm.com>
1	Itamar Holder <iholder@redhat.com>
1	Josh Berkus <josh@agliodbs.com>
1	Kevin Wiesmueller <kwiesmul@redhat.com>
1	Maya Rashish <mrashish@redhat.com>
1	Shirly Radco <sradco@redhat.com>
1	Tomasz Baranski <tbaransk@redhat.com>
1	ansijain <ansi.jain@india.nec.com>
1	yingbai <yingbai@cn.ibm.com>
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
