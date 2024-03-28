KubeVirt v0.4.1
===============

This release follows v0.4.0 and consists of 40 changes, contributed by
8 people, leading to 297 files changed, 42433 insertions(+), 699 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.4.1>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- VM shutdown fixes and tests
- Functional test for CRD validation
- Windows VM test
- DHCP link-local change

Contributors
------------

8 people contributed to this release:

```
        18	David Vossel <dvossel@redhat.com>
         9	Roman Mohr <rmohr@redhat.com>
         8	Artyom Lukianov <alukiano@redhat.com>
         1	Lukas Bednar <lbednar@redhat.com>
         1	Marcus Sorensen <mls@apple.com>
         1	Petr Kotas <pkotas@redhat.com>
         1	Vladik Romanovsky <vromanso@redhat.com>
         1	karmab <karimboumedhel@gmail.com>
```

Test Results
------------

```
> Ran 71 of 77 Specs in 2185.664 seconds
> SUCCESS! -- 71 Passed | 0 Failed | 0 Pending | 6 Skipped PASS
> Ran 6 of 79 Specs in 459.489 seconds
> SUCCESS! -- 6 Passed | 0 Failed | 0 Pending | 73 Skipped PASS
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
