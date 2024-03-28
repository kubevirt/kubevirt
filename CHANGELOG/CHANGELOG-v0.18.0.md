KubeVirt v0.18.0
================

This release follows v0.17.0 and consists of 167 changes, contributed by
16 people, leading to 8712 files changed, 160209 insertions(+), 2178748
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.18.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Build: Use of go modules
- CI: Support for Kubernetes 1.13
- Countless testcase fixes and additions
- Several smaller bug fixes
- Improved upgrade documentation

Contributors
------------

16 people contributed to this release:

```
        44	Stu Gott <sgott@redhat.com>
        32	Arik Hadas <ahadas@redhat.com>
        20	David Vossel <dvossel@redhat.com>
        17	Artyom Lukianov <alukiano@redhat.com>
        15	Roman Mohr <rmohr@redhat.com>
        11	Marc Sluiter <msluiter@redhat.com>
         6	Sebastian Scheinkman <sscheink@redhat.com>
         5	Marcin Franczyk <mfranczy@redhat.com>
         4	Fabian Deutsch <fabiand@redhat.com>
         3	Denis Ollier <dollierp@redhat.com>
         3	Petr Kotas <pkotas@redhat.com>
         3	Vladik Romanovsky <vromanso@redhat.com>
         1	Daniel Hiller <dhiller@redhat.com>
         1	Ihar Hrachyshka <ihar@redhat.com>
         1	Ihar Hrachyshka <ihrachys@redhat.com>
         1	Keith Schincke <keith.schincke@gmail.com>
```

Test Results
------------

```
> Ran 311 of 359 Specs in 9209.761 seconds
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
