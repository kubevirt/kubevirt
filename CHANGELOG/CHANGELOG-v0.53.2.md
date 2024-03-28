KubeVirt v0.53.2
================

This release follows v0.53.1 and consists of 44 changes, contributed by 12 people, leading to 42 files changed, 522 insertions(+), 293 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.53.2.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.53.2`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #7883][kubevirt-bot] Enable to run libguestfs-tools pod to run as noroot user
- [PR #7794][kubevirt-bot] Allow `virtualmachines/migrate` subresource to admin/edit users
- [PR #7866][kubevirt-bot] Adds the reason of a live-migration failure to a recorded event in case EvictionStrategy is set but live-migration is blocked due to its limitations.
- [PR #7726][kubevirt-bot] BugFix: virtctl guestfs incorrectly assumes image name

Contributors
------------
12 people contributed to this release:

```
16	Jed Lejosne <jed@redhat.com>
4	Or Shoval <oshoval@redhat.com>
2	Alex Kalenyuk <akalenyu@redhat.com>
2	fossedihelm <ffossemo@redhat.com>
1	Alice Frosi <afrosi@redhat.com>
1	Daniel Hiller <dhiller@redhat.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Karel Šimon <ksimon@redhat.com>
1	L. Pivarc <lpivarc@redhat.com>
1	Pavel Kratochvil <pakratoc@redhat.com>
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
