# Cgroup v1 support is deprecated

## Introduction

The containerized world, including many technologies like Kubernetes, systemd and KubeVirt,
were originally designed to run on cgroup v1.

At this point, cgroup v2 is the default cgroup manager for most distributions and is widely adopted.

Kubernetes moved cgroup v1 support to maintenance mode in 1.31, and deprecated it in 1.35.
KubeVirt previously followed Kubernetes by moving cgroup v1 to maintenance mode, and is now
formally deprecating it as well.

For more info, please look at the Kubernetes blog post on the subject:
https://kubernetes.io/blog/2024/08/14/kubernetes-1-31-moving-cgroup-v1-support-maintenance-mode/

The Kubernetes enhancement tracking the removal of cgroup v1 support:
https://github.com/kubernetes/enhancements/issues/5573

## What does this mean?

**Cgroup v1 support in KubeVirt is deprecated and will be removed in the next release.**

Users running KubeVirt on nodes that use cgroup v1 should migrate to cgroup v2 before
upgrading to the next KubeVirt release.

During the deprecation phase:
- No new features will be added to cgroup v1 support.
- Critical security fixes will still be provided.
- Major bugs may be fixed if feasible, but some issues might remain unresolved.

Consult your distribution's documentation for instructions on switching from cgroup v1 to
cgroup v2. Kubernetes also provides guidance:
https://kubernetes.io/docs/concepts/architecture/cgroups/

## Background

Quoting from the Kubernetes v1.35 release blog:

> Because cgroup v2 is now the modern standard, Kubernetes is ready to retire the legacy
> cgroup v1 support in v1.35. This is an important notice for cluster administrators: if
> you are still running nodes on older Linux distributions that don't support cgroup v2,
> your kubelet will fail to start. To avoid downtime, you will need to migrate those nodes
> to systems where cgroup v2 is enabled.

KubeVirt is following Kubernetes in deprecating cgroup v1 support, with removal planned
for the next release.
