KubeVirt v1.0.1
===============

This release follows v1.0.0 and consists of 188 changes, contributed by 31 people, leading to 226 files changed, 4540 insertions(+), 7509 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v1.0.1.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v1.0.1`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #10554][kubevirt-bot] fix embed version info of virt-operator
- [PR #10519][kubevirt-bot] A new `instancetype.kubevirt.io:view` `ClusterRole` has been introduced that can be bound to users via a `ClusterRoleBinding` to provide read only access to the cluster scoped `VirtualMachineCluster{Instancetype,Preference}` resources.
- [PR #10493][fossedihelm] Add a Feature Gate to KV CR to automatically set memory limits when a resource quota with memory limits is associated to the creation namespace
- [PR #10433][iholder101] Stop considering nodes without `kubevirt.io/schedulable` label when finding lowest TSC frequency on the cluster
- [PR #10402][kubevirt-bot] BugFix: VMExport now works in a namespace with quotas defined.
- [PR #10397][kubevirt-bot] Bugfix: Allow image-upload to recover from PendingPopulation phase
- [PR #10273][machadovilaca] Change kubevirt_vmi_*_usage_seconds from Gauges to Counters
- [PR #10292][kubevirt-bot] Ensure new hotplug attachment pod is ready before deleting old attachment pod
- [PR #10266][machadovilaca] Remove affinities label from kubevirt_vmi_cpu_affinity and use sum as value
- [PR #10205][AlonaKaplan] hotplug interface bug fix- default interface won't disappear from a hotplugged VM after restart
- [PR #10153][kubevirt-bot] `ControllerRevisions` containing `instancetype.kubevirt.io` `CRDs` are now decorated with labels detailing specific metadata of the underlying stashed object
- [PR #10207][kubevirt-bot] Restrict coordination/lease RBAC permissions to install namespace
- [PR #10195][kubevirt-bot] Deprecate `spec.config.machineType` in KubeVirt CR.
- [PR #10162][kubevirt-bot] Add boot-menu wait time when starting the VM as paused.
- [PR #10191][kubevirt-bot] Use auth API for DataVolumes, stop importing kubevirt.io/containerized-data-importer
- [PR #10193][kubevirt-bot] Bugfix: target virt-launcher pod hangs when migration is cancelled.
- [PR #10176][kubevirt-bot] BugFix: deleting hotplug attachment pod will no longer detach volumes that were not removed.
- [PR #10143][ormergi] Existing detached interfaces with 'absent' state will be cleared from VMI spec.
- [PR #10068][kubevirt-bot] Add perf scale benchmarks for VMIs
- [PR #10051][kubevirt-bot] Fix kubevirt_vmi_phase_count not being created
- [PR #10037][kubevirt-bot] The VM controller now replicates spec interfaces MAC addresses to the corresponding interfaces in the VMI spec.

Contributors
------------
31 people contributed to this release:

```
14	Vasiliy Ulyanov <vulyanov@suse.de>
11	Or Mergi <ormergi@redhat.com>
10	Lee Yarwood <lyarwood@redhat.com>
10	fossedihelm <ffossemo@redhat.com>
9	Alexander Wels <awels@redhat.com>
7	Antonio Cardace <acardace@redhat.com>
5	Alex Kalenyuk <akalenyu@redhat.com>
5	Itamar Holder <iholder@redhat.com>
4	Edward Haas <edwardh@redhat.com>
4	João Vilaça <jvilaca@redhat.com>
4	enp0s3 <ibezukh@redhat.com>
3	Alay Patel <alayp@nvidia.com>
3	Luboslav Pivarc <lpivarc@redhat.com>
3	Pavel Tishkov <pavel.tishkov@flant.com>
2	Alice Frosi <afrosi@redhat.com>
2	Alona Paz <alkaplan@redhat.com>
2	Andrej Krejcir <akrejcir@redhat.com>
2	Arnon Gilboa <agilboa@redhat.com>
2	Jed Lejosne <jed@redhat.com>
2	rokkiter <101091030+rokkiter@users.noreply.github.com>
1	Alay Patel <alay1431@gmail.com>
1	Alvaro Romero <alromero@redhat.com>
1	Assaf Admi <aadmi@redhat.com>
1	Felix Matouschek <fmatouschek@redhat.com>
1	Reficul <xuzhenglun@gmail.com>
1	Roman Mohr <rmohr@google.com>
1	Shelly Kagan <skagan@redhat.com>
1	bmordeha <bmordeha@redhat.com>
1	grass-lu <284555125@qq.com>
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
