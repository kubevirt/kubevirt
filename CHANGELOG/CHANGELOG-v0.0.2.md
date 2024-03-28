KubeVirt v0.0.2
===============

This release follows v0.0.1-alpha.5 and consists of 378 changes, contributed by
16 people, leading to 267 files changed, 13559 insertions(+), 17180
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.0.2>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Usage of CRDs
- Moved libvirt to a pod
- Introduction of `virtctl`
- Use glide instead of govendor
- Container based ephermal disks
- Contributing guide improvements
- Support for Kubernetes Namespaces

Contributors
------------

16 people contributed to this release:

```
       130	Roman Mohr <rmohr@redhat.com>
        50	Stu Gott <sgott@redhat.com>
        49	Fabian Deutsch <fabiand@redhat.com>
        47	David Vossel <davidvossel@gmail.com>
        31	Daniel Berrange <berrange@redhat.com>
        30	Adam Young <ayoung@redhat.com>
        15	Martin Polednik <mpolednik@redhat.com>
        10	Vladik Romanovsky <vromanso@redhat.com>
         8	Lukianov Artyom <alukiano@redhat.com>
         2	dankenigsberg <danken@gmail.com>
         1	Alexis Monville <alexis@monville.com>
         1	Allon Mureinik <amureini@redhat.com>
         1	Arik Hadas <ahadas@redhat.com>
         1	Bohdan <cyberbond95@gmail.com>
         1	Martin Sivak <msivak@redhat.com>
         1	Petr Kotas <pkotas@redhat.com>
```

Test Results
------------

```
> Ran 31 of 31 Specs in 192.789 seconds
> SUCCESS! -- 31 Passed | 0 Failed | 0 Pending | 0 Skipped PASS
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
