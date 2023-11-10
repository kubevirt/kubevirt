KubeVirt v0.6.2
===============

This release follows v0.6.1 and consists of 17 changes, contributed by
3 people, leading to 41 files changed, 343 insertions(+), 73 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.6.2>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Binary relocation for packaging
- QEMU Process detection
- Role aggregation
- CPU Model selection
- VM Rename fix

Contributors
------------

3 people contributed to this release:

```
         8	Fabian Deutsch <fabiand@redhat.com>
         5	Artyom Lukianov <alukiano@redhat.com>
         4	Roman Mohr <rmohr@redhat.com>
```

Test Results
------------

```
> Ran 117 of 128 Specs in 3774.947 seconds
> SUCCESS! -- 117 Passed | 0 Failed | 0 Pending | 11 Skipped PASS
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
