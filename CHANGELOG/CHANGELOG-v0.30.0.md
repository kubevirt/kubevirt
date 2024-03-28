KubeVirt v0.30.0
================

This release follows v0.29.0 and consists of 168 changes, contributed by
32 people, leading to 143 files changed, 22257 insertions(+), 6718 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.30.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Tests: Many more test fixes
- Security: Introduce a custom SELinux policy for virt-launcher
- More user friendly IPv6 default CIDR for IPv6 addresses
- Fix OpenAPI compatibility issues by switching to openapi-gen
- Improved support for EFI boot (configurable OVMF path and test fixes)
- Improved VMI IP reporting
- Support propagation of annotations from VMI to pods
- Support for more fine grained (NET_RAW( capability granting to virt-launcher
- Support for eventual consistency with DataVolumes

Contributors
------------

32 people contributed to this release:

```
        14	Roman Mohr <rmohr@redhat.com>
        12	Jed Lejosne <jed@redhat.com>
        12	Miguel Duarte Barroso <mdbarroso@redhat.com>
         8	Edward Haas <edwardh@redhat.com>
         8	L. Pivarc <lpivarc@redhat.com>
         8	Marcus Sorensen <marcus_sorensen@apple.com>
         5	Petr Horacek <phoracek@redhat.com>
         5	Stu Gott <sgott@redhat.com>
         4	Daniel Belenky <dbelenky@redhat.com>
         3	Igor Bezukh <ibezukh@redhat.com>
         3	Karel Simon <ksimon@redhat.com>
         3	Or Shoval <oshoval@redhat.com>
         3	Prashanth Buddhala <pbudds@gmail.com>
         2	Alona Kaplan <alkaplan@redhat.com>
         2	Andrea Bolognani <abologna@redhat.com>
         2	Dan Kenigsberg <danken@redhat.com>
         2	David Vossel <dvossel@redhat.com>
         2	Omer Yahud <oyahud@redhat.com>
         2	Or Mergi <ormergi@redhat.com>
         2	rnetser <rnetser@redhat.com>
         1	Daniel Hiller <daniel.hiller.1972@gmail.com>
         1	Doron Fediuck <doron-fediuck@users.noreply.github.com>
         1	Jim Fehlig <jfehlig@suse.com>
         1	Kunal Kushwaha <kunalkushwaha453@gmail.com>
         1	Maya Rashish <mrashish@redhat.com>
         1	Murilo Fossa Vicentini <muvic@linux.ibm.com>
         1	Pedro Ibáñez <pedro@redhat.com>
         1	Tomasz Baranski <tbaransk@redhat.com>
         1	Vatsal Parekh <vparekh@redhat.com>
         1	ipinto <ipinto@redhat.com>
         1	pnavarro <pednape@gmail.com>
```

Test Results
------------

```
> Ran 470 of 554 Specs in 11936.748 seconds
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
