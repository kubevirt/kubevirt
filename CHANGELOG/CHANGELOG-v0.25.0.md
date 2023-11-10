KubeVirt v0.25.0
================

This release follows v0.24.0 and consists of 39 changes, contributed by
11 people, leading to 82 files changed, 1989 insertions(+), 434 deletions(-).

The source code and selected binaries are available for download at:
<https://github.com/kubevirt/kubevirt/releases/tag/v0.25.0>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.

Notable changes
---------------

- CI: Support for Kubernetes 1.17
- Support emulator thread pinning
- Support virtctl restart --force
- Support virtctl migrate to trigger live migrations from the CLI

Contributors
------------

11 people contributed to this release:

```
         7	Vatsal Parekh <vparekh@redhat.com>
         7	Vladik Romanovsky <vromanso@redhat.com>
         3	Roman Mohr <rmohr@redhat.com>
         2	Daniel Belenky <dbelenky@redhat.com>
         2	Daniel Hiller <daniel.hiller.1972@gmail.com>
         2	Fabian Deutsch <fabiand@redhat.com>
         1	Arik Hadas <ahadas@redhat.com>
         1	Ihar Hrachyshka <ihrachys@redhat.com>
         1	Michael Henriksen <mhenriks@redhat.com>
         1	Tareq Alayan <talayan@redhat.com>
```

Test Results
------------

```
> Ran 407 of 485 Specs in 12424.177 seconds
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
