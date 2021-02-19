How to update the golang version
================================

To update the go version we need to update the builder image so that it uses the new version.
We are using [gimme go] for fetching the go version.

- update image tag in [hack/builder/version.sh](../hack/builder/version.sh)

  (note: we don't really have some procedure to determine the new version, but we at least sync the version to the used libvirt version IIRC)
- change the `GIMME_GO` version in the [hack/builder/Dockerfile](../hack/builder/Dockerfile)
- rebuild the image

  `make builder-build`
- change the image sha in [hack/dockerized](../hack/dockerized)
- push the image

  `make builder-push`

  (note: for this you of course need push access rights)
- update WORKSPACE with new [go rules for bazel](https://github.com/bazelbuild/rules_go/releases)
- upload new dependencies to dependency mirror

  https://github.com/kubevirt/project-infra/blob/master/plugins/cmd/uploader/README.md

[gimme go]: https://github.com/travis-ci/gimme
