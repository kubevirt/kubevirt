# VirtualMachine Controller Synchronizer Interface

This document describes the synchronizer interface contract used by the VirtualMachine controller for managing Spec and Status modifications.

## Overview

The VirtualMachine controller uses a synchronizer pattern to separate business logic from API persistence. Synchronizers are responsible for computing desired state changes, while the controller persists ordinary VM changes. Writes to related resources and the existing Expand exception are described below.

## Synchronizer Interface Contract

All synchronizers in the VirtualMachine controller must follow this contract:

```go
type synchronizer interface {
    Sync(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
}
```

### Contract Requirements

1. **MUST only modify the local VM object** passed as a parameter (ObjectMeta, Spec and Status fields are allowed). The controller passes a DeepCopy, so synchronizers may modify it in place.
2. **MUST NOT make direct API calls** to `Update()`, `UpdateStatus()`, `Patch()`, or `PatchStatus()` on the VirtualMachine
3. **MUST return the modified VM object** (or the original if no changes were made)
4. **MAY return an error** if synchronization fails

### Responsibility Separation

The synchronizer is responsible for:

- Computing desired state based on current VM and VMI state
- Modifying the local VM object supplied by the controller
- Returning the modified VM object

The VirtualMachine controller is responsible for:

- Detecting changes via `DeepEqual` checks on Spec/Status
- Persisting Spec/ObjectMeta changes via `Update()`
- Persisting Status changes via `UpdateStatus()` in the `updateStatus()` function
- Handling API errors and retries

## Implementation Pattern

### Example Synchronizer

```go
func (s *MySynchronizer) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
    // Compute desired state
    desiredValue := s.computeValue(vm, vmi)

    // The controller already supplied a copy of the cached VM
    vm.Status.SomeField = desiredValue

    // Return the modified VM
    return vm, nil
}
```

### Controller Integration

The controller integrates synchronizers in the `sync()` function:

```go
func (c *Controller) sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, key string) (*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance, common.SyncError, error) {
    // ... other logic ...

    // Call synchronizer
    if c.mySynchronizer != nil {
        syncedVM, err := c.mySynchronizer.Sync(vm, vmi)
        if err != nil {
            return vm, vmi, handleSynchronizerErr(err), nil
        }
        vm.ObjectMeta = syncedVM.ObjectMeta
        vm.Spec = syncedVM.Spec
        vm.Status = syncedVM.Status
    }

    // ... controller persists changes later ...

    return vm, vmi, nil, nil
}
```

In the `execute()` function, the controller detects and persists changes:

```go
func (c *Controller) execute(key string) error {
    obj, exists, err := c.vmIndexer.GetByKey(key)
    // ...
    originalVM := obj.(*virtv1.VirtualMachine)
    vm := originalVM.DeepCopy()

    // Run sync (which calls all synchronizers)
    vm, vmi, syncErr, err = c.sync(vm, vmi, key)

    // Persist Status changes
    err = c.updateStatus(vm, originalVM, vmi, syncErr, logger)

    return syncErr
}
```

The `updateStatus()` function compares Status and persists changes:

```go
func (c *Controller) updateStatus(vm, vmOrig *virtv1.VirtualMachine, ...) error {
    // ... compute status fields ...

    // Only update if Status changed
    if !equality.Semantic.DeepEqual(vm.Status, vmOrig.Status) {
        if _, err := c.clientset.VirtualMachine(vm.Namespace).UpdateStatus(context.Background(), vm, v1.UpdateOptions{}); err != nil {
            return err
        }
    }
    return nil
}
```

## Writes to Other Resources

The contract restricts writes to the VirtualMachine being synchronized. Synchronizers may write other resources when needed, including creating ControllerRevisions and patching VMI interfaces in the network synchronizer.

### ControllerRevision Management

Instancetype synchronizers may create ControllerRevisions as side effects during `Sync()`:

```go
func (c *controller) Store(vm *virtv1.VirtualMachine) error {
    // Calls storeControllerRevision() which creates ControllerRevision via API
    if err := c.storeInstancetypeRevision(vm); err != nil {
        return err
    }

    // storeInstancetypeRevision updates vm.Status.InstancetypeRef locally (no API call)
    return nil
}
```

**Rationale**: ControllerRevisions are immutable snapshots that need to exist before Status refs can point to them. Creating them as side effects is acceptable. Upgrade retains the original snapshots until the VM controller successfully persists their replacement status references, then the controller deletes the replaced revisions. If persistence fails, a fresh reconcile can still read the originals and recreate or verify the exact replacements by their deterministic names. Missing snapshots return an error; upgrades never choose a replacement by listing eligible revisions.

Upgrade cleanup is best effort: a failed delete is logged and is not retried after the new references have been persisted. The revision's VM owner reference lets Kubernetes garbage collection remove it when the VM is deleted.

## Existing Exception: Expand

The instancetype Expand flow makes direct API calls to the VM:

```go
func (c *controller) handleExpand(vm *virtv1.VirtualMachine, referencePolicy virtv1.InstancetypeReferencePolicy) (*virtv1.VirtualMachine, error) {
    // ... expand instancetype/preference into VM Spec ...

    updatedVM, err := c.clientset.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
    // ... continue ...
}
```

Expand replaces matcher references with inline definitions and clears their status references. This PR retains its existing `Update()` and `UpdateStatus()` calls, followed by snapshot deletion after both writes succeed. Immediate persistence is not a requirement of expansion itself.

Moving Expand under the controller's persistence is follow-up work. It needs to preserve the expanded spec and cleared status references across retries after either write fails, and delete snapshots only once neither the persisted spec nor status needs them. The current two writes are separate API operations.

## Benefits of This Pattern

1. **Separation of Concerns**: Business logic (synchronizers) is decoupled from API operations (controller)
2. **Testability**: Synchronizers can be tested without mocking Kubernetes clients
3. **Consistency**: All synchronizers follow the same pattern
4. **Clarity**: It's clear who is responsible for what
5. **Avoids Conflicts**: No intermediate API calls reduce ResourceVersion conflicts
6. **Follows Kubernetes Patterns**: Status as a subresource with dedicated `UpdateStatus()` calls

## Common Pitfalls

### Don't: Make API calls in synchronizers

```go
// WRONG - Don't do this
func (s *MySynchronizer) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
    vm.Status.SomeField = "value"

    // This violates the contract
    updatedVM, err := s.client.VirtualMachine(vm.Namespace).UpdateStatus(...)
    return updatedVM, err
}
```

### Don't: Short-circuit the controller flow on synchronizer changes

```go
// WRONG - in the controller's orchestration of synchronizers
syncedVM, err := c.firstSynchronizer.Sync(vm, vmi)
if err != nil {
    return vm, vmi, handleSynchronizerErr(err), nil
}
if !equality.Semantic.DeepEqual(vm, syncedVM) {
    return syncedVM, vmi, nil, nil // Skips the remaining synchronizers
}
// ... remaining synchronizers ...
```

The controller should continue through the remaining synchronizers after a successful call and accumulate their changes before persistence. A guard clause returning `vm, nil` inside an individual synchronizer is fine: it returns control to the controller, which can call the next synchronizer.

### Do: Modify local object and return

```go
// CORRECT
func (s *MySynchronizer) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
    if !needsUpdate(vm) {
        return vm, nil  // No changes needed
    }

    vm.Status.SomeField = computeValue(vm, vmi)
    return vm, nil  // Return the modified object supplied by the controller
}
```

## Historical Context

Prior to this refactoring, synchronizers made direct API calls:

```go
// OLD PATTERN (before refactoring)
func (h *revisionHandler) Store(vm *virtv1.VirtualMachine) error {
    // Set Status locally
    vm.Status.InstancetypeRef = statusRef

    // Direct API call
    patchedVM, err := h.virtClient.VirtualMachine(vm.Namespace).PatchStatus(...)

    // Update local object with API response
    vm.ObjectMeta = patchedVM.ObjectMeta
    return err
}
```

This pattern had several issues:

- Synchronizers were tightly coupled to Kubernetes clients
- Intermediate API calls could cause ResourceVersion conflicts
- Testing required complex client mocking
- Unclear responsibility boundaries

The new pattern resolves these issues by establishing clear contracts and responsibilities.

## References

- VirtualMachine controller: `pkg/virt-controller/watch/vm/vm.go`
- Instancetype controller: `pkg/instancetype/controller/vm/controller.go`
- Network synchronizer: `pkg/network/controllers/vm.go`
