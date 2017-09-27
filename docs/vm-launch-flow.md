# Overview and Concepts

This proposal outlines an approach for tieing a virtual machine process
lifecycle to a kubernetes POD lifecycle in a way that does not depend on pid
namespaces.

The goal here is to provide guarantees that a virtual machine process will be
unable to permanently outlive the lifecycle of the corresponding POD while
keeping the virtual machine process in the same pid namespace as libvirt.

## Kubernetes Prestop Hooks

Kubernetes has a container feature called a *"Prestop hook"* that results in
a script being executed in a container before Kubernetes sends the signal to
terminate the container. As long as the kubernetes runtime is invoking a POD's
termination, this script will get executed.

The Prestop hook is not bound to the container’s grace-period, meaning there’s
no timeout associated with a Prestop hook. It can wait/block indefinitely.

https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#hook-handler-execution 
*"If the hook hangs during execution, the Pod phase stays in a running state and
never reaches failed.”*

## Watchdog Timer

https://en.wikipedia.org/wiki/Watchdog_timer

*"A watchdog timer is an electronic timer that is used to detect and recover
from computer malfunctions. During normal operation, the computer regularly
resets the watchdog timer to prevent it from elapsing, or "timing out". If, due
to a hardware fault or program error, the computer fails to reset the watchdog,
the timer will elapse and generate a timeout signal. The timeout signal is used
to initiate corrective action or actions."*

Use of watchdog timers puts the responsibility of reporting liveness on the
component being watched. The watcher is simply ensuring that the *"something"*
is updating the timer. If the timer expires the watcher executes a recovery
action.

### File based Watchdog Timers

A simplistic implementation of a watchdog timer is have the watched component
update a file on the local filesystem on a recurring interval. The watcher is
looking at the file’s modified time. Once the difference in
(epoch time - file’s modified time) exceeds the watchdog timeout, the watchdog
recovery actions occur.

# Proposed Changes

## Component Responsibilities 

**Virt-launcher** is responsible for invoking the startup conditions required
by virt-handler to launch the corresponding virtual machine. It is also
responsible for monitoring the virtual machine’s pid which ties the POD's
lifecycle to the virtual machine’s lifecycle under normal operation. During
k8s invoked termination, the virt-launcher Prestop hook is responsible for
signalling virt-handler to perform virtual machine termination and waiting
for confirmation termination has occurred before exiting.

**Virt-handler** is responsible for maintaining all things related to the local
lifecycle of the virtual machine process. It coordinates with virt-controller
and virt-launcher to achieve this.

## Virt-launcher Container Prestop Hook

A Prestop hook script is used by the virt-launcher container to signal
virt-handler to terminate the corresponding virtual machine process. This
occurs when the virt-launcher container is being terminated by k8s. This
Prestop hook guarantees the virt-launcher container will not exit until the
corresponding virtual machine process has terminated.

This prestop hook involves a single action. It signals to virt-handler it
would like to exit and waits indefinitely for confirmation from virt-handler
it is okay to exit.  

When virt-handler receives a virt-launcher signal for termination, the
virt-handler handles gracefully terminating the corresponding virtual machine
process. Once the virtual machine process has exited, virt-handler confirms
the virt-launcher’s Prestop hook request for termination.

One simple way to implement the virt-handler confirmation is by creating a file
in a shared directory between virt-launcher and virt-handler. Virt-launcher
creates a file in this directory requesting termination, virt-handler confirms
the request for termination by deleting the file.  This means the prestop hook
is simply creating a file and waiting for it to disappear before exiting. This
method is naturally resilient to virt-handler being temporarily unavailable. 

## Virt-launcher Watchdog Timer Safeguard

In the event that a virt-launcher process crashes or the virt-launcher container
is killed forcibly, virt-handler needs a container runtime agnostic way of
detecting a virt-launcher process has exited in an unrecoverable way.

This can be achieved through the use of a file based watchdog timer. 

Every virt-launcher must create a watchdog file in a shared directory with
virt-handler and touch that file every ‘X’ number of seconds.

Virt-handler must consider the presence of the watchdog file a virtual machine
startup condition similar to how the shared unix socket file is treated. 

If virt-handler detects that the system’s epoch time minus the watchdog file’s
access time exceeds the watchdog timeout value, then the virtual machine is
terminated. The virt-launcher process must continue to touch the watchdog file
in order to keep the virtual machine process alive. 

# New Launch Flow Examples

In the new launch flow, virt-launcher, virt-handler, and libvirt all share
the same PID namespace.

## Startup Flow

- Virt-controller creates a virt-launcher POD that corresponds to a
  VirtualMachine spec that needs to launch.
- K8s schedules the virt-launcher POD
- Virt-launcher initializes on a node, creates socket and watchdog file, waits
  for virtual machine pid file to appear in a shared directory so it can begin
  monitoring the virtual machine process.
- Virt-controller sees the POD readiness checks have passed and sets
  NodeSelector to the POD’s target node on corresponding VirtualMachine spec
- Virt-handler sees a new VirtualMachine spec is assigned to its local node and
  checks to see if virt-launcher startup conditions pass (socket and unexpired
  watchdog file must be present)
- Virt-handler invokes libvirt to launch virtual machine process.
- Qemu-wrapper script launches process in virt-launcher’s cgroups and writes
  its pid to a pid file visible to virt-launcher.
- Virt-launcher sees the pidfile and begins monitoring the virtual machine process.

## Shutdown Flow: Cluster Level User Initiated

- Virt-controller removes VirtualMachine object and deletes virt-launcher pod.
- Virt-handler detects VirtualMachine cluster object is gone and begins
  executing graceful shutdown of the virtual machine process.
- The virt-launcher Prestop hook is executed as part of POD termination flow
  which requests virtual machine termination and waits for confirmation.
- Virt-handler sees request for termination from virt-launcher and knows that
  virtual machine is already in the process of being torn down.
- Eventually the virtual machine pid exits.
- Virt-handler confirms any virt-launcher requests for termination related to
  the terminated virtual machine.
- Virt-launcher POD exits once it detects virt-handler confirmation.

## Shutdown Flow: Pod Termination Initiated
- Someone/something requests k8s to terminate a virt-launcher pod.
- The virt-launcher Prestop hook is executed as part of POD termination flow
  which requests virtual machine termination and waits for confirmation.
- Virt-handler sees termination request from virt-launcher and begins
  gracefully shutting down the virtual machine. 
- Eventually the virtual machine pid exits.
- Virt-handler confirms any virt-launcher requests for termination related to
  the terminated virtual machine.
- Virt-launcher POD exits once it detects virt-handler confirmation.
- Shutdown Flow: Pod Abnormal Exit
- The virt-launcher process fails or the virt-launcher container is forcibly
  removed.
- The corresponding virt-launcher watchdog file expires
- Virt-handler detects the watchdog file has expired and begins gracefully
  tearing down the virtual machine process.
- Eventually the virtual machine pid exits. 

# Virtual Machine Lifecycle Management Guarantees

Given the ability to automatically recover a failed virt-handler daemon, the
following guarantees are provided by this proposal.

- VM will not start without a corresponding POD being active or very recently
  being active (within watchdog timeout period)
- POD will not exit without resulting in the eventual virtual machine
  termination.

In the future, an unrecoverable virt-handler component should result in a
fencing operation taking place on the failed node. Fencing closes any
remaining gap that could cause virtual machine processes to outlive their
corresponding POD as a result of a permanent virt-handler failure. 

