# Software Test Plan: CNV-71612

## Restore: Predictable PVC name

---

## Section I: Motivation and Requirements Review

### 1.1 Requirements Review Checklist

- [x] **User stories defined** -- Two user stories covering default and custom PVC naming
- [x] **Acceptance criteria identifiable** -- Restore with original names, restore with custom names
- [x] **Component scope clear** -- Storage Platform, snapshot/restore subsystem
- [ ] **Edge cases documented** -- Duplicate names, validation rules not specified
- [x] **Non-requirements listed** -- Explicitly called out in ticket

### 1.2 Known Limitations

- Override validation rules are not specified in the ticket
- Behavior when target PVC name already exists is undefined
- Maximum name length constraints not documented

### 1.3 Technology Review

- **API:** VirtualMachineRestore CRD with new `volumeOverrides` field
- **Storage:** PersistentVolumeClaim creation during restore workflow
- **Snapshot:** VirtualMachineSnapshot and VirtualMachineSnapshotContent CRDs
- **Platform:** OpenShift Virtualization 4.19+

---

## Section II: Software Test Plan

### 2.1 Scope

This test plan covers the predictable PVC naming feature for VM snapshot
restore operations. Testing focuses on:

- Default behavior (PVC names match source)
- Custom PVC name overrides via volumeOverrides
- Annotation and Label overrides on restored PVCs
- Validation and error handling for invalid overrides
- Interaction with existing PVCs and storage classes

### 2.2 Goals

- Verify users can restore VMs with predictable (unchanged) PVC names
- Verify users can specify custom PVC names during restore
- Verify Annotation/Label overrides are applied correctly
- Verify proper error handling for invalid or conflicting names
- Ensure backward compatibility with existing restore workflows

### 2.3 Test Strategy

| Tier | Framework | Scope |
|------|-----------|-------|
| Tier 1 (Functional) | Go/Ginkgo | API-level restore with overrides |
| Tier 2 (End-to-End) | Python/pytest | Full workflow: create VM, snapshot, restore with overrides |

### 2.4 Environment Requirements

- OpenShift 4.19+ cluster with CNV installed
- Storage class supporting dynamic provisioning (e.g., ocs-storagecluster-ceph-rbd)
- At least 2 worker nodes for live migration testing
- VirtualMachineSnapshot feature gate enabled

### 2.5 Risks

- **Storage class compatibility** -- Some storage providers may handle PVC
  naming differently
- **Concurrent restores** -- Race conditions when multiple restores target
  the same PVC names
- **Name collision** -- Behavior when a PVC with the target name already
  exists needs validation

---

## Section III: Test Scenarios and Traceability

### Requirements to Test Scenarios Mapping

- **[CNV-71612]** -- Restore VM from snapshot with predictable PVC names

  - **Scenario 1:** Restore VM with default PVC names (no overrides)
    - Priority: Critical
    - Tier: Tier 1, Tier 2

  - **Scenario 2:** Restore VM with custom PVC names via volumeOverrides
    - Priority: Critical
    - Tier: Tier 1, Tier 2

  - **Scenario 3:** Restore VM with Annotation overrides on PVCs
    - Priority: Major
    - Tier: Tier 1

  - **Scenario 4:** Restore VM with Label overrides on PVCs
    - Priority: Major
    - Tier: Tier 1

  - **Scenario 5:** Restore with duplicate/conflicting PVC name (negative)
    - Priority: Major
    - Tier: Tier 1

  - **Scenario 6:** Restore with invalid PVC name format (negative)
    - Priority: Major
    - Tier: Tier 1

  - **Scenario 7:** Restore VM with multiple volumes, each with custom name
    - Priority: Critical
    - Tier: Tier 1, Tier 2

  - **Scenario 8:** Backward compatibility -- restore without volumeOverrides field
    - Priority: Critical
    - Tier: Tier 1

  - **Scenario 9:** Restore with PVC name exceeding Kubernetes length limit
    - Priority: Minor
    - Tier: Tier 1

  - **Scenario 10:** End-to-end: create VM, write data, snapshot, restore with custom names, verify data
    - Priority: Critical
    - Tier: Tier 2

---

## Section IV: Sign-off and Approval

| Role | Name | Date | Status |
|------|------|------|--------|
| QE Lead | | | Pending |
| Dev Lead | | | Pending |
| PM | | | Pending |
