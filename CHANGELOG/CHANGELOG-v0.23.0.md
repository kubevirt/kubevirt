KubeVirt v0.23.0
================

This release follows v0.22.0 and consists of 91 changes, contributed by
15 people, leading to 216 files changed, 9040 insertions(+), 2258 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Guest OS Information is available under the VMI status now
- Updated to Go 1.12.8 and latest bazel
- Updated go-yaml to v2.2.4, which has a ddos vulnerability fixed
- Cleaned up and fixed CRD scheme registration
- Several bugfixes
- Many CI improvements (e.g. more logs in case of test failures)

Contributors
------------

15 people contributed to this release:

```
        24	Roman Mohr <rmohr@redhat.com>
        14	ipinto <ipinto@redhat.com>
        10	Federico Paolinelli <fpaoline@redhat.com>
         5	Marc Sluiter <msluiter@redhat.com>
         3	Francesco Romani <fromani@redhat.com>
         3	Marcin Franczyk <mfranczy@redhat.com>
         3	Petr Kotas <pkotas@redhat.com>
         3	Prashanth Buddhala <pbudds@gmail.com>
         2	Artyom Lukianov <alukiano@redhat.com>
         2	Fabian Deutsch <fabiand@redhat.com>
         2	Vladik Romanovsky <vromanso@redhat.com>
         1	Alvaro Aleman <alv2412@googlemail.com>
         1	Guangming Wang <guangming.wang@daocloud.io>
         1	Kedar Bidarkar <kbidarka@redhat.com>
```

Test Results
------------

```
> Ran 393 of 461 Specs in 14846.610 seconds
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
