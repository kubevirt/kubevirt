KubeVirt v0.7.0
===============

This release follows v0.6.0 and consists of 351 changes, contributed by
23 people, leading to 747 files changed, 395521 insertions(+), 57735
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.7.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- CI: Move test storage to hostPath
- CI: Add support for Kubernetes 1.10.4
- CI: Improved network tests for multiple-interfaces
- CI: Drop Origin 3.9 support
- CI: Add test for testing templates on Origin
- VM to VMI rename
- VM affinity and anti-affinity
- Add awareness for multiple networks
- Add hugepage support
- Add device-plugin based kvm
- Add support for setting the network interface model
- Add (basic and inital) Kubernetes compatible networking approach (SLIRP)
- Add role aggregation for our roles
- Add support for setting a disks serial number
- Add support for specyfing the CPU model
- Add support for setting an network intefraces MAC address
- Relocate binaries for FHS conformance
- Logging improvements
- Template fixes
- Fix OpenShift CRD validation
- virtctl: Improve vnc logging improvements
- virtctl: Add expose
- virtctl: Use PATCH instead of PUT

Contributors
------------

23 people contributed to this release:

```
        72	Roman Mohr <rmohr@redhat.com>
        63	Artyom Lukianov <alukiano@redhat.com>
        48	Stu Gott <sgott@redhat.com>
        44	Ihar Hrachyshka <ihar@redhat.com>
        36	Sebastian Scheinkman <sscheink@redhat.com>
        19	David Vossel <dvossel@redhat.com>
        15	Fabian Deutsch <fabiand@redhat.com>
        12	Francesco Romani <fromani@redhat.com>
        11	Tzvi Avni <tavni@redhat.com>
         7	Marc Sluiter <msluiter@redhat.com>
         4	Vladik Romanovsky <vromanso@redhat.com>
         3	j-griffith <john.griffith8@gmail.com>
         3	tchughesiv <tchughesiv@gmail.com>
         2	Alexander Wels <awels@redhat.com>
         2	Gabriel Szasz <gszasz@redhat.com>
         2	Karim Boumedhel <kboumedh@redhat.com>
         2	Ryan Hallisey <rhallise@redhat.com>
         1	Adam Litke <alitke@redhat.com>
         1	Lukas Bednar <lbednar@redhat.com>
         1	Nelly Credi <ncredi@redhat.com>
         1	Shiyang Wang <shiywang@redhat.com>
         1	Thiago da Silva <thiago@redhat.com>
         1	Yanir Quinn <yquinn@redhat.com>
```

Test Results
------------

```
> Ran 119 of 135 Specs in 4126.052 seconds
> SUCCESS! -- 119 Passed | 0 Failed | 0 Pending | 16 Skipped PASS
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
