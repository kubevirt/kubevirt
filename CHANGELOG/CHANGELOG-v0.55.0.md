KubeVirt v0.55.0
================

This release follows v0.54.0 and consists of 391 changes, contributed by 39 people, leading to 480 files changed, 29803 insertions(+), 6373 deletions(-).
v0.55.0 is a promotion of release candidate v0.55.0-rc.0 which was originally published 2022-07-07
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.55.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.55.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #7336][iholder-redhat] Introduce clone CRD, controller and API
- [PR #7791][iholder-redhat] Introduction of an initial deprecation policy
- [PR #7875][lyarwood] `ControllerRevisions` of any `VirtualMachineFlavorSpec` or `VirtualMachinePreferenceSpec` are stored during the initial start of a `VirtualMachine` and used for subsequent restarts ensuring changes to the original `VirtualMachineFlavor` or `VirtualMachinePreference` do not modify the `VirtualMachine` and the `VirtualMachineInstance` it creates.
- [PR #8011][fossedihelm] Increase virt-launcher memory overhead
- [PR #7963][qinqon] Bump alpine_with_test_tooling
- [PR #7881][ShellyKa13] Enable memory dump to be included in VMSnapshot
- [PR #7926][qinqon] tests: Move main clean function to global AfterEach and create a VM per each infra_test.go Entry.
- [PR #7845][janeczku] Fixed a bug that caused `make generate` to fail when API code comments contain backticks. (#7844, @janeczku)
- [PR #7932][marceloamaral] Addition of kubevirt_vmi_migration_phase_transition_time_from_creation_seconds metric to monitor how long it takes to transition a VMI Migration object to a specific phase from creation time.
- [PR #7879][marceloamaral] Faster VM phase transitions thanks to an increased virt-controller QPS/Burst
- [PR #7807][acardace] make cloud-init 'instance-id' persistent across reboots
- [PR #7928][iholder-redhat] bugfix: node-labeller now removes "host-model-cpu.node.kubevirt.io/" and "host-model-required-features.node.kubevirt.io/" prefixes
- [PR #7841][jean-edouard] Non-root VMs will now migrate to root VMs after a cluster disables non-root.
- [PR #7933][akalenyu] BugFix: Fix vm restore in case of restore size bigger then PVC requested size
- [PR #7919][lyarwood] Device preferences are now applied to any default network interfaces or missing volume disks added to a `VirtualMachineInstance` at runtime.
- [PR #7910][qinqon] tests: Create the expected readiness probe instead of liveness
- [PR #7732][acardace] Prevent virt-handler from starting a migration twice
- [PR #7594][alicefr] Enable to run libguestfs-tools pod to run as noroot user
- [PR #7811][raspbeep] User now gets information about the type of commands which the guest agent does not support.
- [PR #7590][awels] VMExport allows filesystem PVCs to be exported as either disks or directories.
- [PR #7683][alicefr] Add --command and --local-ssh-opts" options to virtctl ssh to execute remote command using local ssh method

Contributors
------------
39 people contributed to this release:

```
38	Itamar Holder <iholder@redhat.com>
20	Alexander Wels <awels@redhat.com>
20	Marcelo Amaral <marcelo.amaral1@ibm.com>
19	Michael Henriksen <mhenriks@redhat.com>
18	Miguel Duarte Barroso <mdbarroso@redhat.com>
16	Shelly Kagan <skagan@redhat.com>
15	Ben Oukhanov <boukhanov@redhat.com>
14	Dan Kenigsberg <danken@redhat.com>
13	Edward Haas <edwardh@redhat.com>
13	Lee Yarwood <lyarwood@redhat.com>
10	Enrique Llorente <ellorent@redhat.com>
8	Alice Frosi <afrosi@redhat.com>
8	Jed Lejosne <jed@redhat.com>
8	bmordeha <bmodeha@redhat.com>
6	Andrej Krejcir <akrejcir@redhat.com>
6	Or Shoval <oshoval@redhat.com>
5	Antonio Cardace <acardace@redhat.com>
5	Janusz Marcinkiewicz <januszm@nvidia.com>
5	Roman Mohr <rmohr@google.com>
4	L. Pivarc <lpivarc@redhat.com>
4	Ram Lavi <ralavi@redhat.com>
3	Pavel Kratochvil <pakratoc@redhat.com>
3	prnaraya <prnaraya@redhat.com>
2	Alex Kalenyuk <akalenyu@redhat.com>
2	Bartosz Rybacki <brybacki@redhat.com>
2	Shirly Radco <sradco@redhat.com>
2	fossedihelm <ffossemo@redhat.com>
1	Arnon Gilboa <agilboa@redhat.com>
1	Daniel Hiller <dhiller@redhat.com>
1	Haibo Xu <haibo1.xu@intel.com>
1	João Vilaça <jvilaca@redhat.com>
1	Karel Šimon <ksimon@redhat.com>
1	Or Mergi <ormergi@redhat.com>
1	Stu Gott <sgott@redhat.com>
1	borod108 <boris.od@gmail.com>
1	janeczku <jabruder@gmail.com>
1	tgfree <tgfree7@gmail.com>
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
