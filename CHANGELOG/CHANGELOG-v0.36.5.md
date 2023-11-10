KubeVirt v0.36.5
================

This release follows v0.36.3 and consists of 47 changes, contributed by 14 people, leading to 61 files changed, 1612 insertions(+), 359 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.36.5.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.36.5`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6837][dhiller] Backport prow release automation changes from 0.40 to 0.36
- [PR #6291][kubevirt-bot] Fix goroutine leak in virt-handler, potentially causing issues with a high turnover of VMIs.
- [PR #6591][jean-edouard] VMs with cloud-init data should now properly migrate from older KubeVirt versions
- [PR #6512][xpivarc] Fix cases where migration will not be processed if previous migration failed.
- [PR #6400][rmohr] Fix rbac permissions for freeze/unfreeze, addvolume/removevolume, guestosinfo, filesystemlist and userlist
- [PR #6147][rmohr] Fix rbac permissions for freeze/unfreeze, addvolume/removevolume, guestosinfo, filesystemlist and userlist
- [PR #6269][awels] BugFix: Generate ISO images 4k aligned for node storage with 4k blocksize
- [PR #5919][davidvossel] Fixes event recording causing a segfault in virt-controller
- [PR #5865][xpivarc] Fix: Kubevirt build with golang 1.14+ will not fail on validation of container disk with memory allocation error
- [PR #5851][rthallisey] Prometheus metrics scraped from virt-handler are now served from the VMI informer cache, rather than calling back to the Kubernetes API for VMI information.
- [PR #5655][acardace] virt-launcher now populates domain's guestOS info and interfaces status according guest agent also when doing periodic resyncs.
- [PR #5708][kubevirt-bot] Fixes null pointer dereference in migration controller
- [PR #5705][davidvossel] Fix virt-controller clobbering in progress vmi migration state during virt handler handoff
- [PR #5672][davidvossel] Validation/Mutation webhooks now explicitly define a 10 second timeout period

Contributors
------------
14 people contributed to this release:

```
8	David Vossel <dvossel@redhat.com>
5	Jed Lejosne <jed@redhat.com>
4	Or Shoval <oshoval@redhat.com>
3	Alexander Wels <awels@redhat.com>
2	Marcus Sorensen <mls@apple.com>
1	Bartosz Rybacki <brybacki@redhat.com>
1	Daniel Hiller <dhiller@redhat.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Kevin Wiesmueller <kwiesmul@redhat.com>
1	L. Pivarc <lpivarc@redhat.com>
1	Ryan Hallisey <rhallisey@nvidia.com>
1	Vladik Romanovsky <vromanso@redhat.com>
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
