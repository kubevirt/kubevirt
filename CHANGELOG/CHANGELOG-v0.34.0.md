KubeVirt v0.34.0
================

This release follows v0.33.0 and consists of 366 changes, contributed by 35 people, leading to 1042 files changed, 110966 insertions(+), 117125 deletions(-).
v0.34.0 is a promotion of release candidate v0.34.0-rc.2 which was originally published 2020-10-07
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.34.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.34.0`.

Pre-built containers are published on Docker Hub and can be viewed at: <https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- [PR #4315][kubevirt-bot] PVCs populated by DVs are now allowed as volumes.
- [PR #3837][jean-edouard] VM interfaces with no `bootOrder` will no longer be candidates for boot when using the BIOS bootloader, as documented
- [PR #3879][ashleyschuett] KubeVirt should now be configured through the KubeVirt CR `configuration` key. The usage of the kubevirt-confg ConfigMap will be deprecated in the future.
- [PR #4074][stu-gott] Fixed bug preventing non-admin users from pausing/unpausing VMs
- [PR #4252][rhrazdil] Fixes https://bugzilla.redhat.com/show_bug.cgi?id=1853911
- [PR #4016][ashleyschuett] Allow for post copy VMI migrations
- [PR #4235][davidvossel] Fixes timeout failure that occurs when pulling large containerDisk images
- [PR #4263][rmohr] Add readiness and liveness probes to virt-handler, to clearly indicate readiness
- [PR #4248][maiqueb] always compile KubeVirt with selinux support on pure go builds.
- [PR #4012][danielBelenky] Added support for the eviction API for VMIs with eviction strategy. This enables VMIs to be live-migrated when the node is drained or when the descheduler wants to move a VMI to a different node.
- [PR #4075][ArthurSens] Metric kubevirt_vmi_vcpu_seconds' state label is now exposed as a human-readable state instead of an integer
- [PR #4162][vladikr] introduce a cpuAllocationRatio config parameter to normalize the number of CPUs requested for a pod, based on the number of vCPUs
- [PR #4177][maiqueb] Use vishvananda/netlink instead of songgao/water to create tap devices.
- [PR #4092][stu-gott] Allow specifying nodeSelectors, affinity and tolerations to control where KubeVirt components will run
- [PR #3927][ArthurSens] Adds new metric kubevirt_vmi_memory_unused_bytes
- [PR #3493][vladikr] virtIO-FS is being added as experimental, protected by a feature-gate that needs to be enabled in the kubevirt config by the administrator
- [PR #4193][mhenriks] Add snapshot.kubevirt.io to admin/edit/view roles
- [PR #4149][qinqon] Bump kubevirtci to k8s-1.19
- [PR #3471][crobinso] Allow hiding that the VM is running on KVM, so that Nvidia graphics cards can be passed through
- [PR #4115][phoracek] Add conformance automation and manifest publishing
- [PR #3733][davidvossel] each PRs description.
- [PR #4082][mhenriks] VirtualMachineRestore API and implementation
- [PR #4154][davidvossel] Fixes issue with Serivce endpoints not being updated properly in place during KubeVirt updates.
- [PR #3289][vatsalparekh] Add option to run only VNC Proxy in virtctl
- [PR #4027][alicefr] Added memfd as default memory backend for hugepages. This introduces the new annotation kubevirt.io/memfd to disable memfd as default and fallback to the previous behavior.
- [PR #3612][ashleyschuett] Adds `customizeComponents` to the kubevirt api
- [PR #4029][cchengleo] Fix an issue which prevented virt-operator from installing monitoring resources in custom namespaces.
- [PR #4031][rmohr] Initial support for sonobuoy for conformance testing

Contributors
------------
35 people contributed to this release:

```
38	Ashley Schuett <ashleyns1992@gmail.com>
33	Roman Mohr <rmohr@redhat.com>
26	Michael Henriksen <mhenriks@redhat.com>
25	Miguel Duarte Barroso <mdbarroso@redhat.com>
21	David Vossel <dvossel@redhat.com>
19	rmohr <rmohr@redhat.com>
17	Stu Gott <sgott@redhat.com>
15	Vladik Romanovsky <vromanso@redhat.com>
11	Jed Lejosne <jed@redhat.com>
10	Or Shoval <oshoval@redhat.com>
8	Daniel Belenky <dbelenky@redhat.com>
8	Quique Llorente <ellorent@redhat.com>
7	Alice Frosi <afrosi@redhat.com>
6	Bartosz Rybacki <brybacki@redhat.com>
5	Cheng Cheng <cheng@ccheng.us>
5	Edward Haas <edwardh@redhat.com>
5	Vatsal Parekh <vparekh@redhat.com>
4	Petr Horacek <phoracek@redhat.com>
4	arthursens <arthursens2005@gmail.com>
3	Cole Robinson <crobinso@redhat.com>
2	Alexander Wels <awels@redhat.com>
2	Cheng Cheng <ccheng@ccheng.us>
2	Cheng Cheng <chengcheng@apple.com>
2	Victor Toso <victortoso@redhat.com>
1	Alex Kalenyuk <akalenyu@redhat.com>
1	Andrea Bolognani <abologna@redhat.com>
1	Daniel Hiller <daniel.hiller.1972@gmail.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Kedar Bidarkar <kbidarka@redhat.com>
1	L. Pivarc <lpivarc@redhat.com>
1	Maya Rashish <mrashish@redhat.com>
1	Radim Hrazdil <rhrazdil@redhat.com>
1	Ram Lavi <ralavi@redhat.com>
```

Additional Resources
--------------------

- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]

[contributing]: https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/master/LICENSE
