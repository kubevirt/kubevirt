KubeVirt v0.16.0
================

This release follows v0.15.0 and consists of 228 changes, contributed by
24 people, leading to 790 files changed, 63334 insertions(+), 2560 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.16.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Bazel fixes
- Initial work to support upgrades (not finalized)
- Initial support for HyperV features
- Support propagation of MAC addresses to multus
- Support live migration cancellation
- Support for table input devices
- Support for generating OLM metadata
- Support for triggering VM live migration on node taints

Contributors
------------

24 people contributed to this release:

```
        68	Roman Mohr <rmohr@redhat.com>
        38	David Vossel <dvossel@redhat.com>
        33	Vladik Romanovsky <vromanso@redhat.com>
        20	Marc Sluiter <msluiter@redhat.com>
        19	Greg Bock <greg.bock@stackpath.com>
         9	Denis Ollier <dollierp@redhat.com>
         7	Ihar Hrachyshka <ihar@redhat.com>
         6	Artyom Lukianov <alukiano@redhat.com>
         4	Arik Hadas <ahadas@redhat.com>
         3	Francesco Romani <fromani@redhat.com>
         3	Ihar Hrachyshka <ihrachys@redhat.com>
         3	Marc Koderer <marc@koderer.com>
         3	Sebastian Scheinkman <sscheink@redhat.com>
         2	Gladkov Alexey <agladkov@redhat.com>
         1	10240987 <ji.yuan@zte.com.cn>
         1	Kedar Bidarkar <kbidarka@redhat.com>
         1	Michael Henkel <michael.henkel@gmail.com>
         1	Michael Henriksen <mhenriks@redhat.com>
         1	Petr Kotas <pkotas@redhat.com>
         1	Quique Llorente <ellorent@redhat.com>
         1	Stu Gott <sgott@redhat.com>
         1	Tareq Alayan <talayan@redhat.com>
         1	Yossi Segev <ysegev@redhat.com>
         1	nitkon <niteshkonkar@in.ibm.com>
```

Test Results
------------

```
> Ran 260 of 308 Specs in 7643.227 seconds
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
