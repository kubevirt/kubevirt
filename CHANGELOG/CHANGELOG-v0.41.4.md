KubeVirt v0.41.4
================

This release follows v0.41.3 and consists of 37 changes, contributed by 11 people, leading to 70 files changed, 2270 insertions(+), 624 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.41.4.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.41.4`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6573][acardace] mutate migration PDBs instead of creating an additional one for the duration of the migration.
- [PR #6517][jean-edouard] VMs with cloud-init data should now properly migrate from older KubeVirt versions
- [PR #6333][acardace] Fix virt-launcher exit pod race condition
- [PR #6401][rmohr] Fix rbac permissions for freeze/unfreeze, addvolume/removevolume, guestosinfo, filesystemlist and userlist
- [PR #6147][rmohr] Fix rbac permissions for freeze/unfreeze, addvolume/removevolume, guestosinfo, filesystemlist and userlist
- [PR #5673][kubevirt-bot] Improved logging around VM/VMI shutdown and restart
- [PR #6227][kwiesmueller] Fix goroutine leak in virt-handler, potentially causing issues with a high turnover of VMIs.

Contributors
------------
11 people contributed to this release:

```
7	Jed Lejosne <jed@redhat.com>
6	Antonio Cardace <acardace@redhat.com>
3	Itamar Holder <iholder@redhat.com>
1	David Vossel <dvossel@redhat.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Kevin Wiesmueller <kwiesmul@redhat.com>
1	L. Pivarc <lpivarc@redhat.com>
1	Marcus Sorensen <mls@apple.com>
1	Shelly Kagan <skagan@redhat.com>
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
