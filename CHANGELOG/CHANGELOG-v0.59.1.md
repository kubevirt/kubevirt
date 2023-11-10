KubeVirt v0.59.1
================

This release follows v0.59.0 and consists of 106 changes, contributed by 22 people, leading to 175 files changed, 5961 insertions(+), 2222 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.59.1.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.59.1`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #9825][kubevirt-bot] BugFix: allow multiple NFS disks to be used/hotplugged
- [PR #9743][kubevirt-bot] virtctl supports retrieving vm manifest for VM export
- [PR #9660][kubevirt-bot] TSC-enabled VMs can now migrate to a node with a non-identical (but close-enough) frequency
- [PR #9581][kubevirt-bot] BugFix: virtualmachineclusterinstancetypes/preferences show up for get all -n <namespace>
- [PR #9500][kubevirt-bot] Requests to update the target `Name` of a `{Instancetype,Preference}Matcher` without also updating the `RevisionName` are now rejected.
- [PR #9507][kubevirt-bot] Bug fix: Fixes case when migration is not retried if the migration Pod gets denied.
- [PR #9413][kubevirt-bot] Default RBAC for clone and export
- [PR #9408][kubevirt-bot] Fix vmrestore with WFFC snapshotable storage class
- [PR #9380][kubevirt-bot] Bug fix: DNS integration continues to work after migration
- [PR #9362][iholder101] Add guest-to-request memory headroom ratio.
- [PR #9145][awels] Show VirtualMachine name in the VMExport status
- [PR #9345][kubevirt-bot] Use ECDSA instead of RSA for key generation
- [PR #9343][kubevirt-bot] externally created mediated devices will not be deleted by virt-handler

Contributors
------------
22 people contributed to this release:

```
7	Alexander Wels <awels@redhat.com>
6	Itamar Holder <iholder@redhat.com>
6	L. Pivarc <lpivarc@redhat.com>
6	Lee Yarwood <lyarwood@redhat.com>
6	fossedihelm <ffossemo@redhat.com>
5	enp0s3 <ibezukh@redhat.com>
4	Alex Kalenyuk <akalenyu@redhat.com>
3	Alice Frosi <afrosi@redhat.com>
3	Jed Lejosne <jed@redhat.com>
3	Vasiliy Ulyanov <vulyanov@suse.de>
3	Vladik Romanovsky <vromanso@redhat.com>
2	Antonio Cardace <acardace@redhat.com>
2	bmordeha <bmodeha@redhat.com>
1	Alvaro Romero <alromero@redhat.com>
1	Brian Carey <bcarey@redhat.com>
1	Luboslav Pivarc <lpivarc@redhat.com>
1	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
1	Orel Misan <omisan@redhat.com>
1	Shelly Kagan <skagan@redhat.com>
1	prnaraya <prnaraya@redhat.com>
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
