KubeVirt v0.42.2
================

This release follows v0.42.1 and consists of 27 changes, contributed by 6 people, leading to 63 files changed, 1265 insertions(+), 177 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.42.2.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.42.2`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6554][jean-edouard] VMs with cloud-init data should now properly migrate from older KubeVirt versions
- [PR #5887][ashleyschuett] Allow virtctl to stop VM and ignore the graceful shutdown period
- [PR #5907][kubevirt-bot] Fix: ioerrors don't cause crash-looping of notify server
- [PR #5871][maiqueb] Fix: do not override with the DHCP server advertising IP with the gateway info.
- [PR #5875][kubevirt-bot] Update ca-bundle if it is unable to be parsed

Contributors
------------
6 people contributed to this release:

```
10	Ashley Schuett <aschuett@redhat.com>
7	Miguel Duarte Barroso <mdbarroso@redhat.com>
4	Jed Lejosne <jed@redhat.com>
1	L. Pivarc <lpivarc@redhat.com>
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
