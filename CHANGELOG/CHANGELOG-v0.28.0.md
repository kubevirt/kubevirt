KubeVirt v0.28.0
================

This release follows v0.27.0 and consists of 131 changes, contributed by
20 people, leading to 172 files changed, 7960 insertions(+), 1796 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.28.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- CI: Try to discover flaky tests before merge
- Fix the use of priorityClasses
- Fix guest memory overhead calculation
- Fix SR-IOV device overhead requirements
- Fix loading of tun module during virt-handler initialization
- Fixes for several test cases
- Fixes to support running with container_t
- Support for renaming a vM
- Support ioEmulator thread pinning
- Support a couple of alerts for virt-handler
- Support for filesystem listing using the guest agent
- Support for retrieving data from the guest agent
- Support for device role tagging
- Support for assigning devices to the PCI root bus
- Support for guest overhead override
- Rewrite container-disk in C to in order to reduce it's memory footprint

Contributors
------------

20 people contributed to this release:

```
        18	Vladik Romanovsky <vromanso@redhat.com>
        16	Roman Mohr <rmohr@redhat.com>
        13	Omer Yahud <oyahud@redhat.com>
         7	Daniel Belenky <dbelenky@redhat.com>
         6	Petr Kotas <pkotas@redhat.com>
         6	Stu Gott <sgott@redhat.com>
         5	Daniel Hiller <daniel.hiller.1972@gmail.com>
         4	Igor Bezukh <ibezukh@redhat.com>
         3	Jed Lejosne <jed@redhat.com>
         2	Andrej Krejcir <akrejcir@redhat.com>
         2	Marcus Sorensen <marcus_sorensen@apple.com>
         2	Petr Horacek <phoracek@redhat.com>
         2	Quique Llorente <ellorent@redhat.com>
         2	ipinto <ipinto@redhat.com>
         1	Artyom Lukianov <alukiano@redhat.com>
         1	Jim Fehlig <jfehlig@suse.com>
         1	Miguel Duarte Barroso <mdbarroso@redhat.com>
         1	Omer Yahud <oyahud@oyahud.tlv.csb>
         1	ge.jin <ge.jin@woqutech.com>
```

Test Results
------------

```
> Ran 455 of 540 Specs in 11312.197 seconds
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
