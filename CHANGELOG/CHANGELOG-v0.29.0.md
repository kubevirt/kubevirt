KubeVirt v0.29.0
================

This release follows v0.28.0 and consists of 241 changes, contributed by
28 people, leading to 302 files changed, 12931 insertions(+), 5829 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.29.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Tests: Many many test fixes
- Tests: Many more test fixes
- CI: Add lane with SELinux enabled
- CI: Drop PPC64 support for now
- Drop Genie support
- Drop the use of hostPaths in the virt-launcher for improved security
- Support priority classes for important componenets
- Support IPv6 over masquerade binding
- Support certificate rotations based on shared secrets
- Support for VM ready condition
- Support for advanced node labelling (supported CPU Families and machine types)

Contributors
------------

28 people contributed to this release:

```
        38	David Vossel <dvossel@redhat.com>
        25	Roman Mohr <rmohr@redhat.com>
        21	Alona Kaplan <alkaplan@redhat.com>
        16	Vladik Romanovsky <vromanso@redhat.com>
        14	Stu Gott <sgott@redhat.com>
        10	Petr Horacek <phoracek@redhat.com>
         9	Jed Lejosne <jed@redhat.com>
         6	Karel Simon <ksimon@redhat.com>
         6	Miguel Duarte Barroso <mdbarroso@redhat.com>
         6	Quique Llorente <ellorent@redhat.com>
         4	Daniel Hiller <daniel.hiller.1972@gmail.com>
         4	Igor Bezukh <ibezukh@redhat.com>
         3	Kedar Bidarkar <kbidarka@redhat.com>
         3	Marcus Sorensen <marcus_sorensen@apple.com>
         2	Daniel Belenky <dbelenky@redhat.com>
         2	Edward Haas <edwardh@redhat.com>
         2	Omer Yahud <oyahud@redhat.com>
         2	Vatsal Parekh <vparekh@redhat.com>
         1	Andrej Krejcir <akrejcir@redhat.com>
         1	Fabian Deutsch <fabiand@redhat.com>
         1	Or Shoval <oshoval@redhat.com>
         1	Petr Kotas <pkotas@redhat.com>
         1	Ravid Brown <>
         1	Ravid Brown <ravid@redhat.com>
         1	Yan Du <yadu@redhat.com>
         1	oatakan <oatakan@gmail.com>
         1	root <root@zeus06.eng.lab.tlv.redhat.com>
```

Test Results
------------

```
> Ran 470 of 552 Specs in 11677.537 seconds
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
