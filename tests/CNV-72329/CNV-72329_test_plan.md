# Openshift-virtualization-tests Test plan

## **Support changing the VM attached network NAD ref using hotplug - Quality Engineering Plan**

### **Metadata & Tracking**

| Field | Details |
|:------|:--------|
| **Enhancement(s)** | [VEP 140: Live update NAD reference](https://github.com/kubevirt/enhancements/issues/140) |
| **Feature in Jira** | [CNV-72329](https://issues.redhat.com/browse/CNV-72329) |
| **Jira Tracking** | [CNV-72329](https://issues.redhat.com/browse/CNV-72329) |
| **QE Owner(s)** | TBD |
| **Owning SIG** | sig/network |
| **Participating SIGs** | sig/compute |
| **Current Status** | Draft |

### **Related GitHub Pull Requests**

| PR Link | Repository | Source Jira Issue | Source Type | Description |
|:--------|:-----------|:------------------|:------------|:------------|
| [#16412](https://github.com/kubevirt/kubevirt/pull/16412) | kubevirt/kubevirt | CNV-72329 | GitHub Search | Implement Live Update of NAD Reference |
| [#138](https://github.com/kubevirt/enhancements/pull/138) | kubevirt/enhancements | CNV-72329 | GitHub Search | VEP 140: Live update NAD reference (Design) |

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value, technology, and testability prior to formal test planning.

#### **1. Requirement & User Story Review Checklist**

| Check | Done | Details/Notes | Comments |
|:------|:-----|:--------------|:---------|
| **Review Requirements** | [ ] | Allow customers to change the network a VM is connected to without rebooting | Feature enables dynamic network switching via NAD reference update |
| **Understand Value** | [ ] | Eliminates VM downtime for network changes | Users can change VLAN, network segment without guest disruption |
| **Customer Use Cases** | [ ] | VM admin swaps guest uplink between networks transparently | Better/worse link, different VLAN, isolated segment scenarios |
| **Testability** | [ ] | Requires cluster with multiple NADs and VMs | Live migration capability required for NAD update |
| **Acceptance Criteria** | [ ] | VM network reference changes without VM restart | Network connectivity preserved through change |
| **Non-Functional Requirements (NFRs)** | [ ] | Network switch should be transparent to guest workloads | TCP connections should survive the change |

#### **2. Technology and Design Review**

| Check | Done | Details/Notes | Comments |
|:------|:-----|:--------------|:---------|
| **Developer Handoff/QE Kickoff** | [ ] | PR #16412 in active development | Work in progress, coordinating with sig/network |
| **Technology Challenges** | [ ] | Uses live migration to implement NAD reference change | Migration evaluator updated to handle NAD updates |
| **Test Environment Needs** | [ ] | Multiple NADs with different network configurations | At least two NADs required for testing |
| **API Extensions** | [ ] | VM spec network interface NAD reference field | Feature gate: LiveUpdateNADRefEnabled |
| **Topology Considerations** | [ ] | Works with bridge and SR-IOV bindings | May have different behaviors per binding type |

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

This test plan covers the Live Update of NAD Reference feature, which allows users to change the NetworkAttachmentDefinition reference on a running VM's network interface. The change is implemented through live migration, ensuring guest workloads are not disrupted.

**In Scope:**

- NAD reference update on running VMs via VM spec modification
- Live migration triggered by NAD reference change
- Network connectivity verification after NAD change
- Feature gate enable/disable behavior
- Error handling for invalid NAD references
- Multi-interface VM scenarios
- Workload continuity during NAD update

#### **2. Testing Goals**

##### **Positive Use Cases (Happy Path)**

- Verify NAD reference can be updated on running VM
- Verify live migration completes successfully during NAD update
- Verify network connectivity preserved after NAD reference change
- Verify guest TCP connections survive NAD reference change
- Verify feature gate enables/disables NAD reference update capability
- Verify update works with multiple network interfaces

##### **Negative Use Cases (Error Handling & Edge Cases)**

- Verify appropriate error when referencing non-existent NAD
- Verify appropriate error when VM is not running
- Verify appropriate error when feature gate is disabled
- Verify behavior when NAD reference update fails mid-migration
- Verify rollback behavior on failed NAD update

#### **3. Non-Goals (Testing Scope Exclusions)**

| Non-Goal | Rationale | PM/ Lead Agreement |
|:---------|:----------|:-------------------|
| Testing network plugin internals (Multus, CNI) | Platform-level testing outside KubeVirt scope | [ ] |
| Performance benchmarking of migration speed | Not part of functional feature validation | [ ] |
| Testing NAD creation/deletion | NAD lifecycle is Multus responsibility | [ ] |
| Node-level network configuration | Infrastructure concern, not virtualization | [ ] |

#### **4. Test Strategy**

##### **A. Types of Testing**

| Item (Testing Type) | Applicable (Y/N or N/A) | Comments |
|:--------------------|:------------------------|:---------|
| **Functional Testing** | Y | Core feature validation - NAD reference update workflow |
| **Automation Testing** | Y | Test cases will be automated in kubevirt/tests |
| **Performance Testing** | N/A | Not performance-sensitive feature |
| **Security Testing** | N/A | No new security boundaries introduced |
| **Usability Testing** | N/A | API-only feature, no UI |
| **Compatibility Testing** | Y | Test with different network binding types |
| **Regression Testing** | Y | Ensure existing network hotplug not affected |
| **Upgrade Testing** | N/A | Feature under development, upgrade N/A for new feature |
| **Backward Compatibility Testing** | Y | Ensure VMs without feature gate work normally |

##### **B. Potential Areas to Consider**

| Item | Description | Applicable (Y/N or N/A) | Comment |
|:-----|:------------|:------------------------|:--------|
| **Dependencies** | Dependent on deliverables from other components/products? Identify what is tested by which team. | Y | Depends on Multus NAD support |
| **Monitoring** | Does the feature require metrics and/or alerts? | N | No new metrics required |
| **Cross Integrations** | Does the feature affect other features/require testing by other components? | Y | Interacts with live migration, network hotplug |
| **UI** | Does the feature require UI? If so, ensure the UI aligns with the requirements. | N | API-only feature |

#### **5. Test Environment**

| Environment Component | Configuration | Specification Examples |
|:----------------------|:--------------|:-----------------------|
| **Cluster Topology** | Multi-node cluster | 3+ nodes for migration testing |
| **OCP & OpenShift Virtualization Version(s)** | Latest development | OCP 4.19+, CNV development |
| **CPU Virtualization** | Standard | Intel VT-x / AMD-V |
| **Compute Resources** | Standard | 4+ vCPU, 8GB+ RAM per node |
| **Special Hardware** | Optional SR-IOV NICs | For SR-IOV binding testing |
| **Storage** | Shared storage | Required for live migration |
| **Network** | Multiple VLANs/networks | At least 2 NADs with different configs |
| **Required Operators** | CNV, SRIOV Network Operator (optional) | Standard CNV deployment |
| **Platform** | OpenShift | Bare metal or cloud |
| **Special Configurations** | LiveUpdateNADRefEnabled feature gate | Must be enabled for testing |

#### **5.5. Testing Tools & Frameworks**

| Category | Tools/Frameworks |
|:---------|:-----------------|
| **Test Framework** | Ginkgo/Gomega (Tier 1), pytest (Tier 2) |
| **CI/CD** | Prow, OpenShift CI |
| **Other Tools** | kubectl, virtctl |

#### **6. Entry Criteria**

The following conditions must be met before testing can begin:

- PR #16412 merged or available in test build
- LiveUpdateNADRefEnabled feature gate available
- Multi-node cluster with shared storage configured
- At least two NetworkAttachmentDefinitions available
- Live migration functional on test cluster

#### **7. Risks and Limitations**

| Risk Category | Specific Risk for This Feature | Mitigation Strategy | Status |
|:--------------|:-------------------------------|:--------------------|:-------|
| Timeline/Schedule | Feature PR still in development (WIP) | Monitor PR progress, coordinate with dev | [ ] |
| Test Coverage | Limited binding type combinations | Prioritize bridge, expand to SR-IOV | [ ] |
| Test Environment | Multiple NADs with different networks required | Pre-configure test clusters with standard NADs | [ ] |
| Untestable Aspects | Guest-level network verification during transition | Use long-running TCP connections as proxy | [ ] |
| Resource Constraints | Live migration requires shared storage | Ensure test clusters have proper storage | [ ] |
| Dependencies | Multus and CNI plugin behavior | Coordinate with network team on expected behaviors | [ ] |
| Other | Feature gate interactions | Test with gate enabled and disabled | [ ] |

#### **8. Known Limitations**

- Feature requires live migration to implement NAD change - VMs that cannot migrate cannot use this feature
- Network binding type must support the operation
- Guest OS must handle network device reconnection gracefully

---

### **III. Test Scenarios & Traceability**

This section provides a **high-level overview** of test scenarios mapped to requirements.

#### **1. Requirements-to-Tests Mapping**

| Requirement ID | Requirement Summary | Test Scenario(s) | Test Type(s) | Priority |
|:---------------|:--------------------|:-----------------|:-------------|:---------|
| CNV-72329-01 | NAD reference update triggers live migration | Verify NAD reference change initiates migration | Tier 1 (Functional) | P0 |
| CNV-72329-02 | Network connectivity preserved after NAD change | Verify VM network accessible after NAD update completes | Tier 1 (Functional) | P0 |
| CNV-72329-03 | Feature gate controls NAD update capability | Verify NAD update rejected when feature gate disabled | Tier 1 (Functional) | P1 |
| CNV-72329-04 | Invalid NAD reference rejected | Verify error returned for non-existent NAD reference | Tier 1 (Functional) | P1 |
| CNV-72329-05 | VM must be running for NAD update | Verify NAD update rejected for stopped VM | Tier 1 (Functional) | P1 |
| CNV-72329-06 | Multi-interface VM NAD update | Verify single interface NAD update on multi-interface VM | Tier 1 (Functional) | P1 |
| CNV-72329-07 | Workload continuity during NAD change | Verify TCP connection survives NAD reference change via migration | Tier 2 (End-to-End) | P0 |
| CNV-72329-08 | Complete NAD swap workflow | Verify VM creation -> connect -> change NAD -> verify connectivity | Tier 2 (End-to-End) | P0 |
| CNV-72329-09 | VLAN change use case | Verify VM network changes VLAN after NAD reference update | Tier 2 (End-to-End) | P1 |
| CNV-72329-10 | Failed migration rollback | Verify VM remains functional if NAD update migration fails | Tier 2 (End-to-End) | P1 |

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

- **Reviewers:**
  - [Name / @github-username]
  - [Name / @github-username]
- **Approvers:**
  - [Name / @github-username]
  - [Name / @github-username]
