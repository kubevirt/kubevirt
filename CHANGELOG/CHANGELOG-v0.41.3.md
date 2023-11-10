KubeVirt v0.41.3
================

This release follows v0.41.0 and consists of 84 changes, contributed by 18 people, leading to 81 files changed, 2480 insertions(+), 3221 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.41.3.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.41.3`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6196][ashleyschuett] Allow multiple shutdown events to ensure the event is received by ACPI
- [PR #6194][kubevirt-bot] Allow Failed VMs to be stopped when using `--force --gracePeriod 0`
- [PR #6039][akalenyu] BugFix: Pending VMIs when creating concurrent bulk of VMs backed by WFFC DVs
- [PR #5917][davidvossel] Fixes event recording causing a segfault in virt-controller
- [PR #5886][ashleyschuett] Allow virtctl to stop VM and ignore the graceful shutdown period
- [PR #5866][xpivarc] Fix: Kubevirt build with golang 1.14+ will not fail on validation of container disk with memory allocation error
- [PR #5873][kubevirt-bot] Update ca-bundle if it is unable to be parsed
- [PR #5822][kubevirt-bot] migrated references of authorization/v1beta1 to authorization/v1
- [PR #5704][davidvossel] Fix virt-controller clobbering in progress vmi migration state during virt handler handoff
- [PR #5707][kubevirt-bot] Fixes null pointer dereference in migration controller
- [PR #5685][stu-gott] [bugfix] - reject VM defined with volume with no matching disk
- [PR #5670][stu-gott] Validation/Mutation webhooks now explicitly define a 10 second timeout period
- [PR #5653][kubevirt-bot] virt-launcher now populates domain's guestOS info and interfaces status according guest agent also when doing periodic resyncs.
- [PR #5644][kubevirt-bot] Fix live-migration failing when VM with masquarade iface has explicitly specified any of these ports: 22222, 49152, 49153
- [PR #5646][kubevirt-bot] virtctl rename support is dropped

Contributors
------------
18 people contributed to this release:

```
17	Ashley Schuett <aschuett@redhat.com>
8	David Vossel <dvossel@redhat.com>
5	Roman Mohr <rmohr@redhat.com>
4	L. Pivarc <lpivarc@redhat.com>
4	Radim Hrazdil <rhrazdil@redhat.com>
3	Antonio Cardace <acardace@redhat.com>
2	Itamar Holder <iholder@redhat.com>
2	Jed Lejosne <jed@redhat.com>
1	Alex Kalenyuk <akalenyu@redhat.com>
1	Alexander Wels <awels@redhat.com>
1	Bartosz Rybacki <brybacki@redhat.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Karel Šimon <ksimon@redhat.com>
1	Omer Yahud <oyahud@redhat.com>
1	Or Shoval <oshoval@redhat.com>
1	Petr Horáček <phoracek@redhat.com>
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
