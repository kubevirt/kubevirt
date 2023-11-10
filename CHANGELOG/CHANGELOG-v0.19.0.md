KubeVirt v0.19.0
================

This release follows v0.18.0 and consists of 216 changes, contributed by
26 people, leading to 621 files changed, 21307 insertions(+), 11875
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.19.0-rc.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Fixes when run on kind
- Fixes for sub-resource RBAC
- Limit pod network interface bindings
- Many additional bug fixes in many areas
- Additional testcases for updates, disk types, live migration with NFS
- Additional testcases for memory over-commit, block storage, cpu manager,
headless mode
- Improvements around HyperV
- Improved error handling for runStartegies
- Improved update procedure
- Improved network metrics reporting (packets and errors)
- Improved guest overhead calculation
- Improved SR-IOV testsuite
- Support for live migration auto-converge
- Support for config-drive disks
- Support for setting a pullPolicy con containerDisks
- Support for unprivileged VMs when using SR-IOV
- Introduction of a project security policy

Contributors
------------

26 people contributed to this release:

```
        33	Marc Sluiter <msluiter@redhat.com>
        28	Roman Mohr <rmohr@redhat.com>
        27	Vladik Romanovsky <vromanso@redhat.com>
        24	David Vossel <dvossel@redhat.com>
        12	Artyom Lukianov <alukiano@redhat.com>
        11	Arik Hadas <ahadas@redhat.com>
        11	Francesco Romani <fromani@redhat.com>
         9	Daniel Gonzalez <daniel@gonzalez-nothnagel.de>
         8	Daniel Hiller <daniel.hiller.1972@gmail.com>
         7	Ihar Hrachyshka <ihar@redhat.com>
         7	Marcin Franczyk <mfranczy@redhat.com>
         6	Daniel Hiller <daniel.hiller.1972@googlemail.com>
         5	Petr Kotas <pkotas@redhat.com>
         5	Stu Gott <sgott@redhat.com>
         4	Ihar Hrachyshka <ihrachys@redhat.com>
         4	Sebastian Scheinkman <sscheink@redhat.com>
         3	Kedar Bidarkar <kbidarka@redhat.com>
         2	Federico Paolinelli <fpaoline@redhat.com>
         2	Kunal Kushwaha <kushwaha_kunal_v7@lab.ntt.co.jp>
         2	j-griffith <john.griffith8@gmail.com>
         1	Fabian Deutsch <fabiand@redhat.com>
         1	Federico Paolinelli <fedepaol@gmail.com>
         1	Jim Ma <ema@redhat.com>
         1	Mark Knowles <mknowles@redhat.com>
         1	Petr Horacek <phoracek@redhat.com>
         1	Vatsal Parekh <vparekh@redhat.com>
```

Test Results
------------

```
> Ran 356 of 404 Specs in 11020.915 seconds
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
