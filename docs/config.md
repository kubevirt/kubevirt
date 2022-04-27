# Configuration
The philosophy we will use when exposing HCO configuration options is that only
options that all component operators consume can be exposed on the default HCO
CR.  Configuration options that are for specific component operators, will be
hidden but accepted as valid configuration.

## 1.0 Config Options
All component operators must express each of the HCO `Config Options` on their CR.
The component variable:
  - does __not__ have to be the same name as the HCO variable
  - can be expressed by multiple granular variables
The HCO is responsible for mapping a "top level" variable to the component
operator's variable that express the same description.

For example, KubeVirt exposes on its CR `imageTag`.  The variable `imageTag` is
the version of KubeVirt that will be deployed. The mapping for the HCO `Version`
to KubeVirt is:
```
Version -> imageTag
```

#### Proposed configuration
| Config Options | Description |
|----|----|
| Version  |  Product Version |
| ContainerRegistry | Link to Registry in the form of <registry/org> |
| ImagePullPolicy | Always, IfNotPreset, Never  |

Exposed variables are expressed in the HCO [CSV file](https://github.com/kubevirt/hyperconverged-cluster-operator/blob/dfdd4ac492c1d91c130eec03af1e4f6b04d54c7e/deploy/converged/olm-catalog/kubevirt-hyperconverged/0.0.1/kubevirt-hyperconverged-operator.v0.0.1.clusterserviceversion.yaml#L18-L20)
```yaml
          "spec": {
            "Version":"2.0"
            "ContainerRegistry":"quay.io/kubevirt"
            "ImagePullPolicy":"IfNotPresent",
          }
```

## 1.X Configuration Plan
For the long term, the HCO can offer more granular configuration for component
operators.  Each component operator CR spec can be vendored in and configuration
can be expressed at the top level.  This will be handy when consuming the HCO
upstream.

## Adding or Removing configuration
Everytime we add or remove a config object, it changes the API which introduces
potential breakage for upgrades or downgrades. In order to mitigate this, we can
roll out configuration changes slowly over multiple releases.

#### Beta
Adding configuration options
 - Configuration option is available from all component operators
 - Option is off by default
 - Option is hidden on the HCO CR by default

Removing configuration options
 - Configuration option is available from all component operators
 - Validation warns that config option is deprecated
 - Option is hidden on the HCO CR by default

#### GA
Adding configuration options
 - Configuration option is available from all relevant component operators
 - Option is __on__ by default
 - Option __may__ be visible on the HCO CR if all operators use it

Removing configuration options
 - Configuration option is __removed__ from all component operators
 - Validation __prevents__ CR being created with config option
