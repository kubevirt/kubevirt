KubeVirt v0.31.0
================

This release follows v0.30.3 and consists of 209 changes, contributed by 30 people, leading to 659 files changed, 132453 insertions(+), 40469 deletions(-).
v0.31.0 is a promotion of release candidate v0.31.0-rc.1 which was originally published 2020-07-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.31.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.31.0`.

Pre-built containers are published on Docker Hub and can be viewed at: <https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- [PR 3690][davidvossel] Update go-grpc dependency to v1.30.0 in order to improve stability
- [PR 3628][AlonaKaplan] Avoid virt-handler crash in case of virt-launcher network configuration error
- [PR 3635][jean-edouard] The "HostDisk" feature gate has to be enabled to use hostDisks
- [PR 3641][vatsalparekh] Reverts kubevirt/kubevirt#3488 because CI seems to have merged it without all tests passing
- [PR 3488][vatsalparekh] Add a way to update VMI Status with latest Pod IP for Masquerade bindings
- [PR 3406][tomob] If a PVC was created by a DataVolume, it cannot be used as a Volume Source for a VM. The owning DataVolume has to be used instead.
- [PR 3566][kraxel] added: tigervnc support for linux & windows
- [PR 3529][jean-edouard] Enabling EFI will also enable Secure Boot, which requires SMM to be enabled.
- [PR 3455][ashleyschuett] Add KubevirtConfiguration, MigrationConfiguration, DeveloperConfiguration and NetworkConfiguration to API-types
- [PR 3520][rmohr] Fix hot-looping on the  VMI sync-condition if errors happen during the Scheduled phase of a VMI
- [PR 3220][mhenriks] API and controller/webhook for VirtualMachineSnapshots

Contributors
------------
30 people contributed to this release:

```
37	Roman Mohr <rmohr@redhat.com>
21	Michael Henriksen <mhenriks@redhat.com>
17	Vatsal Parekh <vparekh@redhat.com>
12	Alona Kaplan <alkaplan@redhat.com>
11	Jed Lejosne <jed@redhat.com>
7	David Vossel <dvossel@redhat.com>
7	Miguel Duarte Barroso <mdbarroso@redhat.com>
7	Or Shoval <oshoval@redhat.com>
7	Stu Gott <sgott@redhat.com>
6	Edward Haas <edwardh@redhat.com>
4	Ashley Schuett <ashleyns1992@gmail.com>
3	Igor Bezukh <ibezukh@redhat.com>
3	Kedar Bidarkar <kbidarka@redhat.com>
3	Maya Rashish <mrashish@redhat.com>
2	Daniel Belenky <dbelenky@redhat.com>
2	Daniel Hiller <daniel.hiller.1972@gmail.com>
2	Gerd Hoffmann <kraxel@redhat.com>
2	Howard Zhang <howard.zhang@arm.com>
2	Or Mergi <ormergi@redhat.com>
2	Vladik Romanovsky <vromanso@redhat.com>
1	Adam Litke <alitke@redhat.com>
1	Dan Kenigsberg <danken@redhat.com>
1	Jed Lejosne <jean-edouard@users.noreply.github.com>
1	Jim Fehlig <jfehlig@suse.com>
1	Joowon Cheong <jwcheong0420@gmail.com>
1	Petr Horacek <phoracek@redhat.com>
1	Shweta Padubidri <spadubid@localhost.localdomain>
1	Tomasz Baranski <tbaransk@redhat.com>
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
