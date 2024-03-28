KubeVirt v0.40.0
================

This release follows v0.39.0 and consists of 450 changes, contributed by 51 people, leading to 646 files changed, 42768 insertions(+), 6668 deletions(-).
v0.40.0 is a promotion of release candidate v0.40.0-rc.2 which was originally published 2021-04-16
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.40.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.40.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #5467][rmohr] Fixes upgrades from KubeVirt v0.36
- [PR #5350][jean-edouard] Removal of entire `permittedHostDevices` section will now remove all user-defined host device plugins.
- [PR #5242][jean-edouard] Creating more than 1 migration at the same time for a given VMI will now fail
- [PR #4907][vasiliy-ul] Initial cgroupv2 support
- [PR #5324][jean-edouard] Default feature gates can now be defined in the provider configuration.
- [PR #5006][alicefr] Add discard=unmap option
- [PR #5022][davidvossel] Fixes race condition between operator adding service and webhooks that can result in installs/uninstalls failing
- [PR #5310][ashleyschuett] Reconcile CRD resources
- [PR #5102][iholder-redhat] Go version updated to 1.14.14
- [PR #4746][ashleyschuett] Reconcile Deployments, DaemonSets, MutatingWebhookConfigurations and ValidatingWebhookConfigurations
- [PR #5037][ormergi] Hot-plug SR-IOV VF interfaces to VM's post a successful migration.
- [PR #5269][mlsorensen] Prometheus metrics scraped from virt-handler are now served from the VMI informer cache, rather than calling back to the Kubernetes API for VMI information.
- [PR #5138][davidvossel] virt-handler now waits up to 5 minutes for all migrations on the node to complete before shutting down.
- [PR #5191][yuvalturg] Added a metric for monitoring CPU affinity
- [PR #5215][xphyr] Enable detection of Intel GVT-g vGPU.
- [PR #4760][rmohr] Make virt-handler heartbeat more efficient and robust: Only one combined PATCH and no need to detect different cluster types anymore.
- [PR #5091][iholder-redhat] QEMU SeaBios debug logs are being seen as part of virt-launcher log.
- [PR #5221][rmohr] Remove  workload placement validation webhook which blocks placement updates when VMIs are running
- [PR #5128][yuvalturg] Modified memory related metrics by adding several new metrics and splitting the swap traffic bytes metric
- [PR #5084][ashleyschuett] Add validation to CustomizeComponents object on the KubeVirt resource
- [PR #5182][davidvossel] New [release-blocker] functional test marker to signify tests that can never be disabled before making a release
- [PR #5137][davidvossel] Added our policy around release branch backporting in docs/release-branch-backporting.md
- [PR #5096][yuvalturg] Modified networking metrics by adding new metrics, splitting existing ones by rx/tx and using the device alias for the interface name when available
- [PR #5088][awels] Hotplug works with hostpath storage.
- [PR #4908][dhiller] Move travis tag and master builds to kubevirt prow.
- [PR #4741][EdDev] Allow live migration for SR-IOV VM/s without preserving the VF interfaces.

Contributors
------------
51 people contributed to this release:

```
51	Edward Haas <edwardh@redhat.com>
48	Roman Mohr <rmohr@redhat.com>
23	David Vossel <dvossel@redhat.com>
21	Vasiliy Ulyanov <vulyanov@suse.de>
12	Ashley Schuett <aschuett@redhat.com>
12	Bartosz Rybacki <brybacki@redhat.com>
12	Itamar Holder <iholder@redhat.com>
12	Or Mergi <ormergi@redhat.com>
11	Federico Gimenez <fgimenez@redhat.com>
11	Or Shoval <oshoval@redhat.com>
10	Dan Kenigsberg <danken@redhat.com>
8	Antonio Cardace <acardace@redhat.com>
7	L. Pivarc <lpivarc@redhat.com>
6	Karel Šimon <ksimon@redhat.com>
5	Alexander Wels <awels@redhat.com>
5	Daniel Hiller <dhiller@redhat.com>
5	Jed Lejosne <jed@redhat.com>
4	Andrey Odarenko <andreyo@il.ibm.com>
4	Hao Yu <yuh@us.ibm.com>
4	Maya Rashish <mrashish@redhat.com>
3	Victor Toso <victortoso@redhat.com>
3	Yuval Turgeman <yturgema@redhat.com>
3	alonsadan <asadan@redhat.com>
2	Alice Frosi <afrosi@redhat.com>
2	Andrej Krejcir <akrejcir@redhat.com>
2	Erkan Erol <eerol@redhat.com>
2	Mark DeNeve <markd@xphyr.net>
2	Quique Llorente <ellorent@redhat.com>
2	Vladik Romanovsky <vromanso@redhat.com>
2	ansijain <ansi.jain@india.nec.com>
2	jichenjc <jichenjc@cn.ibm.com>
1	Alex Kalenyuk <akalenyu@redhat.com>
1	Arnon Gilboa <agilboa@redhat.com>
1	Ashley Schuett <ashleyns1992@gmail.com>
1	Cole Robinson <crobinso@redhat.com>
1	Federico Gimenez <fgimenez@users.noreply.github.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Kavya <kavya.g@ibm.com>
1	Marcus Sorensen <mls@apple.com>
1	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
1	Shelly Kagan <skagan@redhat.com>
1	Shweta Padubidri <spadubid@redhat.com>
1	Stu Gott <sgott@redhat.com>
1	Tomas Psota <to.psota@gmail.com>
1	Tomas Psota <tpsota@redhat.com>
1	Vatsal Parekh <vparekh@redhat.com>
1	Yan Du <yadu@redhat.com>
1	alonsadan <alonsadan1@gmail.com>
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
