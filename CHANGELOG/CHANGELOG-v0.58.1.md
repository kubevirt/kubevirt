KubeVirt v0.58.1
================

This release follows v0.58.0 and consists of 213 changes, contributed by 26 people, leading to 397 files changed, 8616 insertions(+), 3933 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.58.1.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.58.1`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #9203][jean-edouard] Most VMIs now run under the SELinux type container_t
- [PR #9191][kubevirt-bot] Default RBAC for clone and export
- [PR #9150][kubevirt-bot] Fix access to portforwarding on VMs/VMIs with the cluster roles kubevirt.io:admin and kubevirt.io:edit
- [PR #9128][kubevirt-bot] Rename migration metrics removing 'total' keyword
- [PR #9034][akalenyu] BugFix: Hotplug pods have hardcoded resource req which don't comply with LimitRange maxLimitRequestRatio of 1
- [PR #9002][iholder101] Bugfix: virt-handler socket leak
- [PR #8907][kubevirt-bot] Bugfix: use virt operator image if provided
- [PR #8784][kubevirt-bot] Use exponential backoff for failing migrations
- [PR #8816][iholder101] Expose new custom components env vars to csv-generator, manifest-templator and gs
- [PR #8798][iholder101] Fix: Align Reenlightenment flows between converter.go and template.go
- [PR #8731][kubevirt-bot] Allow specifying custom images for core components
- [PR #8785][0xFelix] The expand-spec subresource endpoint was renamed to expand-vm-spec and made namespaced
- [PR #8806][kubevirt-bot] Consider the ParallelOutboundMigrationsPerNode when evicting VMs
- [PR #8738][machadovilaca] Use collector to set migration metrics
- [PR #8747][kubevirt-bot] Add alerts for VMs unhealthy states
- [PR #8685][kubevirt-bot] BugFix: Exporter pod does not comply with restricted PSA
- [PR #8647][akalenyu] BugFix: Add an option to specify a TTL for VMExport objects
- [PR #8609][kubevirt-bot] Fix permission denied on on selinux relabeling on some kernel versions
- [PR #8578][rhrazdil] When using Passt binding, virl-launcher has unprivileged_port_start set to 0, so that passt may bind to all ports.

Contributors
------------
26 people contributed to this release:

```
42	Itamar Holder <iholder@redhat.com>
14	Felix Matouschek <fmatouschek@redhat.com>
12	Marcelo Tosatti <mtosatti@redhat.com>
11	bmordeha <bmodeha@redhat.com>
10	Alex Kalenyuk <akalenyu@redhat.com>
10	Jordi Gil <jgil@redhat.com>
8	João Vilaça <jvilaca@redhat.com>
7	Lee Yarwood <lyarwood@redhat.com>
5	Alexander Wels <awels@redhat.com>
3	Alvaro Romero <alromero@redhat.com>
3	Antonio Cardace <acardace@redhat.com>
3	Jed Lejosne <jed@redhat.com>
3	Shelly Kagan <skagan@redhat.com>
3	fossedihelm <ffossemo@redhat.com>
3	prnaraya <prnaraya@redhat.com>
2	L. Pivarc <lpivarc@redhat.com>
2	Radim Hrazdil <rhrazdil@redhat.com>
2	Ram Lavi <ralavi@redhat.com>
2	Roman Mohr <rmohr@google.com>
2	enp0s3 <ibezukh@redhat.com>
1	Arnon Gilboa <agilboa@redhat.com>
1	Brian Carey <bcarey@redhat.com>
1	Edward Haas <edwardh@redhat.com>
1	Igor Bezukh <ibezukh@redhat.com>
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
