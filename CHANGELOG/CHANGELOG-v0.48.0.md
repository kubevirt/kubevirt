KubeVirt v0.48.0
================

This release follows v0.47.1 and consists of 282 changes, contributed by 43 people, leading to 1046 files changed, 32869 insertions(+), 12807 deletions(-).
v0.48.0 is a promotion of release candidate v0.48.0-rc.0 which was originally published 2021-12-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.48.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.48.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6670][futuretea] Added 'virtctl soft-reboot' command to reboot the VMI.
- [PR #6861][orelmisan] virtctl errors are written to stderr instead of stdout
- [PR #6836][enp0s3] Added PHASE and VMI columns for the 'kubectl get vmim' CLI output
- [PR #6784][nunnatsa] kubevirt-config configMap is no longer supported for KubeVirt configuration
- [PR #6839][ShellyKa13] fix restore of VM with RunStrategy
- [PR #6533][zcahana] Paused VMIs are now marked as unready even when no readinessProbe is specified
- [PR #6858][rmohr] Fix a nil pointer in virtctl in combination with some external auth plugins
- [PR #6780][fossedihelm] Add PatchOptions to the Patch request of the VirtualMachineInstanceInterface
- [PR #6773][iholder-redhat] alert if migration for VMI with host-model CPU is stuck since no node is suitable
- [PR #6714][rhrazdil] Shorten timeout for Istio proxy detection
- [PR #6725][fossedihelm] added DryRun mode to virtcl for pause and unpause commands
- [PR #6737][davidvossel] Pending migration target pods timeout after 5 minutes when unschedulable
- [PR #6814][fossedihelm] Changed some terminology to be more inclusive
- [PR #6649][Barakmor1] Designate the apps.kubevirt.io/component label for KubeVirt components.
- [PR #6650][victortoso] Introduces support to ich9 or ac97 sound devices
- [PR #6734][Barakmor1] replacing the command that extract libvirtd's pid  to avoid this error:
- [PR #6802][rmohr] Maintain a separate api package which synchronizes to kubevirt.io/api for better third party integration with client-gen
- [PR #6730][zhhray] change kubevrit cert secret type from Opaque to kubernetes.io/tls
- [PR #6508][oshoval] Add missing domain to guest search list, in case subdomain is used.
- [PR #6664][vladikr] enable the display and ramfb for vGPUs by default
- [PR #6710][iholder-redhat] virt-launcher fix - stop logging successful shutdown when it isn't true
- [PR #6162][vladikr] KVM_HINTS_REALTIME will always be set when dedicatedCpusPlacement is requested
- [PR #6772][zcahana] Bugfix: revert #6565 which prevented upgrades to v0.47.
- [PR #6722][zcahana] Remove obsolete scheduler.alpha.kubernetes.io/critical-pod annotation
- [PR #6723][acardace] remove stale pdbs created by < 0.41.1 virt-controller
- [PR #6721][iholder-redhat] Set default CPU model in VMI spec, even if not defined in KubevirtCR
- [PR #6713][zcahana] Report WaitingForVolumeBinding VM status when PVC/DV-type volumes reference unbound PVCs
- [PR #6681][fossedihelm] Users can use --dry-run flag
- [PR #6663][jean-edouard] The number of virt-api and virt-controller replicas is now configurable in the CSV
- [PR #5981][maya-r] Always resize disk.img files to the largest size at boot.

Contributors
------------
43 people contributed to this release:

```
27	Itamar Holder <iholder@redhat.com>
18	Zvi Cahana <zvic@il.ibm.com>
10	Maya Rashish <mrashish@redhat.com>
9	David Vossel <dvossel@redhat.com>
8	Barak Mordehai <bmordeha@redhat.com>
8	Orel Misan <omisan@redhat.com>
7	Victor Toso <victortoso@redhat.com>
7	Vladik Romanovsky <vromanso@redhat.com>
7	fossedihelm <fossedihelm@gmail.com>
7	futuretea <Hang.Yu@suse.com>
7	hellocloudnative <200922702@qq.com>
6	Alice Frosi <afrosi@redhat.com>
6	Or Mergi <ormergi@redhat.com>
5	L. Pivarc <lpivarc@redhat.com>
4	Edward Haas <edwardh@redhat.com>
4	Igor Bezukh <ibezukh@redhat.com>
4	Roman Mohr <rmohr@redhat.com>
3	Or Shoval <oshoval@redhat.com>
3	Shelly Kagan <skagan@redhat.com>
3	Stu Gott <sgott@redhat.com>
3	huizhang <huizhang@alauda.io>
2	Daniel Hiller <dhiller@redhat.com>
2	Miguel Duarte Barroso <mdbarroso@redhat.com>
2	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
2	Ryan Hallisey <rhallisey@nvidia.com>
2	Vasiliy Ulyanov <vulyanov@suse.de>
1	Antonio Cardace <acardace@redhat.com>
1	Chris Callegari <mazzystr@gmail.com>
1	Denys Shchedrivyi <dshchedr@redhat.com>
1	Erkan Erol <eerol@redhat.com>
1	Federico Gimenez <fgimenez@redhat.com>
1	Hao Yu <yuh@us.ibm.com>
1	Jed Lejosne <jed@redhat.com>
1	Kedar Bidarkar <kbidarka@redhat.com>
1	Michael Henriksen <mhenriks@redhat.com>
1	Radim Hrazdil <rhrazdil@redhat.com>
1	Tomas Psota <tpsota@redhat.com>
1	Zhe Peng <zpeng@redhat.com>
1	dalia-frank <dafrank@redhat.com>
1	dhiller <dhiller@redhat.com>
1	张鹏璇 <200922702@qq.com>
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
