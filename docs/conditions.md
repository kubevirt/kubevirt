# CR Conditions and Readiness Probe
Conditions are..
	   _the latest available observations of an object's state. They are
	   an extension mechanism intended to be used when the details of an
	   observation are not a priori known or would not apply to all
	   instances of a given Kind._

Kubernetes conditions [documentation](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status).

The HCO’s CR is a representation of the all the underlying component operators'
state.  In theory, if the HCO's CR exists, then all component CRs _should_
exist, and all applications _should_ exist.  If the object doesn’t exist, then
all component CRs _should not_ exist, and all applications _should not_ exist.
However, the CR existence can only can tell us if the application should exist and
doesn't help us observe the application's health. This is where the HCO and
component operators will use conditions on their CRs to reflect the health of
the underlying application.  Component operators store conditions that are
watched by the HCO and the HCO will store conditions that reflect the
[worst state](https://github.com/kubevirt/hyperconverged-cluster-operator/blob/main/docs/conditions.md#hco-conditions) of all component operator conditions.

## Outlook Model
There's a long running [discussion](https://github.com/kubernetes/kubernetes/issues/7856) in the Kubernetes
community about the use of Phases, Conditions, and using controllers as
state machines that hasn't been resolved.  This design document follows the
"outlook" approach described in this [comment](https://github.com/kubernetes/kubernetes/issues/7856#issuecomment-99667941), which
we'll use until the Kubernetes community has a resolution we can adopt.

## Condition Struct
We can use some of the CVO's [conditions](https://github.com/openshift/api/blob/b1bcdbc/config/v1/types_cluster_operator.go#L123-L134) to standardize across components.

Here's how the Condition struct will look...

```go
type ApplicationStatusCondition struct {
   // type specifies the state of the operator's reconciliation functionality,
   // which reflects the state of the application
   Type ConditionType `json:"type"`

   // status of the condition, one either True or False.
   Status ConditionStatus `json:"status"`

   // lastTransitionTime is the time of the last update to the current status object.
   LastTransitionTime metav1.Time `json:"lastTransitionTime"`

   // reason is the reason for the condition's last transition.  Reasons are CamelCase
   Reason string `json:"reason,omitempty"`

   // message provides additional information about the current condition.
   // This is only to be consumed by humans.
   Message string `json:"message,omitempty"`
}
```

## Library
We're going to use a shared library to provide the condition types to operators.
This will ensure the code is not specific to any operator and it will allow
products outside of CNV to also consume it.

https://github.com/openshift/custom-resource-status

## ConditionType
`ConditionType` _specifies the state of the operator's reconciliation functionality,
which reflects the state of the application_. `ConditionType`s use `ConditionStatus`
to report state.  The `ConditionStatus`es we will use are either `True` or `False`.
The `ConditionStatus` object can also be `Unknown`, but only the HCO will use
`Unknown` because it's not clear what `Unknown` means in terms of an application's
lifecycle.  The HCO can assume `Unknown` for conditions, while operators are starting up.

#### ApplicationAvailable
```
	ApplicationAvailable ClusterStatusConditionType = "Available"
```
ApplicationAvailable indicates that the binary maintained by the operator
(eg: openshift-apiserver for the openshift-apiserver-operator), is functional
and available in the cluster.

#### OperatorProgressing
```
	OperatorProgressing ClusterStatusConditionType = "Progressing"
```
Progressing indicates that the operator is actively making changes to the binary
maintained by the operator (eg: openshift-apiserver for the
openshift-apiserver-operator).

#### ApplicationDegraded
```
	ApplicationDegraded ClusterStatusConditionType = "Degraded"
```
Degraded indicates that the application is not functioning completely.
An example of a degraded state would be if there should be 5 copies of the
component running but only 4 are running. It may still be available, but it is
degraded.

#### Condition Matrix

| Condition        | Status           | Status  | Status  |
| :------------- |:-------------:|:-----:|:-----:|
| ApplicationAvailable | True | True | True |
| OperatorProgressing | False | True | True |
| ApplicationDegraded | False | False | True |
| Meaning | Application is 100% healthy and the Operator is idle | Application is functional but, either upgrading or healing | Application is functioning below capacity and an upgrade or heal is in progress |

| Condition        | Status           | Status  |
| :------------- |:-------------:|:-----:|
| ApplicationAvailable | False | False |
| OperatorProgressing | False | True |
| ApplicationDegraded | True | True |
| Meaning | Application and/or operator are in a failed state that requires human intervention.  Failed upgrade or failed heal | Application is in a failed state and an operator is healing |

| Condition        | Status           |
| :------------- |:-------------:|
| ApplicationAvailable | False |
| OperatorProgressing | True |
| ApplicationDegraded | False |
| Meaning | Operator is deploying the application |

## Readiness Probe
With a standardized set of conditions, the HCO should report the health of the
overall application back to OLM and the user.  This will be critial for sensitive
operations like upgrade, because OLM needs to know it shouldn't replace an
operator when it is in the middle of important work.

See this [issue](https://github.com/operator-framework/operator-lifecycle-manager/issues/922) for why we only want to report a readiness probe on the HCO
instead of on all component operators.

## Reason
`Reason` is _a one-word CamelCase reason for the condition's last transition_.

Components will be responsible for reporting `Reason`, which will explain their
condition.

## Message
`Message` is a _human-readable message indicating details about last transition_.

Explain why your CR has `Reason`.

## HCO conditions
It's important to point out that the `ConditionTypes` on the HCO don't represent
the condition the HCO is in, rather the condition of the component CRs.

The HCO will use the same `ConditionType`s and `Reason`s, but it will be the
only operator to use `Unknown` for `Status`.  If the HCO notices that a component
CR is missing condition fields, the HCO will assume the status of
`ApplicationAvailable = false`, `OperatorProgressing = unknown` and
`ApplicationDegraded = unknown`.  If the HCO also detects there is no `Reason` value
too, then it will assume `Reason = "InstallInvalid"`.

The HCO's `ConditionType`s will always represent the _worst_ `Status` and the
corresponding `Reason`.

The _worst_ `Status` for each `ConditionType`:

| Condition   | Status  |
| :------------- |:-------------:|
| ApplicationAvailable | False |
| OperatorProgressing | True |
| ApplicationDegraded | True |
