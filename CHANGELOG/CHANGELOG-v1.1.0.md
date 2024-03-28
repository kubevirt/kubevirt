# KubeVirt v1.1.0

This release follows v1.0.1 and consists of 1071 changes, leading to 1108 files changed, 82781 insertions(+), 33012 deletions(-).
v1.1.0 is a promotion of release candidate v1.1.0-rc.1, which was originally published on 2023-11-03.

The primary release artifact of KubeVirt is the git tree. The release tag is signed and can be verified using `git tag -v v1.1.0`.

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v1.1.0  
Pre-built containers are published on Quay and can be viewed at: https://quay.io/kubevirt/

## Notable changes

- [#10669](https://github.com/kubevirt/kubevirt/pull/10669) ([@kubevirt-bot](https://github.com/kubevirt-bot)) Introduce network binding plugin for Passt networking, interfacing with Kubevirt new network binding plugin API.
- [#10646](https://github.com/kubevirt/kubevirt/pull/10646) ([@jean-edouard](https://github.com/jean-edouard)) The dedicated migration network should now always be properly detected by virt-handler
- [#10602](https://github.com/kubevirt/kubevirt/pull/10602) ([@kubevirt-bot](https://github.com/kubevirt-bot)) Fix LowKVMNodesCount not firing
- [#10566](https://github.com/kubevirt/kubevirt/pull/10566) ([@fossedihelm](https://github.com/fossedihelm)) Add 100Mi of memory overhead for vmi with dedicatedCPU or that wants GuaranteedQos
- [#10568](https://github.com/kubevirt/kubevirt/pull/10568) ([@ormergi](https://github.com/ormergi)) Network binding plugin API support CNIs, new integration point on virt-launcher pod creation.
- [#10496](https://github.com/kubevirt/kubevirt/pull/10496) ([@fossedihelm](https://github.com/fossedihelm)) Automatically set cpu limits when a resource quota with cpu limits is associated to the creation namespace and the `AutoResourceLimits` FeatureGate is enabled
- [#10309](https://github.com/kubevirt/kubevirt/pull/10309) ([@lyarwood](https://github.com/lyarwood)) cluster-wide [`common-instancetypes`](https://github.com/kubevirt/common-instancetypes) resources can now deployed by `virt-operator` using the `CommonInstancetypesDeploymentGate` feature gate.
- [#10543](https://github.com/kubevirt/kubevirt/pull/10543) ([@0xFelix](https://github.com/0xFelix)) Clear VM guest memory when ignoring inference failures
- [#9590](https://github.com/kubevirt/kubevirt/pull/9590) ([@xuzhenglun](https://github.com/xuzhenglun)) fix embed version info of virt-operator
- [#10532](https://github.com/kubevirt/kubevirt/pull/10532) ([@alromeros](https://github.com/alromeros)) Add --volume-mode flag in image-upload
- [#10515](https://github.com/kubevirt/kubevirt/pull/10515) ([@iholder101](https://github.com/iholder101)) Bug-fix: Stop copying VMI spec to VM during snapshots
- [#10320](https://github.com/kubevirt/kubevirt/pull/10320) ([@victortoso](https://github.com/victortoso)) sidecar-shim implements PreCloudInitIso hook
- [#10463](https://github.com/kubevirt/kubevirt/pull/10463) ([@0xFelix](https://github.com/0xFelix)) VirtualMachines: Introduce InferFromVolumeFailurePolicy in Instancetype- and PreferenceMatchers
- [#10393](https://github.com/kubevirt/kubevirt/pull/10393) ([@iholder101](https://github.com/iholder101)) [Bugfix] [Clone API] Double-cloning is now working as expected.
- [#10486](https://github.com/kubevirt/kubevirt/pull/10486) ([@assafad](https://github.com/assafad)) Deprecation notice for the metrics listed in the PR. Please update your systems to use the new metrics names.
- [#10438](https://github.com/kubevirt/kubevirt/pull/10438) ([@lyarwood](https://github.com/lyarwood)) A new `instancetype.kubevirt.io:view` `ClusterRole` has been introduced that can be bound to users via a `ClusterRoleBinding` to provide read only access to the cluster scoped `VirtualMachineCluster{Instancetype,Preference}` resources.
- [#10477](https://github.com/kubevirt/kubevirt/pull/10477) ([@jean-edouard](https://github.com/jean-edouard)) Dynamic KSM enabling and configuration
- [#10110](https://github.com/kubevirt/kubevirt/pull/10110) ([@tiraboschi](https://github.com/tiraboschi)) Stream guest serial console logs from a dedicated container
- [#10015](https://github.com/kubevirt/kubevirt/pull/10015) ([@victortoso](https://github.com/victortoso)) Implements USB host passthrough in permittedHostDevices of KubeVirt CRD
- [#10184](https://github.com/kubevirt/kubevirt/pull/10184) ([@acardace](https://github.com/acardace)) Add memory hotplug feature
- [#10044](https://github.com/kubevirt/kubevirt/pull/10044) ([@machadovilaca](https://github.com/machadovilaca)) Add operator-observability package
- [#10489](https://github.com/kubevirt/kubevirt/pull/10489) ([@maiqueb](https://github.com/maiqueb)) Remove the network-attachment-definition `list` and `watch` verbs from virt-controller's RBAC
- [#10450](https://github.com/kubevirt/kubevirt/pull/10450) ([@0xFelix](https://github.com/0xFelix)) virtctl: Enable inference in create vm subcommand by default
- [#10447](https://github.com/kubevirt/kubevirt/pull/10447) ([@fossedihelm](https://github.com/fossedihelm)) Add a Feature Gate to KV CR to automatically set memory limits when a resource quota with memory limits is associated to the creation namespace
- [#10253](https://github.com/kubevirt/kubevirt/pull/10253) ([@rmohr](https://github.com/rmohr)) Stop trying to create unused directory /var/run/kubevirt-ephemeral-disk in virt-controller
- [#10231](https://github.com/kubevirt/kubevirt/pull/10231) ([@kvaps](https://github.com/kvaps)) Propogate public-keys to cloud-init NoCloud meta-data
- [#10400](https://github.com/kubevirt/kubevirt/pull/10400) ([@alromeros](https://github.com/alromeros)) Add new vmexport flags to download raw images, either directly (--raw) or by decompressing (--decompress) them
- [#9673](https://github.com/kubevirt/kubevirt/pull/9673) ([@germag](https://github.com/germag)) DownwardMetrics: Expose DownwardMetrics through virtio-serial channel.
- [#10086](https://github.com/kubevirt/kubevirt/pull/10086) ([@vladikr](https://github.com/vladikr)) allow live updating VM affinity and node selector
- [#10050](https://github.com/kubevirt/kubevirt/pull/10050) ([@victortoso](https://github.com/victortoso)) Updating the virt stack: QEMU 8.0.0, libvirt to 9.5.0, edk2 20230524,
- [#10370](https://github.com/kubevirt/kubevirt/pull/10370) ([@benjx1990](https://github.com/benjx1990)) N/A
- [#10391](https://github.com/kubevirt/kubevirt/pull/10391) ([@awels](https://github.com/awels)) BugFix: VMExport now works in a namespace with quotas defined.
- [#10386](https://github.com/kubevirt/kubevirt/pull/10386) ([@liuzhen21](https://github.com/liuzhen21)) KubeSphere added to the adopter's file!
- [#10380](https://github.com/kubevirt/kubevirt/pull/10380) ([@alromeros](https://github.com/alromeros)) Bugfix: Allow image-upload to recover from PendingPopulation phase
- [#10366](https://github.com/kubevirt/kubevirt/pull/10366) ([@ormergi](https://github.com/ormergi)) Kubevirt now delegates Slirp networking configuration to Slirp network binding plugin.  In case you haven't registered Slirp network binding plugin image yet (i.e.: specify in Kubevirt config) the following default image would be used: `quay.io/kubevirt/network-slirp-binding:20230830_638c60fc8`. On next release (v1.2.0) no default image will be set and registering an image would be mandatory.
- [#10167](https://github.com/kubevirt/kubevirt/pull/10167) ([@0xFelix](https://github.com/0xFelix)) virtctl: Apply namespace to created manifests
- [#10148](https://github.com/kubevirt/kubevirt/pull/10148) ([@alromeros](https://github.com/alromeros)) Add port-forward functionalities to vmexport
- [#9821](https://github.com/kubevirt/kubevirt/pull/9821) ([@sradco](https://github.com/sradco)) Deprecation notice for the metrics listed in the PR. Please update your systems to use the new metrics names.
- [#10272](https://github.com/kubevirt/kubevirt/pull/10272) ([@ormergi](https://github.com/ormergi)) Introduce network binding plugin for Slirp networking, interfacing with Kubevirt new network binding plugin API.
- [#10284](https://github.com/kubevirt/kubevirt/pull/10284) ([@AlonaKaplan](https://github.com/AlonaKaplan)) Introduce an API for network binding plugins. The feature is behind "NetworkBindingPlugins" gate.
- [#10275](https://github.com/kubevirt/kubevirt/pull/10275) ([@awels](https://github.com/awels)) Ensure new hotplug attachment pod is ready before deleting old attachment pod
- [#9231](https://github.com/kubevirt/kubevirt/pull/9231) ([@victortoso](https://github.com/victortoso)) Introduces sidecar-shim container image
- [#10254](https://github.com/kubevirt/kubevirt/pull/10254) ([@rmohr](https://github.com/rmohr)) Don't mark the KubeVirt "Available" condition as false on up-to-date and ready but misscheduled virt-handler pods.
- [#10185](https://github.com/kubevirt/kubevirt/pull/10185) ([@AlonaKaplan](https://github.com/AlonaKaplan)) Add support to migration based SRIOV hotplug.
- [#10182](https://github.com/kubevirt/kubevirt/pull/10182) ([@iholder101](https://github.com/iholder101)) Stop considering nodes without `kubevirt.io/schedulable` label when finding lowest TSC frequency on the cluster
- [#10138](https://github.com/kubevirt/kubevirt/pull/10138) ([@machadovilaca](https://github.com/machadovilaca)) Change kubevirt_vmi_*_usage_seconds from Gauge to Counter
- [#10173](https://github.com/kubevirt/kubevirt/pull/10173) ([@rmohr](https://github.com/rmohr))
- [#10101](https://github.com/kubevirt/kubevirt/pull/10101) ([@acardace](https://github.com/acardace)) Deprecate `spec.config.machineType` in KubeVirt CR.
- [#10020](https://github.com/kubevirt/kubevirt/pull/10020) ([@akalenyu](https://github.com/akalenyu)) Use auth API for DataVolumes, stop importing kubevirt.io/containerized-data-importer
- [#10107](https://github.com/kubevirt/kubevirt/pull/10107) ([@PiotrProkop](https://github.com/PiotrProkop)) Expose kubevirt_vmi_vcpu_delay_seconds_total reporting amount of seconds VM spent in  waiting in the queue instead of running.
- [#10099](https://github.com/kubevirt/kubevirt/pull/10099) ([@iholder101](https://github.com/iholder101)) Bugfix: target virt-launcher pod hangs when migration is cancelled.
- [#10056](https://github.com/kubevirt/kubevirt/pull/10056) ([@jean-edouard](https://github.com/jean-edouard)) UEFI guests now use Bochs display instead of VGA emulation
- [#10070](https://github.com/kubevirt/kubevirt/pull/10070) ([@machadovilaca](https://github.com/machadovilaca)) Remove affinities label from kubevirt_vmi_cpu_affinity and use sum as value
- [#10165](https://github.com/kubevirt/kubevirt/pull/10165) ([@awels](https://github.com/awels)) BugFix: deleting hotplug attachment pod will no longer detach volumes that were not removed.
- [#9878](https://github.com/kubevirt/kubevirt/pull/9878) ([@jean-edouard](https://github.com/jean-edouard)) The EFI NVRAM can now be configured to persist across reboots
- [#9932](https://github.com/kubevirt/kubevirt/pull/9932) ([@lyarwood](https://github.com/lyarwood)) `ControllerRevisions` containing `instancetype.kubevirt.io` `CRDs` are now decorated with labels detailing specific metadata of the underlying stashed object
- [#10039](https://github.com/kubevirt/kubevirt/pull/10039) ([@simonyangcj](https://github.com/simonyangcj)) fix guaranteed qos of virt-launcher pod broken when use virtiofs
- [#10116](https://github.com/kubevirt/kubevirt/pull/10116) ([@ormergi](https://github.com/ormergi)) Existing detached interfaces with 'absent' state will be cleared from VMI spec.
- [#9982](https://github.com/kubevirt/kubevirt/pull/9982) ([@fabiand](https://github.com/fabiand)) Introduce a support lifecycle and Kubernetes target version.
- [#10118](https://github.com/kubevirt/kubevirt/pull/10118) ([@akalenyu](https://github.com/akalenyu)) Change exportserver default UID to succeed exporting CDI standalone PVCs (not attached to VM)
- [#10106](https://github.com/kubevirt/kubevirt/pull/10106) ([@acardace](https://github.com/acardace)) Add boot-menu wait time when starting the VM as paused.
- [#10058](https://github.com/kubevirt/kubevirt/pull/10058) ([@alicefr](https://github.com/alicefr)) Add field errorPolicy for disks
- [#10004](https://github.com/kubevirt/kubevirt/pull/10004) ([@AlonaKaplan](https://github.com/AlonaKaplan)) Hoyplug/unplug interfaces should be done by updating the VM spec template. virtctl and REST API endpoints were removed.
- [#10067](https://github.com/kubevirt/kubevirt/pull/10067) ([@iholder101](https://github.com/iholder101)) Bug fix: `virtctl create clone` marshalling and replacement of `kubectl` with `kubectl virt`
- [#9989](https://github.com/kubevirt/kubevirt/pull/9989) ([@alaypatel07](https://github.com/alaypatel07)) Add perf scale benchmarks for VMIs
- [#10001](https://github.com/kubevirt/kubevirt/pull/10001) ([@machadovilaca](https://github.com/machadovilaca)) Fix kubevirt_vmi_phase_count not being created
- [#9896](https://github.com/kubevirt/kubevirt/pull/9896) ([@ormergi](https://github.com/ormergi)) The VM controller now replicates spec interfaces MAC addresses to the corresponding interfaces in the VMI spec.
- [#9840](https://github.com/kubevirt/kubevirt/pull/9840) ([@dhiller](https://github.com/dhiller)) Increase probability for flake checker script to find flakes
- [#9988](https://github.com/kubevirt/kubevirt/pull/9988) ([@enp0s3](https://github.com/enp0s3)) always deploy the outdated VMI workload alert
- [#7708](https://github.com/kubevirt/kubevirt/pull/7708) ([@VirrageS](https://github.com/VirrageS)) `nodeSelector` and `schedulerName` fields have been added to VirtualMachineInstancetype spec.
- [#7197](https://github.com/kubevirt/kubevirt/pull/7197) ([@vasiliy-ul](https://github.com/vasiliy-ul)) Experimantal support of SEV attestation via the new API endpoints
- [#9958](https://github.com/kubevirt/kubevirt/pull/9958) ([@AlonaKaplan](https://github.com/AlonaKaplan)) Disable network interface hotplug/unplug for VMIs. It will be supported for VMs only.
- [#9882](https://github.com/kubevirt/kubevirt/pull/9882) ([@dhiller](https://github.com/dhiller)) Add some context for initial contributors about automated testing and draft pull requests.
- [#9935](https://github.com/kubevirt/kubevirt/pull/9935) ([@xpivarc](https://github.com/xpivarc)) Bug fix - correct logging in container disk
- [#9552](https://github.com/kubevirt/kubevirt/pull/9552) ([@phoracek](https://github.com/phoracek)) gRPC client now works correctly with non-Go gRPC servers
- [#9918](https://github.com/kubevirt/kubevirt/pull/9918) ([@ShellyKa13](https://github.com/ShellyKa13)) Fix for hotplug with WFFC SCI storage class which uses CDI populators
- [#9737](https://github.com/kubevirt/kubevirt/pull/9737) ([@AlonaKaplan](https://github.com/AlonaKaplan)) On hotunplug - remove bridge, tap and dummy interface from virt-launcher and the caches (file and volatile) from the node.
- [#9861](https://github.com/kubevirt/kubevirt/pull/9861) ([@rmohr](https://github.com/rmohr)) Fix the possibility of data corruption when requestin a force-restart via "virtctl restart"
- [#9818](https://github.com/kubevirt/kubevirt/pull/9818) ([@akrejcir](https://github.com/akrejcir)) Added "virtctl credentials" commands to dynamically change SSH keys in a VM, and to set user's password.
- [#9872](https://github.com/kubevirt/kubevirt/pull/9872) ([@alromeros](https://github.com/alromeros)) Bugfix: Allow lun disks to be mapped to DataVolume sources
- [#9073](https://github.com/kubevirt/kubevirt/pull/9073) ([@machadovilaca](https://github.com/machadovilaca)) Fix incorrect KubevirtVmHighMemoryUsage description

## Contributors

76 people contributed to this release:

```
62	Victor Toso <victortoso@redhat.com>
55	Edward Haas <edwardh@redhat.com>
43	Or Mergi <ormergi@redhat.com>
42	fossedihelm <ffossemo@redhat.com>
39	Itamar Holder <iholder@redhat.com>
38	Alona Paz <alkaplan@redhat.com>
36	Vasiliy Ulyanov <vulyanov@suse.de>
27	Ondrej Pokorny <opokorny@redhat.com>
26	Daniel Hiller <dhiller@redhat.com>
26	Fabian Deutsch <fabiand@redhat.com>
21	Lee Yarwood <lyarwood@redhat.com>
19	Antonio Cardace <acardace@redhat.com>
19	Felix Matouschek <fmatouschek@redhat.com>
16	Luboslav Pivarc <lpivarc@redhat.com>
15	Jed Lejosne <jed@redhat.com>
14	Alexander Wels <awels@redhat.com>
12	Alvaro Romero <alromero@redhat.com>
12	João Vilaça <jvilaca@redhat.com>
11	Roman Mohr <rmohr@google.com>
10	enp0s3 <ibezukh@redhat.com>
9	Varun Ramachandra Sekar <varun.sekar1994@gmail.com>
9	prnaraya <prnaraya@redhat.com>
9	stirabos <stirabos@redhat.com>
8	Alex Kalenyuk <akalenyu@redhat.com>
8	Alice Frosi <afrosi@redhat.com>
8	Brian Carey <bcarey@redhat.com>
7	Andrew Burden <aburden@redhat.com>
6	L. Pivarc <lpivarc@redhat.com>
6	Vladik Romanovsky <vromanso@redhat.com>
5	Andrej Krejcir <akrejcir@redhat.com>
5	German Maglione <gmaglione@redhat.com>
4	Javier Cano Cano <jcanocan@redhat.com>
4	Michael Henriksen <mhenriks@redhat.com>
4	Miguel Duarte Barroso <mdbarroso@redhat.com>
3	Alay Patel <alayp@nvidia.com>
3	Dan Kenigsberg <danken@redhat.com>
3	Daniel Hiller <daniel.hiller.1972@googlemail.com>
3	Dharmit Shah <shahdharmit@gmail.com>
3	HHHskkk <913596231@qq.com>
3	Janusz Marcinkiewicz <januszm@nvidia.com>
3	Or Shoval <oshoval@redhat.com>
3	Orel Misan <omisan@redhat.com>
3	Pavel Tishkov <pavel.tishkov@flant.com>
3	Shelly Kagan <skagan@redhat.com>
3	Shirly Radco <sradco@redhat.com>
3	bmordeha <bmordeha@redhat.com>
2	Andrei Kvapil <kvapss@gmail.com>
2	Arnon Gilboa <agilboa@redhat.com>
2	Assaf Admi <aadmi@redhat.com>
2	Benjamin <72671586+benjx1990@users.noreply.github.com>
2	Oliver Sabiniarz <o_sabiniarz@yahoo.de>
2	PiotrProkop <pprokop@nvidia.com>
2	howard zhang <howard.zhang@arm.com>
2	liuzhen <liuzhen@yunify.com>
2	rkishner <rkishner@redhat.com>
2	rokkiter <101091030+rokkiter@users.noreply.github.com>
2	yojay11717 <lanyujie@inspur.com>
1	Alay Patel <alay1431@gmail.com>
1	Andrea Bolognani <abologna@redhat.com>
1	Aviv Litman <alitman@alitman-thinkpadp1gen4i.tlv.csb>
1	Aviv Litman <alitman@alitman.tlv.csb>
1	Aviv Litman <alitman@redhat.com>
1	Eng Zer Jun <engzerjun@gmail.com>
1	Itamar Holder <77444623+iholder101@users.noreply.github.com>
1	Marcelo Tosatti <mtosatti@redhat.com>
1	Maya Rashish <mrashish@redhat.com>
1	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
1	Nijin Ashok <nashok@redhat.com>
1	Petr Horacek <hrck@protonmail.com>
1	Reficul <xuzhenglun@gmail.com>
1	SIMON COTER <simon.coter@oracle.com>
1	akrgupta <akrgupta@redhat.com>
1	grass-lu <284555125@qq.com>
1	rokkiter <yongen.pan@daocloud.io>
1	wangzihao05 <wangzihao05@inspur.com>
1	yangchenjun <yang.chenjun@99cloud.net>
```

## Additional Resources

- Mailing list: https://groups.google.com/forum/#!forum/kubevirt-dev
- Slack: https://kubernetes.slack.com/messages/virtualization
- An easy to use demo: https://github.com/kubevirt/demo
- How to contribute: https://github.com/kubevirt/kubevirt/blob/main/CONTRIBUTING.md
- License: https://github.com/kubevirt/kubevirt/blob/main/LICENSE
