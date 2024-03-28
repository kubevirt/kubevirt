KubeVirt v0.2.0
===============

This release follows v0.1.0 and consists of 131 changes, contributed by
6 people, leading to 148 files changed, 9096 insertions(+), 5871 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.2.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- VM launch and shutdown flow improvements
- VirtualMachine API redesign
- Removal of HAProxy
- Redesign of VNC/Console access
- Initial support for different vagrant providers

Contributors
------------

6 people contributed to this release:

```
        65	Roman Mohr <rmohr@redhat.com>
        60	David Vossel <dvossel@redhat.com>
         2	Fabian Deutsch <fabiand@redhat.com>
         2	Stu Gott <sgott@redhat.com>
         1	Marek Libra <mlibra@redhat.com>
         1	Martin Kletzander <mkletzan@redhat.com>
```

Test Results
------------

```
> Ran 40 of 42 Specs in 703.532 seconds
> SUCCESS! -- 40 Passed | 0 Failed | 0 Pending | 2 Skipped PASS
```

Additional Resources
--------------------

- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- IRC: <irc://irc.freenode.net/#kubevirt>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]

[git-evtag]: https://github.com/cgwalters/git-evtag#using-git-evtag
[contributing]: https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/master/LICENSE
