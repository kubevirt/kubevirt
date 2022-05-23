# Deprecation Policy

In order for any software to be reliable enough for production use, it needs to have a stable API. This means that
users would have a high confidence that the way they use Kubevirt would not become obsolete, or at least would give
the proper time to prepare if it does.

This document aims to specify a process for several aspects of deprecation revolving Kubevirt. This is intended to
be a "live document" which updates and refines itself over time.

## Process

### Feature Gates
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

## Discussion
An ongoing discussion takes part in the following issue: https://github.com/kubevirt/kubevirt/issues/7745
