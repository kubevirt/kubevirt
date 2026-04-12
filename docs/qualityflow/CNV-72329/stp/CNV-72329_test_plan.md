# Openshift-virtualization-tests Test plan

## **Support Changing the VM Attached Network NAD Ref Using Hotplug - Quality Engineering Plan**

### **Metadata & Tracking**

| Field                  | Details                                                           |
|:-----------------------|:------------------------------------------------------------------|
| **Enhancement(s)**     | [VEP #140 / kubevirt#16412](https://github.com/kubevirt/kubevirt/pull/16412) |
| **Feature in Jira**    | [VIRTSTRAT-560](https://issues.redhat.com/browse/VIRTSTRAT-560) - Allow changing network VLAN on the fly |
| **Jira Tracking**      | Epic: [CNV-72329](https://issues.redhat.com/browse/CNV-72329), Parent: [VIRTSTRAT-560](https://issues.redhat.com/browse/VIRTSTRAT-560) |
| **QE Owner(s)**        | TBD                                                               |
| **Owning SIG**         | sig-network                                                       |
| **Participating SIGs** | sig-compute (VM controller restart-required logic)                |
| **Current Status**     | Draft                                                             |

**Document Conventions (if applicable):** NAD = NetworkAttachmentDefinition; VMI = VirtualMachineInstance; NIC = Network Interface Card; VLAN = Virtual LAN; VEP = Virtualization Enhancement Proposal

### **Feature Overview**

This feature allows VM administrators to change the NetworkAttachmentDefinition (NAD) reference on a running VM's secondary network interface without requiring a VM restart. Instead, the change is applied through a live migration. This enables scenarios such as moving a VM to a different VLAN or network segment on the fly. The feature is gated behind the `LiveUpdateNADRef` feature gate (with graduation to default-on tracked in [kubevirt#17049](https://github.com/kubevirt/kubevirt/pull/17049)) and is supported exclusively for bridge-binding interfaces. Guest interface properties (MAC address, interface name) are preserved during the change, ensuring the guest OS is not disrupted.

**Key implementation points** (from PR analysis):

- `haveCurrentNetsChanged()` in `pkg/network/vmliveupdate/restart.go` uses `reflect.DeepEqual` on Network objects; the PR modifies this to exempt bridge-bound NAD ref changes when `LiveUpdateNADRef` is enabled, avoiding unnecessary restarts
- `applyDynamicIfaceRequestOnVMI()` in `pkg/network/controllers/vm.go` gains a new path for NAD ref updates on existing interfaces
- `shouldVMIBeMarkedForAutoMigration()` in `pkg/network/migration/evaluator.go` is extended to recognize NAD ref changes as migration triggers
- New feature gate constants added in `pkg/virt-config/feature-gates.go` and `pkg/virt-config/featuregate/active.go`

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

| Check                                  | Done | Details/Notes                                                                                                                                                                           | Comments |
|:---------------------------------------|:-----|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:---------|
| **Review Requirements**                | [ ]  | Reviewed the relevant requirements.                                                                                                                                                     | CNV-72329 defines a clear user story: swap guest uplink from one network to another without the VM noticing. Parent VIRTSTRAT-560 frames the value as "Allow changing network VLAN on the fly." Linked issues: CNV-71605 (RFE), CNV-60118 (Reattach VM interface to different network while running), CNV-78912 (user feedback visibility gap). |
| **Understand Value**                   | [ ]  | Confirmed clear user stories and understood.  <br/>Understand the difference between U/S and D/S requirements<br/> **What is the value of the feature for RH customers**.               | Customers can move VMs between VLANs or network segments without downtime, reducing maintenance windows and enabling dynamic network reconfiguration. Upstream user story: "As a VM admin, I want to swap the guests uplink from one network to another without the VM noticing." |
| **Customer Use Cases**                 | [ ]  | Ensured requirements contain relevant **customer use cases**.                                                                                                                           | Primary use case: VM admin changes VLAN assignment by updating the NAD reference, avoiding VM restart. Applies to datacenter migrations, network segmentation changes, and maintenance operations. |
| **Testability**                        | [ ]  | Confirmed requirements are **testable and unambiguous**.                                                                                                                                | All acceptance criteria are testable: NAD ref change triggers live migration (not restart), connectivity verification on new network, bridge-only support, feature gate control, property preservation. E2e test already exists in PR #16412 at `tests/network/nad_live_update.go`. |
| **Acceptance Criteria**                | [ ]  | Ensured acceptance criteria are **defined clearly** (clear user stories; D/S requirements clearly defined in Jira).                                                                     | Acceptance criteria derived from the epic description, VEP #140 design, and PR implementation: (1) NAD ref change without restart, (2) connectivity to new network, (3) bridge-only support, (4) feature gate controls behavior, (5) guest properties preserved. Error handling for non-existent NAD is implied but not formally documented. CNV-78912 (user feedback) still in design. |
| **Non-Functional Requirements (NFRs)** | [ ]  | Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs. | No explicit performance or scalability NFRs defined. Downtime is implicitly zero (live migration, not restart). Monitoring/alerts not specified. User feedback visibility tracked as CNV-78912. Documentation subtask CNV-72336 exists but is in New status. |

#### **2. Technology and Design Review**

| Check                            | Done | Details/Notes                                                                                                                                           | Comments |
|:---------------------------------|:-----|:--------------------------------------------------------------------------------------------------------------------------------------------------------|:---------|
| **Developer Handoff/QE Kickoff** | [ ]  | A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.** | QE kickoff should be scheduled during design phase. VEP #140 provides the design document. PR kubevirt/kubevirt#16412 is the implementation (merged). Subtask CNV-72331 tracks upstream design. |
| **Technology Challenges**        | [ ]  | Identified potential testing challenges related to the underlying technology.                                                                            | `haveCurrentNetsChanged()` at restart.go uses `reflect.DeepEqual` on Network objects; the PR modifies this logic to exempt bridge-bound NAD ref changes gated by `LiveUpdateNADRef`. `applyDynamicIfaceRequestOnVMI` now handles NAD ref updates on existing interfaces. `shouldVMIBeMarkedForAutoMigration()` in evaluator.go is extended to trigger migration for NAD ref changes. The migration-based approach (vs. in-place update) must be thoroughly tested. |
| **Test Environment Needs**       | [ ]  | Determined necessary **test environment setups and tools**.                                                                                              | Requires multi-node cluster with Multus and bridge CNI. Multiple NADs with different configurations (different VLANs/network segments) needed. Standard CNV test infrastructure sufficient. RWX storage required for live migration. |
| **API Extensions**               | [ ]  | Reviewed new or modified APIs and their impact on testing.                                                                                               | VM spec `Network.Multus.NetworkName` field becomes live-updatable for bridge-binding interfaces. The migration evaluator now considers NAD ref changes as migration triggers. New `LiveUpdateNADRef` feature gate added to HyperConverged CR (graduation PR #17049 removes the gate). |
| **Topology Considerations**      | [ ]  | Evaluated multi-cluster, network topology, and architectural impacts.                                                                                    | Multi-node cluster required for live migration scenarios. Bridge-based NADs must be available on all schedulable worker nodes. Single-cluster topology is sufficient. |

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

This test plan covers the ability for a VM administrator to change the NetworkAttachmentDefinition (NAD) reference on a running VM's bridge-bound network interface. The change is applied through a live migration rather than a VM restart. Testing validates that the VM connects to the new network after migration, guest interface properties are preserved, the feature gate controls availability, unsupported binding types are rejected, and existing NIC hotplug/unplug operations remain unaffected.

**Testing Goals**

- **P0:** Verify that a NAD reference change on a bridge-bound interface takes effect on a running VM through live migration without restart
- **P0:** Verify that the feature gate `LiveUpdateNADRef` controls whether NAD ref changes are live-applied or require restart
- **P1:** Verify that guest interface properties (MAC address, interface name) are preserved after a NAD ref change and migration
- **P1:** Verify that NAD ref changes are rejected for non-bridge bindings (masquerade, SRIOV)
- **P1:** Verify that existing NIC hotplug and unplug operations are not disrupted when the feature gate is enabled
- **P1:** Verify that a VM retains connectivity to peers on the new network after migration
- **P1:** Verify that VMI status accurately reflects the new NAD after the change
- **P2:** Verify that multiple sequential NAD ref changes are handled correctly (each triggers migration)
- **P2:** Verify graceful error handling when a non-existent NAD is referenced
- **P2:** Verify that concurrent hotplug and NAD ref change operations complete without conflict

**Out of Scope (Testing Scope Exclusions)**

| Out-of-Scope Item | Rationale | PM/ Lead Agreement |
|:-------------------|:----------|:-------------------|
| SRIOV binding NAD ref changes | Only bridge binding is supported in this release | [ ] Name/Date |
| Masquerade binding NAD ref changes | Only bridge binding is supported in this release | [ ] Name/Date |
| Binding plugin NAD ref changes | Only bridge binding is supported in this release | [ ] Name/Date |
| Multus CNI internals | CNI plugin behavior is tested by the network platform team | [ ] Name/Date |
| VLAN configuration at the CNI level | VLAN is a CNI-level concept; we test NAD ref changes, not CNI configuration | [ ] Name/Date |
| User feedback visibility (CNV-78912) | Tracked as a separate linked issue with its own design | [ ] Name/Date |
| Multus Dynamic Networks Controller interaction | PR #16412 explicitly notes this is not tested with Dynamic Networks Controller | [ ] Name/Date |

#### **2. Test Strategy**

| Item                           | Description                                                                                                                                                  | Applicable (Y/N or N/A) | Comments |
|:-------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------|:------------------------|:---------|
| Functional Testing             | Validates that the feature works according to specified requirements and user stories                                                                        | Y | Core NAD ref change via migration, feature gate behavior, binding type validation, property preservation, error handling |
| Automation Testing             | Ensures test cases are automated for continuous integration and regression coverage                                                                          | Y | All test scenarios will be automated in Ginkgo (Tier 1) and pytest (Tier 2). Upstream e2e test exists in `tests/network/nad_live_update.go` |
| Performance Testing            | Validates feature performance meets requirements (latency, throughput, resource usage)                                                                       | N/A | No performance NFRs defined; NAD ref change triggers a one-time migration, not a throughput-sensitive path |
| Security Testing               | Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning                                                              | N/A | No new RBAC roles or security boundaries introduced; uses existing VM update permissions |
| Usability Testing              | Validates user experience, UI/UX consistency, and accessibility requirements. Does the feature require UI? If so, ensure the UI aligns with the requirements | N/A | No UI component; API-only feature |
| Compatibility Testing          | Ensures feature works across supported platforms, versions, and configurations                                                                               | Y | Verify bridge binding works across supported OCP versions; verify coexistence with existing hotplug |
| Regression Testing             | Verifies that new changes do not break existing functionality                                                                                                | Y | Existing NIC hotplug/unplug must continue working; non-NAD network property changes must still require restart. Code analysis shows `IsRestartRequired()` is called from `pkg/virt-controller/watch/vm/vm.go:3033` -- changes to restart logic must not regress existing VM lifecycle behavior |
| Upgrade Testing                | Validates upgrade paths from previous versions, data migration, and configuration preservation                                                               | N/A | Feature is a one-time spec patch operation triggering migration; no persistent state requiring conversion across upgrades. Subtask CNV-72333 tracks upgrade considerations |
| Backward Compatibility Testing | Ensures feature maintains compatibility with previous API versions and configurations                                                                        | Y | With gate disabled, behavior must match pre-feature behavior (NAD ref change triggers restart). PR #17049 (gate graduation) changes this -- when gate is removed, live update becomes default |
| Dependencies                   | Dependent on deliverables from other components/products? Identify what is tested by which team.                                                             | N/A | No other team must deliver new components; Multus and bridge CNI are pre-existing platform infrastructure |
| Cross Integrations             | Does the feature affect other features/require testing by other components? Identify what is tested by which team.                                           | Y | Live migration (sig-compute): NAD ref change triggers migration; migration evaluator extended. NIC hotplug (sig-network): shared controller sync path in `pkg/network/controllers/vm.go` must be regression-tested |
| Monitoring                     | Does the feature require metrics and/or alerts?                                                                                                              | N/A | No new metrics or alerts defined for this feature |
| Cloud Testing                  | Does the feature require multi-cloud platform testing? Consider cloud-specific features.                                                                     | N/A | Bridge-based networking; not cloud-specific |

#### **3. Test Environment**

| Environment Component                         | Configuration | Specification Examples |
|:----------------------------------------------|:--------------|:-----------------------|
| **Cluster Topology**                          | Multi-node cluster, minimum 2 schedulable worker nodes | Required for live migration scenarios after NAD ref change |
| **OCP & OpenShift Virtualization Version(s)** | OCP 4.22+, OpenShift Virtualization 4.22+ | Derived from CNV-72329 target version |
| **CPU Virtualization**                        | Standard (Intel VT-x / AMD-V) | No special CPU features required |
| **Compute Resources**                         | Sufficient for running 2+ VMs concurrently with migration capacity | Standard test cluster sizing |
| **Special Hardware**                          | N/A | No special hardware required; bridge networking uses software bridges |
| **Storage**                                   | ocs-storagecluster-ceph-rbd | RWX storage for live migration support |
| **Network**                                   | OVN-Kubernetes with Multus CNI; multiple bridge-based NADs configured with different network segments | At least 3 bridge NADs for sequential change testing |
| **Required Operators**                        | OpenShift Virtualization (openshift-cnv), HyperConverged Cluster Operator | Standard CNV operator stack |
| **Platform**                                  | OCP (bare metal or virtualized) | Bridge networking available on all supported platforms |
| **Special Configurations**                    | `LiveUpdateNADRef` feature gate enabled on HyperConverged CR; `WorkloadUpdateMethods` includes `LiveMigrate` | Feature gate must be toggled for gate-control tests |

#### **3.1. Testing Tools & Frameworks**

| Category           | Tools/Frameworks |
|:-------------------|:-----------------|
| **Test Framework** | Ginkgo v2 + Gomega (Tier 1), pytest (Tier 2) |
| **CI/CD**          | Prow |
| **Other Tools**    | virtctl, oc, kubectl |

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] VEP #140 design document is finalized and approved
- [ ] PR kubevirt/kubevirt#16412 is merged and included in the target build (DONE - merged)
- [ ] `LiveUpdateNADRef` feature gate is available in HyperConverged CR
- [ ] At least 3 bridge-based NADs are deployed on the test cluster
- [ ] Multi-node cluster with live migration capability is available
- [ ] RWX-capable storage class is configured

#### **5. Risks**

| Risk Category        | Specific Risk for This Feature | Mitigation Strategy | Status |
|:---------------------|:-------------------------------|:--------------------|:-------|
| Timeline/Schedule    | Feature gate graduation PR #17049 is still open; behavior may change if gate is removed before release | Track PR status weekly; test both gated and ungated behavior | [ ] |
| Test Coverage        | User feedback visibility (CNV-78912) is not yet designed; may affect how test results are verified | Coordinate with dev on CNV-78912 timeline; use VMI status and migration completion as interim verification methods | [ ] |
| Test Coverage        | Multus Dynamic Networks Controller interaction is explicitly untested upstream | Document as known gap; evaluate if downstream testing is needed | [ ] |
| Test Environment     | Bridge NADs must be pre-configured with distinct network segments to verify actual connectivity changes | Document NAD setup in test environment provisioning scripts | [ ] |
| Untestable Aspects   | Internal controller reconciliation timing and migration trigger timing are not directly observable; only the end result can be verified | Test observable outcomes (VM on new network post-migration) with appropriate wait conditions | [ ] |
| Resource Constraints | N/A | N/A | [ ] |
| Dependencies         | N/A | N/A | [ ] |
| Other                | Concurrent operations (hotplug + NAD ref change) may have race conditions that are timing-dependent in `applyDynamicIfaceRequestOnVMI()` | Use retry logic and condition-based waits rather than fixed timeouts | [ ] |

#### **6. Known Limitations**

- Only bridge binding is supported for NAD ref changes; SRIOV, masquerade, and binding plugin interfaces are not supported
- NAD ref changes are applied through live migration, not in-place; a migratable VM with RWX storage is required
- User feedback visibility for NAD ref change completion is limited (tracked as CNV-78912)
- Non-NAD network property changes (e.g., changing the binding type itself) still require a VM restart
- The feature requires the `LiveUpdateNADRef` feature gate to be explicitly enabled (until gate graduation via PR #17049)
- Feature is not tested with Multus Dynamic Networks Controller (per PR #16412 note)

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

| Requirement ID | Requirement Summary | Test Scenario(s) | Tier | Priority |
|:---------------|:--------------------|:-----------------|:-----|:---------|
| CNV-72329 | As a VM admin, I want to change the NAD reference on a running VM without restarting it | Verify VM connects to new network after NAD ref change via live migration without restart | Tier 1 | P0 |
| CNV-72329 | As a VM admin, I want the LiveUpdateNADRef feature gate to control whether NAD ref changes are live-applied | Verify NAD ref change applies via migration when gate enabled | Tier 1 | P0 |
| CNV-72329 | As a VM admin, I want the LiveUpdateNADRef feature gate to control whether NAD ref changes are live-applied | Verify NAD ref change requires restart when gate disabled | Tier 1 | P0 |
| CNV-72329 | As a VM admin, I want guest interface properties preserved after NAD ref change so the guest OS is not disrupted | Verify MAC address unchanged after NAD ref change and migration | Tier 1 | P1 |
| CNV-72329 | As a VM admin, I want guest interface properties preserved after NAD ref change so the guest OS is not disrupted | Verify guest interface name unchanged after NAD ref change and migration | Tier 1 | P1 |
| CNV-72329 | As a VM admin, I want NAD ref changes rejected for non-bridge bindings so only supported configurations are allowed | Verify NAD ref change rejected for masquerade binding | Tier 1 | P1 |
| CNV-72329 | As a VM admin, I want NAD ref changes rejected for non-bridge bindings so only supported configurations are allowed | Verify NAD ref change rejected for SRIOV binding | Tier 1 | P1 |
| CNV-72329 | As a VM admin, I want NAD ref changes rejected for non-bridge bindings so only supported configurations are allowed | Verify NAD ref change succeeds for bridge binding | Tier 1 | P1 |
| CNV-72329 | As a VM admin, I want existing NIC hotplug/unplug to work when the LiveUpdateNADRef gate is enabled | Verify NIC hotplug succeeds with feature gate enabled | Tier 1 | P1 |
| CNV-72329 | As a VM admin, I want existing NIC hotplug/unplug to work when the LiveUpdateNADRef gate is enabled | Verify NIC unplug succeeds with feature gate enabled | Tier 1 | P1 |
| CNV-72329 | As a VM admin, I want the VMI status to reflect the new network after a NAD ref change | Verify VMI status shows updated NAD name after migration | Tier 1 | P1 |
| CNV-72329 | As a VM admin, I want the VM to reach peers on the new network after changing the NAD reference | Verify connectivity to peer on new VLAN after NAD ref change and migration | Tier 2 | P1 |
| CNV-72329 | As a VM admin, I want to migrate a VM after changing its NAD reference and retain network connectivity | Verify second migration after NAD ref change succeeds and retains connectivity | Tier 2 | P1 |
| CNV-72329 | As a VM admin, I want to perform multiple sequential NAD ref changes on a running VM | Verify sequential NAD ref changes each trigger migration and take effect | Tier 2 | P2 |
| CNV-72329 | As a VM admin, I want a clear error when I reference a non-existent NAD | Verify error when target NAD does not exist | Tier 1 | P2 |
| CNV-72329 | As a VM admin, I want a clear error when I reference a non-existent NAD | Verify VM stays on original network after failed change | Tier 1 | P2 |
| CNV-72329 | As a VM admin, I want concurrent hotplug and NAD ref change operations to complete without conflict | Verify concurrent hotplug and NAD ref change both complete | Tier 2 | P2 |
| CNV-72329-REG | Regression: Existing NIC hotplug must not be affected by restart logic changes | Verify bridge NIC hotplug still triggers migration correctly with new restart.go logic | Tier 1 | P1 |
| CNV-72329-REG | Regression: Non-NAD network changes must still require restart | Verify changing binding type on existing interface still requires restart | Tier 1 | P1 |

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD]
  - [TBD]
* **Approvers:**
  - [TBD]
  - [TBD]
