KubeVirt v0.40.1
================

This release follows v0.40.0 and consists of 19 changes, contributed by 7 people, leading to 40 files changed, 1153 insertions(+), 94 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.40.1.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.40.1`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6598][jean-edouard] VMs with cloud-init data should now properly migrate from older KubeVirt versions
- [PR #6287][kubevirt-bot] Fix goroutine leak in virt-handler, potentially causing issues with a high turnover of VMIs.
- [PR #5559][kubevirt-bot] Fix `docker save` issues with kubevirt images
- [PR #5500][kubevirt-bot] Support hotplug with virtctl using addvolume and removevolume commands

Contributors
------------
7 people contributed to this release:

```
7	Alexander Wels <awels@redhat.com>
4	Jed Lejosne <jed@redhat.com>
1	Kevin Wiesmueller <kwiesmul@redhat.com>
1	Roman Mohr <rmohr@redhat.com>
1	Vasiliy Ulyanov <vulyanov@suse.de>
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
