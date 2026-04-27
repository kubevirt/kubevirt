# Deprecation Policy

In order for any software to be reliable enough for production use, it needs to have a stable API. This means that
users would have a high confidence that the way they use Kubevirt would not become obsolete, or at least would give
the proper time to prepare if it does.

This document aims to specify a process for several aspects of deprecation revolving Kubevirt. This is intended to
be a "live document" which updates and refines itself over time.

## Process

### Feature Gates
#### Introduction
A feature gate should be introduced for every new feature, capability, configuration or any other change that
is not yet stable and can jeopardize the cluster.

A feature gate serves as:
* A way to warn the user about a possible danger that could be caused by an unstable logic.
* A way for the users to explicitly state that they would like to take the risk and enable the feature.
* A way for the user to explicitly confirm that some feature / HW is available on the cluster.

Over time the new features would stabilize and graduate their API versions (i.e. from alpha, to beta to stable). When
that happens, and therefore the risk of jeopardizing the user decreases significantly, the feature should be enabled
by default and the feature gate should be deprecated (see section below).

#### Lifecycle

A feature gate progresses through the following states:

| State | Default | Description |
|-------|---------|-------------|
| **Alpha** | Disabled | Feature is under experimentation. Can be enabled explicitly through the feature gate. |
| **Beta** | Disabled | Feature is under evaluation. Can be enabled explicitly through the feature gate. |
| **GA** | Enabled | Feature has reached General Availability. The feature gate is a no-op and can be safely removed from the KubeVirt CR. A warning is presented if the feature gate is still listed. |
| **Deprecated** | Disabled | Feature gate is going to be discontinued in the following release. A warning is presented if the feature gate is still listed. |
| **Discontinued** | Disabled | Feature gate has been removed and has no effect. A warning is presented if the feature gate is still listed, directing users to any replacement. |

#### Deprecation
A feature gate should be deprecated for one of the following reasons:
* When a feature stabilizes and there is no need to guard it by a feature gate.
* When a feature (that's guarded with a feature gate) did not show as useful, therefore is deprecated.

In order to deprecate a feature gate the following conditions must be met:
* All the features that used to be guarded by it need to be enabled by default.
  * The above point means that all of the features that were guarded by this feature gate
    would be enabled without the need to enable this feature gate. In other words, the
    feature gate needs to turn into a no-op.
* When the feature gate is enabled (by creating / updating Kubevit CR), a warning should be presented to the user. The
  best fit to trigger the warning is probably our validating webhooks.
* A mail should be sent to kubevirt-dev mailing list (kubevirt-dev@googlegroups.com) to inform about the deprecation.

#### Discontinuation
In the release following the deprecation, the feature gate should be moved to the **Discontinued** state:
* Any remaining functional code that checked the feature gate should be removed.
* The feature gate registration should remain so that users who still have it configured receive a warning directing
  them to any replacement.

## Discussion
An ongoing discussion takes part in the following issue: https://github.com/kubevirt/kubevirt/issues/7745
