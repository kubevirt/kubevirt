# Introduction

Let's start with the relationship between the two important components:

* **Kubernetes** is a container orchestration system, and is used to run
  containers on a cluster
* **KubeVirt** is an add-on which is installed on-top of Kubernetes, to be able
  to add basic virtualization functionality to Kubernetes.

KubeVirt is an add-on to Kubernetes, and they have several things in
common:

* Mostly written in golang
* Often related to distributed microservice architectures
* Declarative and Reactive (Operator pattern) approach

This short page shall help to get started with the projects and topics
surrounding them


## Contributing to KubeVirt

### Our workflow

Contributing to KubeVirt should be as simple as possible. Have a question? Want
to discuss something? Want to contribute something? Just open an
[Issue](https://github.com/kubevirt/kubevirt/issues), a [Pull
Request](https://github.com/kubevirt/kubevirt/pulls), or send a mail to our
[Google Group](https://groups.google.com/forum/#!forum/kubevirt-dev).

If you spot a bug or want to change something pretty simple, just go
ahead and open an Issue and/or a Pull Request, including your changes
at [kubevirt/kubevirt](https://github.com/kubevirt/kubevirt).

For bigger changes, please create a tracker Issue, describing what you want to
do. Then either as the first commit in a Pull Request, or as an independent
Pull Request, provide an **informal** design proposal of your intended changes.
The location for such propoals is
[/docs](docs/) in the KubeVirt
core repository. Make sure that all your Pull Requests link back to the
relevant Issues.

### Getting started

To make yourself comfortable with the code, you might want to work on some
Issues marked with one or more of the following labels
[beginner](https://github.com/kubevirt/kubevirt/issues?q=is%3Aissue+is%3Aopen+label%3Abeginner),
[help
wanted](https://github.com/kubevirt/kubevirt/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)
or [bug](https://github.com/kubevirt/kubevirt/labels/bug). Any help is highly
appreciated.

### Testing

**Untested features do not exist**. To ensure that what we code really works,
relevant flows should be covered via unit tests and functional tests. So when
thinking about a contribution, also think about testability. All tests can be
run local without the need of CI. Have a look at the
[Testing](docs/getting-started.md#testing)
section in the [Developer Guide](docs/getting-started.md).

### Getting your code reviewed/merged

Maintainers are here to help you enabling your use-case in a reasonable amount
of time. The maintainers will try to review your code and give you productive
feedback in a reasonable amount of time. However, if you are blocked on a
review, or your Pull Request does not get the attention you think it deserves,
reach out for us via Comments in your Issues, or ping us on IRC
[#kubevirt @irc.freenode.net](https://kiwiirc.com/client/irc.freenode.net/kubevirt).

Maintainers are:

 * @admiyo
 * @berrange
 * @davidvossel
 * @fabiand
 * @rmohr
 * @stu-gott
 * @vladikr

## Projects & Communities

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
