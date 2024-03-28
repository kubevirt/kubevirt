KubeVirt v0.44.3
================

This release follows v0.44.2 and consists of 14 changes, contributed by 6 people, leading to 47 files changed, 1024 insertions(+), 147 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.44.3.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.44.3`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6518][jean-edouard] VMs with cloud-init data should now properly migrate from older KubeVirt versions
- [PR #6532][kubevirt-bot] mutate migration PDBs instead of creating an additional one for the duration of the migration.
- [PR #6536][kubevirt-bot] Fix corrupted DHCP Gateway Option from local DHCP server, leading to rejected IP configuration on Windows VMs.

Contributors
------------
6 people contributed to this release:

```
4	Antonio Cardace <acardace@redhat.com>
4	Jed Lejosne <jed@redhat.com>
1	Adam Litke <alitke@redhat.com>
1	Peter Salanki <peter@salanki.st>
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
