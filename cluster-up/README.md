# How to use cluster-up

This directory provides a wrapper around gocli. It can be vendored into other
git repos and integrated to provide in the kubevirt well-known cluster commands
like `make cluster-up` and `make cluster-down`.

In order to properly use it, one has to vendor this folder from a git tag,
which can be found on the github release page.

Then, before calling one of the make targets, the environment variable
`KUBEVIRTCI_TAG` must be exported and set to the tag which was used to vendor
kubevirtci. It allow the content to find the right `gocli` version.

```
export KUBEVIRTCI_TAG=`curl -L -Ss https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest`
```

Find more kubevirtci tags at https://quay.io/repository/kubevirtci/gocli?tab=tags.
