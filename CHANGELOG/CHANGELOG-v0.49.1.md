KubeVirt v0.49.1
================

This release follows v0.49.0 and consists of 200 changes, contributed by 28 people, leading to 1929 files changed, 148241 insertions(+), 56732 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.49.1.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.49.1`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #8323][jean-edouard] Improve path handling for non-root virt-launcher workloads
- [PR #8306][kubevirt-bot] Fixed `KubeVirtComponentExceedsRequestedMemory` alert complaining about many-to-many matching not allowed.
- [PR #8109][Barakmor1] Bump the version of emicklei/go-restful from 2.15.0 to 2.16.0
- [PR #7984][ShellyKa13] BugFix: Fix vm restore in case of restore size bigger then PVC requested size
- [PR #7718][akalenyu] BugFix: virtctl guestfs incorrectly assumes image name
- [PR #7617][machadovilaca] Add Virtual Machine name label to virt-launcher pod
- [PR #7610][acardace] Fix failed reported migrations when actually they were successful.
- [PR #7478][orelmisan] Fixed setting custom guest pciAddress and bootOrder parameter(s) to a list of SR-IOV NICs.
- [PR #7514][kubevirt-bot] BugFix: Fixed RBAC for admin/edit user to allow virtualmachine/addvolume and removevolume. This allows for persistent disks
- [PR #7247][kubevirt-bot] New and resized disks are now always 1MiB-aligned
- [PR #7179][kubevirt-bot] Improve device plugin de-registration in virt-handler and some test stabilizations
- [PR #7166][kubevirt-bot] Garbage collect finalized migration objects only leaving the most recent 5 objects
- [PR #7154][davidvossel] Switch from reflects.DeepEquals to equality.Semantic.DeepEquals() across the entire project
- [PR #7146][kubevirt-bot] Updated recording rule "kubevirt_vm_container_free_memory_bytes"
- [PR #7140][kubevirt-bot] Fixes issue associated with blocked uninstalls when VMIs exist during removal
- [PR #7073][kubevirt-bot] When expanding disk images, take the minimum between the request and the capacity - avoid using the full underlying file system on storage like NFS, local.
- [PR #7042][kubevirt-bot] Fix issue with ssh being unreachable on VMIs with Istio proxy
- [PR #7034][kubevirt-bot] Add infoSource field to vmi.status.interfaces.
- [PR #7043][kubevirt-bot] Migrating VMIs that contain dedicated CPUs will now have properly dedicated CPUs on target

Contributors
------------
28 people contributed to this release:

```
16	Orel Misan <omisan@redhat.com>
16	Shelly Kagan <skagan@redhat.com>
15	Roman Mohr <rmohr@google.com>
13	Roman Mohr <rmohr@redhat.com>
12	Jed Lejosne <jed@redhat.com>
12	fossedihelm <ffossemo@redhat.com>
9	David Vossel <dvossel@redhat.com>
6	Michael Henriksen <mhenriks@redhat.com>
5	Daniel Hiller <dhiller@redhat.com>
4	Alex Kalenyuk <akalenyu@redhat.com>
4	L. Pivarc <lpivarc@redhat.com>
4	Omer Yahud <oyahud@redhat.com>
4	bmordeha <bmodeha@redhat.com>
3	Alexander Wels <awels@redhat.com>
3	Or Shoval <oshoval@redhat.com>
2	Antonio Cardace <acardace@redhat.com>
2	Barak Mordehai <bmordeha@redhat.com>
2	Itamar Holder <iholder@redhat.com>
2	João Vilaça <jvilaca@redhat.com>
2	Radim Hrazdil <rhrazdil@redhat.com>
2	Shirly Radco <sradco@redhat.com>
1	Alice Frosi <afrosi@redhat.com>
1	Erkan Erol <eerol@redhat.com>
1	Maya Rashish <mrashish@redhat.com>
1	Sascha Grunert <sgrunert@redhat.com>
1	Vasiliy Ulyanov <vulyanov@suse.de>
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
