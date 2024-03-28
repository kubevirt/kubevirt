KubeVirt v0.1.0
===============

This release follows v0.0.4 and consists of 115 changes, contributed by
11 people, leading to 121 files changed, 5278 insertions(+), 1916 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.1.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Many API improvements for a proper OpenAPI reference
- Add watchdog support
- Drastically improve the deployment on non-vagrant setups
  - Dropped nodeSelectors
  - Separated inner component deployment from edge component deployment
  - Created separate manifests for developer, test, and release deployments
- Moved komponents to kube-system namespace
- Improved and unified flag parsing

Contributors
------------

11 people contributed to this release:

```
        42	Roman Mohr <rmohr@redhat.com>
        20	David Vossel <dvossel@redhat.com>
        18	Lukas Bednar <lbednar@redhat.com>
        14	Martin Polednik <mpolednik@redhat.com>
         7	Fabian Deutsch <fabiand@redhat.com>
         6	Lukianov Artyom <alukiano@redhat.com>
         3	Vladik Romanovsky <vromanso@redhat.com>
         2	Petr Kotas <petr.kotas@gmail.com>
         1	Barak Korren <bkorren@redhat.com>
         1	Francois Deppierraz <francois@ctrlaltdel.ch>
         1	Saravanan KR <skramaja@redhat.com>
```

Test Results
------------

```
> Ran 44 of 46 Specs in 851.185 seconds
> SUCCESS! -- 44 Passed | 0 Failed | 0 Pending | 2 Skipped PASS
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
