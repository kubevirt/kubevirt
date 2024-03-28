KubeVirt v0.41.0
================

This release follows v0.40.0 and consists of 398 changes, contributed by 46 people, leading to 398 files changed, 20967 insertions(+), 6926 deletions(-).
v0.41.0 is a promotion of release candidate v0.41.0-rc.1 which was originally published 2021-05-11
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.41.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.41.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #5586][kubevirt-bot] This version of KubeVirt includes upgraded virtualization technology based on libvirt 7.0.0 and QEMU 5.2.0.
- [PR #5344][ashleyschuett] Reconcile PrometheusRules and ServiceMonitor resources
- [PR #5542][andreyod] Add startStrategy field to VMI spec to allow Virtual Machine start in paused state.
- [PR #5459][ashleyschuett] Reconcile service resource
- [PR #5520][ashleyschuett] Reconcile required labels and annotations on ConfigMap resources
- [PR #5533][rmohr] Fix `docker save` and `docker push` issues with released kubevirt images
- [PR #5428][oshoval] virt-launcher now populates domain's guestOS info and interfaces status according guest agent also when doing periodic resyncs.
- [PR #5410][ashleyschuett] Reconcile ServiceAccount resources
- [PR #5109][Omar007] Add support for specifying a logical and physical block size for disk devices
- [PR #5471][ashleyschuett] Reconcile APIService resources
- [PR #5513][ashleyschuett] Reconcile Secret resources
- [PR #5496][davidvossel] Improvements to migration proxy logging
- [PR #5376][ashleyschuett] Reconcile CustomResourceDefinition resources
- [PR #5435][AlonaKaplan] Support dual stack service on "virtctl expose"-
- [PR #5425][davidvossel] Fixes VM restart during eviction when EvictionStrategy=LiveMigrate
- [PR #5423][ashleyschuett] Add resource requests to virt-controller, virt-api, virt-operator and virt-handler
- [PR #5343][erkanerol] Some cleanups and small additions to the storage metrics
- [PR #4682][stu-gott] Updated Guest Agent Version compatibility check. The new approach is much more accurate.
- [PR #5485][rmohr] Fix fallback to iptables if nftables is not used on the host on arm64
- [PR #5426][rmohr] Fix fallback to iptables if nftables is not used on the host
- [PR #5403][tiraboschi] Added a kubevirt_ prefix to several recording rules and metrics
- [PR #5241][stu-gott] Introduced Duration and RenewBefore parameters for cert rotation. Previous values are now deprecated.
- [PR #5463][acardace] Fixes upgrades from KubeVirt v0.36
- [PR #5456][zhlhahaha] Enable arm64 cross-compilation
- [PR #3310][davidvossel] Doc outlines our Kubernetes version compatibility commitment
- [PR #3383][EdDev] Add `vmIPv6NetworkCIDR` under `NetworkSource.pod` to support custom IPv6 CIDR for the vm network when using masquerade binding.
- [PR #3415][zhlhahaha] Make kubevirt code fit for arm64 support. No testing is at this stage performed against arm64 at this point.
- [PR #5147][xpivarc] Remove CAP_NET_ADMIN from the virt-launcher pod(second take).
- [PR #5351][awels] Support hotplug with virtctl using addvolume and removevolume commands
- [PR #5050][ashleyschuett] Fire Prometheus Alert when a vmi is orphaned for more than an hour

Contributors
------------
46 people contributed to this release:

```
25	David Vossel <dvossel@redhat.com>
21	Stu Gott <sgott@redhat.com>
20	Ashley Schuett <aschuett@redhat.com>
18	Miguel Duarte Barroso <mdbarroso@redhat.com>
13	Itamar Holder <iholder@redhat.com>
11	Alexander Wels <awels@redhat.com>
11	Or Mergi <ormergi@redhat.com>
10	Vladik Romanovsky <vromanso@redhat.com>
9	Alona Kaplan <alkaplan@redhat.com>
8	Federico Gimenez <fgimenez@redhat.com>
8	Howard Zhang <howard.zhang@arm.com>
8	L. Pivarc <lpivarc@redhat.com>
8	Quique Llorente <ellorent@redhat.com>
8	Roman Mohr <rmohr@redhat.com>
8	Shelly Kagan <skagan@redhat.com>
7	Andrey Odarenko <andreyo@il.ibm.com>
7	Ezra Silvera <ezra@il.ibm.com>
7	Or Shoval <oshoval@redhat.com>
6	Antonio Cardace <acardace@redhat.com>
6	Edward Haas <edwardh@redhat.com>
6	Karel Šimon <ksimon@redhat.com>
5	Erkan Erol <eerol@redhat.com>
4	Andrea Bolognani <abologna@redhat.com>
3	Yuval Turgeman <yturgema@redhat.com>
2	Alice Frosi <afrosi@redhat.com>
2	Bartosz Rybacki <brybacki@redhat.com>
2	Dan Kenigsberg <danken@redhat.com>
2	Federico Gimenez <fgimenez@users.noreply.github.com>
2	Omar Pakker <Omar007@users.noreply.github.com>
2	Vasiliy Ulyanov <vulyanov@suse.de>
2	Vatsal Parekh <vparekh@redhat.com>
1	Alex <alexsimonjones@gmail.com>
1	Andrej Krejcir <akrejcir@redhat.com>
1	Daniel Hiller <dhiller@redhat.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Maya Rashish <mrashish@redhat.com>
1	Michael Henriksen <mhenriks@redhat.com>
1	Radim Hrazdil <rhrazdil@redhat.com>
1	Ram Lavi <ralavi@redhat.com>
1	Shirly Radco <sradco@redhat.com>
1	Tomas Psota <tpsota@redhat.com>
1	cchen <actor168@gmail.com>
1	jichenjc <jichenjc@cn.ibm.com>
1	root <root@viosd2.watson.ibm.com>
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
