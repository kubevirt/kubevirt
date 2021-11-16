# Kubernetes Version Compatibility

Kubernetes supports 3 minor release concurrently. This means if the latest
Kubernetes release is v1.19, that v1.18 and v1.17 will remain supported for
backport fixes but that v1.16 will lose support.

Similarly, each KubeVirt release maintains compatibility with the latest 3
Kubernetes releases that are out at the time the KubeVirt release is made.

# Compatibility Matrix Examples.

Here are some theoretical examples (note that the versions are made up to
illustrate the example)

## New KubeVirt Release

KubeVirt release v0.1 is cut. At that point in time the latest Kubernetes
version is v1.3. This means KubeVirt v0.1 will forever be compatible with
Kubernetes v1.3, v1.2, and v1.1.

## KubeVirt Main

KubeVirt main always follows the latest 3 Kubernetes releases. If a new
Kubernetes v1.4 release is cut, that means support for v1.1 will be dropped
for KubeVirt main.

Note that this support for the latest Kubernetes releases doesn't happen
immediately. There is a period of time, usually a few weeks, where the new
version of Kubernetes must be integrated into KubeVirt's CI. The goal here
is to integrate the latest Kubernetes version into CI before the next
KubeVirt release is cut.

## Old KubeVirt Release

KubeVirt main supports Kubernetes releases v1.4, v1.3, and v1.2. However, the
KubeVirt v0.1 release was cut when the latest Kubernetes release was v1.3.

This means that KubeVirt v0.1 supports Kubernetes v1.3, v1.2, v1.1 while
KubeVirt main is tracking support for Kubernetes v1.4, v1.3, v1.2.

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


