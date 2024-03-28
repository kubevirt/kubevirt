KubeVirt v0.24.0
================

This release follows v0.23.0 and consists of 73 changes, contributed by
17 people, leading to 124 files changed, 3303 insertions(+), 617 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.24.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- CI: Support for Kubernetes 1.15
- CI: Support for Kubernetes 1.16
- Add and fix a couple of test cases
- Support for pause and unpausing VMs
- Update of libvirt to 5.6.0
- Fix bug related to parallel scraping of Prometheus endpoints
- Fix to reliably test VNC

Contributors
------------

17 people contributed to this release:

```
        15	Marc Sluiter <msluiter@redhat.com>
         8	Daniel Hiller <daniel.hiller.1972@gmail.com>
         8	Roman Mohr <rmohr@redhat.com>
         5	Francesco Romani <fromani@redhat.com>
         3	Michael Henriksen <mhenriks@redhat.com>
         2	Quique Llorente <ellorent@redhat.com>
         2	ipinto <ipinto@redhat.com>
         1	Andrea Bolognani <abologna@redhat.com>
         1	Artyom Lukianov <alukiano@redhat.com>
         1	Kedar Bidarkar <kbidarka@redhat.com>
         1	Marcin Franczyk <marcin0franczyk@gmail.com>
         1	Pep Turró Mauri <pep@redhat.com>
         1	Sebastian Scheinkman <sscheink@redhat.com>
         1	Vladik Romanovsky <vromanso@redhat.com>
         1	alonSadan <asadan@redhat.com>
         1	yinchengfeng <yinchengfeng@baidu.com>
```

Test Results
------------

```
> Ran 403 of 481 Specs in 12434.458 seconds
> PASS
```

Additional Resources
--------------------

- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]

[git-evtag]: https://github.com/cgwalters/git-evtag#using-git-evtag
[contributing]: https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/master/LICENSE
