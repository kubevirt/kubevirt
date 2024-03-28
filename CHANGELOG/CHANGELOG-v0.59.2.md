KubeVirt v0.59.2
================

This release follows v0.59.1 and consists of 19 changes, contributed by 9 people, leading to 25 files changed, 493 insertions(+), 49 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.59.2.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.59.2`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #9910][kubevirt-bot] Bugfix: Allow lun disks to be mapped to DataVolume sources
- [PR #9853][machadovilaca] Remove mixin query not available on all clusters
- [PR #9826][Barakmor1] Add condition to migrations that indicates that migration was rejected by ResourceQuot

Contributors
------------
9 people contributed to this release:

```
3	Marcelo Tosatti <mtosatti@redhat.com>
3	bmordeha <bmodeha@redhat.com>
1	Alexander Wels <awels@redhat.com>
1	Alvaro Romero <alromero@redhat.com>
1	Felix Matouschek <fmatouschek@redhat.com>
1	João Vilaça <jvilaca@redhat.com>
1	Maya Rashish <mrashish@redhat.com>
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
