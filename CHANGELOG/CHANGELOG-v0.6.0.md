KubeVirt v0.6.0
===============

This release follows v0.5.0 and consists of 247 changes, contributed by
19 people, leading to 168 files changed, 6035 insertions(+), 2389 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.6.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- A range of flakyness reducing test fixes
- Vagrant setup got deprectated
- Updated Docker and CentOS versions
- Add Kubernetes 1.10.3 to test matrix
- A couple of ginkgo concurrency fixes
- A couple of spelling fixes
- A range if infra updates

- Use /dev/kvm if possible, otherwise fallback to emulation
- Add default view/edit/admin RBAC Roles
- Network MTU fixes
- CDRom drives are now read-only
- Secrets can now be correctly referenced on VMs
- Add disk boot ordering
- Add virtctl version
- Add virtctl expose
- Fix virtual machine memory calculations
- Add basic virtual machine Network API

Contributors
------------

19 people contributed to this release:

```
        89	Roman Mohr <rmohr@redhat.com>
        32	Yuval Lifshitz <ylifshit@redhat.com>
        22	Stu Gott <sgott@redhat.com>
        16	David Vossel <dvossel@redhat.com>
        13	Artyom Lukianov <alukiano@redhat.com>
        13	Ihar Hrachyshka <ihar@redhat.com>
        13	Marcus Sorensen <mls@apple.com>
        13	Sebastian Scheinkman <sscheink@redhat.com>
         9	Marcin Franczyk <mfranczy@redhat.com>
         8	Vladik Romanovsky <vromanso@redhat.com>
         5	Alexander Wels <awels@redhat.com>
         4	Marc Sluiter <msluiter@redhat.com>
         3	Fabian Deutsch <fabiand@redhat.com>
         2	nmm <nmm@localhost.localdomain>
         1	Gabriel Szasz <gszasz@redhat.com>
         1	Jason Brooks <jbrooks@redhat.com>
         1	Lukas Bednar <lbednar@redhat.com>
         1	Martin Kletzander <mkletzan@redhat.com>
         1	Phi|eas |ebada <norpol@users.noreply.github.com>
```

Test Results
------------

```
> Ran 105 of 115 Specs in 3638.019 seconds
> SUCCESS! -- 105 Passed | 0 Failed | 0 Pending | 10 Skipped PASS
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
