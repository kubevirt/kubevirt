# Introduction

Let's start with the relationship between several related projects:

* **Kubernetes** is a container orchestration system, and is used to run
  containers on a cluster
* **containerized-data-importer (CDI)** is an add-on which solves the problem of
  populating Kubernetes Persistent Volumes with data.  It was written to be
  general purpose but with the virtualization use case in mind.  Therefore, it
  has a close relationship and special integration with KubeVirt.
* **KubeVirt** is an add-on which is installed on-top of Kubernetes, to be able
  to add basic virtualization functionality to Kubernetes.

As an add-on to Kubernetes, CDI shares some philosophy and design choices:

* Mostly written in golang
* Often related to distributed microservice architectures
* Declarative and Reactive (Operator pattern) approach

This short page shall help to get started with the projects and topics
surrounding them.  If you notice a strong similarity with the [KubeVirt contribution guidelines](https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md) it's because we have taken inspiration from their success.


## Contributing to CDI

### Our workflow

Contributing to CDI should be as simple as possible. Have a question? Want
to discuss something? Want to contribute something? Just open an
[Issue](https://github.com/kubevirt/containerized-data-importer/issues) or a [Pull
Request](https://github.com/kubevirt/containerized-data-importer/pulls).  For discussion, we use the [KubeVirt Google Group](https://groups.google.com/forum/#!forum/kubevirt-dev).

If you spot a bug or want to change something pretty simple, just go
ahead and open an Issue and/or a Pull Request, including your changes
at [kubevirt/containerized-data-importer](https://github.com/kubevirt/containerized-data-importercontainerized-data-importer).

For bigger changes, please create a tracker Issue, describing what you want to
do. Then either as the first commit in a Pull Request, or as an independent
Pull Request, provide an **informal** design proposal of your intended changes.
The location for such propoals is
[/docs](docs/) in the CDI repository. Make sure that all your Pull Requests link back to the
relevant Issues.

### Getting started

To make yourself comfortable with the code, you might want to work on some
Issues marked with one or more of the following labels
[help wanted](https://github.com/kubevirt/containerized-data-importer/labels/help%20wanted),
[good first issue](https://github.com/kubevirt/containerized-data-importer/labels/good%20first%20issue),
or [bug](https://github.com/kubevirt/containerized-data-importer/labels/kind%2Fbug).
Any help is greatly appreciated.

### Testing

**Untested features do not exist**. To ensure that what we code really works,
relevant flows should be covered via unit tests and functional tests. So when
thinking about a contribution, also think about testability. All tests can be
run local without the need of CI. Have a look at the
[Developer Guide](hack/README.md).

### Getting your code reviewed/merged

Maintainers are here to help you enabling your use-case in a reasonable amount
of time. The maintainers will try to review your code and give you productive
feedback in a reasonable amount of time. However, if you are blocked on a
review, or your Pull Request does not get the attention you think it deserves,
reach out for us via Comments in your Issues, or ping us on IRC
[#kubevirt @irc.freenode.net](https://kiwiirc.com/client/irc.freenode.net/kubevirt).

Maintainers are:

* @awels
* @j-griffith
* @aglitke

### PR Checklist

Before your PR can be merged it must meet the following criteria:
* [README.md](README.md) has been updated if core functionality is affected.
* Complex features need standalone documentation in [doc/](doc/).
* Functionality must be fully tested.  Unit test code coverage as reported by
  [Goveralls](https://coveralls.io/github/kubevirt/containerized-data-importer?branch=master)
  must not decrease unless justification is given (ie. you're adding generated
  code).


## Projects & Communities

### [CDI](https://github.com/kubevirt/containerized-data-importer)

* Getting started
  * [Developer Guide](hack/README.md)
  * [Other Documentation](doc/)

### [KubeVirt](https://github.com/kubevirt/)

* Getting started
  * [Developer Guide](docs/getting-started.md)
  * [Demo](https://github.com/kubevirt/demo)
  * [Documentation](docs/)

### [Kubernetes](http://kubernetes.io/)

* Getting started
  * [http://kubernetesbyexample.com](http://kubernetesbyexample.com)
  * [Hello Minikube - Kubernetes](https://kubernetes.io/docs/tutorials/stateless-application/hello-minikube/)
  * [User Guide - Kubernetes](https://kubernetes.io/docs/user-guide/)
* Details
  * [Declarative Management of Kubernetes Objects Using Configuration Files - Kubernetes](https://kubernetes.io/docs/concepts/tools/kubectl/object-management-using-declarative-config/)
  * [Kubernetes Architecture](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/architecture.md)

## Additional Topics

* Golang
  * [Documentation - The Go Programming Language](https://golang.org/doc/)
  * [Getting Started - The Go Programming Language](https://golang.org/doc/install)
* Patterns
  * [Introducing Operators: Putting Operational Knowledge into Software](https://coreos.com/blog/introducing-operators.html)
  * [Microservices](https://martinfowler.com/articles/microservices.html) nice
    content by Martin Fowler
* Testing
  * [Ginkgo - A Golang BDD Testing Framework](https://onsi.github.io/ginkgo/)
