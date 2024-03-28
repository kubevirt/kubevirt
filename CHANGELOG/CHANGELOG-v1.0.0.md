KubeVirt v1.0.0
===============

This release follows v0.59.2 and consists of 1089 changes, contributed by 74 people, leading to 2849 files changed, 232018 insertions(+), 168449 deletions(-)
v1.0.0 is a promotion of release candidate v1.0.0-rc.1 which was originally published 2023-06-30
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v1.0.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v1.0.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #10037][kubevirt-bot] The VM controller now replicates spec interfaces MAC addresses to the corresponding interfaces in the VMI spec.
- [PR #9992][machadovilaca] Fix incorrect KubevirtVmHighMemoryUsage description
- [PR #9965][kubevirt-bot] Disable network interface hotplug/unplug for VMIs. It will be supported for VMs only.
- [PR #9931][kubevirt-bot] Fix for hotplug with WFFC SCI storage class which uses CDI populators
- [PR #9946][kubevirt-bot] On hotunplug - remove bridge, tap and dummy interface from virt-launcher and the caches (file and volatile) from the node.
- [PR #9757][enp0s3] Introduce CPU hotplug
- [PR #9811][machadovilaca] Remove unnecessary marketplace tool
- [PR #7742][Fuzzy-Math] Experimental support for AMD SEV-ES
- [PR #9799][vladikr] Introduce an ability to set memory overcommit percentage in instanceType spec
- [PR #8780][lyarwood] Add basic support for expressing minimum resource requirements for CPU and Memory within VirtualMachine{Preferences,ClusterPreferences}
- [PR #9812][mhenriks] Handle DataVolume PendingPopulation phase
- [PR #9858][fossedihelm] build virtctl for all os/architectures when `KUBEVIRT_RELEASE` env var is true
- [PR #9765][lyarwood] Allow to define preferred cpu features in VirtualMachine{Preferences,ClusterPreferences}
- [PR #9844][EdDev] Drop the `kubevirt.io/interface` resource name API for reserving domain resources for network interfaces.
- [PR #9841][ormergi] Support hot-unplug of network interfaces on VirtualMachine objects
- [PR #9851][lxs137] virt-api: portfowrad can handle IPv6 VM
- [PR #9845][lxs137] DHCPv6 server handle request without iana option
- [PR #9769][lyarwood] Allow to define the preferred subdomain in VirtualMachine{Preferences,ClusterPreferences}
- [PR #9246][jean-edouard] Fixed migration issue for VMIs that have RWX disks backed by filesystem storage classes.
- [PR #9808][jcanocan] DownwardMetrics: Rename AllocatedToVirtualServers metric to AllocatedToVirtualServers and add ResourceProcessorLimit metric
- [PR #9832][tiraboschi] build virtctl also for arm64 for linux, darwin and windows
- [PR #9744][lyarwood] Allow to define the preferred termination grace period in VirtualMachine{Preferences,ClusterPreferences}
- [PR #9828][rthallisey] Publish multiarch manifests with each release
- [PR #9761][lyarwood] Allow to define the preferred masquerade configuration in VirtualMachine{Preferences,ClusterPreferences}
- [PR #9768][jean-edouard] New CR option to enable auto CPU limits for virt-launcher on some namespaces
- [PR #9779][EdDev] Support hot-unplug of network interfaces on VMI objects
- [PR #9688][xpivarc] Users are warned about the usage of deprecated fields
- [PR #9798][rmohr] Add LiveMigrateIfPossible eviction strategy to allow admins to express a live migration preference instead of a live migration requirement for evictions.
- [PR #9764][fossedihelm] Cluster admins can enable ksm in a set of nodes via kv configuration
- [PR #9753][lyarwood] The following flags have been added to the `virtctl image-upload` command allowing users to associate a default instance type and/or preference with an image during upload. `--default-instancetype`,  `--default-instancetype-kind`, `--default-preference` and `--default-preference-kind`. [See the user-guide documentation](https://kubevirt.io/user-guide/virtual_machines/instancetypes/#inferfromvolume) for more details on using the uploaded image with the `inferFromVolume` feature during `VirtualMachine` creation.
- [PR #9575][lyarwood] A new `v1beta1` version of the `instancetype.kubevirt.io` API and CRDs has been introduced.
- [PR #9738][Barakmor1] Add condition to migrations that indicates that migration was rejected by ResourceQuota
- [PR #9730][assafad] Add `kubevirt_vmi_memory_cached_bytes` metric
- [PR #9674][fossedihelm] Introduce cluster configuration `VirtualMachineOptions` to specify virtual machine behavior at cluster level
- [PR #9724][0xFelix] An alert which triggers when KubeVirt APIs marked as deprecated are used was added.
- [PR #9623][rmohr] Bump to apimachinery 1.26
- [PR #9747][lyarwood] action required - With the `v1.0.0` release of KubeVirt the storage version of all core `kubevirt.io` APIs will be moving to version `v1`. To accommodate the eventual removal of the `v1alpha3` version with KubeVirt >=`v1.2.0` it is recommended that operators deploy the [`kube-storage-version-migrator`](https://github.com/kubernetes-sigs/kube-storage-version-migrator) tool within their environment. This will ensure any existing `v1alpha3` stored objects are migrated to `v1` well in advance of the removal of the underlying `v1alpha3` version.
- [PR #9268][ormergi] virt-launcher pods network interfaces name scheme is changed to hashed names (SHA256), based on the VMI spec network names.
- [PR #9746][EdDev] Introduce the `kubevirt.io/interface` resource name to reserve domain resources for network interfaces.
- [PR #9652][machadovilaca] Add kubevirt_number_of_vms recording rule
- [PR #9691][fossedihelm] ksm enabled nodes will have `kubevirt.io/ksm-enabled` label
- [PR #9628][lyarwood] * The `kubevirt.io/v1` `apiVersion` is now the default storage version for newly created objects
- [PR #8293][daghaian] Add multi-arch support to KubeVirt. This allows a single KubeVirt installation to run VMs on different node architectures in the same cluster.
- [PR #9686][maiqueb] Fix ownership of macvtap's char devices on non-root pods
- [PR #9631][0xFelix] virtctl: Allow to infer instancetype or preference from specified volume when creating VMs
- [PR #9665][rmohr] Expose the final resolved qemu machine type on the VMI on status.machine
- [PR #9609][germag] Add support for running virtiofsd in an unprivileged container when sharing configuration volumes.
- [PR #9651][0xFelix] virtctl: Allow to specify memory of created VMs. Default to 512Mi if no instancetype was specified or is inferred.
- [PR #9640][jean-edouard] TSC-enabled VMs can now migrate to a node with a non-identical (but close-enough) frequency
- [PR #9629][0xFelix] virtctl: Allow to specify the boot order of volumes when creating VMs
- [PR #9632][toelke] * Add Genesis Cloud to the adopters list
- [PR #9572][fossedihelm] Enable freePageReporting for new non high performance vmi
- [PR #9435][rmohr] Ensure existence of all PVCs attached to the VMI before creating the VM target pod.
- [PR #8156][jean-edouard] TPM VM device can now be set to persistent
- [PR #8575][iholder101] QEMU-level migration parallelism (a.k.a. multifd) + Upgrade QEMU to 7.2.0-11.el9
- [PR #9603][qinqon] Adapt node-labeller.sh script to work at non kvm envs with emulation.
- [PR #9591][awels] BugFix: allow multiple NFS disks to be used/hotplugged
- [PR #9596][iholder101] Add "virtctl create clone" command
- [PR #9422][awels] Ability to specify cpu/mem request limit for supporting containers (hotplug/container disk/virtiofs/side car)
- [PR #9536][akalenyu] BugFix: virtualmachineclusterinstancetypes/preferences show up for get all -n <namespace>
- [PR #9177][alicefr] Adding SCSI persistent reservation
- [PR #9470][machadovilaca] Enable libvirt GetDomainStats on paused VMs
- [PR #9407][assafad] Use env `RUNBOOK_URL_TEMPLATE` for the runbooks URL template
- [PR #9399][maiqueb] Compute the interfaces to be hotplugged based on the current domain info, rather than on the interface status.
- [PR #9491][orelmisan] API, AddInterfaceOptions: Rename NetworkName to NetworkAttachmentDefinitionName and InterfaceName to Name
- [PR #9327][jcanocan] DownwardMetrics: Swap KubeVirt build info with qemu version in VirtProductInfo field
- [PR #9478][xpivarc] Bug fix: Fixes case when migration is not retried if the migration Pod gets denied.
- [PR #9421][lyarwood] Requests to update the target `Name` of a `{Instancetype,Preference}Matcher` without also updating the `RevisionName` are now rejected.
- [PR #9367][machadovilaca] Add VM instancetype and preference label to vmi_phase_count metric
- [PR #9392][awels] virtctl supports retrieving vm manifest for VM export
- [PR #9442][EdDev] Remove the VMI Status interface `podConfigDone` field in favor of a new source option in `infoSource`.
- [PR #9376][ShellyKa13] Fix vmrestore with WFFC snapshotable storage class
- [PR #6852][maiqueb] Dev preview: Enables network interface hotplug for VMs / VMIs
- [PR #9300][xpivarc] Bug fix: API and virtctl invoked migration is not rejected when the VM is paused
- [PR #9189][xpivarc] Bug fix: DNS integration continues to work after migration
- [PR #9322][iholder101] Add guest-to-request memory headroom ratio.
- [PR #8906][machadovilaca] Alert if there are no available nodes to run VMs
- [PR #9320][darfux] node-labeller: Check arch on the handler side
- [PR #9127][fossedihelm] Use ECDSA instead of RSA for key generation
- [PR #9330][qinqon] Skip label kubevirt.io/migrationTargetNodeName from virtctl expose service selector
- [PR #9163][vladikr] fixes the requests/limits CPU number mismatch for VMs with isolatedEmulatorThread
- [PR #9250][vladikr] externally created mediated devices will not be deleted by virt-handler
- [PR #9193][qinqon] Add annotation for live migration and bridged pod interface
- [PR #9260][ShellyKa13] Fix bug of possible re-trigger of memory dump
- [PR #9241][akalenyu] BugFix: Guestfs image url not constructed correctly
- [PR #9220][orelmisan] client-go: Added context to VirtualMachine's methods.
- [PR #9228][rumans] Bump virtiofs container limit
- [PR #9169][lyarwood] The `dedicatedCPUPlacement` attribute is once again supported within the `VirtualMachineInstancetype` and `VirtualMachineClusterInstancetype` CRDs after a recent bugfix improved `VirtualMachine` validations, ensuring defaults are applied before any attempt to validate.
- [PR #9159][andreabolognani] This version of KubeVirt includes upgraded virtualization technology based on libvirt 9.0.0 and QEMU 7.2.0.
- [PR #8989][rthallisey] Integrate multi-architecture container manifests into the bazel make recipes
- [PR #9188][awels] Default RBAC for clone and export
- [PR #9145][awels] Show VirtualMachine name in the VMExport status
- [PR #8937][fossedihelm] Added foreground finalizer to  virtual machine
- [PR #9133][ShellyKa13] Fix addvolume not rejecting adding existing volume source, fix removevolume allowing to remove non hotpluggable volume
- [PR #9047][machadovilaca] Deprecate VM stuck in status alerts

Contributors
------------
74 people contributed to this release:

```
50	Edward Haas <edwardh@redhat.com>
46	Lee Yarwood <lyarwood@redhat.com>
39	Orel Misan <omisan@redhat.com>
37	fossedihelm <ffossemo@redhat.com>
36	Alice Frosi <afrosi@redhat.com>
31	Felix Matouschek <fmatouschek@redhat.com>
30	Miguel Duarte Barroso <mdbarroso@redhat.com>
28	German Maglione <gmaglione@redhat.com>
27	Or Mergi <ormergi@redhat.com>
24	Itamar Holder <iholder@redhat.com>
24	L. Pivarc <lpivarc@redhat.com>
21	Alona Paz <alkaplan@redhat.com>
20	Roman Mohr <rmohr@google.com>
19	João Vilaça <jvilaca@redhat.com>
18	Jed Lejosne <jed@redhat.com>
17	Alexander Wels <awels@redhat.com>
16	Vladik Romanovsky <vromanso@redhat.com>
16	enp0s3 <ibezukh@redhat.com>
14	aghaiand <david.aghaian@panasonic.aero>
12	Ondrej Pokorny <opokorny@redhat.com>
11	Daniel Hiller <dhiller@redhat.com>
11	Victor Toso <victortoso@redhat.com>
11	howard zhang <howard.zhang@arm.com>
10	Alex Kalenyuk <akalenyu@redhat.com>
10	Maya Rashish <mrashish@redhat.com>
10	Shelly Kagan <skagan@redhat.com>
9	Vasiliy Ulyanov <vulyanov@suse.de>
8	Andrea Bolognani <abologna@redhat.com>
7	Michael Henriksen <mhenriks@redhat.com>
7	Ryan Hallisey <rhallisey@nvidia.com>
7	bmordeha <bmodeha@redhat.com>
6	David Aghaian <16483722+daghaian@users.noreply.github.com>
6	Fabian Deutsch <fabiand@redhat.com>
6	Nithish <nithishkarthik01@gmail.com>
6	Or Shoval <oshoval@redhat.com>
5	Alvaro Romero <alromero@redhat.com>
5	Brian Carey <bcarey@redhat.com>
4	Caleb Crane <ccrane@suse.de>
4	Luboslav Pivarc <lpivarc@redhat.com>
4	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
3	David Vossel <dvossel@redhat.com>
3	Enrique Llorente <ellorent@redhat.com>
3	Janusz Marcinkiewicz <januszm@nvidia.com>
2	Alay Patel <alayp@nvidia.com>
2	Andrej Krejcir <akrejcir@redhat.com>
2	Andrew Imeson <andrew@andrewimeson.com>
2	Antonio Cardace <acardace@redhat.com>
2	Jan Wozniak <wozniak.jan@gmail.com>
2	Kyle Lane <kylelane@google.com>
2	Marcelo Tosatti <mtosatti@redhat.com>
2	Vicente Cheng <vicente.cheng@suse.com>
2	assaf-admi <aadmi@redhat.com>
2	menyakun <lxs137@hotmail.com>
1	Chris Ho <chris.he@suse.com>
1	HF <crazytaxii666@gmail.com>
1	Javier Cano Cano <jcanocan@redhat.com>
1	Justin Cichra <jcichra@cloudflare.com>
1	Li Yuxuan <liyuxuan.darfux@bytedance.com>
1	Mark <mlavi@users.noreply.github.com>
1	Petr Horacek <hrck@protonmail.com>
1	Philipp Riederer <philipp@riederer.email>
1	Pritam Saha <saha7pritam@gmail.com>
1	Ram Lavi <ralavi@redhat.com>
1	Romà Llorens <roma.llorens@gmail.com>
1	Tomasz Knopik <tknopik@nvidia.com>
1	Zhuchen Wang <zcwang@google.com>
1	alitman <alitman@redhat.com>
1	dalia-frank <dafrank@redhat.com>
1	prnaraya <prnaraya@redhat.com>
1	stirabos <stirabos@redhat.com>
1	xpivarc <41989919+xpivarc@users.noreply.github.com>
1	zhuanlan <zhuanlan_yewu@cmss.chinamobile.com>
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
