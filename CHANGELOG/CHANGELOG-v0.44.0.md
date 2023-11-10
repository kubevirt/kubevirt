KubeVirt v0.44.0
================

This release follows v0.43.0 and consists of 389 changes, contributed by 41 people, leading to 508 files changed, 28369 insertions(+), 24278 deletions(-).
v0.44.0 is a promotion of release candidate v0.44.0-rc.0 which was originally published 2021-08-02
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.44.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.44.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6058][acardace] Fix virt-launcher exit pod race condition
- [PR #6035][davidvossel] Addition of perfscale-audit tool for auditing performance of control plane during stress tests
- [PR #6145][acardace] virt-launcher: disable unencrypted TCP socket for libvirtd.
- [PR #6163][davidvossel] Handle qemu processes in defunc (zombie) state
- [PR #6105][ashleyschuett] Add VirtualMachineInstancesPerNode to KubeVirt CR under Spec.Configuration
- [PR #6104][zcahana] Report FailedUnschedulable VM status when scheduling errors occur
- [PR #5905][davidvossel] VM CrashLoop detection and Exponential Backoff
- [PR #6070][acardace] Initiate Live-Migration using a unix socket (exposed by virt-handler) instead of an additional TCP<->Unix migration proxy started by virt-launcher
- [PR #5728][vasiliy-ul] Live migration of VMs with hotplug volumes is now enabled
- [PR #6109][rmohr] Fix virt-controller SCC: Reflect the need for NET_BIND_SERVICE in the virt-controller SCC.
- [PR #5942][ShellyKa13] Integrate guest agent to online VM snapshot
- [PR #6034][ashleyschuett] Go version updated to version 1.16.6
- [PR #6040][yuhaohaoyu] Improved debuggability by keeping the environment of a failed VMI alive.
- [PR #6068][dhiller] Add check that not all tests have been skipped
- [PR #6041][xpivarc] [Experimental] Virt-launcher can run as non-root user
- [PR #6062][iholder-redhat] replace dead "stress" binary with new, maintained, "stress-ng" binary
- [PR #6029][mhenriks] CDI to 1.36.0 with DataSource support
- [PR #4089][victortoso] Add support to USB Redirection with usbredir
- [PR #5946][vatsalparekh] Add guest-agent based ping probe
- [PR #6005][acardace] make containerDisk validation memory usage limit configurable
- [PR #5791][zcahana] Added a READY column to the tabular output of "kubectl get vm/vmi"
- [PR #6006][awels] DataVolumes created by DataVolumeTemplates will follow the associated VMs priority class.
- [PR #5982][davidvossel] Reduce vmi Update collisions (http code 409) during startup
- [PR #5891][akalenyu] BugFix: Pending VMIs when creating concurrent bulk of VMs backed by WFFC DVs
- [PR #5925][rhrazdil] Fix issue with Windows VMs not being assigned IP address configured in network-attachment-definition IPAM.
- [PR #6007][rmohr] Fix: The bandwidth limitation on migrations is no longer ignored. Caution: The default bandwidth limitation of 64Mi is changed to "unlimited" to not break existing installations.
- [PR #4944][kwiesmueller] Add `/portforward` subresource to `VirtualMachine` and `VirtualMachineInstance` that can tunnel TCP traffic through the API Server using a websocket stream.
- [PR #5402][alicefr] Integration of libguestfs-tools and added new command `guestfs` to virtctl
- [PR #5953][ashleyschuett] Allow Failed VMs to be stopped when using `--force --gracePeriod 0`
- [PR #5876][mlsorensen] KubeVirt CR supports specifying a runtime class for virt-launcher pods via 'launcherRuntimeClass'.

Contributors
------------
41 people contributed to this release:

```
27	David Vossel <dvossel@redhat.com>
24	Zvi Cahana <zvic@il.ibm.com>
22	L. Pivarc <lpivarc@redhat.com>
16	Quique Llorente <ellorent@redhat.com>
16	Shelly Kagan <skagan@redhat.com>
16	Vasiliy Ulyanov <vulyanov@suse.de>
14	Roman Mohr <rmohr@redhat.com>
11	Antonio Cardace <acardace@redhat.com>
10	Alice Frosi <afrosi@redhat.com>
10	Alona Kaplan <alkaplan@redhat.com>
9	Michael Henriksen <mhenriks@redhat.com>
8	Marcelo Amaral <marcelo.amaral1@ibm.com>
7	Ashley Schuett <aschuett@redhat.com>
6	Ben Ukhanov <ben1zuk321@gmail.com>
6	Igor Bezukh <ibezukh@redhat.com>
6	Itamar Holder <iholder@redhat.com>
6	Victor Toso <victortoso@redhat.com>
5	Radim Hrazdil <rhrazdil@redhat.com>
4	Alexander Wels <awels@redhat.com>
4	Daniel Hiller <dhiller@redhat.com>
4	Miguel Duarte Barroso <mdbarroso@redhat.com>
4	Or Shoval <oshoval@redhat.com>
3	Federico Gimenez <fgimenez@redhat.com>
3	Marcus Sorensen <mls@apple.com>
3	Vatsal Parekh <vparekh@redhat.com>
3	alonsadan <asadan@redhat.com>
2	Kevin Wiesmueller <kwiesmul@redhat.com>
2	Marcus Sorensen <marcus_sorensen@apple.com>
1	Alex Kalenyuk <akalenyu@redhat.com>
1	Andrea Bolognani <abologna@redhat.com>
1	Chris Callegari <mazzystr@gmail.com>
1	Hao Yu <yuh@us.ibm.com>
1	Howard Zhang <howard.zhang@arm.com>
1	Jed Lejosne <jed@redhat.com>
1	LiHui <andrewli@yunify.com>
1	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
1	Simone Tiraboschi <stirabos@redhat.com>
1	Stu Gott <sgott@redhat.com>
1	borod108 <boris.od@gmail.com>
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
