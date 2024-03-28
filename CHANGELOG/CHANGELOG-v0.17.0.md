KubeVirt v0.17.0
================

This release follows v0.16.0 and consists of 178 changes, contributed by
16 people, leading to 439 files changed, 17189 insertions(+), 4807 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.17.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Several testcase additions
- Improved virt-controller node distribution
- Improved support between version migrations
- Support for a configurable MachineType default
- Support for live-migration of a VM on node taints
- Support for VM swap metrics
- Support for versioned virt-launcher / virt-handler communication
- Support for HyperV flags
- Support for different VM run strategies (i.e manual and rerunOnFailure)
- Several fixes for live-migration (TLS support, protected pods)

Contributors
------------

16 people contributed to this release:

```
        46	David Vossel <dvossel@redhat.com>
        35	Roman Mohr <rmohr@redhat.com>
        20	Vladik Romanovsky <vromanso@redhat.com>
        18	Marc Sluiter <msluiter@redhat.com>
        18	Stu Gott <sgott@redhat.com>
        17	Arik Hadas <ahadas@redhat.com>
         7	Francesco Romani <fromani@redhat.com>
         5	Artyom Lukianov <alukiano@redhat.com>
         2	Alexander Wels <awels@redhat.com>
         2	Ihar Hrachyshka <ihar@redhat.com>
         2	Karel Šimon <ksimon@redhat.com>
         2	Marcin Franczyk <mfranczy@redhat.com>
         1	Denis Ollier <dollierp@redhat.com>
         1	Irit goihman <igoihman@redhat.com>
         1	Mariusz Mazur <mmazur@redhat.com>
         1	Petr Kotas <pkotas@redhat.com>
```

Test Results
------------

```
> Ran 293 of 338 Specs in 8297.680 seconds
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
