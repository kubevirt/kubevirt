KubeVirt v0.44.2
================

This release follows v0.44.1 and consists of 72 changes, contributed by 12 people, leading to 140 files changed, 4706 insertions(+), 1275 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.44.2.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.44.2`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6479][kubevirt-bot] BugFix: Fixed hotplug race between kubelet and virt-handler when virt-launcher dies unexpectedly.
- [PR #6392][rmohr] Better place vcpu threads on host cpus to form more efficient passthrough architectures
- [PR #6251][rmohr] Better place vcpu threads on host cpus to form more efficient passthrough architectures
- [PR #6344][kubevirt-bot] BugFix: hotplug was broken when using it with a hostpath volume that was on a separate device.
- [PR #6263][rmohr] Make k8s client rate limits configurable
- [PR #6207][kubevirt-bot] Fix goroutine leak in virt-handler, potentially causing issues with a high turnover of VMIs.
- [PR #6101][rmohr] Make k8s client rate limits configurable
- [PR #6249][kubevirt-bot] Fix rbac permissions for freeze/unfreeze, addvolume/removevolume, guestosinfo, filesystemlist and userlist

Contributors
------------
12 people contributed to this release:

```
15	Roman Mohr <rmohr@redhat.com>
14	Shelly Kagan <skagan@redhat.com>
5	Igor Bezukh <ibezukh@redhat.com>
4	Alexander Wels <awels@redhat.com>
3	Israel Pinto <ipinto@redhat.com>
2	Jed Lejosne <jed@redhat.com>
1	Adam Litke <alitke@redhat.com>
1	Kedar Bidarkar <kbidarka@redhat.com>
1	Kevin Wiesmueller <kwiesmul@redhat.com>
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
