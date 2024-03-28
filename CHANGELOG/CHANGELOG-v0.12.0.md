KubeVirt v0.12.0
================

This release follows v0.11.0 and consists of 207 changes, contributed by
27 people, leading to 1107 files changed, 137791 insertions(+), 20497
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.12.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Introduce a KubeVirt Operator for KubeVirt life-cycle management
- Introduce dedicated kubevirt namespace
- Support VMI ready conditions
- Support vCPU threads and sockets
- Support scale and HPA for VMIRS
- Support to pass NTP related DHCP options
- Support guest IP address reporting via qemu guest agent
- Support for live migration with shared storage
- Support scheduling of VMs based on CPU family
- Support masquerade network interface binding

Contributors
------------

27 people contributed to this release:

```
        37	Marc Sluiter <msluiter@redhat.com>
        30	Vladik Romanovsky <vromanso@redhat.com>
        22	Roman Mohr <rmohr@redhat.com>
        20	Artyom Lukianov <alukiano@redhat.com>
        14	David Vossel <dvossel@redhat.com>
        14	Lukas Bednar <lbednar@redhat.com>
        12	Arik Hadas <ahadas@redhat.com>
        11	Dylan Redding <dylan.redding@stackpath.com>
         7	Sebastian Scheinkman <sscheink@redhat.com>
         7	Yanir Quinn <yquinn@redhat.com>
         6	Stu Gott <sgott@redhat.com>
         5	Ihar Hrachyshka <ihar@redhat.com>
         3	Fabian Deutsch <fabiand@redhat.com>
         3	Quique Llorente <ellorent@redhat.com>
         2	Justin Barrick <justin.m.barrick@gmail.com>
         2	Marcin Franczyk <mfranczy@redhat.com>
         2	Michael Henriksen <mhenriks@redhat.com>
         1	Frederik Carlier <frederik.carlier@quamotion.mobi>
         1	Ihar Hrachyshka <ihrachys@redhat.com>
         1	Karel Šimon <ksimon@redhat.com>
         1	Kunal Kushwaha <kushwaha_kunal_v7@lab.ntt.co.jp>
         1	Marcin Mirecki <mmirecki@redhat.com>
         1	Petr Kotas <pkotas@redhat.com>
         1	Richard Su <rwsu@redhat.com>
         1	Tzvi Avni <tavni@redhat.com>
         1	Yossi Segev <ysegev@redhat.com>
         1	ipinto <ipinto@redhat.com>
```

Test Results
------------

```
> Ran 217 of 257 Specs in 6555.536 seconds
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
