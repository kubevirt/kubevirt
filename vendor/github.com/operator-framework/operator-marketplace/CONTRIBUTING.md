# How to contribute

Operator Marketplace is Apache 2.0 licensed and accepts contributions via GitHub pull requests. This document outlines some of the conventions on commit message formatting, instructions on how to set up a dev environment, and other resources to help get contributions into operator-marketplace.  

## Setting up a dev environment

- Fork the repository on GitHub
- Clone the forked repository in your go path.
- Get access to an OpenShift cluster. The marketplace-operator is currently supported on OpenShift 4.0. You can find instructions on installing an OpenShift cluster [here](https://github.com/openshift/installer).
- The Cluster-Version-Operator(CVO) manages the lifecycle of the marketplace-operator along with other internal OpenShift components. Before deleting the marketplace-operator deployment, instruct CVO to stop managing the marketplace-operator 
```
$ oc apply -f examples/cvo.override.yaml
``` 
- Delete the marketplace-operator deployment
```
$ oc delete deployment openshift-marketplace -n marketplace-operator
```
- Install [Operator-SDK](https://github.com/operator-framework/operator-sdk).
- Compile the marketplace-operator and start the operator in your dev environment
```
$ operator-sdk up local --namespace=openshift-marketplace --kubeconfig=<path-to-kubeconfig-file>  
```

### Testing changes to RBAC policy
- If there's any modification in existing RBAC policies, the modifications need to be tested using a separate deployment. Easiest way to do this is to build an image locally using [this dockerfile](./Dockerfile), push the image to a registry, and replace the [image here](https://github.com/operator-framework/operator-marketplace/blob/master/deploy/operator.yaml#L23) with the newly built image. The operator can then be redeployed with the new changes.   

```
$ cd <path-to-operator-marketplace-repo>

$ operator-sdk build . --tag=<REGISTRYHOST>/<USERNAME>/marketplace-operator

$ docker push <REGISTRYHOST>/<USERNAME>/marketplace-operator 

$ sed -i -e 's?quay.io/openshift/origin-operator-marketplace:latest?<REGISTRYHOST>/<USERNAME>/marketplace-operator?g' deploy/operator.yaml

$ oc apply -f deploy/operator.yaml
```

## Marketplace-operator on vanilla Kubernetes

- To set up marketplace-operator on vanilla Kubernetes, [install Operator-Lifecycle-Manager(OLM)](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/install/install.md) in a Kubernetes Cluster. 
- `cd` into the marketplace-operator repository and install the marketplace-operator using 
```
$ kubectl apply -f deploy/upstream
```
- To delete the marketplace-operator deployment use 
```
$ kubectl delete deployment marketplace
``` 

### Testing changes to RBAC policy on vanilla Kubernetes
- If there's any modification in existing RBAC policies, the modifications need to be tested using a separate deployment. Easiest way to do this is to build an image locally using [this dockerfile](./Dockerfile), push the image to a registry, and replace the [image here](https://github.com/operator-framework/operator-marketplace/blob/master/deploy/upstream/07_operator.yaml#L28) with the newly built image. The operator can then be redeployed with the new changes.

```
$ cd <path-to-operator-marketplace-repo>

$ operator-sdk build . --tag=<REGISTRYHOST>/<USERNAME>/marketplace-operator

$ docker push <REGISTRYHOST>/<USERNAME>/marketplace-operator 

$ sed -i -e 's?quay.io/openshift/origin-operator-marketplace:latest?<REGISTRYHOST>/<USERNAME>/marketplace-operator?g' deploy/upstream/07_operator.yaml

$ kubectl apply -f deploy/upstream/operator.yaml
```

## Reporting bugs and creating issues

If any part of the operator-marketplace project has bugs or documentation mistakes, please let us know by [opening an issue](https://github.com/operator-framework/operator-marketplace/issues/new) or a PR. 

## Contribution flow

This is an outline of what a contributor's workflow looks like:

- Fork the repository on GitHub
- Clone the forked repository in your go path.
- Create a topic branch from where to base the contribution. This is usually master.
- Make commits of logical units. A commit should typically add a feature or fix a bug, but never both at the same time. A PR should also have one single commit, i.e all the changes made should be consolidated into one single commit.  
- Make sure commit messages are in the proper format (see below).
- Push changes in a topic branch to a personal fork of the repository.
- To make sure that your topic branch is in sync with the remote master branch, follow a rebase workflow.
- Submit a pull request to operator-framework/operator-marketplace.
- The PR must receive one `/lgtm` and one `/approve` comments from the maintainers of the operator-marketplace project.

Thanks for contributing!

### Format of the commit message

We follow a convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```
bug 1683422: [csc] Make CLI output less verbose

- Fixes #116
- Removed TARGETNAMESPACE and PACKAGES fields from output
- Modified catalogsourceconfig.crd to remove fields from `additionalPrinterColumns`

```

The format can be described more formally as follows:

```
<subsystem>: <what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

The first line is the subject and should be no longer than 50 characters, the second line is always blank, and other lines should be wrapped at 80 characters. This allows the message to be easier to read on GitHub as well as in various git tools.
