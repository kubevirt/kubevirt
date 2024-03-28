KubeVirt v0.46.0
================

This release follows v0.45.0 and consists of 256 changes, contributed by 35 people, leading to 732 files changed, 31209 insertions(+), 20471 deletions(-).
v0.46.0 is a promotion of release candidate v0.46.0-rc.0 which was originally published 2021-10-01
The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.46.0.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.46.0`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #6425][awels] Hotplug disks are possible when iothreads are enabled.
- [PR #6297][acardace] mutate migration PDBs instead of creating an additional one for the duration of the migration.
- [PR #6464][awels] BugFix: Fixed hotplug race between kubelet and virt-handler when virt-launcher dies unexpectedly.
- [PR #6465][salanki] Fix corrupted DHCP Gateway Option from local DHCP server, leading to rejected IP configuration on Windows VMs.
- [PR #6458][vladikr] Tagged SR-IOV interfaces will now appear in the config drive metadata
- [PR #6446][brybacki] Access mode for virtctl image upload is now optional. This version of virtctl now requires CDI v1.34 or greater
- [PR #6391][zcahana] Cleanup obsolete permissions from virt-operator's ClusterRole
- [PR #6419][rthallisey] Fix virt-controller panic caused by lots of deleted VMI events
- [PR #5972][kwiesmueller] Add a `ssh` command to `virtctl` that can be used to open SSH sessions to VMs/VMIs.
- [PR #6403][jrife] Removed go module pinning to an old version (v0.3.0) of github.com/go-kit/kit
- [PR #6367][brybacki] virtctl imageupload now uses DataVolume.spec.storage
- [PR #6198][iholder-redhat] Fire a Prometheus alert when a lot of REST failures are detected in virt-api
- [PR #6211][davidvossel] cluster-profiler pprof gathering tool and corresponding "ClusterProfiler" feature gate
- [PR #6323][vladikr] switch live migration to use unix sockets
- [PR #6374][vladikr] Fix the default setting of CPU requests on vmipods
- [PR #6283][rthallisey] Record the time it takes to delete a VMI and expose it as a metric
- [PR #6251][rmohr] Better place vcpu threads on host cpus to form more efficient passthrough architectures
- [PR #6377][rmohr] Don't fail on failed selinux relabel attempts if selinux is permissive
- [PR #6308][awels] BugFix: hotplug was broken when using it with a hostpath volume that was on a separate device.
- [PR #6186][davidvossel] Add resource and verb labels to rest_client_requests_total metric

Contributors
------------
35 people contributed to this release:

```
31	David Vossel <dvossel@redhat.com>
30	Roman Mohr <rmohr@redhat.com>
10	Maya Rashish <mrashish@redhat.com>
10	Vladik Romanovsky <vromanso@redhat.com>
8	Ryan Hallisey <rhallisey@nvidia.com>
7	Jed Lejosne <jed@redhat.com>
7	Vasiliy Ulyanov <vulyanov@suse.de>
6	Dan Kenigsberg <danken@redhat.com>
5	Alexander Wels <awels@redhat.com>
5	Bartosz Rybacki <brybacki@redhat.com>
5	Federico Gimenez <fgimenez@redhat.com>
5	Itamar Holder <iholder@redhat.com>
5	L. Pivarc <lpivarc@redhat.com>
4	Antonio Cardace <acardace@redhat.com>
3	Igor Bezukh <ibezukh@redhat.com>
3	Kevin Wiesmueller <kwiesmul@redhat.com>
3	Miguel Duarte Barroso <mdbarroso@redhat.com>
3	Or Shoval <oshoval@redhat.com>
2	Daniel Hiller <dhiller@redhat.com>
1	Adam Litke <alitke@redhat.com>
1	Alice Frosi <afrosi@redhat.com>
1	Chris Callegari <mazzystr@gmail.com>
1	Denis Ollier <dollierp@redhat.com>
1	Erkan Erol <erkanerol92@gmail.com>
1	Howard Zhang <howard.zhang@arm.com>
1	Jordan Rife <jrife@google.com>
1	Kedar Bidarkar <kbidarka@redhat.com>
1	Marcelo Carneiro do Amaral <marcelo.amaral1@ibm.com>
1	Or Mergi <ormergi@redhat.com>
1	Peter Salanki <peter@salanki.st>
1	Shelly Kagan <skagan@redhat.com>
1	Vatsal Parekh <vparekh@redhat.com>
1	Zvi Cahana <zvic@il.ibm.com>
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
