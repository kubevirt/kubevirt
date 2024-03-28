KubeVirt v0.32.0
================

This release follows v0.31.0 and consists of 189 changes, contributed by 26 people, leading to 460 files changed, 17395 insertions(+), 19058 deletions(-).
v0.32.0 is a promotion of release candidate v0.32.0-rc.2 which was originally published 2020-08-10
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.32.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.32.0`.

Pre-built containers are published on Docker Hub and can be viewed at: <https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- [PR #3921][vladikr] use correct memory units in libvirt xml
- [PR #3893][davidvossel] Adds recurring period that resyncs virt-launcher domains with virt-handler
- [PR #3880][sgarbour] Better error message when input parameters are not the expected number of parameters for each argument. Help menu will popup in case the number of parameters is incorrect.
- [PR #3785][xpivarc] Vcpu wait metrics available
- [PR #3642][vatsalparekh] Add a way to update VMI Status with latest Pod IP for Masquerade bindings
- [PR #3636][ArthurSens] Adds kubernetes metadata.labels as VMI metrics' label
- [PR #3825][awels] Virtctl now prints error messages from the response body on upload errors.
- [PR #3830][davidvossel] Fixes re-establishing domain notify client connections when domain notify server restarts due to an error event.
- [PR #3778][danielBelenky] Do not emit a SyncFailed event if we fail to sync a VMI in a final state
- [PR #3803][andreabolognani] Not sure what to write here (see above)
- [PR #2694][rmohr] Use native go libraries for selinux to not rely on python-selinux tools like semanage, which are not always present.
- [PR #3692][victortoso] QEMU logs can now be fetched from outside the pod
- [PR #3738][enp0s3] Restrict creation of VMI if it has labels that are used internally by Kubevirt components.
- [PR #3725][danielBelenky] The tests binary is now part of the release and can be consumed from the GitHub release page.
- [PR #3684][rmohr] Log if critical devices, like kvm, which virt-handler wants to expose are not present on the node.
- [PR #3166][petrkotas] Introduce new virtctl commands:
- [PR #3708][andreabolognani] Make qemu work on GCE by pulling in a fix for https://bugzilla.redhat.com/show_bug.cgi?id=1822682

Contributors
------------
26 people contributed to this release:

```
19	arthursens <arthursens2005@gmail.com>
14	Igor Bezukh <ibezukh@redhat.com>
11	Or Shoval <oshoval@redhat.com>
11	Roman Mohr <rmohr@redhat.com>
9	David Vossel <dvossel@redhat.com>
8	Jed Lejosne <jed@redhat.com>
8	Or Mergi <ormergi@redhat.com>
7	Daniel Belenky <dbelenky@redhat.com>
7	Edward Haas <edwardh@redhat.com>
5	Andrea Bolognani <abologna@redhat.com>
5	L. Pivarc <lpivarc@redhat.com>
3	Ashley Schuett <ashleyns1992@gmail.com>
2	Daniel Hiller <daniel.hiller.1972@gmail.com>
2	Kedar Bidarkar <kbidarka@redhat.com>
2	Maya Rashish <mrashish@redhat.com>
2	Shaul Garbourg <sgarbour@redhat.com>
2	Victor Toso <victortoso@redhat.com>
1	Alexander Wels <awels@redhat.com>
1	Alona Kaplan <alkaplan@redhat.com>
1	Petr Kotas <pkotas@redhat.com>
1	Vatsal Parekh <vparekh@redhat.com>
1	Vladik Romanovsky <vromanso@redhat.com>
1	alonSadan <asadan@redhat.com>
1	rmohr <rmohr@redhat.com>
```

Additional Resources
--------------------

- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]

[contributing]: https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/master/LICENSE
