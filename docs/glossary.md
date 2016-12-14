# KubeVirt Glossary

## VM
See [VM Definition](##vm-definition)

## TPR
Abbreviation for Kuberenetes [Third Party Resource](##third-party-resource).

## Third Party Resource
Kubernetes has an extensible API which allows extending its REST-API.
Resources using this extension mechanism are called Third Party Resource.
See [extensible-api](https://github.com/kubernetes/kubernetes/blob/master/docs/design/extending-api.md)
for more information.

## VM Spec
Descrition of a VM on the cluster level. This is part of the [VM Definition](##vm-definition), but contains purely VM related information (Devices, Disks, Networks, ...)

## VM Definition
The whole definition of the VM how cluster-wide. Including Kubernetes Metadata, ...
The [VM Spec](##vm-spec) is part of it.

## Domain
Libvirt domain. `virt-handler` can derive a Domain XML out of a [VM Spec](##vm-spec). 
This is the host centric view of the cluster wide [VM Spec](##vm-spec).
