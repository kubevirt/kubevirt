KubeVirt v0.55.2
================

This release follows v0.55.1 and consists of 16 changes, contributed by 4 people, leading to 57 files changed, 1642 insertions(+), 198 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.55.2.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.55.2`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #8490][kubevirt-bot] Fixed migration failure of VMs with containerdisks on systems with containerd

Contributors
------------
4 people contributed to this release:

```
11	Ram Lavi <ralavi@redhat.com>
2	Vasiliy Ulyanov <vulyanov@suse.de>
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
