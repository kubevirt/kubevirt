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

After the above deprecation process, the feature gate should remain as a no-op with a warning popping up.

### Metadata (e.g. labels / annotations)
Kubevirt adds certain labels / annotations to both objects that are created by us (e.g. VMs) and cluster-wide
objects (e.g. Nodes). This metadata is exposed to the end-user which might use it and depend on it.

Usually, metadata needs to be deprecated if it has a non-intuitive / irrelevant name or reflects data that is no
longer relevant / useful.

#### Deprecation
In order to deprecate metadata the following conditions must be met:
* A mail should be sent to kubevirt-dev mailing list (kubevirt-dev@googlegroups.com) to inform about the deprecation.
* In case the metadata is being renamed / changed, the new metadata (e.g. label) should be added alongside the deprecated
one.

For example, if we want to deprecate a label named `kubevirt.io/badNonIntuitiveName` to
`kubevirt.io/greatNewName`, both labels should appear.

#### Removal
Deprecated metadata will not be removed for at least 3 release cycles, which means roughly 1 year.

Afterwards, a mail should be sent to kubevirt-dev mailing list (kubevirt-dev@googlegroups.com) to inform
about the removal. If there are no objections, the metadata can be removed. Otherwise, a new removal date
can be discussed according to the context.

### API
Although the following only mentions API objects, it's also relevant for their APIs (e.g. object fields),
Kubevirt CR configuration, etc. Obviously, for fields / configs deprecation the step about adding a
deprecated annotations can be skipped. However, a warning should always be triggered for any usage of all
of the above mentioned.

#### Deprecation
In order to deprecate an API object the following actions need to be taken:
* The deprecated object needs to contain a warning about the object being deprecated. The warning should also
  state whenever the object would be removed (see below).
* A mail should be sent to kubevirt-dev mailing list (kubevirt-dev@googlegroups.com) to inform about the deprecation.

#### Removal
A deprecated object will not be removed for at least 5 release cycles, which means roughly 1.5 years.

Afterwards, a mail should be sent to kubevirt-dev mailing list (kubevirt-dev@googlegroups.com) to inform
about the removal. If there are no objections, the object can be removed. Otherwise, a new removal date
can be discussed according to the context.

### Feature Behavior
This refers to any changes in features behavior that might surprise the user or break backwards compatibility.

As an example, let's say that a certain feature can be enabled via Kubevirt CR that affects only worker nodes,
and that feature is changed to also affect control-plane nodes. This is considered a feature behavior change.
Internal details that are not exposed to the end-user as guarantees that are part of our API aren't considered
as feature behavior changes.

The process is identical to the process of deprecating API (see above), except the number of releases before
the final behavior change should be 3 releases (roughly 1 year).


## Discussion
An ongoing discussion takes part in the following issue: https://github.com/kubevirt/kubevirt/issues/7745
