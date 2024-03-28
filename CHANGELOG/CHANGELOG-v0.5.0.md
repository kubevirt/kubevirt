KubeVirt v0.5.0
===============

This release follows v0.4.1 and consists of 151 changes, contributed by
12 people, leading to 1415 files changed, 138035 insertions(+), 9848
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.5.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Better controller health signaling
- Better virtctl error messages
- Improvements to enable CRI-O support
- Run CI on stable OpenShift
- Add test coverage for multiple PVCs
- Improved controller life-cycle guarantees
- Add Webhook validation
- Add tests coverage for node eviction
- OfflineVirtualMachine status improvements
- RegistryDisk API update

Contributors
------------

12 people contributed to this release:

```
        71	Roman Mohr <rmohr@redhat.com>
        53	David Vossel <dvossel@redhat.com>
         7	Artyom Lukianov <alukiano@redhat.com>
         6	Fabian Deutsch <fabiand@redhat.com>
         4	Petr Kotas <pkotas@redhat.com>
         2	Lukas Bednar <lbednar@redhat.com>
         2	Marcus Sorensen <mls@apple.com>
         2	Yuval Lifshitz <ylifshit@redhat.com>
         1	Alexander Wels <awels@redhat.com>
         1	Guohua Ouyang <gouyang@redhat.com>
         1	Karim Boumedhel <kboumedh@redhat.com>
         1	Travis CI <travis@travis-ci.org>
```

Test Results
------------

```
> Ran 82 of 90 Specs in 3224.968 seconds
> FAIL! -- 81 Passed | 1 Failed | 0 Pending | 8 Skipped --- FAIL: TestTests (3224.98s)
```

Note: The tests are a little flaky, thus not a complete pass for this release.

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
