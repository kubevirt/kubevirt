KubeVirt v0.57.0
================

This release follows v0.56.0 and consists of 253 changes, contributed by 50 people, leading to 382 files changed, 28741 insertions(+), 4384 deletions(-).
v0.57.0 is a promotion of release candidate v0.57.0-rc.0 which was originally published 2022-09-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.57.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.57.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #8129][mlhnono68] Fixes virtctl to support connection to clusters proxied by RANCHER or having special paths
- [PR #8337][0xFelix] virtctl's native SSH client is now useable in the Windows console without workarounds
- [PR #8257][awels] VirtualMachineExport now supports VM export source type.
- [PR #8367][vladikr] fix the guest memory conversion by setting it to resources.requests.memory when guest memory is not explicitly provided
- [PR #7990][ormergi] Deprecate SR-IOV live migration feature gate.
- [PR #8069][lyarwood] The VirtualMachineInstancePreset resource has been deprecated ahead of removal in a future release. Users should instead use the VirtualMachineInstancetype and VirtualMachinePreference resources to encapsulate any shared resource or preferences characteristics shared by their VirtualMachines.
- [PR #8326][0xFelix] virtctl: Do not log wrapped ssh command by default
- [PR #8325][rhrazdil] Enable route_localnet sysctl option for masquerade binding at virt-handler
- [PR #8159][acardace] Add support for USB disks
- [PR #8006][lyarwood] `AutoattachInputDevice` has been added to `Devices` allowing an `Input` device to be automatically attached to a `VirtualMachine` on start up.  `PreferredAutoattachInputDevice` has also been added to `DevicePreferences` allowing users to control this behaviour with a set of preferences.
- [PR #8134][arnongilboa] Support DataVolume garbage collection
- [PR #8157][StefanKro] TrilioVault for Kubernetes now supports KubeVirt for backup and recovery.
- [PR #8273][alaypatel07] add server-side validations for spec.topologySpreadConstraints during object creation
- [PR #8049][alicefr] Set RunAsNonRoot as default for the guestfs pod
- [PR #8107][awels] Allow VirtualMachineSnapshot as a VirtualMachineExport source
- [PR #7846][janeczku] Added support for configuring topology spread constraints for virtual machines.
- [PR #8215][alaypatel07] support validation for spec.affinity fields during vmi creation
- [PR #8071][oshoval] Relax networkInterfaceMultiqueue semantics: multi queue will configure only what it can (virtio interfaces).
- [PR #7549][akrejcir] Added new API subresources to expand instancetype and preference.

Contributors
------------
50 people contributed to this release:

```
19	Alexander Wels <awels@redhat.com>
16	Miguel Duarte Barroso <mdbarroso@redhat.com>
10	Edward Haas <edwardh@redhat.com>
10	Ram Lavi <ralavi@redhat.com>
8	Alex Kalenyuk <akalenyu@redhat.com>
8	Lee Yarwood <lyarwood@redhat.com>
8	bmordeha <bmodeha@redhat.com>
7	L. Pivarc <lpivarc@redhat.com>
7	Shelly Kagan <skagan@redhat.com>
5	Alay Patel <alayp@nvidia.com>
4	Alice Frosi <afrosi@redhat.com>
4	Andrej Krejcir <akrejcir@redhat.com>
3	David Aghaian <16483722+daghaian@users.noreply.github.com>
3	Igor Bezukh <ibezukh@redhat.com>
3	Jed Lejosne <jed@redhat.com>
3	Michael Henriksen <mhenriks@redhat.com>
3	Or Shoval <oshoval@redhat.com>
2	Antonio Cardace <acardace@redhat.com>
2	Arnon Gilboa <agilboa@redhat.com>
2	Diana Teplits <dteplits@redhat.com>
2	Felix Matouschek <fmatouschek@redhat.com>
2	Howard Zhang <howard.zhang@arm.com>
2	Maya Rashish <mrashish@redhat.com>
2	Prashanth Dintyala <vdintyala@nvidia.com>
2	Radim Hrazdil <rhrazdil@redhat.com>
2	Shirly Radco <sradco@redhat.com>
2	daghaian <16483722+daghaian@users.noreply.github.com>
1	Abirdcfly <fp544037857@gmail.com>
1	Alona Paz <alkaplan@redhat.com>
1	Andrea Bolognani <abologna@redhat.com>
1	Arnaud Aubert <aaubert@magesi.com>
1	Dan Kenigsberg <danken@redhat.com>
1	HF <crazytaxii666@gmail.com>
1	Itamar Holder <iholder@redhat.com>
1	João Vilaça <jvilaca@redhat.com>
1	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
1	Or Mergi <ormergi@redhat.com>
1	Roman Mohr <rmohr@google.com>
1	Roman Mohr <rmohr@redhat.com>
1	Stefan Kroll <stefan.kroll@trilio.io>
1	Stu Gott <sgott@redhat.com>
1	Vasiliy Ulyanov <vulyanov@suse.de>
1	Vladik Romanovsky <vromanso@redhat.com>
1	Ygal Blum <ygal.blum@gmail.com>
1	assaf-admi <aadmi@redhat.com>
1	crazytaxii <hua.feng@99cloud.net>
1	howard zhang <howard.zhang@arm.com>
1	janeczku <jabruder@gmail.com>
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
