Monitoring VMs
==============

Summary
-------

VM metrics are collected through libvirt APIs and reported using a prometheus endpoint.
The metrics are scraped when the endpoint is queried, implementing the [Collector](https://godoc.org/github.com/prometheus/client_golang/prometheus#Collector) interface. There is no caching.
The collecting code is robust with respect to slow or unresponsive VMs.
Each VM on a given node is queried by a goroutine.
At any given time no more than one single goroutine can be querying the VM for metrics.

Design
------

The requirements for the collecting code are (not in priority order)
1. be as fast as possible
2. be as lightweight as possible
3. deal gracefully with unresponsive source (more on that below)

While the first two bullet points are easy to understand, the third needs some more context

Unresponsive metrics sources
----------------------------

When we use QEMU on shared storage, like we want to do with Kubevirt, any network issue could cause
one or more storage operations to delay, or to be lost entirely.

In that case, the userspace process that requested the operation can end up in the D state,
and become unresponsive, and unkillable.

A robust monitoring application must deal with the fact that
the libvirt API can block for a long time, or forever. This is not an issue or a bug of one specific
API, but it is rather a byproduct of how libvirt and QEMU interact.

Whenever we query more than one VM, we should take care to avoid that a blocked VM prevent other,
well behaving VMs to be queried. IOW, we don't want one rogue VM to disrupt well-behaving VMs.
Unfortunately, any way we enumerate VMs, either implicitly, using the libvirt bulk stats API,
or explicitly, listing all libvirt domains and query each one in turn, we may unpredictably encounter
unresponsive VMs.


Dealing with unresponsive metrics source
----------------------------------------

From a monitoring perspective, _any_ monitoring-related libvirt call could be unresponsive any given time.
To deal with that:
1. the monitoring of each VM is done in a separate goroutine. Goroutines are cheap, so we don't recycle them (e.g. nothing like a goroutine pool).
Each monitoring goroutine ends once it collected the metrics.
2. we don't want monitoring goroutines to pile up on unresponsive VMs. To avoid that we track the business of metrics source. No more than one goroutine may
query a VM for metrics at any given time. If the VM is unresponsive, no more than a goroutine waits for it. This also act as simple throttling mechanism.
3. it is possible that a libvirt API call _eventually_ unblocks. Thus the monitoring goroutine must take care of checking that the data it is going to submit
is still fresh, and avoid overriding fresh data with stale data.


Appendix: high level recap: libvirt client, libvirt daemon, QEMU
-----------------------------------------------------------------

Let's review how the client application (anything using the pkg.monitoring/vms/processes/prometheus package),
the libvirtd daemon and the QEMU processes interact with each other.

The libvirt daemon talks to QEMU using the JSON QMP protocol over UNIX domain sockets. This happens in the virt-launcher pod.
The details of the protocol are not important now, but the key part is that the protocol
is a simple request/response, meaning that libvirtd must serialize all the interactions
with the QEMU monitor, and must protects its endpoint with a lock.
No out of order request/responses are possible (e.g. no pipelining or async replies).
This means that if for any reason a QMP request could not be completed, any other caller
trying to access the QEMU monitor will block until the blocked caller returns.

To retrieve some key informations, most notably about the block device state or the balloon
device state, the libvirtd daemon *must* use the QMP protocol.

The QEMU core, including the handling of the QMP protocol, is single-threaded.
All the above combined make it possible for a client to block forever waiting for a QMP
request, if QEMU itself is blocked. The most likely cause of block is I/O, and this is especially
true considering how QEMU is used.

