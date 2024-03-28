KubeVirt v0.8.0
===============

This release follows v0.7.0 and consists of 354 changes, contributed by
36 people, leading to 2612 files changed, 183877 insertions(+), 49008
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.8.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Support for DataVolume
- Support for a subprotocol for webbrowser terminals
- Support for virtio-rng
- Support disconnected VMs
- Support for setting host model
- Support for host CPU passthrough
- Support setting a vNICs mac and PCI address
- Support for memory over-commit
- Support booting from network devices
- Use less devices by default, aka disable unused ones
- Improved VMI shutdown status
- More logging to improve debugability
- A lot of small fixes, including typos and documentation fixes
- Race detection in tests
- Hook improvements
- Update to use Fedora 28 (includes updates of dependencies like libvirt and
  qemu)
- Move CI to support Kubernetes 1.11

Contributors
------------

36 people contributed to this release:

```
        76	Artyom Lukianov <alukiano@redhat.com>
        59	David Vossel <dvossel@redhat.com>
        43	Roman Mohr <rmohr@redhat.com>
        27	Sebastian Scheinkman <sscheink@redhat.com>
        23	Petr Kotas <pkotas@redhat.com>
        14	Petr Horáček <phoracek@redhat.com>
        14	Stu Gott <sgott@redhat.com>
        13	Marc Sluiter <msluiter@redhat.com>
        12	Shiyang Wang <shiywang@redhat.com>
        12	Vladik Romanovsky <vromanso@redhat.com>
         7	Ihar Hrachyshka <ihar@redhat.com>
         6	Ben Warren <bawarren@cisco.com>
         5	Fabian Deutsch <fabiand@redhat.com>
         5	dankenigsberg <danken@redhat.com>
         4	Ihar Hrachyshka <ihrachys@redhat.com>
         3	Arik Hadas <ahadas@redhat.com>
         3	Michael Henriksen <mhenriks@redhat.com>
         3	Yanir Quinn <yquinn@redhat.com>
         3	Yuval Lifshitz <ylifshit@redhat.com>
         3	root <root@sscheink.tlv.csb>
         2	Daniel Belenky <dbelenky@redhat.com>
         2	Gonzalo Rafuls <grafuls@redhat.com>
         2	dankenigsberg <danken@gmail.com>
         1	Alexander Wels <awels@redhat.com>
         1	Alvaro Aleman <alv2412@googlemail.com>
         1	Barak Korren <bkorren@redhat.com>
         1	Boaz Shuster <boaz.shuster.github@gmail.com>
         1	Francesco Romani <fromani@redhat.com>
         1	Gabriel Szasz <gszasz@redhat.com>
         1	Lukas Bednar <lbednar@redhat.com>
         1	Ryan Hallisey <rhallise@redhat.com>
         1	Simone Tiraboschi <stirabos@redhat.com>
         1	William Zhang <warmchang@outlook.com>
         1	gbenhaim <galbh2@gmail.com>
         1	imjoey <majunjiev@gmail.com>
         1	j-griffith <john.griffith8@gmail.com>
```

Test Results
------------

```
> Ran 149 of 163 Specs in 3851.878 seconds
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
