KubeVirt v0.27.0
================

This release follows v0.26.0 and consists of 165 changes, contributed by
22 people, leading to 197 files changed, 7671 insertions(+), 1256 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.27.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Support for more guest agent informations in the API
- Support setting priorityClasses on VMs
- Support for additional control plane alerts via prometheus
- Support for io and emulator thread pinning
- Support setting a custom SELinux type for the launcher
- Support to perform network configurations from handler instead of launcher
- Support to opt-out of auto attaching the serial console
- Support for different uninstallStaretgies for data protection
- Fix to let qemu run in the qemu group
- Fix guest agen connectivity check after i.e. live migrations

Contributors
------------

22 people contributed to this release:

```
        58	Ihar Hrachyshka <ihrachys@redhat.com>
        11	Stu Gott <sgott@redhat.com>
        10	L. Pivarc <lpivarc@redhat.com>
        10	Omer Yahud <oyahud@oyahud.tlv.csb>
         7	Roman Mohr <rmohr@redhat.com>
         6	Petr Kotas <pkotas@redhat.com>
         5	Daniel Hiller <daniel.hiller.1972@gmail.com>
         5	Igor Bezukh <ibezukh@redhat.com>
         4	Daniel Belenky <dbelenky@redhat.com>
         3	Or Shoval <oshoval@redhat.com>
         2	Alberto Losada <alosadag@redhat.com>
         2	David Vossel <dvossel@redhat.com>
         2	Vladik Romanovsky <vromanso@redhat.com>
         1	Alexander Wels <awels@redhat.com>
         1	Jed Lejosne <jed@redhat.com>
         1	Jim Fehlig <jfehlig@suse.com>
         1	Joowon Cheong <jwcheong0420@gmail.com>
         1	L. Pivarc <456130@mail.muni.cz>
         1	Murilo Fossa Vicentini <muvic@linux.ibm.com>
         1	Sally O'Malley <somalley@redhat.com>
         1	ipinto <ipinto@redhat.com>
```

Test Results
------------

```
> Ran 428 of 509 Specs in 13013.221 seconds
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
