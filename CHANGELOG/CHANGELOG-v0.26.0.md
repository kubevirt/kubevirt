KubeVirt v0.26.0
================

This release follows v0.25.0 and consists of 116 changes, contributed by
19 people, leading to 1556 files changed, 156060 insertions(+), 56779
deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.26.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- Fix incorrect ownerReferences to avoid VMs getting GCed
- Fixes for several tests
- Fix greedy permissions around Secrets by delegating them to kubelet
- Fix OOM infra pod by increasing it's memory request
- Clarify device support around live migrations
- Support for an uninstall strategy to protect workloads during uninstallation
- Support for more prometheus metrics and alert rules
- Support for testing SRIOV connectivity in functional tests
- Update Kubernetes client-go to 1.16.4
- FOSSA fixes and status

Contributors
------------

19 people contributed to this release:

```
        25	Roman Mohr <rmohr@redhat.com>
        14	Vatsal Parekh <vparekh@redhat.com>
         9	Daniel Belenky <dbelenky@redhat.com>
         7	Omer Yahud <oyahud@oyahud.tlv.csb>
         6	Ihar Hrachyshka <ihrachys@redhat.com>
         4	Daniel Hiller <daniel.hiller.1972@gmail.com>
         3	Or Shoval <oshoval@redhat.com>
         3	Stu Gott <sgott@redhat.com>
         2	Federico Paolinelli <fpaoline@redhat.com>
         2	Ihar Hrachyshka <ihar@redhat.com>
         2	Michael Henriksen <mhenriks@redhat.com>
         2	Petr Kotas <pkotas@redhat.com>
         2	fossabot <badges@fossa.io>
         1	Alberto Losada <alosadag@redhat.com>
         1	Dan Kenigsberg <danken@redhat.com>
         1	Igor Bezukh <ibezukh@redhat.com>
         1	Marc Sluiter <msluiter@redhat.com>
         1	Peter White <peter.white@metaswitch.com>
```

Test Results
------------

```
> Ran 417 of 498 Specs in 12827.215 seconds
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
