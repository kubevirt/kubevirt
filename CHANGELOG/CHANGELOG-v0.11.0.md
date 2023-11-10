KubeVirt v0.11.0
================

This release follows v0.10.0 and consists of 170 changes, contributed by
25 people, leading to 349 files changed, 8497 insertions(+), 3065 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.11.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- API: registryDisk got renamed to containreDisk
- CI: User OKD 3.11
- Fix: Tolerate if the PVC has less capacity than expected
- Aligned to use ownerReferences
- Update to libvirt-4.10.0
- Support for VNC on MAC OSX
- Support for network SR-IOV interfaces
- Support for custom DHCP options
- Support for VM restarts via a custom endpoint
- Support for liveness and readiness probes

Contributors
------------

25 people contributed to this release:

```
        46	Roman Mohr <rmohr@redhat.com>
        24	Ihar Hrachyshka <ihar@redhat.com>
        17	Marc Sluiter <msluiter@redhat.com>
        15	Gage Orsburn <gageorsburn@live.com>
        10	Artyom Lukianov <alukiano@redhat.com>
         7	Petr Kotas <pkotas@redhat.com>
         6	Arik Hadas <ahadas@redhat.com>
         6	Marcin Franczyk <mfranczy@redhat.com>
         5	Quique Llorente <ellorent@redhat.com>
         4	Fabian Deutsch <fabiand@redhat.com>
         4	Frederik Carlier <frederik.carlier@quamotion.mobi>
         4	Vladik Romanovsky <vromanso@redhat.com>
         3	Dan Kenigsberg <danken@redhat.com>
         3	Daniel Belenky <dbelenky@redhat.com>
         3	Ihar Hrachyshka <ihrachys@redhat.com>
         2	Karim Boumedhel <kboumedh@redhat.com>
         2	Marcus Sorensen <mls@apple.com>
         2	Michael Henriksen <mhenriks@redhat.com>
         1	Adam Litke <alitke@redhat.com>
         1	Dan Kenigsberg <danken@gmail.com>
         1	Karel Šimon <ksimon@redhat.com>
         1	Sebastian Scheinkman <sscheink@redhat.com>
         1	Shiyang Wang <shiywang@redhat.com>
         1	Stu Gott <sgott@redhat.com>
         1	imjoey <majunjiev@gmail.com>
```

Test Results
------------

```
> Ran 200 of 239 Specs in 7228.750 seconds
> PASS
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
