KubeVirt v0.56.0
================

This release follows v0.55.1 and consists of 324 changes, contributed by 38 people, leading to 970 files changed, 18998 insertions(+), 11069 deletions(-).
v0.56.0 is a promotion of release candidate v0.56.0-rc.1 which was originally published 2022-08-17
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.56.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.56.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #7599][iholder-redhat] Introduce a mechanism to abort non-running migrations - fixes "Unable to cancel live-migration if virt-launcher pod in pending state" bug
- [PR #8027][alaypatel07] Wait deletion to succeed all the way till objects are finalized in perfscale tests
- [PR #8198][rmohr] Improve path handling for non-root virt-launcher workloads
- [PR #8136][iholder-redhat] Fix cgroups unit tests: mock out underlying runc cgroup manager
- [PR #8047][iholder-redhat] Deprecate live migration feature gate
- [PR #7986][iholder-redhat] [Bug-fix]: Windows VM with WSL2 guest fails to migrate
- [PR #7814][machadovilaca] Add VMI filesystem usage metrics
- [PR #7849][AlonaKaplan] [TECH PREVIEW] Introducing passt - a new approach to user-mode networking for virtual machines
- [PR #7991][ShellyKa13] Virtctl memory dump with create flag to create a new pvc
- [PR #8039][lyarwood] The flavor API and associated CRDs of `VirtualMachine{Flavor,ClusterFlavor}` are renamed to instancetype and `VirtualMachine{Instancetype,ClusterInstancetype}`.
- [PR #8112][AlonaKaplan] Changing the default of `virtctl expose` `ip-family` parameter to be empty value instead of IPv4.
- [PR #8073][orenc1] Bump runc to v1.1.2
- [PR #8092][Barakmor1] Bump the version of emicklei/go-restful from 2.15.0 to 2.16.0
- [PR #8053][alromeros] [Bug-fix]: Fix mechanism to fetch fs overhead when CDI resource has a different name
- [PR #8035][0xFelix] Add option to wrap local scp client to scp command
- [PR #7981][lyarwood] Conflicts will now be raised when using flavors if the `VirtualMachine` defines any `CPU` or `Memory` resource requests.
- [PR #8068][awels] Set cache mode to match regular disks on hotplugged disks.

Contributors
------------
38 people contributed to this release:

```
23	Itamar Holder <iholder@redhat.com>
22	Alona Paz <alkaplan@redhat.com>
20	Miguel Duarte Barroso <mdbarroso@redhat.com>
19	Roman Mohr <rmohr@google.com>
16	Nahshon Unna-Tsameret <nunnatsa@redhat.com>
12	Dan Kenigsberg <danken@redhat.com>
11	Edward Haas <edwardh@redhat.com>
11	Felix Matouschek <fmatouschek@redhat.com>
11	Michael Henriksen <mhenriks@redhat.com>
11	Shelly Kagan <skagan@redhat.com>
9	Igor Bezukh <ibezukh@redhat.com>
7	Lee Yarwood <lyarwood@redhat.com>
6	Alexander Wels <awels@redhat.com>
5	Andrej Krejcir <akrejcir@redhat.com>
5	L. Pivarc <lpivarc@redhat.com>
5	bmordeha <bmodeha@redhat.com>
4	Alay Patel <alayp@nvidia.com>
4	Bartosz Rybacki <brybacki@redhat.com>
3	Alvaro Romero <alromero@redhat.com>
3	João Vilaça <jvilaca@redhat.com>
3	Or Shoval <oshoval@redhat.com>
3	Vasiliy Ulyanov <vulyanov@suse.de>
2	Alice Frosi <afrosi@redhat.com>
2	Jed Lejosne <jed@redhat.com>
2	Maya Rashish <mrashish@redhat.com>
2	orenc1 <ocohen@redhat.com>
1	Alona Paz <alkaplan@alkaplan.tlv.csb>
1	Andrei Kvapil <kvapss@gmail.com>
1	Brian Carey <bcarey@redhat.com>
1	Enrique Llorente <ellorent@redhat.com>
1	Karel Šimon <ksimon@redhat.com>
1	Radim Hrazdil <rhrazdil@redhat.com>
1	Ram Lavi <ralavi@redhat.com>
1	Ryan Hallisey <rhallisey@nvidia.com>
1	Vladimir Markelov <vmatroskin@gmail.com>
1	fossedihelm <ffossemo@redhat.com>
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
