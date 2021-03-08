# Contributing to Hyperconverged Cluster Operator

## ***This document is a work in progress***

## Contributing to the HyperConverged API

The Hyperconverged Cluster Operator represents an opinionated deployment of KubeVirt. Its purpose is to deploy KubeVirt
and accompanying projects with good defaults and hard-coded values, so they work together well for most people, in a testable and reproducible
manner.

This means that the API of HCO should be kept simple. HCO should do everything right out of the box, so it is easy to
test and deploy. Sometimes, however, HCO cannot guess what is the right thing to do. On these rare occasions, a
configurable is exposed in its Resource. Each configurable must be documented, so it is clear for a human operator when
it should be used, and why the correct value cannot be guessed automatically.

### Add new Feature Gate

Think twice before you do. Feature gates make HCO very hard to test; each of them essentially duplicates our test
matrix. They also complicate life for the human operator, who has to read the documentation to understand the
implications of pressing a knob. You should add new featureGate only if hard-coding it to true considerably harms our
typical users. Think if you can just add the new feature gate in a hard code manner to the requested operator.

If there is a real need for a new feature gate, please follow these steps:

1. In the PR message, describe the new feature gate, what it does and why it needed to be added to the HCO API.
1. Add the new feature gate to the HyperConvergedFeatureGates struct
   in [pkg/apis/hco/v1beta1/hyperconverged_types.go](pkg/apis/hco/v1beta1/hyperconverged_types.go)
    - make sure the name of the feature gate field is as the feature gate field in the target operand, including casing.
      It also must start with a capital letter, to be exposed from the api package.
    - Set the field type to `FeatureGate`.
    - Make sure the json name in the json tag is valid (e.g. starts with a small cap).
    - add open-api annotations:
        - add detailed description in the comment
        - default annotation
        - optional annotation
for example:
    ```golang
	// Allow migrating a virtual machine with CPU host-passthrough mode. This should be
    // enabled only when the Cluster is homogeneous from CPU HW perspective doc here
    // +optional
    // +kubebuilder:default=false
    WithHostPassthroughCPU FeatureGate `json:"withHostPassthroughCPU,omitempty"`
    ```

1. Add `IsEnabled` method for the new feature gate. It should be something like this:
   ```golang
   func (fgs *HyperConvergedFeatureGates) IsWithHostPassthroughCPUEnabled() bool {
      return (fgs != nil) && (fgs.WithHostPassthroughCPU != nil) && (*fgs.WithHostPassthroughCPU)
   }
   ```
1. Run openapi-gen code generation (the GOPATH below is an example. use the right value for your settings)
    ```shell
    GOPATH=~/go GO111MODULE=auto openapi-gen --output-file-base zz_generated.openapi --input-dirs="github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1" --output-package github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1
    ```
1. Run deepcopy code generation (the GOPATH below is an example. use the right value for your settings):
    ```shell
    GOPATH=~/go GO111MODULE=auto deepcopy-gen --input-dirs="github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1" --output-file-base zz_generated.deepcopy ~/go/src/github.com/kubevirt/hyperconverged-cluster-operator
    ```
1. Add a set of unit tests
   in [pkg/apis/hco/v1beta1/hyperconverged_types_test.go](pkg/apis/hco/v1beta1/hyperconverged_types_test.go)
   to check this new function.
1. Add the new feature gate to the relevant operator handler. Currently, this is only supported for KubeVirt. For
   KubeVirt, do the following:
   In [pkg/controller/operands/kubevirt.go](pkg/controller/operands/kubevirt.go)
    - Add a constant for the feature gate name in the constant block marked with
      the `// KubeVirt feature gates that are exposed in HCO API`
      comment.
    - Add the new feature gate and the new IsEnabled function to the map in the `getFeatureGateChecks` function.
1. Rebuild the manifests:
    ```shell
    make build-manifests
    ```
    