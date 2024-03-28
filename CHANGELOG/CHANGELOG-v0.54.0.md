KubeVirt v0.54.0
================

This release follows v0.53.1 and consists of 223 changes, contributed by 38 people, leading to 215 files changed, 15237 insertions(+), 1800 deletions(-).
v0.54.0 is a promotion of release candidate v0.54.0-rc.0 which was originally published 2022-06-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.54.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.54.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #7757][orenc1] new alert for excessive number of VMI migrations in a period of time.
- [PR #7517][ShellyKa13] Add virtctl Memory Dump command
- [PR #7801][VirrageS] Empty (`nil` values) of `Address` and `Driver` fields in XML will be omitted.
- [PR #7475][raspbeep] Adds the reason of a live-migration failure to a recorded event in case EvictionStrategy is set but live-migration is blocked due to its limitations.
- [PR #7739][fossedihelm] Allow `virtualmachines/migrate` subresource to admin/edit users
- [PR #7618][lyarwood] The requirement to define a `Disk` or `Filesystem` for each `Volume` associated with a `VirtualMachine` has been removed. Any `Volumes` without a `Disk` or `Filesystem` defined will have a `Disk` defined within the `VirtualMachineInstance` at runtime.
- [PR #7529][xpivarc] NoReadyVirtController and NoReadyVirtOperator should be properly fired.
- [PR #7465][machadovilaca] Add metrics for migrations and respective phases
- [PR #7592][akalenyu] BugFix: virtctl guestfs incorrectly assumes image name

Contributors
------------
38 people contributed to this release:

```
27	Lee Yarwood <lyarwood@redhat.com>
16	Jed Lejosne <jed@redhat.com>
15	Shelly Kagan <skagan@redhat.com>
12	Miguel Duarte Barroso <mdbarroso@redhat.com>
9	bmordeha <bmodeha@redhat.com>
8	Andrea Bolognani <abologna@redhat.com>
7	Janusz Marcinkiewicz <januszm@nvidia.com>
6	L. Pivarc <lpivarc@redhat.com>
5	Vasiliy Ulyanov <vulyanov@suse.de>
4	Dan Kenigsberg <danken@redhat.com>
4	Edward Haas <edwardh@redhat.com>
4	Or Shoval <oshoval@redhat.com>
3	Alex Kalenyuk <akalenyu@redhat.com>
3	Itamar Holder <iholder@redhat.com>
2	Alice Frosi <afrosi@redhat.com>
2	Andrey Odarenko <andreyo@il.ibm.com>
2	Daniel Hiller <dhiller@redhat.com>
2	Fabian Deutsch <fabiand@redhat.com>
2	Igor Bezukh <ibezukh@redhat.com>
2	Marcelo Amaral <marcelo.amaral1@ibm.com>
2	akriti gupta <akrgupta@redhat.com>
2	fossedihelm <ffossemo@redhat.com>
1	Andrej Krejcir <akrejcir@redhat.com>
1	Ben Oukhanov <boukhanov@redhat.com>
1	Diana Teplits <dteplits@redhat.com>
1	Howard Zhang <howard.zhang@arm.com>
1	João Vilaça <jvilaca@redhat.com>
1	Joël Séguillon <joel.seguillon@gmail.com>
1	Karel Šimon <ksimon@redhat.com>
1	Nik Paushkin <63355212+NikPaushkin@users.noreply.github.com>
1	Pavel Kratochvil <pakratoc@redhat.com>
1	Petr Horáček <phoracek@redhat.com>
1	Ram Lavi <ralavi@redhat.com>
1	Ryan Hallisey <rhallisey@nvidia.com>
1	borod108 <boris.od@gmail.com>
1	orenc1 <ocohen@redhat.com>
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
