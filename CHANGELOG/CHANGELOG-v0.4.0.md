KubeVirt v0.4.0
===============

This release follows v0.3.0 and consists of 180 changes, contributed by
16 people, leading to 2272 files changed, 146578 insertions(+), 285629
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.4.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Fix several networking issues
- Add and enable OpenShift support to CI
- Add conditional Windows tests (if an image is present)
- Add subresources for console access
- virtctl config alignmnet with kubectl
- Fix API reference generation
- Stable UUIDs for OfflineVirtualMachines
- Build virtctl for MacOS and Windows
- Set default architecture to x86_64
- Major improvement to the CI infrastructure (all containerized)
- virtctl convenience functions for starting and stopping a VM

Contributors
------------

16 people contributed to this release:

```
        53	David Vossel <dvossel@redhat.com>
        44	Roman Mohr <rmohr@redhat.com>
        20	Marcus Sorensen <mls@apple.com>
        16	Stu Gott <sgott@redhat.com>
        11	Fabian Deutsch <fabiand@redhat.com>
         9	Lukianov Artyom <alukiano@redhat.com>
         7	Vladik Romanovsky <vromanso@redhat.com>
         4	Francesco Romani <fromani@redhat.com>
         4	Lukas Bednar <lbednar@redhat.com>
         3	Artyom Lukianov <alukiano@redhat.com>
         3	Marek Libra <mlibra@redhat.com>
         2	Marc Sluiter <marc@slintes.net>
         1	Petr Kotas <petr.kotas@gmail.com>
         1	Raghavendra Talur <rtalur@redhat.com>
         1	Ryan Hallisey <rhallise@redhat.com>
         1	mlsorensen <shadowsor@gmail.com>
```

Test Results
------------

```
> Ran 65 of 67 Specs in 2341.639 seconds
> FAIL! -- 60 Passed | 5 Failed | 0 Pending | 2 Skipped --- FAIL: TestTests (2341.64s)
```

Note: The test failures are due to CI problems.

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
