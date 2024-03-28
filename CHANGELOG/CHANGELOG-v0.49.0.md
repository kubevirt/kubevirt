KubeVirt v0.49.0
================

This release follows v0.48.1 and consists of 298 changes, contributed by 36 people, leading to 652 files changed, 53600 insertions(+), 8784 deletions(-).
v0.49.0 is a promotion of release candidate v0.49.0-rc.0 which was originally published 2022-01-04
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.49.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.49.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #7004][iholder-redhat] Bugfix: Avoid setting block migration for volumes used by read-only disks
- [PR #6959][enp0s3] generate event when target pod enters unschedulable phase
- [PR #6888][assafad] Added common labels into alert definitions
- [PR #6166][vasiliy-ul] Experimental support of AMD SEV
- [PR #6980][vasiliy-ul] Updated the dependencies to include the fix for CVE-2021-43565 (KubeVirt is not affected)
- [PR #6944][iholder-redhat] Remove disabling TLS configuration from Live Migration Policies
- [PR #6800][jean-edouard] CPU pinning doesn't require hardware-assisted virtualization anymore
- [PR #6501][ShellyKa13] Use virtctl image-upload to upload archive content
- [PR #6918][iholder-redhat] Bug fix: Unscheduable host-model VMI alert is now properly triggered
- [PR #6796][Barakmor1] 'kubevirt-operator' changed to 'virt-operator' on 'managed-by' label in kubevirt's components made by virt-operator
- [PR #6036][jean-edouard] Migrations can now be done over a dedicated multus network
- [PR #6933][erkanerol] Add a new lane for monitoring tests
- [PR #6949][jean-edouard] KubeVirt components should now be successfully removed on CR deletion, even when using only 1 replica for virt-api and virt-controller
- [PR #6954][maiqueb] Update the `virtctl` exposed services `IPFamilyPolicyType` default to `IPFamilyPolicyPreferDualStack`
- [PR #6931][fossedihelm] added DryRun to AddVolumeOptions and RemoveVolumeOptions
- [PR #6379][nunnatsa] Fix issue https://bugzilla.redhat.com/show_bug.cgi?id=1945593
- [PR #6399][iholder-redhat] Introduce live migration policies that allow system-admins to have fine-grained control over migration configuration for different sets of VMs.
- [PR #6880][iholder-redhat] Add full Podman support for `make` and `make test`
- [PR #6702][acardace] implement virt-handler canary upgrade and rollback for faster and safer rollouts
- [PR #6717][davidvossel] Introducing the VirtualMachinePools feature for managing stateful VMs at scale
- [PR #6698][rthallisey] Add tracing to the virt-controller work queue
- [PR #6762][fossedihelm] added DryRun mode to virtcl to migrate command
- [PR #6891][rmohr] Fix "Make raw terminal failed: The handle is invalid?" issue with "virtctl console" when not executed in a pty
- [PR #6783][rmohr] Skip SSH RSA auth if no RSA key was explicitly provided and not key exists at the default location

Contributors
------------
36 people contributed to this release:

```
55	Itamar Holder <iholder@redhat.com>
23	Edward Haas <edwardh@redhat.com>
21	Vasiliy Ulyanov <vulyanov@suse.de>
19	David Vossel <dvossel@redhat.com>
12	Jed Lejosne <jed@redhat.com>
9	prnaraya <prnaraya@redhat.com>
7	Barak Mordehai <bmordeha@redhat.com>
7	Ryan Hallisey <rhallisey@nvidia.com>
6	Antonio Cardace <acardace@redhat.com>
6	Vladik Romanovsky <vromanso@redhat.com>
5	fossedihelm <fossedihelm@gmail.com>
4	Daniel Hiller <dhiller@redhat.com>
4	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
3	Maya Rashish <mrashish@redhat.com>
3	Or Mergi <ormergi@redhat.com>
3	Roman Mohr <rmohr@redhat.com>
2	Igor Bezukh <ibezukh@redhat.com>
2	Kedar Bidarkar <kbidarka@redhat.com>
2	Or Shoval <oshoval@redhat.com>
2	Zhe Peng <zpeng@redhat.com>
2	Zvi Cahana <zvic@il.ibm.com>
1	Alex Kalenyuk <akalenyu@redhat.com>
1	Ashley Schuett <aschuett@redhat.com>
1	Dan Kenigsberg <danken@redhat.com>
1	Diana Teplits <dteplits@redhat.com>
1	Erkan Erol <eerol@redhat.com>
1	Helene Durand <helene@kubermatic.com>
1	L. Pivarc <lpivarc@redhat.com>
1	Michael Henriksen <mhenriks@redhat.com>
1	Miguel Duarte Barroso <mdbarroso@redhat.com>
1	Orel Misan <omisan@redhat.com>
1	Petr Horáček <phoracek@redhat.com>
1	Shelly Kagan <skagan@redhat.com>
1	assaf-admi <aadmi@redhat.com>
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
