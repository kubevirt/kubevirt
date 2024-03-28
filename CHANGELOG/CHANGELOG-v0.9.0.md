KubeVirt v0.9.0
===============

This release follows v0.8.0 and consists of 211 changes, contributed by
20 people, leading to 1955 files changed, 112474 insertions(+), 32444
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.9.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- CI: NetworkPolicy tests
- CI: Support for an external provider (use a preconfigured cluster for tests)
- Fix virtctl console issues with CRI-O
- Support to initialize empty PVs
- Support for basic CPU pinning
- Support for setting IO Threads
- Support for block volumes
- Move preset logic to mutating webhook
- Introduce basic metrics reporting using prometheus metrics
- Many stabilizing fixes in many places

Contributors
------------

20 people contributed to this release:

```
        48	Roman Mohr <rmohr@redhat.com>
        36	Stu Gott <sgott@redhat.com>
        32	Vladik Romanovsky <vromanso@redhat.com>
        22	Marc Sluiter <msluiter@redhat.com>
        15	Artyom Lukianov <alukiano@redhat.com>
        15	Marcin Franczyk <mfranczy@redhat.com>
        10	David Vossel <dvossel@redhat.com>
         8	Petr Kotas <pkotas@redhat.com>
         5	Guohua Ouyang <gouyang@redhat.com>
         4	Yan Du <yadu@redhat.com>
         3	Ben Warren <bawarren@cisco.com>
         3	Ihar Hrachyshka <ihar@redhat.com>
         2	Gabriel Szasz <gszasz@redhat.com>
         2	j-griffith <john.griffith8@gmail.com>
         1	Dan Kenigsberg <danken@redhat.com>
         1	Dylan Redding <dylan.redding@stackpath.com>
         1	Fred Rolland <frolland@redhat.com>
         1	Itamar Heim <iheim@localhost.localdomain>
         1	Lukas Bednar <lbednar@redhat.com>
         1	Piotr Kliczewski <piotr.kliczewski@gmail.com>
```

Test Results
------------

```
> Ran 166 of 193 Specs in 4192.048 seconds
> PASS
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
