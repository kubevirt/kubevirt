KubeVirt v0.10.0
================

This release follows v0.9.0 and consists of 253 changes, contributed by
26 people, leading to 1376 files changed, 268565 insertions(+), 9773
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.10.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Support for vhost-net
- Support for block multi-queue
- Support for custom PCI addresses for virtio devices
- Support for deploying KubeVirt to a custom namespace
- Support for ServiceAccount token disks
- Support for multus backed networks
- Support for genie backed networks
- Support for kuryr backed networks
- Support for block PVs
- Support for configurable disk device caches
- Support for pinned IO threads
- Support for virtio net multi-queue
- Support for image upload (depending on CDI)
- Support for custom entity lists with more VM details (cusomt columns)
- Support for IP and MAC address reporting of all vNICs
- Basic support for guest agent status reporting
- More structured logging
- Better libvirt error reporting
- Stricter CR validation
- Better ownership references
- Several test improvements

Contributors
------------

26 people contributed to this release:

```
        54	Roman Mohr <rmohr@redhat.com>
        50	David Vossel <dvossel@redhat.com>
        29	Vladik Romanovsky <vromanso@redhat.com>
        20	Stu Gott <sgott@redhat.com>
        20	Yuval Lifshitz <ylifshit@redhat.com>
        13	Marcin Franczyk <mfranczy@redhat.com>
        11	Marc Sluiter <msluiter@redhat.com>
         8	Gabriel Szasz <gszasz@redhat.com>
         7	Artyom Lukianov <alukiano@redhat.com>
         6	Michael Henriksen <mhenriks@redhat.com>
         5	Petr Kotas <pkotas@redhat.com>
         4	Koichiro Den <den@klaipeden.com>
         4	Sebastian Scheinkman <sscheink@redhat.com>
         3	Arik Hadas <ahadas@redhat.com>
         3	imjoey <majunjiev@gmail.com>
         3	steigr <me@stei.gr>
         2	Alexander Gallego <gallego.alexx@gmail.com>
         2	Gage Orsburn <gageorsburn@live.com>
         2	Rich Renner <renner@osi.io>
         1	Alexander Wels <awels@redhat.com>
         1	Fabian Deutsch <fabiand@redhat.com>
         1	Ihar Hrachyshka <ihar@redhat.com>
         1	Karim Boumedhel <kboumedh@redhat.com>
         1	Marcin Mirecki <mmirecki@redhat.com>
         1	Shiyang Wang <shiywang@redhat.com>
         1	renner <renner@pop-os.localdomain>
```

Test Results
------------

```
> Ran 180 of 216 Specs in 5647.016 seconds
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
