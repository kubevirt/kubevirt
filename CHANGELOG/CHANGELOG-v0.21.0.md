KubeVirt v0.21.0
================

his release follows v0.20.0 and consists of 176 changes, contributed by
28 people, leading to 234 files changed, 23892 insertions(+), 1616 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.21.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- CI: Support for Kubernetes 1.14
- Many bug fixes in several areas
- Support for `virtctl migrate`
- Support configurable number of controller threads
- Support to opt-out of bridge binding for podnetwork
- Support for OpenShift Prometheus monitoring
- Support for setting more SMBIOS fields
- Improved containerDisk memory usage and speed
- Fix CRI-O memory limit
- Drop spc_t from launcher
- Add feature gates to security sensitive features

Contributors
------------

28 people contributed to this release:

```
        40	Marc Sluiter <msluiter@redhat.com>
        29	Arik Hadas <ahadas@redhat.com>
        24	Stu Gott <sgott@redhat.com>
        13	kubevirt-bot <rmohr+kubebot@redhat.com>
        10	Vatsal Parekh <vatsalparekh@outlook.com>
         6	Marcin Franczyk <mfranczy@redhat.com>
         5	Federico Paolinelli <fpaoline@redhat.com>
         5	Ihar Hrachyshka <ihrachys@redhat.com>
         5	Petr Kotas <pkotas@redhat.com>
         5	Vladik Romanovsky <vromanso@redhat.com>
         3	Daniel Hiller <daniel.hiller.1972@googlemail.com>
         3	Prashanth Buddhala <pbudds@gmail.com>
         3	alonSadan <asadan@redhat.com>
         3	jichenjc <jichenjc@cn.ibm.com>
         2	Artyom Lukianov <alukiano@redhat.com>
         2	Daniel Belenky <dbelenky@redhat.com>
         2	Daniel Hiller <daniel.hiller.1972@gmail.com>
         2	Fabian Deutsch <fabiand@redhat.com>
         2	Francesco Romani <fromani@redhat.com>
         2	Ihar Hrachyshka <ihar@redhat.com>
         2	Kedar Bidarkar <kbidarka@redhat.com>
         2	Vatsal Parekh <vparekh@redhat.com>
         1	David Vossel <dvossel@redhat.com>
         1	Denys Shchedrivyi <dshchedr@redhat.com>
         1	Ido Rosenzwig <irosenzw@redhat.com>
         1	Sheng Lin <shelin@nvidia.com>
         1	architb <architb@nvidia.com>
         1	rnetser <rnetser@redhat.com>
```

Test Results
------------

```
> Ran 377 of 454 Specs in 13898.938 seconds
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
