# Containerized Data Importer

[![Build Status](https://travis-ci.org/kubevirt/containerized-data-importer.svg?branch=master)](https://travis-ci.org/kubevirt/containerized-data-importer)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubevirt/containerized-data-importer)](https://goreportcard.com/report/github.com/kubevirt/containerized-data-importer)
[![Coverage Status](https://img.shields.io/coveralls/kubevirt/containerized-data-importer/master.svg)](https://coveralls.io/github/kubevirt/containerized-data-importer?branch=master)
[![Licensed under Apache License version 2.0](https://img.shields.io/github/license/kubevirt/containerized-data-importer.svg)](https://www.apache.org/licenses/LICENSE-2.0)

**Containerized-Data-Importer (CDI)** is a persistent storage management add-on for Kubernetes.
It's primary goal is to provide a declarative way to build Virtual Machine Disks on PVCs for [Kubevirt](https://github.com/kubevirt/kubevirt) VMs

CDI works with standard core Kubernetes resources and is storage device agnostic, while its primary focus is to build disk images for Kubevirt, it's also useful outside of a Kubevirt context to use for initializing your Kubernetes Volumes with data.


# Introduction

## Kubernetes extension to populate PVCs with VM images
CDI provides the ability to populate PVCs with VM images upon creation.  Multiple image formats and sources are supported:

### Current supported Image formats
* .tar
* .gz
* .xz
* .img
* .iso
* .qcow2

### Current supported image endpoints
* http
* S3
* local directory

## DataVolumes
CDI also includes a CRD, that provides an object of type DataVolume.  The DataVolume is an abstraction on top of the standard Kubernetes PVC and can be used to automate creation and population of a PVC for consumption in a Kubevirt VM.

## Deploy it

Deploying the CDI controller is straightforward. In this document the _default_ namespace is used, but in a production setup a [protected namespace](#protecting-the-golden-image-namespace) that is inaccessible to regular users should be used instead.

  ```
  $ export VERSION=$(curl https://github.com/kubevirt/containerized-data-importer/releases/latest | grep -o "v[0-9]\.[0-9]*\.[0-9]*")
  $ kubectl create -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-operator.yaml
  $ kubectl create -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-operator-cr.yaml
  ```

## Use it

Create a DataVolume and populate it with data from an http source

```
$ kubectl create -f https://raw.githubusercontent.com/kubevirt/containerized-data-importer/$VERSION/manifests/example/datavolume.yaml
```

There are quite a few examples in the [example manifests](https://github.com/kubevirt/containerized-data-importer/tree/master/manifests/example), check them out as a reference to create DataVolumes from additional sources like registries, S3 and your local system.

## Hack it

CDI includes a self contained development and test environment.  We use Docker to build, and we provide a simple way to get a test cluster up and running on your laptop.

```
$ mkdir $GOPATH/src/kubevirt.io && cd $GOPATH/src/kubevirt.io
$ git clone https://github.com/kubevirt/containerized-data-importer && cd containerized-data-importer
$ make cluster-up
$ make cluster-sync
$ ./cluster/kubect.sh .....
```

## Connect with us

We'd love to hear from you, reach out on Github via Issues or Pull Requests!

Hit us up on Slack, under the Kubernetes Virtualization channel

Shoot us an email at: kubevirt-dev@googlegroups.com


## More details

1. [Hacking details](hack/README.md#getting-started-for-developers)
1. [Design docs](/doc/design.md#design)
1. [Kubevirt documentation](https://kubevirt.io)
