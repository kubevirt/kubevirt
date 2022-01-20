# KubeVirt Scalability and Performance SLIs/SLOs

This document is inspired by the [Kubernetes slos document](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/slos.md) and is meant to serve the same purpose but for KubeVirt's control plane.

## KubeVirt Performance and Scale Testing
The KubeVirt SIG-scale is looking to use two tests to define scale and perf: Burst and Steady State.

The goal for a **Burst Test** is to evaluate the system's performance under heavy load when the user is most
interested in performing an operation as quickly as possible.  A good example of a Burst workload is a sudden
spike in demand for compute resources to handle a massive increase of users.

The Burst Test measures batch VM/VMI creation latency so here's how KubeVirt tests it. The Burst Test will
ramp up object count in a datacenter, measure, then ramp down object count in a datacenter. The measurement
will include the Object Creation rate (objects/second), VMI Phase Transition Rate, and count Kubernetes REST API calls.

The **Steady State Test** takes a different approach than the burst test in that instead of measuring how quickly
a system can move between different capacities, the Steady State Test focuses on how well the system can maintain
max capacity or near max capacity.  For example, a system may create a certain number of warm resources
(an active resource that isn't being used by a customer) to reduce user wait time.

The Steady State Test measures how quickly the system returns to the expected capacity after churn is applied.
This test will ramp up object count in a datacenter, create churn - deleting and recreating objects - then measure.
The Steady State Test will measure Object Creation rate (objects/second), VMI Phase Transition Rate, and count
Kubernetes REST API calls.

| Test | Creation Rate (VMI) | Creation Rate (VM) | Creation Rate (VMPool) |
| --- | --- | --- | --- |
| Burst |  |  |  |
| Steady State |  |  |  |

| Test | Phase Transition Rate (Pending) | Phase Transition Rate (Scheduling) | Phase Transition Rate (Scheduled) | Phase Transition Rate (Running) |
| --- | --- | --- | --- | --- |
| Burst |  |  |  |  |
| Steady State |  |  |  |  |

| Test | K8s REST Verb (CREATE) | K8s REST Verb (LIST) | K8s REST Verb (PATCH) | K8s REST Verb (UPDATE) |
| --- | --- | --- | --- | --- |
| Burst |  |  |  |  |
| Steady State |  |  |  |  |
