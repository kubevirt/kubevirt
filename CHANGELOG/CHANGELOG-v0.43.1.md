KubeVirt v0.43.1
================

This release follows v0.43.0 and consists of 13 changes, contributed by 6 people, leading to 45 files changed, 812 insertions(+), 87 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.43.1.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.43.1`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6555][jean-edouard] VMs with cloud-init data should now properly migrate from older KubeVirt versions
- [PR #6052][kubevirt-bot] make containerDisk validation memory usage limit configurable

Contributors
------------
6 people contributed to this release:

```
4	Jed Lejosne <jed@redhat.com>
3	Antonio Cardace <acardace@redhat.com>
1	Peter Salanki <peter@salanki.st>
1	Roman Mohr <rmohr@redhat.com>
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
