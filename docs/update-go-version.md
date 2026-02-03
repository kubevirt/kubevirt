# How to update Go version 
A quick guide to update KubeVirt's Go version.

To update the Go version we need to update the builder image so that it uses the new version,
push it to the registry and finally let KubeVirt use the new builder image.

## Updating Go Version
### Updating builder image

* Change the `GIMME_GO_VERSION` in the [hack/builder/Dockerfile](../hack/builder/Dockerfile) to the desired Go version.
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
* Change `kubevirt_builder_version` variable in [hack/dockerized](../hack/dockerized) to the tag from the previous step.
* Update `go.mod` to specify the new Go version.
