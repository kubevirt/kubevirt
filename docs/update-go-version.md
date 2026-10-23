# How to update Go version 
A quick guide to update KubeVirt's Go version.

To update the Go version we need to update the builder image so that it uses the new version,
push it to the registry and finally let KubeVirt use the new builder image.

In addition, [go rules for bazel](https://github.com/bazelbuild/rules_go) have to be updated if the current version does not support the target Go version.

## Updating Go Version
### Updating builder image

* Change `GIMME_GO_VERSION` in [hack/builder/Dockerfile](../hack/builder/Dockerfile) and [hack/builder/Dockerfile.cs10](../hack/builder/Dockerfile.cs10) to the desired Go version.
* Rebuild the builder image by executing `make builder-build`.
  
### Publishing builder image
* Publish new builder image with `make builder-publish`.
  * Note: Proper access rights are required in order to publish builder image.
  * When publish is finished, the builder image tag will be presented. For instance, if the output of `make builder-publish` is:
    ```shell
    2103210933-9be558add-amd64: digest: sha256:cc83534b5d99da35643f8a2a87830b0dabcb4f130c1db181a835dc8def09174b size: 3271
    + TMP_IMAGES=' quay.io/kubevirt/builder:2103210933-9be558add-amd64'
    + export DOCKER_CLI_EXPERIMENTAL=enabled
    + DOCKER_CLI_EXPERIMENTAL=enabled
    + docker manifest create --amend quay.io/kubevirt/builder:2103210933-9be558add quay.io/kubevirt/builder:2103210933-9be558add-amd64
      Created manifest list quay.io/kubevirt/builder:2103210933-9be558add
    + docker manifest push quay.io/kubevirt/builder:2103210933-9be558add
      sha256:d828eb647e7ef3115f39ff4cb2d5d41da39b4134056e429be16c3a019b521957
    + cleanup
    + rm manifests/ -rf
    ```
  * The image tag is `2103210933-9be558add`
* Update `kubevirt_builder_version` (and `kubevirt_cs10_builder_version` for the CentOS Stream 10 builder image) in [hack/dockerized](../hack/dockerized) to the published image tags.
* In [MODULE.bazel](../MODULE.bazel), update `version` in `go_sdk.download(...)` alongside `GIMME_GO_VERSION` to the desired Go version:
  ```starlark
  go_sdk = use_extension("@io_bazel_rules_go//go:extensions.bzl", "go_sdk")
  go_sdk.download(version = "1.26.10")
  ```
* Refresh [MODULE.bazel.lock](../MODULE.bazel.lock) by running `./hack/dockerized bazel mod deps`.

## Update go rules for bazel
* In [MODULE.bazel](../MODULE.bazel), check the current `rules_go` version:
  ```starlark
  bazel_dep(name = "rules_go", version = "0.60.0", repo_name = "io_bazel_rules_go")
  ```
* Visit [Bazel's releases page](https://github.com/bazel-contrib/rules_go/releases) and check whether the current `rules_go` version supports the new Go version.
  * If it is not supported, update `version` in `bazel_dep(name = "rules_go", ...)` to a supporting release on the [Bazel Central Registry](https://registry.bazel.build/modules/rules_go) and run `./hack/dockerized bazel mod deps` to update [MODULE.bazel.lock](../MODULE.bazel.lock).
