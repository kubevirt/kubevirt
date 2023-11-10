KubeVirt v0.59.0
================

This release follows v0.58.1 and consists of 940 changes, contributed by 73 people, leading to 1435 files changed, 121668 insertions(+), 40676 deletions(-).
v0.59.0 is a promotion of release candidate v0.59.0-rc.2 which was originally published 2023-03-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.59.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.59.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #9311][kubevirt-bot] fixes the requests/limits CPU number mismatch for VMs with isolatedEmulatorThread
- [PR #9276][fossedihelm] Added foreground finalizer to  virtual machine
- [PR #9295][kubevirt-bot] Fix bug of possible re-trigger of memory dump
- [PR #9270][kubevirt-bot] BugFix: Guestfs image url not constructed correctly
- [PR #9234][kubevirt-bot] The `dedicatedCPUPlacement` attribute is once again supported within the `VirtualMachineInstancetype` and `VirtualMachineClusterInstancetype` CRDs after a recent bugfix improved `VirtualMachine` validations, ensuring defaults are applied before any attempt to validate.
- [PR #9267][fossedihelm] This version of KubeVirt includes upgraded virtualization technology based on libvirt 9.0.0 and QEMU 7.2.0.
- [PR #9197][kubevirt-bot] Fix addvolume not rejecting adding existing volume source, fix removevolume allowing to remove non hotpluggable volume
- [PR #9120][0xFelix] Fix access to portforwarding on VMs/VMIs with the cluster roles kubevirt.io:admin and kubevirt.io:edit
- [PR #9116][EdDev] Allow the specification of the ACPI Index on a network interface.
- [PR #8774][avlitman] Added new Virtual machines CPU metrics:
- [PR #9087][zhuchenwang] Open `/dev/vhost-vsock` explicitly to ensure that the right vsock module is loaded
- [PR #9020][feitnomore] Adding support for status/scale subresources so that VirtualMachinePool now supports HorizontalPodAutoscaler
- [PR #9085][0xFelix] virtctl: Add options to infer instancetype and preference when creating a VM
- [PR #8917][xpivarc] Kubevirt can be configured with Seccomp profile. It now ships a custom profile for the launcher.
- [PR #9054][enp0s3] do not inject LimitRange defaults into VMI
- [PR #7862][vladikr] Store the finalized VMI migration status in the migration objects.
- [PR #8878][0xFelix] Add 'create vm' command to virtctl
- [PR #9048][jean-edouard] DisableCustomSELinuxPolicy feature gate introduced to disable our custom SELinux policy
- [PR #8953][awels] VMExport now has endpoint containing entire VM definition.
- [PR #8976][iholder101] Fix podman CRI detection
- [PR #9043][iholder101] Adjust operator functional tests to custom images specification
- [PR #8875][machadovilaca] Rename migration metrics removing 'total' keyword
- [PR #9040][lyarwood] `inferFromVolume` now uses labels instead of annotations to lookup default instance type and preference details from a referenced `Volume`. This has changed in order to provide users with a way of looking up suitably decorated resources through these labels before pointing to them within the `VirtualMachine`.
- [PR #9039][orelmisan] client-go: Added context to additional VirtualMachineInstance's methods.
- [PR #9018][orelmisan] client-go: Added context to additional VirtualMachineInstance's methods.
- [PR #9025][akalenyu] BugFix: Hotplug pods have hardcoded resource req which don't comply with LimitRange maxLimitRequestRatio of 1
- [PR #8908][orelmisan] client-go: Added context to some of VirtualMachineInstance's methods.
- [PR #6863][rmohr] The install strategy job will respect the infra node placement from now on
- [PR #8948][iholder101] Bugfix: virt-handler socket leak
- [PR #8649][acardace] KubeVirt is now able to run VMs inside restricted namespaces.
- [PR #8992][iholder101] Align with k8s fix for default limit range requirements
- [PR #8889][rmohr] Add basic TLS encryption support for vsock websocket connections
- [PR #8660][huyinhou] Fix remoteAddress field in virt-api log being truncated when it is an ipv6 address
- [PR #8961][rmohr] Bump distroless base images
- [PR #8952][rmohr] Fix read-only sata disk validation
- [PR #8657][fossedihelm] Use an increasingly exponential backoff before retrying to start the VM, when an I/O error occurs.
- [PR #8480][lyarwood] New `inferFromVolume` attributes have been introduced to the `{Instancetype,Preference}Matchers` of a `VirtualMachine`. When provided the `Volume` referenced by the attribute is checked for the following annotations with which to populate the `{Instancetype,Preference}Matchers`:
- [PR #7762][VirrageS] Service `kubevirt-prometheus-metrics` now sets `ClusterIP` to `None` to make it a headless service.
- [PR #8599][machadovilaca] Change KubevirtVmHighMemoryUsage threshold from 20MB to 50MB
- [PR #7761][VirrageS] imagePullSecrets field has been added to KubeVirt CR to support deployments form private registries
- [PR #8887][iholder101] Bugfix: use virt operator image if provided
- [PR #8750][jordigilh] Fixes an issue that prevented running real time workloads in non-root configurations due to libvirt's dependency on CAP_SYS_NICE to change the vcpu's thread's scheduling and priority to FIFO and 1. The change of priority and scheduling is now executed in the virt-launcher for both root and non-root configurations, removing the dependency in libvirt.
- [PR #8845][lyarwood] An empty `Timer` is now correctly omitted from `Clock` fixing bug #8844.
- [PR #8842][andreabolognani] The virt-launcher pod no longer needs the SYS_PTRACE capability.
- [PR #8734][alicefr] Change libguestfs-tools image using root appliance in qcow2 format
- [PR #8764][ShellyKa13] Add list of included and excluded volumes in vmSnapshot
- [PR #8811][iholder101] Custom components: support gs
- [PR #8770][dhiller] Add Ginkgo V2 Serial decorator to serial tests as preparation to simplify parallel vs. serial test run logic
- [PR #8808][acardace] Apply migration backoff only for evacuation migrations.
- [PR #8525][jean-edouard] CR option mediatedDevicesTypes is deprecated in favor of mediatedDeviceTypes
- [PR #8792][iholder101] Expose new custom components env vars to csv-generator and manifest-templator
- [PR #8701][enp0s3] Consider the ParallelOutboundMigrationsPerNode when evicting VMs
- [PR #8740][iholder101] Fix: Align Reenlightenment flows between converter.go and template.go
- [PR #8530][acardace] Use exponential backoff for failing migrations
- [PR #8720][0xFelix] The expand-spec subresource endpoint was renamed to expand-vm-spec and made namespaced
- [PR #8458][iholder101] Introduce support for clones with a snapshot source (e.g. clone snapshot -> VM)
- [PR #8716][rhrazdil] Add overhead of interface with Passt binding when no ports are specified
- [PR #8619][fossedihelm] virt-launcher: use `virtqemud` daemon instead of `libvirtd`
- [PR #8736][knopt] Added more precise rest_client_request_latency_seconds histogram buckets
- [PR #8624][zhuchenwang] Add the REST API to be able to talk to the application in the guest VM via VSOCK.
- [PR #8625][AlonaKaplan] iptables are no longer used by masquerade binding. Nodes with iptables only won't be able to run VMs with masquerade binding.
- [PR #8673][iholder101] Allow specifying custom images for core components
- [PR #8622][jean-edouard] Built with golang 1.19
- [PR #8336][alicefr] Flag for setting the guestfs uid and gid
- [PR #8667][huyinhou] connect VM vnc failed when virt-launcher work directory is not /
- [PR #8368][machadovilaca] Use collector to set migration metrics
- [PR #8558][xpivarc] Bug-fix: LimitRange integration now works when VMI is missing namespace
- [PR #8404][andreabolognani] This version of KubeVirt includes upgraded virtualization technology based on libvirt 8.7.0, QEMU 7.1.0 and CentOS Stream 9.
- [PR #8652][akalenyu] BugFix: Exporter pod does not comply with restricted PSA
- [PR #8563][xpivarc] Kubevirt now runs with nonroot user by default
- [PR #8442][kvaps] Add Deckhouse to the Adopters list
- [PR #8546][zhuchenwang] Provides the Vsock feature for KubeVirt VMs.
- [PR #8598][acardace] VMs configured with hugepages can now run using the default container_t SELinux type
- [PR #8594][kylealexlane] Fix permission denied on on selinux relabeling on some kernel versions
- [PR #8521][akalenyu] Add an option to specify a TTL for VMExport objects
- [PR #7918][machadovilaca] Add alerts for VMs unhealthy states
- [PR #8516][rhrazdil] When using Passt binding, virl-launcher has unprivileged_port_start set to 0, so that passt may bind to all ports.
- [PR #7772][jean-edouard] The SELinux policy for virt-launcher is down to 4 rules, 1 for hugepages and 3 for virtiofs.
- [PR #8402][jean-edouard] Most VMIs now run under the SELinux type container_t
- [PR #8513][alromeros] [Bug-fix] Fix error handling in virtctl image-upload

Contributors
------------
73 people contributed to this release:

```
62	Itamar Holder <iholder@redhat.com>
39	L. Pivarc <lpivarc@redhat.com>
36	Lee Yarwood <lyarwood@redhat.com>
33	Andrea Bolognani <abologna@redhat.com>
29	Edward Haas <edwardh@redhat.com>
28	fossedihelm <ffossemo@redhat.com>
25	Antonio Cardace <acardace@redhat.com>
23	Felix Matouschek <fmatouschek@redhat.com>
23	Jed Lejosne <jed@redhat.com>
22	bmordeha <bmodeha@redhat.com>
20	Roman Mohr <rmohr@google.com>
18	Alex Kalenyuk <akalenyu@redhat.com>
18	Orel Misan <omisan@redhat.com>
17	Shelly Kagan <skagan@redhat.com>
16	Alice Frosi <afrosi@redhat.com>
14	Alexander Wels <awels@redhat.com>
12	Marcelo Tosatti <mtosatti@redhat.com>
11	Jordi Gil <jgil@redhat.com>
10	Alvaro Romero <alromero@redhat.com>
10	Andrej Krejcir <akrejcir@redhat.com>
9	Dan Kenigsberg <danken@redhat.com>
9	João Vilaça <jvilaca@redhat.com>
8	Or Shoval <oshoval@redhat.com>
8	Radim Hrazdil <rhrazdil@redhat.com>
7	Maya Rashish <mrashish@redhat.com>
6	Brian Carey <bcarey@redhat.com>
6	Ram Lavi <ralavi@redhat.com>
6	feitnomore <feitnomore@users.noreply.github.com>
5	Bartosz Rybacki <brybacki@redhat.com>
5	Ben Oukhanov <boukhanov@redhat.com>
5	Janusz Marcinkiewicz <januszm@nvidia.com>
5	Vasiliy Ulyanov <vulyanov@suse.de>
5	Zhuchen Wang <zcwang@google.com>
4	Alona Paz <alkaplan@redhat.com>
4	Daniel Hiller <dhiller@redhat.com>
4	Howard Zhang <howard.zhang@arm.com>
4	Vladik Romanovsky <vromanso@redhat.com>
4	enp0s3 <ibezukh@redhat.com>
3	Javier Cano Cano <jcanocan@redhat.com>
3	Michael Henriksen <mhenriks@redhat.com>
3	howard zhang <howard.zhang@arm.com>
3	huyinhou <huyinhou@bytedance.com>
3	prnaraya <prnaraya@redhat.com>
2	Alay Patel <alayp@nvidia.com>
2	Arnon Gilboa <agilboa@redhat.com>
2	Ondrej Pokorny <opokorny@redhat.com>
2	Petr Horáček <phoracek@redhat.com>
2	윤세준 <sjyoon@sjyoon02.local>
1	Andrei Kvapil <kvapss@gmail.com>
1	Arnaud Aubert <aaubert@magesi.com>
1	Aviv Litman <alitman@redhat.com>
1	Fabian Deutsch <fabiand@redhat.com>
1	Geetika Kapoor <gkapoor@redhat.com>
1	HF <crazytaxii666@gmail.com>
1	Igor Bezukh <ibezukh@redhat.com>
1	Miguel Duarte Barroso <mdbarroso@redhat.com>
1	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
1	Petr Horacek <hrck@protonmail.com>
1	PiotrProkop <pprokop@nvidia.com>
1	Ryan Hallisey <rhallisey@nvidia.com>
1	Shirly Radco <sradco@redhat.com>
1	Simone Tiraboschi <stirabos@redhat.com>
1	Stu Gott <sgott@redhat.com>
1	Tomasz Knopik <tknopik@nvidia.com>
1	Yan Du <yadu@redhat.com>
1	Yufeng Duan <55268016+didovesei@users.noreply.github.com>
1	akriti gupta <akrgupta@redhat.com>
1	assaf-admi <aadmi@redhat.com>
1	dalia-frank <dafrank@redhat.com>
1	jia.dong <jia.dong@i-tudou.com>
1	kfox1111 <Kevin.Fox@pnnl.gov>
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
