# Openshift-virtualization-tests Test plan

## **CPU Hotplug vCPU Limit Enforcement - Quality Engineering Plan**

### **Metadata & Tracking**

| Field | Details |
|:------|:--------|
| **Enhancement(s)** | N/A (Bug Fix) |
| **Feature in Jira** | [CNV-57352](https://issues.redhat.com/browse/CNV-57352) |
| **Jira Tracking** | [CNV-61263](https://issues.redhat.com/browse/CNV-61263) |
| **QE Owner(s)** | Kedar Bidarkar |
| **Owning SIG** | sig/compute |
| **Participating SIGs** | N/A |
| **Current Status** | Closed |

### **Related GitHub Pull Requests**

| PR Link | Repository | Source Jira Issue | Source Type | Description |
|:--------|:-----------|:------------------|:------------|:------------|
| [#14338](https://github.com/kubevirt/kubevirt/pull/14338) | kubevirt/kubevirt | CNV-57352 | custom_field | Limit MaxSockets based on maximum of vcpus |
| [#14511](https://github.com/kubevirt/kubevirt/pull/14511) | kubevirt/kubevirt | CNV-57352 | custom_field | Cherry-pick to release-1.5 |

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value, technology, and testability prior to formal test planning.

#### **1. Requirement & User Story Review Checklist**

| Check | Done | Details/Notes | Comments |
|:------|:-----|:--------------|:---------|
| **Review Requirements** | [x] | CPU hotplug logic must not exceed machine type vCPU limits | Hotplug logic was multiplying CPU count by 4, exceeding the 710 hard limit |
| **Understand Value** | [x] | Allows VMs with high vCPU counts (64+) to start successfully | Critical for enterprise workloads requiring high CPU counts |
| **Customer Use Cases** | [x] | Customers with 64+ vCPU VMs were unable to start their VMs | Multiple internal and customer reports |
| **Testability** | [x] | Fully testable via VM creation with high vCPU counts | Requires host with sufficient CPUs to test high counts |
| **Acceptance Criteria** | [x] | VMs with high vCPU counts (e.g., 216 cores) start successfully | No "Maximum CPUs greater than specified machine type limit" error |
| **Non-Functional Requirements (NFRs)** | [x] | Performance at scale with high vCPU counts | Scale/Performance testing recommended |

#### **2. Technology and Design Review**

| Check | Done | Details/Notes | Comments |
|:------|:-----|:--------------|:---------|
| **Developer Handoff/QE Kickoff** | [x] | Fix limits MaxSockets to cap vCPUs at 512 | adjustedSockets = maxVCPUs / (cores * threads) |
| **Technology Challenges** | [x] | Machine type limits vary (710 for pc-q35-rhel9) | Upper bound of 512 chosen as safe default |
| **Test Environment Needs** | [x] | Host with 200+ CPUs for full validation | Multi-socket systems recommended |
| **API Extensions** | [x] | No API changes, defaults behavior change | MaxSockets auto-calculation modified |
| **Topology Considerations** | [x] | NUMA-aware guests may have additional constraints | guestMappingPassthrough affects socket topology |

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

This test plan covers the verification of the CPU hotplug vCPU limit enforcement fix. The fix ensures that when CPU hotplug is enabled, the calculated MaxSockets value does not result in total vCPUs exceeding 512, which prevents exceeding machine type limits.

**In Scope:**

- VM creation with high vCPU counts (64, 100, 178+, 216 cores)
- Verification of MaxSockets calculation with upper bound enforcement
- VM startup success with various CPU topologies (cores, sockets, threads combinations)
- Regression testing for standard CPU hotplug functionality

#### **2. Testing Goals**

##### **Positive Use Cases (Happy Path)**

- Verify VM with 216 cores, 1 socket, 1 thread starts successfully
- Verify VM with 100 sockets starts successfully (original reported case)
- Verify MaxSockets is correctly calculated to stay within 512 vCPU limit
- Verify CPU hotplug functionality works after fix (can add CPUs up to MaxSockets)
- Verify existing VMs with maxSockets already defined are not affected

##### **Negative Use Cases (Error Handling & Edge Cases)**

- Verify proper error handling when host has insufficient CPUs
- Verify behavior with NUMA-aware guests (guestMappingPassthrough)
- Verify edge case: cores * threads already exceeds 512
- Verify behavior when user explicitly sets maxSockets > calculated limit

#### **3. Non-Goals (Testing Scope Exclusions)**

| Non-Goal | Rationale | PM/ Lead Agreement |
|:---------|:----------|:-------------------|
| Performance benchmarking with 500+ vCPUs | Covered by Scale/Performance team | Agreed per QE comments |
| Testing every machine type limit | 512 upper bound covers common cases | Default safe limit chosen |
| libvirt/QEMU internal validation | RHEL component responsibility | Tracked in RHEL-65844 |

#### **4. Test Strategy**

##### **A. Types of Testing**

| Item (Testing Type) | Applicable (Y/N or N/A) | Comments |
|:--------------------|:------------------------|:---------|
| **Functional Testing** | Y | Core test: VM start with high vCPU counts |
| **Automation Testing** | Y | Add to existing CPU hotplug test suite |
| **Performance Testing** | N/A | Deferred to Scale/Performance team |
| **Security Testing** | N/A | No security implications |
| **Usability Testing** | N/A | No UI changes |
| **Compatibility Testing** | Y | Test across CNV versions (4.16.z, 4.17, 4.18, 4.19) |
| **Regression Testing** | Y | Ensure existing CPU hotplug scenarios still work |
| **Upgrade Testing** | Y | Verify fix persists after upgrade |
| **Backward Compatibility Testing** | Y | Existing VMs with maxSockets should work unchanged |

##### **B. Potential Areas to Consider**

| Item | Description | Applicable (Y/N or N/A) | Comment |
|:-----|:------------|:------------------------|:--------|
| **Dependencies** | libvirt version with eim support (RHEL-69724) | Y | libvirt-libs-10.0.0-6.15.el9_4 or later required for 4.16.z |
| **Monitoring** | No metrics/alerts needed | N/A | Bug fix, no monitoring changes |
| **Cross Integrations** | Live Migration with high vCPU VMs | Y | Should verify migration works with capped MaxSockets |
| **UI** | No UI changes | N/A | API/defaults change only |

#### **5. Test Environment**

| Environment Component | Configuration | Specification Examples |
|:----------------------|:--------------|:-----------------------|
| **Cluster Topology** | Multi-node cluster | 3+ node cluster with high-CPU workers |
| **OCP & OpenShift Virtualization Version(s)** | CNV 4.16.7+, 4.17.z, 4.18.z, 4.19.0+ | Verify fix in all supported versions |
| **CPU Virtualization** | Host with 200+ CPUs | 4-socket Cascade Lake or similar |
| **Compute Resources** | High memory nodes | 256GB+ RAM for large vCPU VMs |
| **Special Hardware** | Multi-socket NUMA systems | Required for 178+ vCPU testing |
| **Storage** | Standard | Container disk sufficient |
| **Network** | Pod networking | Default masquerade |
| **Required Operators** | OpenShift Virtualization | No additional operators |
| **Platform** | x86_64 | Intel/AMD with virtualization extensions |
| **Special Configurations** | None | Standard OCP installation |

#### **5.5. Testing Tools & Frameworks**

| Category | Tools/Frameworks |
|:---------|:-----------------|
| **Test Framework** | Ginkgo/Gomega (Tier 1), pytest (Tier 2) |
| **CI/CD** | Prow, OpenShift CI |
| **Other Tools** | virtctl, oc CLI |

#### **6. Entry Criteria**

The following conditions must be met before testing can begin:

- CNV build containing PR #14338 is available
- Test cluster with sufficient CPU resources (200+ CPUs recommended)
- libvirt version with eim auto-enable support (for 4.16.z)
- Access to multi-socket NUMA hardware for edge case testing

#### **7. Risks and Limitations**

| Risk Category | Specific Risk for This Feature | Mitigation Strategy | Status |
|:--------------|:-------------------------------|:--------------------|:-------|
| Timeline/Schedule | Z-stream backport timing | Track CNV-61665, CNV-61668 for z-stream status | [x] |
| Test Coverage | Cannot test all machine types | Use 512 upper bound as safe default | [x] |
| Test Environment | Insufficient CPU count on test hosts | Use Scale/Perf cluster resources | [ ] |
| Untestable Aspects | Machine type limits beyond 512 | Document as known limitation | [x] |
| Resource Constraints | High-CPU hosts are limited | Coordinate with lab team | [ ] |
| Dependencies | libvirt version alignment | Verify libvirt version in virt-launcher | [x] |
| Other | NUMA guests may have additional limits | Create separate test scenarios | [ ] |

#### **8. Known Limitations**

- The fix caps MaxSockets to keep total vCPUs at 512, even if the machine type supports more (e.g., 710 for pc-q35)
- NUMA-aware guests with guestMappingPassthrough may have additional socket topology constraints that could prevent starting even with the fix
- Hosts must have sufficient physical CPUs to schedule high-vCPU VMs

---

### **III. Test Scenarios & Traceability**

This section provides a **high-level overview** of test scenarios mapped to requirements.

#### **1. Requirements-to-Tests Mapping**

| Requirement ID | Requirement Summary | Test Scenario(s) | Test Type(s) | Priority |
|:---------------|:--------------------|:-----------------|:-------------|:---------|
| REQ-001 | VM with 216 cores must start successfully | TC-001: Create and start VM with 216 cores, 1 socket, 1 thread | Tier 2 | P1 |
| REQ-002 | VM with 100 sockets must start successfully | TC-002: Create and start VM with 1 core, 100 sockets, 1 thread | Tier 2 | P1 |
| REQ-003 | MaxSockets must not exceed 512 vCPU limit | TC-003: Verify MaxSockets calculation with 32 sockets, 2 cores, 3 threads (expected: 85) | Tier 1 | P1 |
| REQ-004 | CPU hotplug must work with capped MaxSockets | TC-004: Hotplug CPUs on VM with high core count | Tier 2 | P2 |
| REQ-005 | Explicit maxSockets should override default | TC-005: Create VM with explicit maxSockets=2 and 216 cores | Tier 2 | P2 |
| REQ-006 | Standard CPU hotplug regression | TC-006: Verify standard CPU hotplug (4 sockets, 2 cores) still works | Tier 1 | P1 |
| REQ-007 | NUMA-aware guest handling | TC-007: Create high-vCPU VM with guestMappingPassthrough | Tier 2 | P2 |

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

- **Reviewers:**
  - Kedar Bidarkar / @kbidarka
  - Akriti Gupta
- **Approvers:**
  - Daniel Gur / @dagur
  - Guy Chen / @guchen11
