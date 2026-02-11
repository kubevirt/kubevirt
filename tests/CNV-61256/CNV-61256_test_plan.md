# Openshift-virtualization-tests Test plan

## **Disable common-instancetypes deployment from HCO - Quality Engineering Plan**

### **Metadata & Tracking**

| Field | Details |
|:------|:--------|
| **Enhancement(s)** | [CNV-44447](https://issues.redhat.com/browse/CNV-44447) |
| **Feature in Jira** | [CNV-59564](https://issues.redhat.com/browse/CNV-59564) |
| **Jira Tracking** | [CNV-61256](https://issues.redhat.com/browse/CNV-61256) |
| **QE Owner(s)** | Roni Kishner |
| **Owning SIG** | Instance Types |
| **Participating SIGs** | HCO, KubeVirt |
| **Current Status** | Closed |

### **Related GitHub Pull Requests**

| PR Link | Repository | Source Jira Issue | Source Type | Description |
|:--------|:-----------|:------------------|:------------|:------------|
| [#3471](https://github.com/kubevirt/hyperconverged-cluster-operator/pull/3471) | hyperconverged-cluster-operator | CNV-59564 | Comment | Introduce CommonInstancetypesDeployment configurable to HCO CR |

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value, technology, and testability prior to formal test planning.

#### **1. Requirement & User Story Review Checklist**

| Check | Done | Details/Notes | Comments |
|:------|:-----|:--------------|:---------|
| **Review Requirements** | [x] | Allow users to disable common-instancetypes deployment via HCO CR configuration | Feature adds `spec.commonInstancetypesDeployment.enabled` field to HCO CR |
| **Understand Value** | [x] | Customers need control over instance type deployment to manage cluster resources and permissions | In 4.17, the commonInstanceTypesDeploymentGate was mandatory and couldn't be disabled |
| **Customer Use Cases** | [x] | Customer environments where common instance types are not needed or conflict with custom configurations | Addresses customer request for ability to disable instance types in 4.17+ |
| **Testability** | [x] | Feature can be tested by modifying HCO CR and verifying common-instancetypes resources are not deployed | API-level configuration with observable cluster state changes |
| **Acceptance Criteria** | [x] | Setting `spec.commonInstancetypesDeployment.enabled: false` should prevent deployment of common-instancetypes | Verified on CNV-v4.19.0.rhel9-118 |
| **Non-Functional Requirements (NFRs)** | [x] | No performance impact expected; configuration change only affects resource deployment | N/A |

#### **2. Technology and Design Review**

| Check | Done | Details/Notes | Comments |
|:------|:-----|:--------------|:---------|
| **Developer Handoff/QE Kickoff** | [x] | PR merged with documentation and unit tests | PR includes API documentation updates |
| **Technology Challenges** | [x] | None - straightforward API extension | Uses existing KubeVirt CommonInstancetypesDeployment type |
| **Test Environment Needs** | [x] | Standard OpenShift Virtualization cluster | No special hardware requirements |
| **API Extensions** | [x] | New `spec.commonInstancetypesDeployment` field in HyperConverged CR | Mirrors KubeVirt CR structure |
| **Topology Considerations** | [x] | Feature applies cluster-wide | Affects all nodes in the cluster |

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the new `CommonInstancetypesDeployment` configuration in HCO CR that controls whether common-instancetypes cluster-wide resources are deployed by KubeVirt.

**In Scope:**

- Verify `spec.commonInstancetypesDeployment.enabled: false` prevents common-instancetypes deployment
- Verify `spec.commonInstancetypesDeployment.enabled: true` enables common-instancetypes deployment (default behavior)
- Verify nil/unset configuration maintains default behavior (enabled)
- Verify configuration propagates correctly from HCO CR to KubeVirt CR
- Verify upgrade scenarios preserve configuration state
- Verify API validation and error handling

#### **2. Testing Goals**

##### **Positive Use Cases (Happy Path)**

- TC-001: Disable common-instancetypes deployment via HCO CR
- TC-002: Enable common-instancetypes deployment via HCO CR
- TC-003: Verify default behavior when configuration is not set
- TC-004: Verify configuration persists after HCO reconciliation
- TC-005: Verify KubeVirt CR reflects HCO configuration

##### **Negative Use Cases (Error Handling & Edge Cases)**

- TC-006: Verify invalid configuration values are rejected
- TC-007: Verify behavior when toggling configuration while VMs are running
- TC-008: Verify behavior during upgrade from version without this feature

#### **3. Non-Goals (Testing Scope Exclusions)**

| Non-Goal | Rationale | PM/ Lead Agreement |
|:---------|:----------|:-------------------|
| Testing common-instancetypes functionality itself | Already covered by existing KubeVirt tests | [x] |
| Performance testing of instance type operations | No performance impact from configuration change | [x] |
| UI testing for this configuration | No UI changes associated with this feature | [x] |

#### **4. Test Strategy**

##### **A. Types of Testing**

| Item (Testing Type) | Applicable (Y/N or N/A) | Comments |
|:--------------------|:------------------------|:---------|
| **Functional Testing** | Y | Core functionality of enabling/disabling common-instancetypes deployment |
| **Automation Testing** | Y | Tier 2 end-to-end tests for API configuration |
| **Performance Testing** | N/A | No performance impact expected |
| **Security Testing** | N/A | No security implications for this configuration |
| **Usability Testing** | N/A | API-only feature, no UI |
| **Compatibility Testing** | Y | Test across supported OCP versions |
| **Regression Testing** | Y | Ensure existing instance type functionality unaffected |
| **Upgrade Testing** | Y | Test configuration preservation during upgrades |
| **Backward Compatibility Testing** | Y | Test upgrade from versions without this feature |

##### **B. Potential Areas to Consider**

| Item | Description | Applicable (Y/N or N/A) | Comment |
|:-----|:------------|:------------------------|:--------|
| **Dependencies** | KubeVirt CommonInstancetypesDeployment type | Y | HCO passes configuration to KubeVirt CR |
| **Monitoring** | No specific metrics for this feature | N/A | Standard HCO/KubeVirt monitoring applies |
| **Cross Integrations** | Affects VMs that reference common instance types | Y | VMs using common instance types may fail if deployment is disabled |
| **UI** | No UI changes | N/A | Configuration via kubectl/oc only |

#### **5. Test Environment**

| Environment Component | Configuration | Specification Examples |
|:----------------------|:--------------|:-----------------------|
| **Cluster Topology** | Standard | 3 control plane + 2 worker nodes |
| **OCP & OpenShift Virtualization Version(s)** | 4.19+ | OCP 4.19, CNV 4.19 |
| **CPU Virtualization** | Standard | VMX/SVM enabled |
| **Compute Resources** | Standard | 16GB RAM, 4 vCPU per worker |
| **Special Hardware** | None | N/A |
| **Storage** | Standard | ODF or local storage |
| **Network** | Standard | OpenShift SDN or OVN-Kubernetes |
| **Required Operators** | HCO, KubeVirt | Installed via OLM |
| **Platform** | Bare metal or cloud | AWS, Azure, vSphere, or bare metal |
| **Special Configurations** | None | N/A |

#### **5.5. Testing Tools & Frameworks**

| Category | Tools/Frameworks |
|:---------|:-----------------|
| **Test Framework** | Ginkgo/Gomega (Tier 1), pytest (Tier 2) |
| **CI/CD** | OpenShift CI, Prow |
| **Other Tools** | kubectl, oc, virtctl |

#### **6. Entry Criteria**

The following conditions must be met before testing can begin:

- HCO with CommonInstancetypesDeployment support is deployed
- Cluster is healthy and all operators are running
- Access to HCO and KubeVirt CRs is available
- Test framework and dependencies are installed

#### **7. Risks and Limitations**

| Risk Category | Specific Risk for This Feature | Mitigation Strategy | Status |
|:--------------|:-------------------------------|:--------------------|:-------|
| Timeline/Schedule | None | N/A | [x] |
| Test Coverage | Limited edge cases for configuration toggle | Include toggle scenarios in test suite | [x] |
| Test Environment | None | Standard environment sufficient | [x] |
| Untestable Aspects | None | All aspects are testable | [x] |
| Resource Constraints | None | No special resources needed | [x] |
| Dependencies | KubeVirt API compatibility | Use versioned API types | [x] |
| Other | N/A | N/A | [x] |

#### **8. Known Limitations**

- Feature only available in CNV 4.19+ (backported to 4.17 via cloned bugs)
- Disabling common-instancetypes will cause VMs referencing them to fail scheduling

---

### **III. Test Scenarios & Traceability**

This section provides a **high-level overview** of test scenarios mapped to requirements.

#### **1. Requirements-to-Tests Mapping**

| Requirement ID | Requirement Summary | Test Scenario(s) | Test Type(s) | Priority |
|:---------------|:--------------------|:-----------------|:-------------|:---------|
| REQ-001 | Disable common-instancetypes deployment via HCO CR | TC-001: Set `spec.commonInstancetypesDeployment.enabled: false` and verify no common-instancetypes resources exist | Tier 2 | P1 |
| REQ-002 | Enable common-instancetypes deployment via HCO CR | TC-002: Set `spec.commonInstancetypesDeployment.enabled: true` and verify common-instancetypes resources are deployed | Tier 2 | P1 |
| REQ-003 | Default behavior maintains existing functionality | TC-003: Verify unset configuration results in common-instancetypes being deployed | Tier 2 | P1 |
| REQ-004 | Configuration propagates to KubeVirt CR | TC-005: Verify KubeVirt CR `spec.configuration.commonInstancetypesDeployment` matches HCO CR | Tier 1 | P1 |
| REQ-005 | Configuration persists after reconciliation | TC-004: Modify HCO CR and verify configuration persists after operator reconciliation | Tier 2 | P2 |
| REQ-006 | Invalid configuration rejected | TC-006: Attempt to set invalid values and verify API rejection | Tier 1 | P2 |
| REQ-007 | Upgrade preserves configuration | TC-008: Upgrade cluster and verify configuration is preserved | Tier 2 | P2 |
| REQ-008 | Toggle behavior with running VMs | TC-007: Toggle configuration while VMs are running and verify behavior | Tier 2 | P2 |

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

- **Reviewers:**
  - Roni Kishner / @rkishner
  - Lee Yarwood / @lyarwood
- **Approvers:**
  - QE Lead / @qe-lead
  - PM Lead / @pm-lead
