KubeVirt v0.37.0
================

This release follows v0.36.0 and consists of 187 changes, contributed by 28 people, leading to 885 files changed, 68694 insertions(+), 10098 deletions(-).
v0.37.0 is a promotion of release candidate v0.37.0-rc.2 which was originally published 2021-01-18
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.37.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.37.0`.

Pre-built containers are published on Docker Hub and can be viewed at: <https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- [PR #4654][AlonaKaplan] Introduce virt-launcher DHCPv6 server.
- [PR #4669][kwiesmueller] Add nodeSelector to kubevirt components restricting them to run on linux nodes only.
- [PR #4648][davidvossel] Update libvirt base container to be based of packages in rhel-av 8.3
- [PR #4653][qinqon] Allow configure cloud-init with networkData only.
- [PR #4644][ashleyschuett] Operator validation webhook will deny updates to the workloads object of the KubeVirt CR if there are running VMIs
- [PR #3349][davidvossel] KubeVirt v1 GA api
- [PR #4645][maiqueb] Re-introduce the CAP_NET_ADMIN, to allow migration of VMs already having it.
- [PR #4546][yuhaohaoyu] Failure detection and handling for VM with EFI Insecure Boot in KubeVirt environments where EFI Insecure Boot is not supported by design.
- [PR #4625][awels] virtctl upload now shows error when specifying access mode of ReadOnlyMany
- [PR #4396][xpivarc] KubeVirt is now explainable!
- [PR #4517][danielBelenky] Fix guest agent reporting.

Contributors
------------
28 people contributed to this release:

```
19	Alona Kaplan <alkaplan@redhat.com>
13	David Vossel <dvossel@redhat.com>
12	Roman Mohr <rmohr@redhat.com>
11	Edward Haas <edwardh@redhat.com>
8	Dan Kenigsberg <danken@redhat.com>
8	Vasiliy Ulyanov <vulyanov@suse.de>
7	alonSadan <asadan@redhat.com>
6	Miguel Duarte Barroso <mdbarroso@redhat.com>
5	Ashley Schuett <ashleyns1992@gmail.com>
5	Ezra Silvera <ezra@il.ibm.com>
5	Stu Gott <sgott@redhat.com>
4	Zhou Hao <zhouhao@cn.fujitsu.com>
4	xiaobo <zeng.xiaobo@h3c.com>
2	Alexander Wels <awels@redhat.com>
2	Federico Gimenez <fgimenez@redhat.com>
2	L. Pivarc <lpivarc@redhat.com>
2	Quique Llorente <ellorent@redhat.com>
1	Daniel Belenky <dbelenky@redhat.com>
1	Daniel Hiller <dhiller@redhat.com>
1	Hao Yu <yuh@us.ibm.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Jim Fehlig <jfehlig@suse.com>
1	Kevin Wiesmueller <kwiesmul@redhat.com>
1	Marcin Franczyk <marcin0franczyk@gmail.com>
1	Or Shoval <oshoval@redhat.com>
1	ipinto <ipinto@redhat.com>
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
