# Virtual Machine Graceful Shutdown

## Overview

Virtual machine graceful shutdown is the process of signaling a virtual
machine to begin shutting down before forcing the virtual machine off. This
process gives the virtual machine a chance to react to a shutdown request
before the KubeVirt runtime does the equivalent of pulling the power plug on
the virtual machine.

The period between when a virtual machine is signaled to shutdown, and the
point in time that KubeVirt will force the virtual machine off if it is still
active is called a **Grace Period**. The grace period is a configurable value
represented in the Virtual Machine's specification by the
**terminationGracePeriodSeconds** option.

## Usage Examples

### Default: No terminationGracePeriodSeconds specified

By default, if the grace period option is not set a small default grace period
will be observed before killing the virtual machine. At the moment this default
is 30 seconds, which is consistent with value used for containers by
Kubernetes.

### Immediate Force Shutdown: terminationGracePeriodSeconds = 0

A 0 value for the grace period option means that the virtual machine should not
have a grace period observed during shutdown. If a user specifies 0 for this
value, the virtual machine will be immediately force killed on shutdown.

### Grace Period Values > 0

Any value > 0 specified for terminationGracePeriodSeconds represents the number
of seconds the KubeVirt runtime will wait between signaling a virtual machine
to shutdown and killing the virtual machine if it is still active.

## Design and Implementation

At the moment, the only way to shutdown a virtual machine is to remove the
cluster object from kubernetes. Once the virtual machine object has been
removed, we no longer have access to the terminationGracePeriodSeconds value
stored on the Virtual Machine's spec. 

In order to guarantee the value stored in the virtual machine's
terinationGracePeriodSeconds is observed after the cluster object is deleted,
that value is cached locally by virt-handler during the start flow. When a
deleted virtual machine cluster object is detected, the cached grace period
value is observed as virt-handler is shutting the virtual machine down.

### Virt-Controller Involvement

The only change to virt-controller is that it now configurs a custom
grace period for virt-launcher pods that matches the grace period set on
the corresponding virtual machine object. The virt-launcher grace period
is slightly padded in order to ensure under normal operation that
virt-handler will have a chance to force a virtual machine off before the
virt-launcher pod terminates.  If the virt-launcher pod terminates first,
the virtual machine will be forced off as a result of the kubernetes
runtime killing all processes in the virt-launcher cgroup.

### Virt-Handler Involvement

Virt-handler is now responsible for both signaling the virtual machine to
shutdown and ensuring the virtual machine is forced off after the grace
period is observed.

Signaling the beginning of the grace period can come from two sources.

1. The virt-handler virtual machine object informer can notify virt-handler
the cluster object has been removed for currently active virtual machine.

2. A virtual machine's corresponding virt-launcher pod can signal shutdown
by writing to a graceful shutdown trigger file in a shared directory between
virt-launcher and virt-handler.

Once the grace period begins for a virtual machine, virt-handler maintains
the state associated with the grace period in a local cache file. This
allows the grace period to be observed even if the virt-handler process
recovers during this period.

### Virt-Launcher Involvement

Virt-launcher intercepts signals (such as SIGTERM) sent to it by the kubernetes
runtime and notifies virt-handler to begin the gracefull shutdown process by
writting to the graceful shutdown trigger file. 

After writting to the graceful shutdown trigger file, virt-launcher continues to
watch the pid until either the pid exits (as a result of virt-handler shutting
it down) or the kubernetes runtime kills the virt-launcher process with SIGKILL.

A force kill of virt-launcher will result in the corresponding virtual machine
exiting.

### Shutdown Notification Race (Virt-Launcher VS. VMI Object Informer)

When a Virtual Machine object is removed from the cluster, that sets off a race
between two sources used to notify virt-handler it should shutdown the virtual
machine.

1. The virtual machine cluster informer.

2. virt-launcher graceful shutdown trigger.

It doesn't matter which one of these comes first. Once it begins, graceful
shutdown process idempotent.

It is worth noting that this race condition one of the reasons why
virt-launcher needs to signal virt-handler to perform graceful shutdown instead
of virt-launcher acting directly on the process. By centralizing the shutdown
flow to virt-handler, we can guarantee a single grace period is observed
accurately.
