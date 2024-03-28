KubeVirt v0.15.0
================

This release follows v0.14.0 and consists of 273 changes, contributed by
28 people, leading to 2300 files changed, 59757 insertions(+), 4269
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.15.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- CI: Several fixes
- Fix configurable number of KVM devices
- Narrow virt-handler permissions
- Use bazel for development builds
- Support for live migration with shared and non-shared disks
- Support for live migration progress tracking
- Support for EFI boot
- Support for libvirt 5.0
- Support for extra DHCP options
- Support for a hook to manipualte cloud-init metadata
- Support setting a VM serial number
- Support for exposing infra and VM metrics
- Support for a tablet input device
- Support for extra CPU flags
- Support for ignition metadata
- Support to set a default CPU model
- Update to go 1.11.5

Contributors
------------

28 people contributed to this release:

```
        44	Vladik Romanovsky <vromanso@redhat.com>
        43	Artyom Lukianov <alukiano@redhat.com>
        39	David Vossel <dvossel@redhat.com>
        24	Francesco Romani <fromani@redhat.com>
        24	Greg Bock <greg.bock@stackpath.com>
        18	Roman Mohr <rmohr@redhat.com>
         9	Karel Šimon <ksimon@redhat.com>
         9	Tareq Alayan <talayan@redhat.com>
         8	Marc Sluiter <msluiter@redhat.com>
         6	Yanir Quinn <yquinn@redhat.com>
         6	bharat <bharat@cloudflare.com>
         5	Marcus Sorensen <marcus_sorensen@apple.com>
         5	Yossi Segev <ysegev@redhat.com>
         4	Meni Yakove <myakove@redhat.com>
         4	Sebastian Scheinkman <sscheink@redhat.com>
         4	bharatnc <bharatnc@gmail.com>
         3	Marcus Sorensen <mls@apple.com>
         2	Arik Hadas <ahadas@redhat.com>
         2	Bharat Nallan Chakravarthy <bharat@cloudflare.com>
         2	Karim Boumedhel <kboumedh@redhat.com>
         2	Marcin Franczyk <mfranczy@redhat.com>
         2	Nelly Credi <ncredi@redhat.com>
         2	Vijay Bellur <vbellur@redhat.com>
         2	yossisegev <40713576+yossisegev@users.noreply.github.com>
         1	Michael Henriksen <mhenriks@redhat.com>
         1	Stu Gott <sgott@redhat.com>
         1	annastopel <astopel@redhat.com>
         1	mlsorensen <shadowsor@gmail.com>
```

Test Results
------------

```
> Ran 244 of 289 Specs in 7368.455 seconds
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
