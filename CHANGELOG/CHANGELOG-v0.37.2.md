KubeVirt v0.37.2
================

This release follows v0.37.1 and consists of 16 changes, contributed by 4 people, leading to 36 files changed, 804 insertions(+), 457 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.37.2.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.37.2`.

Pre-built containers are published on Docker Hub and can be viewed at: <https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- [PR #4872][kubevirt-bot] Add spec.domain.devices.useVirtioTransitional boolean to support virtio-transitional for old guests
- [PR #4855][kubevirt-bot] Fix an issue where it may not be able to update the KubeVirt CR after creation for up to minutes due to certificate propagation delays

Contributors
------------
4 people contributed to this release:

```
12	Roman Mohr <rmohr@redhat.com>
2	David Vossel <dvossel@redhat.com>
```

Additional Resources
--------------------

- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]

[contributing]: https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/master/LICENSE
