# Overview

Starting with Kubernetes 1.7+, a new construct called the Aggregated API Server
is now available. This Aggregated API Server proxies traffic for various User
API Servers (UAS) that can be added onto Kubernetes dynamically.

# Requirements

In order to provide proper authentication for client requests, the KubeVirt
User Api Server requires a CA certificate and keypair be registered with
Kubernetes. This is a one-time process that is required when first setting up
KubeVirt. See apiserver-pki.md for more information on setting this up.

# Rationale

Before the inception of the Aggregated API Server, Kubevirt used an entirely
independent API server. Traffic to this server, as well as the main Kubernetes
API server, was proxied via haproxy. Queries for KubeVirt related constructs
were directed to the KubeVirt API server, while all other traffic was directed
to the main Kubernetes API server.

This works, but there are some downsides. Firstly, the model doesn't scale when
other projects are introduced. If more than one project requires a proxy in
front of Kubernetes, traffic either needs to be sent through all proxies in
order, or a disjointed experience will occur, where queries for different kinds
of resources need to be passed to disparate API endpoints.

Secondly, using a non-standard API entrypoint requires that we use a script to
set up the proper kubectl configuration.

# Storage

One consideration when writing a UAS is storage. At the time of this writing
(Fall 2017), the Kubernetes project is not in a final state with respect to the
recommended approach. Kubernetes has a future feature on the roadmap to provide
an API that Addon servers such as the KubeVirt apiserver could use to store
various resources. However, this effort is not staffed and no work is expected
to occur for a considerable amount of time.

In the meantime, the approach they recommend is to ship an etcd server that the
UAS can use for resource storage. The downside to this is increased complexity
in terms of setting up a KubeVirt deployment.

Other approaches that could be considered are either using
CustomResourceDefinitions or targetting other arbitrary storage engines
(possibly also incurring increased setup complexity). The downside here is that
increased complexity to code the server is incurred. This also implies that
when the Kubernetes API does provide a storage mechanism, code to use of a
custom storage module will need to be reversed.

# Current State

There are currently a couple issues with this branch:

## Empty update records

Inside the apiserver etcd3 store.go module, Calls to GuaranteedUpdate take an
incorrect code path, this results in the apiserver seeing only empty records on
attempted updates--which obviously never succeed.

## virt-contoller and empty namespaces

When submitting VirtualMachine records to the apiserver, if the namespace is
omitted or default, the cache code in virt-controller will be handed an empty
key. Attempts to look up the correct VirtualMachine from the cache is obviously
impossible this way--so the controller ends up crashing.
