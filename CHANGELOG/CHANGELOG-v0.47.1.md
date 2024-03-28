KubeVirt v0.47.1
================

This release follows v0.46.1 and consists of 311 changes, contributed by 39 people, leading to 740 files changed, 22426 insertions(+), 17615 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.47.1.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.47.1`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6775][kubevirt-bot] Bugfix: revert #6565 which prevented upgrades to v0.47.
- [PR #6703][mhenriks] Fix BZ 2018521 - On upgrade VirtualMachineSnapshots going to Failed
- [PR #6511][knopt] Fixed virt-api significant memory usage when using Cluster Profiler with large KubeVirt deployments. (#6478, @knopt)
- [PR #6629][awels] BugFix: Hotplugging more than one block device would cause IO error (#6564)
- [PR #6657][andreabolognani] This version of KubeVirt includes upgraded virtualization technology based on libvirt 7.6.0 and QEMU 6.0.0.
- [PR #6565][Barakmor1] 'kubevirt-operator' changed to 'virt-operator' on 'managed-by' label in kubevirt's components made by virt-operator
- [PR #6642][ShellyKa13] Include hot-plugged disks in a Online VM Snapshot
- [PR #6513][brybacki] Adds force-bind flag to virtctl imageupload
- [PR #6588][erkanerol] Fix recording rules based on up metrics
- [PR #6575][davidvossel] VM controller now syncs VMI conditions to corresponding VM object
- [PR #6661][rmohr] Make the kubevirt api compatible with client-gen to make selecting compatible k8s golang dependencies easier
- [PR #6535][rmohr] Migrations use digests to reference containerDisks and kernel boot images to ensure disk consistency
- [PR #6651][ormergi] Kubevirt Conformance plugin now supports passing tests images registry.
- [PR #6589][iholder-redhat] custom kernel / initrd to boot from is now pre-pulled which improves stability
- [PR #6199][ormergi] Kubevirt Conformance plugin now supports passing image tag or digest
- [PR #6477][zcahana] Report DataVolumeError VM status when referenced a DataVolume indicates an error
- [PR #6593][rhrazdil] Removed python dependencies from virt-launcher and virt-handler containers
- [PR #6026][akrejcir] Implemented minimal VirtualMachineFlavor functionality.
- [PR #6570][erkanerol] Use honorLabels instead of labelDrop for namespace label on metrics
- [PR #6182][jordigilh] adds support for real time workloads
- [PR #6177][rmohr] Switch the node base images to centos8 stream
- [PR #6171][zcahana] Report ErrorPvcNotFound/ErrorDataVolumeNotFound VM status when PVC/DV-type volumes reference non-existent objects
- [PR #6437][VirrageS] Fix deprecated use of watch API to prevent reporting incorrect metrics.
- [PR #6482][jean-edouard] VMs with cloud-init data should now properly migrate from older KubeVirt versions
- [PR #6375][dhiller] Rely on kubevirtci installing cdi during testing

Contributors
------------
39 people contributed to this release:

```
34	Roman Mohr <rmohr@redhat.com>
15	Andrea Bolognani <abologna@redhat.com>
13	Jed Lejosne <jed@redhat.com>
12	Zvi Cahana <zvic@il.ibm.com>
12	alonsadan <asadan@redhat.com>
10	YitzyD <yitzy.i.dier@gmail.com>
9	Antonio Cardace <acardace@redhat.com>
9	David Vossel <dvossel@redhat.com>
9	Or Mergi <ormergi@redhat.com>
8	Daniel Hiller <dhiller@redhat.com>
8	Shelly Kagan <skagan@redhat.com>
7	Bartosz Rybacki <brybacki@redhat.com>
7	Or Shoval <oshoval@redhat.com>
6	Andrej Krejcir <akrejcir@redhat.com>
4	Alexander Wels <awels@redhat.com>
4	Itamar Holder <iholder@redhat.com>
4	Maya Rashish <mrashish@redhat.com>
4	Michael Henriksen <mhenriks@redhat.com>
4	Radim Hrazdil <rhrazdil@redhat.com>
3	Alice Frosi <afrosi@redhat.com>
3	Erkan Erol <eerol@redhat.com>
3	Howard Zhang <howard.zhang@arm.com>
3	Jordi Gil <jgil@redhat.com>
3	L. Pivarc <lpivarc@redhat.com>
2	Barak <bmordeha@redhat.com>
2	Federico Gimenez <fgimenez@redhat.com>
2	Igor Bezukh <ibezukh@redhat.com>
1	Adam Litke <alitke@redhat.com>
1	Alex Kalenyuk <akalenyu@redhat.com>
1	Israel Pinto <ipinto@redhat.com>
1	Janusz Marcinkiewicz <januszm@nvidia.com>
1	João Vilaça <jvilaca@redhat.com>
1	Petr Horáček <phoracek@redhat.com>
1	Tomasz Knopik <tknopik@nvidia.com>
1	Vasiliy Ulyanov <vulyanov@suse.de>
1	dalia-frank <dafrank@redhat.com>
```

Additional Resources
--------------------

- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]

[contributing]: https://github.com/kubevirt/kubevirt/blob/main/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/main/LICENSE
