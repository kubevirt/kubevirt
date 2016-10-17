# Architecture

KubeVirt is built using a service oriented architecture.

## Stack

* KubeVirt
* Orchestration (K8s)
* Scheduling (K8s)
* Container Runtime
* Operating System
* Virtual
* Physical

Users requiring virtualization services are speaking to the Virtualization API (see below) which in turn is speaking to the Kubernetes cluster to schedule requested VMs.
Scheduling, networking, and storage are all delegated to Kubernetes, while KubeVirt provides the virtualization functionality.

## Services

KubeVirt provides additional functionality to your Kubernetes cluster, including:

* Virtual Machine management
* Network management
* REST API for virtulization functionality

## Application Layout

* Cluster
  * KubeVirt Components
    * virt-controller
    * …
  * KubeVirt Managed Pods
    * VM Foo
    * VM Bar
    * …

## Native Workloads

KubeVirt is deployed on top of a Kubernetes cluster.
This means that you can continue to run your Kubernetes-native workloads next to the VMs managed through KubeVirt.

