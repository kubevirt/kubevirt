# Kubernetes Version Compatibility

Kubernetes supports 3 minor release concurrently. This means if the latest
Kubernetes release is v1.27, that v1.26 and v1.25 will remain supported for
backport fixes but that v1.24 will lose support.

Similarly, each KubeVirt release maintains compatibility with the latest 3
Kubernetes releases that are out at the time the KubeVirt release is made.

See the [KubeVirt to Kubernetes version support matrix](https://github.com/kubevirt/sig-release/blob/main/releases/k8s-support-matrix.md) to see the currently supported versions of KubeVirt and their associated Kubernetes versions.

# Compatibility Matrix Examples.

Here are some theoretical examples (note that the versions are made up to
illustrate the example)

## New KubeVirt Release

KubeVirt release v1.0 is cut. At that point in time the latest Kubernetes
version is v1.27. This means KubeVirt v1.0 will forever be compatible with
Kubernetes v1.27, v1.26, and v1.25.

## KubeVirt Main

KubeVirt main always follows the latest 3 Kubernetes releases. If a new
Kubernetes v1.28 release is cut, that means support for v1.25 will be dropped
for KubeVirt main.

Note that this support for the latest Kubernetes releases doesn't happen
immediately. There is a period of time, usually a few weeks, where the new
version of Kubernetes must be integrated into KubeVirt's CI. The goal here
is to integrate the latest Kubernetes version into CI before the next
KubeVirt release is cut.

## Old KubeVirt Release

KubeVirt main supports Kubernetes releases v1.28, v1.27, and v1.26. However, the
KubeVirt v1.0 release was cut when the latest Kubernetes release was v1.27.

This means that KubeVirt v1.0 supports Kubernetes v1.27, v1.26, v1.25 while
KubeVirt main is tracking support for Kubernetes v1.28, v1.27, v1.26.

# Support Exceptions

The KubeVirt community maintains the ability to extend support for a KubeVirt
release to older and newer Kubernetes versions that may fall outside of the
latest 3 Kubernetes release present at the time of the KubeVirt release.

The KubeVirt commitment to supporting the latest 3 Kubernetes release is a
commitment that this is the minimum guarantee we support and test. We will
not support less than this minimum, however there are times when we may choose
to support _more_ Kubernetes releases than our minimum guideline suggests.

# Unsupported Kubernetes Versions

It is possible, and likely, that KubeVirt releases may continue to work with
unsupported Kubernetes versions. Understand that we make no guarantees for this
support. While it may technically work, there is no testing or CI being
performed to guarantee it remains working with future KubeVirt patch releases. 


