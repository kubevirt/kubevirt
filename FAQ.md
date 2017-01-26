
# Frequently Asked Questions

## Why don't you use the Container Runtime Interface (CRI) to launch VMs?

The short story is that there are two main reasons: Currently there is
no way how VMs can be launched in a parametrized way, and the second
reason is that Kubernetes is built around the assumption of cloud
workloads. This assumption is contradicting the assumptions we have on
pet VMs.

KubeVirt is looking for a solution to manage VMs. This includes both ends:
VMs as they are used in the cloud, thus with few tunables, but also pet
VMs like they are used in traditional data-center virtualization, which expose
a lot more tunables.

Especially the pet case requires the exposure of many tunables, and this can
currently not be established with CRI - in a scalable way. (I.e. it could be
raised that annotations could be used to set VM parameters, but to us this is
rather a workaround than a solution).

Besides that Kubernetes is built around the assumptions of containers and
cloud workloads. And it's difficult to bend Kubernetes to also handle or
support assumptions on which pet VMs are based.

## What architectures are supported?

Currently we are only supporting x86-64.

## Why don't you support my distribution?

In general we aim to be OS independent by shipping all dependencies in
containers (i.e. libvirtd).
But in case that you have issues, please
[file a bug](https://github.com/kubevirt/kubevirt/issues).

## Why don't you answer this question: … ?

We want to, please open an
[issue](https://github.com/kubevirt/kubevirt/issues) and we'll try to answer
and consider to add it to this FAQ.
