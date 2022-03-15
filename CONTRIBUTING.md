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

Please try to be compliant with [Kubernetes api conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md) or, case by case, explicitly justify why doing that is not possible or not reasonable.  

## Expectations from a PR

The items below must be checked per PR by the author and reviewers. Authors are responsible for stating the status of PR and reviewers are responsible for checking/verifying them. 

- ***PR Message:*** PR message must explain the purpose of the PR clearly. If it fixes a bug, describe the bug or mention the github issue/bugzilla id.
If it adds a new feature, explain why we add it and how to use it.
It is totally OK to copy/paste documentation into PR message.

- ***Commit Messages:*** Commits in a PR are squashed and merged into the target branch as a single commit by kubevirt-bot. The message of that single commit consists of PR title and messages of all squashed commits. Therefore, commit messages must be clear, concise and comprehensive.


- ***How to test:***: If automated tests are not applicable for the PR, explain how to test the functionality in the PR. 
  
- ***Unit Tests:*** Production code under `pkg` folder must have unit test. PRs should not decrease the test coverage. 

- ***Functional Tests:*** New features must have functional tests under `tests/func-tests`. 
  
- ***User Documentation:*** If the PR adds/changes something which affect end users, user documents (e.g. /docs/api.md ) must be updated.
  
- ***Developer Documentation:*** If the PR adds/changes something which affect developers of this repo, developer documents (e.g. /docs/run-locally.md ) must be updated.
  
- ***Upgrade Scenario:*** Upgrade scenario must be handled/tested in the PR. For example, if the PR adds new labels to an operand, it must be handled that existing operands during upgrades are labelled as well. If the PR removes an operand from desired state, it must handle removal of the operand during upgrade as well. 
  
- ***Uninstallation Scenario:*** Uninstallation scenario must be handled/tested in the PR. 
  
- ***Backward Compatibility:*** The APIs/interfaces we provide via this repository can be used by anyone, and we don't want to break our consumers. Changes in PRs must be backward compatible. Otherwise, it must state explicitly why we have to do it and how the community agreed on it.
  
- ***Troubleshooting Friendly:*** When there is a failure, the root cause must be able to be pinpointed quickly. Error mechanism must provide enough information (e.g. logs, events etc. ).

<br>

> If one of the checks above is not applicable for a PR, the author must put "N/A" in the PR description like below. 
> - [ ] Upgrade Scenario -> N/A


### Add new Feature Gate

Think twice before you do. Feature gates make HCO very hard to test; each of them essentially duplicates our test
matrix. They also complicate life for the human operator, who has to read the documentation to understand the
implications of pressing a knob. You should add new featureGate only if hard-coding it to true considerably harms our
typical users. Think if you can just add the new feature gate in a hard code manner to the requested operator.

If there is a real need for a new feature gate, please follow these steps:

1. In the PR message, describe the new feature gate, what it does and why it needed to be added to the HCO API.
1. Add the new feature gate to the HyperConvergedFeatureGates struct
   in [api/v1beta1/hyperconverged_types.go](api/v1beta1/hyperconverged_types.go)
    - make sure the name of the feature gate field is as the feature gate field in the target operand, including casing.
      It also must start with a capital letter, to be exposed from the api package.
    - Set the field type to `bool`.
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
    WithHostPassthroughCPU bool `json:"withHostPassthroughCPU,omitempty"`
    ```

1. Run `make generate` to trigger automatic code generation (`deepcopy-gen` and  `openapi-gen`)
1. Add a set of unit tests
   in [api/v1beta1/hyperconverged_types_test.go](api/v1beta1/hyperconverged_types_test.go)
   to check this new function.
1. Add the new feature gate to the relevant operator handler. Currently, this is only supported for KubeVirt. For
   KubeVirt, do the following:
   In [controllers/operands/kubevirt.go](controllers/operands/kubevirt.go)
    - Add a constant for the feature gate name in the constant block marked with
      the `// KubeVirt feature gates that are exposed in HCO API`
      comment.
    - Add the new feature gate and the new IsEnabled function to the map in the `getFeatureGateChecks` function.
1. Rebuild the manifests:
    ```shell
    make build-manifests
    ```
1. If you are specifying a default value, please add a functional test for it in hack/check_defaults.sh (it's a bash script and not a golang code to be sure that the default mechanism is properly working at user eyes regardless of any client-go implementation).
