# Cgroup v1 support has been removed

## Introduction

The containerized world, including many technologies like Kubernetes, systemd and KubeVirt,
were originally designed to run on cgroup v1.

At this point, cgroup v2 is the default cgroup manager for most distributions and is widely adopted.

Kubernetes moved cgroup v1 support to maintenance mode in 1.31, deprecated it in 1.35, and
removed it in 1.37. KubeVirt previously followed Kubernetes through maintenance mode and
formal deprecation, and has now removed cgroup v1 support as well.

For more info, please look at the Kubernetes blog post on the subject:
https://kubernetes.io/blog/2024/08/14/kubernetes-1-31-moving-cgroup-v1-support-maintenance-mode/

The Kubernetes enhancement tracking the removal of cgroup v1 support:
https://github.com/kubernetes/enhancements/issues/5573

## What does this mean?

**Cgroup v1 support in KubeVirt has been removed.**

KubeVirt now requires nodes running cgroup v2. If your nodes still use cgroup v1,
KubeVirt will not function correctly — migrate to cgroup v2 before upgrading.

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

KubeVirt followed Kubernetes in deprecating cgroup v1 support and has now completed its removal.
