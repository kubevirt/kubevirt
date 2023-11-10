KubeVirt v0.13.0
================

This release follows v0.12.0 and consists of 18 changes, contributed by
6 people, leading to 84 files changed, 516 insertions(+), 611 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.13.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- CI: Fix virt-api race
- API: Remove volumeName from disks

Contributors
------------

6 people contributed to this release:

```
         6	David Vossel <dvossel@redhat.com>
         4	Stu Gott <sgott@redhat.com>
         3	Marc Sluiter <msluiter@redhat.com>
         3	Roman Mohr <rmohr@redhat.com>
         1	Fabian Deutsch <fabiand@redhat.com>
         1	Ihar Hrachyshka <ihar@redhat.com>
```

Test Results
------------

```
> Ran 217 of 257 Specs in 6392.444 seconds
> PASS
```

Additional Resources
--------------------

- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]

[git-evtag]: https://github.com/cgwalters/git-evtag#using-git-evtag
[contributing]: https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/master/LICENSE
