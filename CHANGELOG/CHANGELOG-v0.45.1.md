KubeVirt v0.45.1
================

This release follows v0.45.0 and consists of 16 changes, contributed by 6 people, leading to 60 files changed, 1530 insertions(+), 239 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.45.1.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.45.1`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6537][kubevirt-bot] Fix corrupted DHCP Gateway Option from local DHCP server, leading to rejected IP configuration on Windows VMs.
- [PR #6556][jean-edouard] VMs with cloud-init data should now properly migrate from older KubeVirt versions
- [PR #6480][kubevirt-bot] BugFix: Fixed hotplug race between kubelet and virt-handler when virt-launcher dies unexpectedly.
- [PR #6384][kubevirt-bot] Better place vcpu threads on host cpus to form more efficient passthrough architectures

Contributors
------------
6 people contributed to this release:

```
6	Roman Mohr <rmohr@redhat.com>
4	Jed Lejosne <jed@redhat.com>
1	Alexander Wels <awels@redhat.com>
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
