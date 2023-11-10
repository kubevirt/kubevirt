KubeVirt v0.0.4
===============

This release follows v0.0.3 and consists of 133 changes, contributed by
14 people, leading to 109 files changed, 7093 insertions(+), 2437 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.0.4>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Add support for node affinity to VM.Spec
- Add OpenAPI specification
- Drop swagger 1.2 specification
- virt-launcher refactoring
- Leader election mechanism for virt-controller
- Move from glide to dep for dependency management
- Improve virt-handler synchronization loops
- Add support for running the functional tests on oVirt infrastructure
- Several tests fixes (spice, cleanup, ...)
- Add console test tool
- Improve libvirt event notification

Contributors
------------

14 people contributed to this release:

```
        46	David Vossel <dvossel@redhat.com>
        46	Roman Mohr <rmohr@redhat.com>
        12	Lukas Bednar <lbednar@redhat.com>
        11	Lukianov Artyom <alukiano@redhat.com>
         4	Martin Sivak <msivak@redhat.com>
         4	Petr Kotas <pkotas@redhat.com>
         2	Fabian Deutsch <fabiand@redhat.com>
         2	Milan Zamazal <mzamazal@redhat.com>
         1	Artyom Lukianov <alukiano@redhat.com>
         1	Barak Korren <bkorren@redhat.com>
         1	Clifford Perry <coperry94@gmail.com>
         1	Martin Polednik <mpolednik@redhat.com>
         1	Stephen Gordon <sgordon@redhat.com>
         1	Stu Gott <sgott@redhat.com>
```

Test Results
------------

```
> Ran 45 of 47 Specs in 797.286 seconds
> SUCCESS! -- 45 Passed | 0 Failed | 0 Pending | 2 Skipped PASS
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
