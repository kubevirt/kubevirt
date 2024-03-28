KubeVirt v0.58.0
================

This release follows v0.57.1 and consists of 285 changes, contributed by 37 people, leading to 471 files changed, 26960 insertions(+), 6441 deletions(-).
v0.58.0 is a promotion of release candidate v0.58.0-rc.0 which was originally published 2022-10-03
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.58.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.58.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #8578][rhrazdil] When using Passt binding, virl-launcher has unprivileged_port_start set to 0, so that passt may bind to all ports.
- [PR #8463][Barakmor1] Improve metrics documentation
- [PR #8282][akrejcir] Improves instancetype and preference controller revisions. This is a backwards incompatible change and introduces a new v1alpha2 api for instancetype and preferences.
- [PR #8272][jean-edouard] No more empty section in the kubevirt-cr manifest
- [PR #8536][qinqon] Don't show a failure if ConfigDrive cloud init has UserDataSecretRef and not NetworkDataSecretRef
- [PR #8375][xpivarc] Virtiofs can be used with Nonroot feature gate
- [PR #8465][rmohr] Add a vnc screenshot REST endpoint and a "virtctl vnc screenshot" command for UI and script integration
- [PR #8418][alromeros] Enable automatic token generation for VirtualMachineExport objects
- [PR #8488][0xFelix] virtctl: Be less verbose when using the local ssh client
- [PR #8396][alicefr] Add group flag for setting the gid and fsgroup in guestfs
- [PR #8476][iholder-redhat] Allow setting virt-operator log verbosity through Kubevirt CR
- [PR #8366][rthallisey] Move KubeVirt to a 15 week release cadence
- [PR #8479][arnongilboa] Enable DataVolume GC by default in cluster-deploy
- [PR #8474][vasiliy-ul] Fixed migration failure of VMs with containerdisks on systems with containerd
- [PR #8316][ShellyKa13] Fix possible race when deleting unready vmsnapshot and the vm remaining frozen
- [PR #8436][xpivarc] Kubevirt is able to run with restricted Pod Security Standard enabled with an automatic escalation of namespace privileges.
- [PR #8197][alromeros] Add vmexport command to virtctl
- [PR #8252][fossedihelm] Add `tlsConfiguration` to Kubevirt Configuration
- [PR #8431][rmohr] Fix shadow status updates and periodic status updates on VMs, performed by the snapshot controller
- [PR #8359][iholder-redhat] [Bugfix]: HyperV Reenlightenment VMIs should be able to start when TSC Frequency is not exposed
- [PR #8330][jean-edouard] Important: If you use docker with SELinux enabled, set the `DockerSELinuxMCSWorkaround` feature gate before upgrading
- [PR #8401][machadovilaca] Rename metrics to follow the naming convention

Contributors
------------
37 people contributed to this release:

```
20	Alvaro Romero <alromero@redhat.com>
14	L. Pivarc <lpivarc@redhat.com>
14	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
14	Shelly Kagan <skagan@redhat.com>
13	Andrej Krejcir <akrejcir@redhat.com>
13	Roman Mohr <rmohr@google.com>
11	Lee Yarwood <lyarwood@redhat.com>
11	Miguel Duarte Barroso <mdbarroso@redhat.com>
10	Felix Matouschek <fmatouschek@redhat.com>
9	Itamar Holder <iholder@redhat.com>
8	fossedihelm <ffossemo@redhat.com>
7	Alice Frosi <afrosi@redhat.com>
5	Brian Carey <bcarey@redhat.com>
5	Vasiliy Ulyanov <vulyanov@suse.de>
4	Alex Kalenyuk <akalenyu@redhat.com>
4	Jed Lejosne <jed@redhat.com>
4	Ram Lavi <ralavi@redhat.com>
3	Fabian Deutsch <fabiand@redhat.com>
3	Radim Hrazdil <rhrazdil@redhat.com>
2	Bartosz Rybacki <brybacki@redhat.com>
2	Igor Bezukh <ibezukh@redhat.com>
2	Michael Henriksen <mhenriks@redhat.com>
2	Ryan Hallisey <rhallisey@nvidia.com>
1	Alexander Wels <awels@redhat.com>
1	Andrea Bolognani <abologna@redhat.com>
1	Arnon Gilboa <agilboa@redhat.com>
1	Christopher Desiniotis <cdesiniotis@nvidia.com>
1	Enrique Llorente <ellorent@redhat.com>
1	Javier Cano Cano <jcanocan@redhat.com>
1	João Vilaça <jvilaca@redhat.com>
1	Maya Rashish <mrashish@redhat.com>
1	Prashanth Dintyala <vdintyala@nvidia.com>
1	Xiaodong Ye <yeahdongcn@gmail.com>
1	assaf-admi <aadmi@redhat.com>
1	bmordeha <bmodeha@redhat.com>
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
