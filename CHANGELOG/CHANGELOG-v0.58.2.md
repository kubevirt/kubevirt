KubeVirt v0.58.2
================

This release follows v0.58.1 and consists of 67 changes, contributed by 19 people, leading to 84 files changed, 2006 insertions(+), 406 deletions(-).

The source code and selected binaries are available for download at: https://github.com/kubevirt/kubevirt/releases/tag/v0.58.2.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using `git tag -v v0.58.2`.

Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.

Notable changes
---------------

- [PR #9817][jcanocan] virt-controller: fix out-of-bound slice index bug in evacuation controller.
- [PR #9699][kubevirt-bot] The install strategy job will respect the infra node placement from now on
- [PR #9697][kubevirt-bot] fixes the requests/limits CPU number mismatch for VMs with isolatedEmulatorThread
- [PR #9661][kubevirt-bot] TSC-enabled VMs can now migrate to a node with a non-identical (but close-enough) frequency
- [PR #9546][xpivarc] Bug fix: DNS integration continues to work after migration
- [PR #9522][fossedihelm] Use ECDSA instead of RSA for key generation
- [PR #9416][kubevirt-bot] Fix vmrestore with WFFC snapshotable storage class
- [PR #9363][iholder101] Add guest-to-request memory headroom ratio.
- [PR #9230][ShellyKa13] Fix addvolume not rejecting adding existing volume source, fix removevolume allowing to remove non hotpluggable volume

Contributors
------------
19 people contributed to this release:

```
6	Itamar Holder <iholder@redhat.com>
5	Shelly Kagan <skagan@redhat.com>
4	Jed Lejosne <jed@redhat.com>
4	Vladik Romanovsky <vromanso@redhat.com>
3	Vasiliy Ulyanov <vulyanov@suse.de>
3	enp0s3 <ibezukh@redhat.com>
3	fossedihelm <ffossemo@redhat.com>
2	Alex Kalenyuk <akalenyu@redhat.com>
2	L. Pivarc <lpivarc@redhat.com>
2	Marcelo Tosatti <mtosatti@redhat.com>
1	Alexander Wels <awels@redhat.com>
1	Arnon Gilboa <agilboa@redhat.com>
1	Luboslav Pivarc <lpivarc@redhat.com>
1	Maya Rashish <mrashish@redhat.com>
1	Orel Misan <omisan@redhat.com>
1	Roman Mohr <rmohr@google.com>
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
