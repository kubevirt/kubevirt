KubeVirt v0.3.0
===============

This release follows v0.2.0 and consists of 469 changes, contributed by
20 people, leading to 5408 files changed, 2170742 insertions(+), 14691
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/HEAD>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Kubernetes compatible networking
- Kubernetes compatible PV based storage
- VirtualMachinePresets support
- OfflineVirtualMachine support
- RBAC improvements
- Switch to q35 machien type by default
- A large number of test and CI fixes
- Ephemeral disk support

Contributors
------------

20 people contributed to this release:

```
       103	Roman Mohr <rmohr@redhat.com>
        98	David Vossel <dvossel@redhat.com>
        70	Petr Kotas <petr.kotas@gmail.com>
        41	Stu Gott <sgott@redhat.com>
        31	Lukianov Artyom <alukiano@redhat.com>
        24	Fabian Deutsch <fabiand@redhat.com>
        20	Lukas Bednar <lbednar@redhat.com>
        19	Vladik Romanovsky <vromanso@redhat.com>
        16	Martin Polednik <mpolednik@redhat.com>
        14	Francesco Romani <fromani@redhat.com>
        10	Travis CI <travis@travis-ci.org>
         7	Yanir Quinn <yquinn@redhat.com>
         6	Ryan Hallisey <rhallise@redhat.com>
         2	Martin Kletzander <mkletzan@redhat.com>
         2	Saravanan KR <skramaja@redhat.com>
         2	gbenhaim <galbh2@gmail.com>
         1	Alexander Wels <awels@redhat.com>
         1	Suraj Narwade <surajnarwade353@gmail.com>
         1	Vatsal Parekh <vparekh@redhat.com>
         1	karmab <karimboumedhel@gmail.com>
```

Test Results
------------

```
> Ran 57 of 60 Specs in 3490.920 seconds
> SUCCESS! -- 57 Passed | 0 Failed | 0 Pending | 3 Skipped PASS
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
