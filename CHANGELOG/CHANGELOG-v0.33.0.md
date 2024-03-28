KubeVirt v0.33.0
================

This release follows v0.32.0 and consists of 239 changes, contributed by 28 people, leading to 524 files changed, 45482 insertions(+), 28415 deletions(-).
v0.33.0 is a promotion of release candidate v0.33.0-rc.1 which was originally published 2020-09-14
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.33.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.33.0`.

Pre-built containers are published on Docker Hub and can be viewed at: <https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- [PR #3226][vatsalparekh] Added tests to verify custom pciAddress slots and function
- [PR #4048][davidvossel] Improved reliability for failed migration retries
- [PR #3585][mhenriks] "virtctl image-upload pvc ..." will create the PVC if it does not exist
- [PR #3945][xpivarc] KubeVirt is now being built with Go1.13.14
- [PR #3845][ArthurSens] action required: The domain label from VMI metrics is being removed and may break dashboards that use the domain label to identify VMIs. Use name and namespace labels instead
- [PR #4011][dhiller] ppc64le arch has been disabled for the moment, see https://github.com/kubevirt/kubevirt/issues/4037
- [PR #3875][stu-gott] Resources created by KubeVirt are now labelled more clearly in terms of relationship and role.
- [PR #3791][ashleyschuett] make node as kubevirt.io/schedulable=false on virt-handler restart
- [PR #3998][vladikr] the local provider is usable again.
- [PR #3290][maiqueb] Have virt-handler (KubeVirt agent) create the tap devices on behalf of the virt-launchers.
- [PR #3957][AlonaKaplan] virt-launcher support Ipv6 on dual stack cluster.
- [PR #3952][davidvossel] Fixes rare situation where vmi may not properly terminate if failure occurs before domain starts.
- [PR #3973][xpivarc] Fixes VMs with clock.timezone set.
- [PR #3923][danielBelenky] Add support to configure QEMU I/O mode for VMIs
- [PR #3889][rmohr] The status fields for our CRDs are now protected on normal PATCH and PUT operations.The /status subresource is now used where possible for status updates.
- [PR #3568][xpivarc] Guest swap metrics available

Contributors
------------
28 people contributed to this release:

```
23	Alona Kaplan <alkaplan@redhat.com>
23	rmohr <rmohr@redhat.com>
21	David Vossel <dvossel@redhat.com>
16	Roman Mohr <rmohr@redhat.com>
14	Miguel Duarte Barroso <mdbarroso@redhat.com>
12	L. Pivarc <lpivarc@redhat.com>
12	Or Mergi <ormergi@redhat.com>
10	Edward Haas <edwardh@redhat.com>
8	Or Shoval <oshoval@redhat.com>
8	Stu Gott <sgott@redhat.com>
7	Daniel Hiller <daniel.hiller.1972@gmail.com>
5	Michael Henriksen <mhenriks@redhat.com>
5	Petr Horacek <phoracek@redhat.com>
4	Daniel Belenky <dbelenky@redhat.com>
3	Ashley Schuett <ashleyns1992@gmail.com>
2	Alexander Wels <awels@redhat.com>
2	Kedar Bidarkar <kbidarka@redhat.com>
2	Vladik Romanovsky <vromanso@redhat.com>
2	arthursens <arthursens2005@gmail.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Jed Lejosne <jed@redhat.com>
1	Quique Llorente <ellorent@redhat.com>
1	Tomasz Baranski <tbaransk@redhat.com>
1	Vatsal Parekh <vparekh@redhat.com>
1	alonSadan <asadan@redhat.com>
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
