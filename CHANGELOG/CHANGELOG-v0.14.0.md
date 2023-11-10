KubeVirt v0.14.0
================

This release follows v0.13.0 and consists of 115 changes, contributed by
18 people, leading to 991 files changed, 57851 insertions(+), 44285
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.14.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- CI: Several stabilizing fixes
- docs: Document the KubeVirt Razor
- build: golang update
- Update to Kubernetes 1.12
- Update CDI
- Support for Ready and Created Operator conditions
- Support (basic) EFI
- Support for generating cloud-init network-config

Contributors
------------

18 people contributed to this release:

```
        26	Artyom Lukianov <alukiano@redhat.com>
        22	David Vossel <dvossel@redhat.com>
        13	Marc Sluiter <msluiter@redhat.com>
         8	Kedar Bidarkar <kbidarka@redhat.com>
         8	Yossi Segev <ysegev@redhat.com>
         7	Greg Bock <greg.bock@stackpath.com>
         6	gaahrdner <github@philipgardner.com>
         5	Stu Gott <sgott@redhat.com>
         4	Daniel Gonzalez <daniel@gonzalez-nothnagel.de>
         3	Fabian Deutsch <fabiand@redhat.com>
         3	Petr Kotas <pkotas@redhat.com>
         2	Justin Barrick <jbarrick@cloudflare.com>
         2	Marcin Franczyk <mfranczy@redhat.com>
         2	ipinto <ipinto@redhat.com>
         1	Alexander Wels <awels@redhat.com>
         1	Dan Kenigsberg <danken@redhat.com>
         1	Yan Du <yadu@redhat.com>
         1	yossisegev <40713576+yossisegev@users.noreply.github.com>
```

Test Results
------------

```
> Ran 217 of 257 Specs in 6293.783 seconds
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
