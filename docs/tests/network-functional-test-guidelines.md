# Guidelines for Writing Network Functional/e2e Tests

## Prefer testing against VM objects over VMI objects

Most real users create `VirtualMachine` (VM) objects and let KubeVirt manage
the underlying `VirtualMachineInstance` (VMI) for them; standalone VMIs are
the exception, used when a truly ephemeral VM is actually wanted.
The VM is the primary, stable, user-facing object: it owns the supported
lifecycle contract (start/stop/restart, run strategy, live update, etc.),
while the VMI is the lower-level runtime object representing a single
running instance that most users never create directly. Anchoring tests on
the VM means they exercise that supported contract rather than an
implementation detail.
Network functional tests should default to the same shape: create a VM, let it
reconcile to a running VMI, and drive the test through the VM. This exercises
the VM→VMI orchestration path (the reconcile/ownership/status-propagation
code that standalone-VMI tests skip entirely) and keeps the suite closer to
how the feature is actually used in practice.

This is a default, not an absolute rule:

* Testing directly against a VMI is legitimate when the scenario is
  specifically about ephemeral VMs, or when exercising the VM reconcile path
  would add cost/complexity without adding coverage for what the test is
  actually about. Justify the choice in the test/PR rather than defaulting to
  VMI out of habit.
* Not everything can be observed through the VM object. Several status
  fields — interface status being a common one in network tests — are only
  ever populated on the VMI, not mirrored onto the VM. A VM-based test still
  needs to fetch the managed VMI to assert on those fields; that does not
  make it a "VMI test," since orchestration, lifecycle actions, and most
  status checks should still go through the VM wherever the field exists
  there (e.g. the VM's `Ready` condition to know the VM is up, rather than
  waiting on the VMI directly).
  The same applies to guest interaction: console/VNC access (both the test
  helpers and `virtctl console`/`vnc` themselves) addresses the VMI object
  directly — there is no VM-level subresource for it — so a VM-based test
  still fetches the managed VMI to talk to the guest, just as it does to
  read interface status.
* Don't let the "test through the VM" principle leak into breaking
  abstraction: avoid writing helpers that quietly wait on/manipulate the VMI
  internally while presenting a VM-shaped API. If a spec is working with a
  VM, keep working with that VM consistently (e.g. wait for the VM's ready
  condition, not the VMI's phase) and only drop to the VMI object for the
  specific fields that require it.

Much of the existing network test suite predates this guideline and still
creates standalone VMIs directly. That is not a backlog to burn down on its
own, but new tests should default to VM objects per the above, and existing
VMI-based tests are good candidates to convert when they are touched for
other reasons.
