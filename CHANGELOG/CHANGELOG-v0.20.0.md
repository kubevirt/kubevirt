KubeVirt v0.20.0
================

This release follows v0.19.0 and consists of 290 changes, contributed by
26 people, leading to 514 files changed, 24045 insertions(+), 6666 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.20.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Containerdisks are now secure and they are not copied anymore on every start.
Old containerdisks can still be used in the same secure way, but new
containerdisks can't be used on older kubevirt releases
- Create specific SecurityContextConstraints on OKD instead of using the
privileged SCC
- Added clone authorization check for DataVolumes with PVC source
- The sidecar feature is feature-gated now
- Use container image shasums instead of tags for KubeVirt deployments
- Protect control plane components against voluntary evictions with a
PodDisruptionBudget of MinAvailable=1
- Replaced hardcoded `virtctl` by using the basename of the call, this enables
nicer output when installed via krew plugin package manager
- Added RNG device to all Fedora VMs in tests and examples (newer kernels might
block bootimg while waiting for entropy)
- The virtual memory is now set to match the memory limit, if memory limit is
specified and guest memory is not
- Support nftable for CoreOS
- Added a block-volume flag to the virtctl image-upload command
- Improved virtctl console/vnc data flow
- Removed DataVolumes feature gate in favor of auto-detecting CDI support
- Removed SR-IOV feature gate, it is enabled by default now
- VMI-related metrics have been renamed from `kubevirt_vm_` to `kubevirt_vmi_`
to better reflect their purpose
- Added metric to report the VMI count
- Improved integration with HCO by adding a CSV generator tool and modified
KubeVirt CR conditions
- CI Improvements:
  - Added dedicated SR-IOV test lane
  - Improved log gathering
  - Reduced amount of flaky tests

Contributors
------------

26 people contributed to this release:

```
70      Roman Mohr <rmohr@redhat.com>
52      Marc Sluiter <msluiter@redhat.com>
37      Daniel Hiller <daniel.hiller.1972@googlemail.com>
21      Arik Hadas <ahadas@redhat.com>
19      David Vossel <dvossel@redhat.com>
17      Federico Paolinelli <fpaoline@redhat.com>
12      Francesco Romani <fromani@redhat.com>
11      Marcin Franczyk <mfranczy@redhat.com>
8      Artyom Lukianov <alukiano@redhat.com>
7      Gage Orsburn <gageorsburn@live.com>
5      Ihar Hrachyshka <ihrachys@redhat.com>
4      Michael Henriksen <mhenriks@redhat.com>
4      Petr Kotas <pkotas@redhat.com>
3      Ihar Hrachyshka <ihar@redhat.com>
3      Sebastian Scheinkman <sscheink@redhat.com>
3      Vatsal Parekh <vatsalparekh@outlook.com>
2      Fabian Deutsch <fabiand@redhat.com>
2      Kunal Kushwaha <kushwaha_kunal_v7@lab.ntt.co.jp>
2      Xenia Lisovskaia <polnoch@protonmail.com>
2      kubevirt-bot <rmohr+kubebot@redhat.com>
1      Alexander Wels <awels@redhat.com>
1      Denys Shchedrivyi <dshchedr@redhat.com>
1      Niels de Vos <ndevos@redhat.com>
1      Petr Horacek <phoracek@redhat.com>
1      Vatsal Parekh <vparekh@redhat.com>
1      Yossi Segev <ysegev@redhat.com>
```

Test Results
------------

```
> Ran 363 of 415 Specs in 11596.175 seconds
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
