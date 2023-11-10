KubeVirt v0.48.1
================

This release follows v0.48.0 and consists of 4 changes, contributed by 3 people, leading to 4 files changed, 90 insertions(+), 62 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.48.1.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.48.1`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6900][kubevirt-bot] Skip SSH RSA auth if no RSA key was explicitly provided and not key exists at the default location
- [PR #6902][kubevirt-bot] Fix "Make raw terminal failed: The handle is invalid?" issue with "virtctl console" when not executed in a pty

Contributors
------------
3 people contributed to this release:

```
2	Roman Mohr <rmohr@redhat.com>
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
