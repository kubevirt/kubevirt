KubeVirt v0.0.3
===============

This release follows v0.0.2 and consists of 198 changes, contributed by
9 people, leading to 165 files changed, 8321 insertions(+), 1928 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.0.3>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Containerized binary builds
- Socket based container detection
- cloud-init support
- Container based ephemeral disk support
- Basic RBAC profile
- client-go updates
- Rename of VM to VirtualMachine
- Introduction of VirtualMachineReplicaSet
- Improved migration events
- Improved API documentation

Contributors
------------

9 people contributed to this release:

```
        84	Roman Mohr <rmohr@redhat.com>
        70	David Vossel <davidvossel@gmail.com>
        15	Fabian Deutsch <fabiand@redhat.com>
         9	Lukianov Artyom <alukiano@redhat.com>
         8	Martin Polednik <mpolednik@redhat.com>
         5	Lukas Bednar <lbednar@redhat.com>
         3	Martin Kletzander <mkletzan@redhat.com>
         3	Stu Gott <sgott@redhat.com>
         1	jniederm <jniederm@users.noreply.github.com>
```

Test Results
------------

```
> Ran 39 of 41 Specs in 572.670 seconds
> SUCCESS! -- 39 Passed | 0 Failed | 0 Pending | 2 Skipped PASS
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
