# VirtualMachine Controller Synchronizer Interface

This document describes the synchronizer interface contract used by the VirtualMachine controller for managing Spec and Status modifications.

## Overview

The VirtualMachine controller uses a synchronizer pattern to separate business logic from API persistence. Synchronizers are responsible for computing desired state changes, while the controller handles all Kubernetes API interactions.

## Synchronizer Interface Contract

All synchronizers in the VirtualMachine controller must follow this contract:

```go
type synchronizer interface {
    Sync(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
}
```

### Contract Requirements

1. **MUST only modify the local VM object** passed as a parameter (both Spec and Status fields are allowed)
2. **MUST NOT make direct API calls** to `Update()`, `UpdateStatus()`, `Patch()`, or `PatchStatus()` on the VirtualMachine
3. **MUST return the modified VM object** (or the original if no changes were made)
4. **MAY return an error** if synchronization fails

### Responsibility Separation

The synchronizer is responsible for:

- Computing desired state based on current VM and VMI state
- Modifying the local VM object (DeepCopy if needed)
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
    // Create a copy if modifications are needed
    vmCopy := vm.DeepCopy()

    // Compute desired state
    desiredValue := s.computeValue(vm, vmi)

    // Modify the local copy
    vmCopy.Status.SomeField = desiredValue

    // Return the modified VM
    return vmCopy, nil
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

## Current Synchronizers

### 1. Instancetype Synchronizer

**Location**: `pkg/instancetype/controller/vm/controller.go`

**Purpose**: Manages instancetype and preference ControllerRevisions

**Modifications**:

- **Status**: Sets `vm.Status.InstancetypeRef` and `vm.Status.PreferenceRef` with ControllerRevision references
- **Spec**: May modify matchers via the Expand flow

**Special considerations**: Must run early in the sync flow because `startVMI()` calls `ApplyToVMI()` which depends on Status refs being populated.

### 2. Network Synchronizer

**Location**: `pkg/network/admitter/admitter.go`

**Purpose**: Manages network interface configuration

**Modifications**:

- **Spec**: Adds default network interfaces if not present
- **Status**: Updates network-related status fields

### 3. Firmware Synchronizer

**Location**: `pkg/virt-controller/watch/vm/firmware.go`

**Purpose**: Manages firmware UUID generation

**Modifications**:

- **Spec**: Sets `vm.Spec.Template.Spec.Domain.Firmware.UUID` if not present

## Acceptable Exceptions

The synchronizer contract has specific exceptions for operations that require immediate persistence:

### ControllerRevision Management

Instancetype synchronizers may create and delete ControllerRevisions as side effects during `Sync()`:

```go
func (c *controller) Store(vm *virtv1.VirtualMachine) error {
    // Calls storeControllerRevision() which creates ControllerRevision via API
    _, err := h.storeInstancetypeRevision(vm)

    // Updates local Status refs (no API call)
    vm.Status.InstancetypeRef = statusRef

    return nil
}
```

**Rationale**: ControllerRevisions are immutable snapshots that need to exist before Status refs can point to them. Creating them as side effects is acceptable.

### Expand Flow

The instancetype Expand flow makes direct API calls to the VM:

```go
func (c *controller) handleExpand(vm *virtv1.VirtualMachine, referencePolicy virtv1.InstancetypeReferencePolicy) (*virtv1.VirtualMachine, error) {
    // ... expand instancetype/preference into VM Spec ...

    updatedVM, err := c.virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
    // ... continue ...
}
```

**Rationale**: Expand is an intentionally destructive and irreversible operation that replaces matcher references with inline definitions. Immediate persistence ensures the operation is atomic.

## Benefits of This Pattern

1. **Separation of Concerns**: Business logic (synchronizers) is decoupled from API operations (controller)
2. **Testability**: Synchronizers can be tested without mocking Kubernetes clients
3. **Consistency**: All synchronizers follow the same pattern
4. **Clarity**: It's clear who is responsible for what
5. **Avoids Conflicts**: No intermediate API calls reduce ResourceVersion conflicts
6. **Follows Kubernetes Patterns**: Status as a subresource with dedicated `UpdateStatus()` calls

## Common Pitfalls

### ❌ Don't: Make API calls in synchronizers

```go
// WRONG - Don't do this
func (s *MySynchronizer) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
    vm.Status.SomeField = "value"

    // ❌ This violates the contract
    updatedVM, err := s.client.VirtualMachine(vm.Namespace).UpdateStatus(...)
    return updatedVM, err
}
```

### ❌ Don't: Return early without modifications

```go
// WRONG - Don't do this
func (s *MySynchronizer) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
    if someCondition {
        // ❌ This prevents other synchronizers from running
        return vm, nil
    }
    // ...
}
```

The controller should not return early based on synchronizer changes. All synchronizers should run in sequence, and the controller handles persistence at the end.

### ✅ Do: Modify local object and return

```go
// CORRECT
func (s *MySynchronizer) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
    if !needsUpdate(vm) {
        return vm, nil  // No changes needed
    }

    vmCopy := vm.DeepCopy()
    vmCopy.Status.SomeField = computeValue(vm, vmi)
    return vmCopy, nil  // ✅ Return modified local copy
}
```

## Historical Context

Prior to this refactoring, synchronizers made direct API calls:

```go
// OLD PATTERN (before refactoring)
func (h *revisionHandler) Store(vm *virtv1.VirtualMachine) error {
    // Set Status locally
    vm.Status.InstancetypeRef = statusRef

    // ❌ Direct API call
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
- Firmware controller: `pkg/virt-controller/watch/vm/firmware.go`
- Network synchronizer: `pkg/network/admitter/admitter.go`
