KubeVirt v0.22.0
================

This release follows v0.21.0 and consists of 170 changes, contributed by
28 people, leading to 753 files changed, 74964 insertions(+), 19918
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Support for Nvidia GPUs and vGPUs exposed by Nvidia Kubevirt Device Plugin.
- VMIs now successfully start if they get a 0xfe prefixed MAC address assigned from the pod network
- Removed dependency on host semanage in SELinux Permissive mode
- Some changes as result of entering the CNCF sandbox (DCO check, FOSSA check, best practice badge)
- Many bug fixes and improvements in several areas
- CI: Introduced a OKD 4 test lane
- CI: Many improved tests, resulting in less flakyness

Contributors
------------

28 people contributed to this release:

```
        38	Roman Mohr <rmohr@redhat.com>
        22	kubevirt-bot <rmohr+kubebot@redhat.com>
        19	Federico Paolinelli <fpaoline@redhat.com>
        17	Vishesh Tanksale <vtanksale@nvidia.com>
        13	Artyom Lukianov <alukiano@redhat.com>
         8	Marcin Franczyk <mfranczy@redhat.com>
         7	Francesco Romani <fromani@redhat.com>
         7	Ihar Hrachyshka <ihrachys@redhat.com>
         6	Marc Sluiter <msluiter@redhat.com>
         4	Arik Hadas <ahadas@redhat.com>
         4	Ihar Hrachyshka <ihar@redhat.com>
         4	Petr Kotas <pkotas@redhat.com>
         3	lxs <lxs137@hotmail.com>
         2	Kedar Bidarkar <kbidarka@redhat.com>
         2	Vatsal Parekh <vatsalparekh@outlook.com>
         2	Vladik Romanovsky <vromanso@redhat.com>
         1	Alexander Wels <awels@redhat.com>
         1	Bertrand Roussel <broussel@sierrawireless.com>
         1	Daniel Hiller <daniel.hiller.1972@gmail.com>
         1	Denis Ollier <dollierp@redhat.com>
         1	Fabian Deutsch <fabiand@redhat.com>
         1	Pep Turró Mauri <pep@redhat.com>
         1	Vatsal Parekh <vparekh@redhat.com>
         1	alonSadan <asadan@redhat.com>
         1	jichenjc <jichenjc@cn.ibm.com>
         1	johncming <johncming@yahoo.com>
         1	ksimon1 <ksimon@redhat.com>
```

Test Results
------------

```
> Ran 390 of 458 Specs in 13837.058 seconds
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
