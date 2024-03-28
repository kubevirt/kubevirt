KubeVirt v0.42.0
================

This release follows v0.41.0 and consists of 326 changes, contributed by 36 people, leading to 699 files changed, 51909 insertions(+), 23263 deletions(-).
v0.42.0 is a promotion of release candidate v0.42.0-rc.0 which was originally published 2021-06-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.42.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.42.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #5738][rmohr] Stop releasing jinja2 templates of our operator. Kustomize is the preferred way for customizations.
- [PR #5691][ashleyschuett] Allow multiple shutdown events to ensure the event is received by ACPI
- [PR #5558][ormergi] Drop virt-launcher SYS_RESOURCE capability
- [PR #5694][davidvossel] Fixes null pointer dereference in migration controller
- [PR #5416][iholder-redhat] [feature] support booting VMs from a custom kernel/initrd images with custom kernel arguments
- [PR #5495][iholder-redhat] Go version updated to version 1.16.1.
- [PR #5502][rmohr] Add downwardMetrics volume to expose a limited set of hots metrics to guests
- [PR #5601][maya-r] Update libvirt-go to 7.3.0
- [PR #5661][davidvossel] Validation/Mutation webhooks now explicitly define a 10 second timeout period
- [PR #5652][rmohr] Automatically discover kube-prometheus installations and configure kubevirt monitoring
- [PR #5631][davidvossel] Expand backport policy to include logging and debug fixes
- [PR #5528][zcahana] Introduced a "status.printableStatus" field in the VirtualMachine CRD. This field is now displayed in the tabular output of "kubectl get vm".
- [PR #5200][rhrazdil] Add support for Istio proxy traffic routing with masquerade interface. nftables is required for this feature.
- [PR #5560][oshoval] virt-launcher now populates domain's guestOS info and interfaces status according guest agent also when doing periodic resyncs.
- [PR #5514][rhrazdil] Fix live-migration failing when VM with masquarade iface has explicitly specified any of these ports: 22222, 49152, 49153
- [PR #5583][dhiller] Reenable coverage
- [PR #5129][davidvossel] Gracefully shutdown virt-api connections and ensure zero exit code under normal shutdown conditions
- [PR #5582][dhiller] Fix flaky unit tests
- [PR #5600][davidvossel] Improved logging around VM/VMI shutdown and restart
- [PR #5564][omeryahud] virtctl rename support is dropped
- [PR #5585][iholder-redhat] [bugfix] - reject VM defined with volume with no matching disk
- [PR #5595][zcahana] Fixes adoption of orphan DataVolumes
- [PR #5566][davidvossel] Release branches are now cut on the first _business day_ of the month rather than the first day.
- [PR #5108][Omar007] Fixes handling of /proc/<pid>/mountpoint by working on the device information instead of mount information
- [PR #5250][mlsorensen] Controller health checks will no longer actively test connectivity to the Kubernetes API. They will rely in health of their watches to determine if they have API connectivity.
- [PR #5563][ashleyschuett] Set KubeVirt resources flags in the KubeVirt CR
- [PR #5328][andreabolognani] This version of KubeVirt includes upgraded virtualization technology based on libvirt 7.0.0 and QEMU 5.2.0.

Contributors
------------
36 people contributed to this release:

```
38	Roman Mohr <rmohr@redhat.com>
32	Miguel Duarte Barroso <mdbarroso@redhat.com>
24	Itamar Holder <iholder@redhat.com>
15	Ashley Schuett <aschuett@redhat.com>
14	David Vossel <dvossel@redhat.com>
14	Zvi Cahana <zvic@il.ibm.com>
12	Maya Rashish <mrashish@redhat.com>
10	Andrea Bolognani <abologna@redhat.com>
10	Daniel Hiller <dhiller@redhat.com>
9	Radim Hrazdil <rhrazdil@redhat.com>
7	Zhou Hao <zhouhao@fujitsu.com>
5	Omar Pakker <Omar007@users.noreply.github.com>
5	Or Mergi <ormergi@redhat.com>
3	Alexander Wels <awels@redhat.com>
3	Bartosz Rybacki <brybacki@redhat.com>
3	Federico Gimenez <fgimenez@redhat.com>
3	Igor Bezukh <ibezukh@redhat.com>
2	Kevin Wiesmueller <kwiesmul@redhat.com>
2	Marcus Sorensen <mls@apple.com>
2	Omer Yahud <oyahud@redhat.com>
2	Or Shoval <oshoval@redhat.com>
1	Antonio Cardace <acardace@redhat.com>
1	Jed Lejosne <jed@redhat.com>
1	Karel Šimon <ksimon@redhat.com>
1	Krzysztof Majcher <kmajcher@redhat.com>
1	Mark DeNeve <markd@xphyr.net>
1	Petr Horáček <phoracek@redhat.com>
1	Shelly Kagan <skagan@redhat.com>
1	Stu Gott <sgott@redhat.com>
1	Vatsal Parekh <vparekh@redhat.com>
1	Vladik Romanovsky <vromanso@redhat.com>
1	Zou Yu <zouy.fnst@cn.fujitsu.com>
1	dalia-frank <dafrank@redhat.com>
1	ipinto <ipinto@redhat.com>
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
