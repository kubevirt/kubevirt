KubeVirt v0.52.0
================

This release follows v0.51.0 and consists of 314 changes, contributed by 37 people, leading to 1006 files changed, 35687 insertions(+), 24520 deletions(-).
v0.52.0 is a promotion of release candidate v0.52.0-rc.0 which was originally published 2022-04-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.52.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.52.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #7024][fossedihelm] Add an warning message if the client and server virtctl versions are not aligned
- [PR #7486][rmohr] Move stable.txt location to a more appropriate path
- [PR #7372][saschagrunert] Fixed `KubeVirtComponentExceedsRequestedMemory` alert complaining about many-to-many matching not allowed.
- [PR #7426][iholder-redhat] Add warning for manually determining core-component replica count in Kubevirt CR
- [PR #7424][maiqueb] Provide interface binding types descriptions, which will be featured in the KubeVirt API.
- [PR #7422][orelmisan] Fixed setting custom guest pciAddress and bootOrder parameter(s) to a list of SR-IOV NICs.
- [PR #7421][rmohr] Fix knowhosts file corruption for virtctl ssh
- [PR #6854][rmohr] Make virtctl ssh work with ssh-rsa+ preauthentication
- [PR #7267][iholder-redhat] Applied migration configurations can now be found in VMI's status
- [PR #7321][iholder-redhat] [Migration Policies]: precedence to VMI labels over Namespace labels
- [PR #7326][oshoval] The Ginkgo dependency has been upgraded to v2.1.3 (major version upgrade)
- [PR #7361][SeanKnight] Fixed a bug that prevents virtctl from working with clusters accessed via Rancher authentication proxy, or any other cluster where the server URL contains a path component. (#3760)
- [PR #7255][tyleraharrison] Users are now able to specify `--address [ip_address]` when using `virtctl vnc` rather than only using 127.0.0.1
- [PR #7275][enp0s3] Add observedGeneration to virt-operator to have a race-free way to detect KubeVirt config rollouts
- [PR #7233][xpivarc] Bug fix: Successfully aborted migrations should be reported now
- [PR #7158][AlonaKaplan] Add masquerade VMs support to single stack IPv6.
- [PR #7227][rmohr] Remove VMI informer from virt-api to improve scaling characteristics of virt-api
- [PR #7288][raspbeep] Users now don't need to specify container for `kubectl logs <vmi-pod>` and `kubectl exec <vmi-pod>`.
- [PR #6709][xpivarc] Workloads will be migrated to nonroot implementation if NonRoot feature gate is set. (Except VirtioFS)
- [PR #7241][lyarwood] Fixed a bug that prevents only a unattend.xml configmap or secret being provided as contents for a sysprep disk. (#7240, @lyarwood)

Contributors
------------
37 people contributed to this release:

```
40	Itamar Holder <iholder@redhat.com>
30	Dan Kenigsberg <danken@redhat.com>
26	Or Shoval <oshoval@redhat.com>
17	Roman Mohr <rmohr@redhat.com>
13	L. Pivarc <lpivarc@redhat.com>
9	Alona Kaplan <alkaplan@redhat.com>
8	Edward Haas <edwardh@redhat.com>
8	Jed Lejosne <jed@redhat.com>
5	fossedihelm <ffossemo@redhat.com>
4	Antonio Cardace <acardace@redhat.com>
4	Miguel Duarte Barroso <mdbarroso@redhat.com>
4	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
3	Ben Ukhanov <ben1zuk321@gmail.com>
3	Karel Šimon <ksimon@redhat.com>
3	Zhuchen Wang <zcwang@google.com>
3	tyleraharrison <tyleraharrison@gmail.com>
2	Daniel Hiller <dhiller@redhat.com>
2	Igor Bezukh <ibezukh@redhat.com>
2	Zvi Cahana <zcahana@gmail.com>
1	Alex Kalenyuk <akalenyu@redhat.com>
1	Bartosz Rybacki <brybacki@redhat.com>
1	Felix Matouschek <fmatouschek@redhat.com>
1	Frank Yang <poan.yang@suse.com>
1	Lee Yarwood <lyarwood@redhat.com>
1	Marcelo Amaral <marcelo.amaral1@ibm.com>
1	Maya Rashish <mrashish@redhat.com>
1	Orel Misan <omisan@redhat.com>
1	Pavel Kratochvil <pakratoc@redhat.com>
1	Radim Hrazdil <rhrazdil@redhat.com>
1	RamLavi <ralavi@redhat.com>
1	Sascha Grunert <sgrunert@redhat.com>
1	Sean Knight <git@seanknight.com>
1	Shirly Radco <sradco@redhat.com>
1	assaf-admi <aadmi@redhat.com>
1	fossedihelm <fossedihelm@gmail.com>
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
