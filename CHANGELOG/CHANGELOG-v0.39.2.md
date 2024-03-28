KubeVirt v0.39.2
================

This release follows v0.39.1 and consists of 12 changes, contributed by 6 people, leading to 37 files changed, 584 insertions(+), 59 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.39.2.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.39.2`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6597][jean-edouard] VMs with cloud-init data should now properly migrate from older KubeVirt versions
- [PR #5854][rthallisey] Prometheus metrics scraped from virt-handler are now served from the VMI informer cache, rather than calling back to the Kubernetes API for VMI information.
- [PR #5561][kubevirt-bot] Fix `docker save` issues with kubevirt images

Contributors
------------
6 people contributed to this release:

```
4	Jed Lejosne <jed@redhat.com>
3	Roman Mohr <rmohr@redhat.com>
1	Marcus Sorensen <mls@apple.com>
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
