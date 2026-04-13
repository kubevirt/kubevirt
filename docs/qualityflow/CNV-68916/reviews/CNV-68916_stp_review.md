# STP Review Report: CNV-68916

**Reviewed:** outputs/stp/CNV-68916/CNV-68916_test_plan.md
**Date:** 2026-03-19
**Reviewer:** QualityFlow Automated Review (v1.0)

---

## Verdict: APPROVED_WITH_FINDINGS

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 0 |
| Major findings | 7 |
| Minor findings | 9 |
| Confidence | MEDIUM |

---

## Findings by Dimension

### Dimension 1: Rule Compliance (Rules A-O)

| Rule | Status | Finding |
|:-----|:-------|:--------|
| A -- Abstraction Level | WARN | Internal component references found in Scope of Testing and Testing Goals. "declarative volume hotplug behavior" and "VM controller" are implementation-level terms. Scope mentions "feature gate interactions" and "PCI port allocation" which are internal mechanisms. However, Section III scenarios are largely user-perspective. See details below. |
| A.2 -- Language Precision | PASS | Language is precise and professional throughout. No anthropomorphization or colloquial phrasing detected. Measurable criteria are used where appropriate. |
| B -- Section I Meta-Checklist | WARN | Details/Notes column contains feature-specific content rather than preserving the template's standard text. Per the template, the Details/Notes column should retain standard descriptions (e.g., "Reviewed the relevant requirements.") and feature-specific observations should appear only in the Comments column. The STP correctly places content in Comments but also modifies Details/Notes with feature content. |
| C -- Prerequisites vs Scenarios | PASS | No prerequisites masquerading as test scenarios in Section III. Entry criteria and test environment correctly capture configuration prerequisites (feature gate enablement, CDI availability, StorageClass). |
| D -- Dependencies | WARN | Dependencies row is marked Y with Comments listing CDI, virt-controller, virt-launcher, virt-handler, and UI team. CDI, virt-controller, virt-launcher, and virt-handler are internal components of the same product (OpenShift Virtualization), not external team deliveries. Per review rules, these are infrastructure, not dependencies. The UI team reference (CNV-77383) is a valid cross-team dependency. |
| E -- Upgrade Testing | PASS | Upgrade Testing is correctly marked Y. The feature creates persistent state: VMs with hotplugged volumes store configuration in the VM spec (a CRD), and feature gate state is stored in the HyperConverged CR. Both are persistent state indicators per review rules. |
| F -- Version Derivation | PASS | Version "OCP 4.18+, CNV 4.22+" is stated. Without Jira data to cross-reference, the version appears reasonable for a GA feature. No hardcoded stale versions detected. |
| G -- Testing Tools | WARN | Section II.3.1 lists standard tools: Ginkgo v2 + Gomega, pytest, OpenShift CI (Prow), Jenkins, virtctl, oc CLI, kubectl. Per review rules, all of these are standard tools/frameworks. The section should either be empty or list only non-standard tools. |
| G.2 -- Environment Specificity | PASS | Test Environment entries are feature-specific. Storage requirements specify dynamic provisioning with RWO support for DataVolume creation. Special Configurations specifies the DeclarativeHotplugVolumes feature gate enablement. Entries explain why each requirement exists for this feature. |
| H -- Risk Deduplication | PASS | Risks are distinct from Test Environment entries. Storage backend variability risk (II.5) addresses a different concern (test reliability) than the storage environment requirement (II.3). No duplication detected. |
| I -- QE Kickoff Timing | PASS | Developer Handoff/QE Kickoff Comments describe the feature designer (Michael Henriksen) and architecture walkthrough. No indication of post-implementation kickoff. |
| J -- One Tier Per Row | PASS | Every row in Section III has exactly one tier value (either "Tier 1" or "Tier 2"). No multi-tier cells detected. |
| K -- Cross-Section Consistency | PASS | Scope items align with Section III scenarios. Out-of-scope items (UI testing, storage provisioner internals, QEMU/libvirt internals, multi-cluster, performance benchmarking) do not appear in Section III. Testing Goals align with Known Limitations (e.g., SATA hot-detach limitation is documented in both). Monitoring is marked Y in strategy and REQ-EPHEMERAL-02 covers monitoring in Section III. |
| L -- Section Content Validation | WARN | Feature Overview contains implementation-level detail (PCI port allocation scheme, specific PR numbers, internal code paths like "pkg/storage/hotplug/hotplug.go") that could be more concise. While this is the overview section and some detail is acceptable, PR numbers and code paths are excessive for an STP. These details belong in engineering documentation, not the test plan. |
| M -- Deletion Test | PASS | All sections contribute decision-relevant information. No sections contain purely redundant content that could be removed without impacting Go/No-Go decisions. The Feature Overview is detailed but serves as the authoritative feature description for test planning. |
| N -- Link/Reference Validation | PASS | Links are syntactically valid and point to correct domains. Enhancement link points to kubevirt/enhancements (correct upstream project). Jira links point to issues.redhat.com (correct Jira instance). PR URLs reference kubevirt/kubevirt repository (correct for this feature). Enhancement link uses the official repository (kubevirt/enhancements), not a personal fork. |
| O -- Untestable Aspects | PASS | PCI address stability across upgrades is identified as untestable (PR #17029 in progress). This is documented with reason (upstream PR not merged), timeline ("when fix is available"), and corresponding risk entry in II.5 (Untestable Aspects row). No P0 items are marked untestable. |

**Rule A detail -- internal references in Scope/Goals:**

The Scope of Testing (II.1) contains the following internal-mechanism references:

1. "declarative volume hotplug behavior" -- implementation approach; suggest "volume changes applied to running VMs without restart"
2. "feature gate interactions" -- internal mechanism; suggest "feature enablement and configuration behavior"
3. "PCI port allocation" -- internal mechanism; suggest "hotplug capacity management"
4. "RestartRequired condition handling" -- internal condition name; acceptable as it is a user-visible API condition

Testing Goals contain similar references:

1. "Verify declarative hotplug volumes correctly reconcile VM spec changes to running VMIs" -- uses "reconcile" (implementation verb) and "VMI" (internal resource). Suggest: "Verify volume changes in VM configuration are automatically applied to running VMs"
2. "Validate PCI port allocation for hotplug capacity" -- internal mechanism. Suggest: "Validate VMs have sufficient capacity for multiple hotplug operations"

These are MAJOR findings per Rule A, but the scenarios themselves in Section III are mostly user-perspective, which limits the practical impact.

### Dimension 2: Requirement Coverage

| Metric | Value |
|:-------|:------|
| Acceptance criteria covered | N/A (Jira data unavailable) |
| Linked issues reflected | N/A (Jira data unavailable) |
| Negative scenarios present | YES |
| Coverage gaps found | 0 (content-only assessment) |

**Content-only coverage assessment:**

Without Jira data, coverage is assessed based on internal consistency between the Feature Overview and Section III.

- All Feature Overview capabilities have corresponding test scenarios:
  - CD-ROM Inject: REQ-CDROM-INJECT-01 through 04
  - CD-ROM Eject: REQ-CDROM-EJECT-01 through 03
  - CD-ROM Swap: REQ-CDROM-SWAP-01 through 03
  - Empty CD-ROM: REQ-EMPTY-CDROM-01, 02
  - Feature gates: REQ-FGATE-01 through 04
  - RestartRequired: REQ-RESTART-01 through 04
  - Bus types: REQ-BUS-01 through 03
  - PCI ports: REQ-PCI-01 through 03
  - virtctl: REQ-VIRTCTL-01 through 04
  - Ephemeral: REQ-EPHEMERAL-01, 02
  - Hotplug disk: REQ-HOTPLUG-DISK-01, 02
  - E2E lifecycle/migration/snapshot/upgrade: REQ-E2E-*
  - RBAC: REQ-RBAC-01, 02
  - Negative: REQ-NEGATIVE-01, 02

- Negative scenarios (4): REQ-CDROM-INJECT-04, REQ-CDROM-EJECT-03, REQ-NEGATIVE-01, REQ-NEGATIVE-02. This is adequate for a feature with 47 total scenarios.

- UI coverage: Correctly addressed in Out of Scope with PM agreement (CNV-77383).

- Monitoring: Covered by REQ-EPHEMERAL-02 (metric and alert).

- Cross-SIG coverage: Participating SIGs are sig-compute and sig-network. Migration scenarios (REQ-E2E-MIGRATION-01, 02) touch sig-compute. However, no scenarios explicitly test sig-network-related behavior.

**Gaps identified:**

- No explicit sig-network test scenario despite sig-network being listed as a participating SIG. While the feature is primarily sig-storage, the listing of sig-network as a participant suggests some network interaction that should be tested or the SIG should be removed from participating SIGs.

### Dimension 3: Scenario Quality

| Metric | Value |
|:-------|:------|
| Total scenarios | 47 |
| Tier 1 | 37 |
| Tier 2 | 10 |
| P0 | 7 |
| P1 | 27 |
| P2 | 13 |
| Positive scenarios | 43 |
| Negative scenarios | 4 |

**Scenario-level findings:**

1. **[MINOR]** Several scenarios in Section III are verbose (exceeding 15 words). Examples:
   - REQ-CDROM-INJECT-01: "Verify CD-ROM inject with DataVolume source on running VM with empty CD-ROM; confirm volume appears in VMI status and CD-ROM is mountable/readable in guest" (26 words). Suggest splitting into separate verification points or condensing.
   - REQ-E2E-LIFECYCLE-01: "Verify complete lifecycle: create VM with empty CD-ROM, inject media, verify content, swap media, verify new content, eject media, verify 'No medium found', re-inject and verify again" (27 words). This reads more like a test procedure than a scenario description.

2. **[MINOR]** REQ-CDROM-INJECT-01 and REQ-CDROM-INJECT-03 have overlapping verification ("mountable/readable in guest" vs "mountable, readable, and shows expected file count"). Consider consolidating or differentiating more clearly.

3. **[PASS]** Priority distribution is reasonable: P0 (15%), P1 (57%), P2 (28%). Core CD-ROM operations and lifecycle tests are P0. Error handling and edge cases are P2. No priority inflation detected.

4. **[PASS]** Tier distribution is appropriate: Tier 1 (79%) for individual feature validations, Tier 2 (21%) for lifecycle, persistence, migration, snapshot, and upgrade workflows.

5. **[MINOR]** REQ-CDROM-INJECT-01 scenario text references "VMI status" which is an internal resource name. Suggest: "confirm volume appears in VM running state and CD-ROM is mountable in guest."

### Dimension 4: Risk & Limitation Accuracy

**Risks assessment (content-only):**

1. **[PASS]** All 7 risks are genuine uncertainties with actionable mitigation strategies:
   - Timeline risk (feature off by default) -- tracked via CNV-79690
   - Test coverage (upstream flakiness) -- monitored via PR #14998
   - Environment (storage variability) -- test with multiple backends
   - Untestable (PCI stability) -- monitor upstream PR
   - Resources (cluster consumption) -- pre-provision DataVolumes
   - Dependencies (CDI) -- health check as precondition
   - Other (feature gate combinations) -- test all combinations

2. **[MINOR]** All risk statuses are unchecked ([ ]). Expected for a draft STP but noted for tracking.

**Limitations assessment:**

3. **[PASS]** Known limitations are well-documented with 6 specific limitations covering:
   - Ephemeral hotplug incompatibility
   - SATA hot-detach restriction
   - PCI address stability concern
   - Feature off by default
   - VM must be running
   - Volume ordering behavior

4. **[PASS]** No contradictions between limitations and testing goals. The SATA hot-detach limitation is properly reflected in REQ-RESTART-01 (tests the RestartRequired behavior rather than attempting hot-detach).

### Dimension 5: Scope Boundary Assessment

**Content-only assessment:**

1. **[PASS]** Scope aligns with the feature capabilities described in the Feature Overview. All major capabilities are covered.

2. **[PASS]** Out-of-scope items are appropriate with rationale and PM agreement:
   - UI testing (separate epic with dedicated QE)
   - Storage provisioner internals (platform responsibility)
   - QEMU/libvirt internals (upstream responsibility)
   - Multi-cluster (not applicable)
   - Performance benchmarking (not a GA requirement)

3. **[PASS]** As a layered product (OCP-V per review rules), scenarios correctly focus on VM-specific behavior rather than platform-level storage or network functionality.

4. **[MINOR]** Scope mentions "RBAC enforcement" but RBAC scenarios (REQ-RBAC-01, REQ-RBAC-02) test standard Kubernetes RBAC for VM spec modifications, which is not specific to this feature. These scenarios test platform-level RBAC enforcement that exists for any VM spec modification. Consider whether these are feature-specific or should be deferred to general RBAC test suites.

### Dimension 6: Test Strategy Appropriateness

1. **[PASS]** Functional Testing: Correctly marked Y with feature-specific comments.

2. **[PASS]** Automation Testing: Correctly marked Y with references to specific upstream and downstream test files.

3. **[MAJOR]** Performance Testing: Marked Y with Comments "Validate PCI port allocation does not degrade VM startup time; verify hotplug operations complete within acceptable time windows." This describes functional validation (operations complete successfully), not performance testing with specific latency/throughput SLAs. No performance benchmarks or targets are defined. Should be N/A unless specific performance requirements exist. If hotplug latency targets exist, they should be stated explicitly.

4. **[MAJOR]** Security Testing: Marked Y with Comments "RBAC enforcement for VM spec modifications; validate non-privileged users cannot bypass feature gate restrictions." Standard RBAC that applies to any VM spec modification. The feature does not introduce new security boundaries or authentication/authorization mechanisms. Should be N/A unless the feature changes RBAC rules specifically.

5. **[PASS]** Usability Testing: Correctly marked N/A (UI covered by separate epic).

6. **[PASS]** Compatibility Testing: Correctly marked Y with multiple storage backend testing.

7. **[PASS]** Regression Testing: Correctly marked Y with specific regression areas identified.

8. **[PASS]** Upgrade Testing: Correctly marked Y per Rule E (persistent state in CRDs).

9. **[PASS]** Backward Compatibility Testing: Correctly marked Y (feature gate precedence, deprecated --persist flag).

10. **[PASS]** Cloud Testing: Correctly marked N (storage-backend agnostic).

11. **[PASS]** Monitoring: Correctly marked Y with specific metric and alert identified.

### Dimension 7: Metadata Accuracy

1. **[PASS]** Enhancement link: Points to kubevirt/enhancements#31 (correct upstream enhancement tracking for KubeVirt features).

2. **[PASS]** Feature in Jira: CNV-68916 matches the input Jira ID.

3. **[PASS]** Jira Tracking: Lists CNV-68916 (epic) plus three linked issues (CNV-79690, CNV-77383, CNV-64402) with brief descriptions.

4. **[PASS]** QE Owner: "Yan Du" is specified (not TBD).

5. **[PASS]** Owning SIG: sig-storage. Consistent with the feature's storage focus (CD-ROM hotplug volumes).

6. **[MINOR]** Current Status: "GA (off by default; fully-supported when enabled)" -- this describes the feature status, not the STP document status. The template expects the STP's own status (e.g., "Draft", "Reviewed", "Approved"). However, this is a minor interpretation difference and does not impact usability.

7. **[MINOR]** Sign-off section contains placeholder entries "[Name / @github-username]" alongside real names. These should either be filled in or removed.

---

## Recommendations

1. **[MAJOR]** Reclassify Performance Testing as N/A in the Test Strategy (II.2) unless specific latency/throughput SLAs exist for hotplug operations. The current comments describe functional validation, not performance testing.
2. **[MAJOR]** Reclassify Security Testing as N/A in the Test Strategy (II.2) unless this feature introduces new RBAC rules beyond standard VM spec modification permissions. Move RBAC scenarios to general RBAC test coverage or justify why feature-specific RBAC testing is needed.
3. **[MAJOR]** Revise the Dependencies row (II.2) to list only genuine cross-team deliveries. CDI, virt-controller, virt-launcher, and virt-handler are components of the same product and should be listed as infrastructure in Test Environment (II.3), not as dependencies. Retain the UI team (CNV-77383) as a valid dependency.
4. **[MAJOR]** Rewrite internal-mechanism references in Scope of Testing and Testing Goals to use user-perspective language. Replace "declarative volume hotplug behavior" with "volume changes applied to running VMs," "PCI port allocation" with "hotplug capacity," and remove implementation verbs like "reconcile."
5. **[MAJOR]** Remove standard tools from Testing Tools section (II.3.1). Ginkgo, pytest, kubectl, oc, virtctl, OpenShift CI, and Jenkins are all standard project tools. Either leave the section empty or list only non-standard tools specific to this feature.
6. **[MAJOR]** Remove feature-specific content from the Details/Notes column in Section I tables. The Details/Notes column should retain the template's standard text. Move all feature-specific observations to the Comments column.
7. **[MAJOR]** Address the sig-network participating SIG listing: either add test scenarios that exercise sig-network-related behavior, or remove sig-network from the Participating SIGs metadata if it is not relevant to this feature.
8. **[MINOR]** Condense verbose scenario descriptions in Section III (REQ-CDROM-INJECT-01, REQ-E2E-LIFECYCLE-01) to 15 words or fewer. Split multi-step descriptions into separate requirement rows if needed.
9. **[MINOR]** Remove implementation-level detail from Feature Overview (specific PR numbers, code paths like "pkg/storage/hotplug/hotplug.go"). Reference PRs in Section I Comments where they provide traceability value.
10. **[MINOR]** Replace "VMI status" references in scenario descriptions with user-facing terms like "VM running state" per the review rules internal_to_user_mappings.
11. **[MINOR]** Update Current Status to reflect the STP document status (e.g., "Draft") rather than the feature status.
12. **[MINOR]** Fill in or remove placeholder entries in the Sign-off section ("[Name / @github-username]").
13. **[MINOR]** Consider whether RBAC scenarios (REQ-RBAC-01, REQ-RBAC-02) test feature-specific RBAC or general Kubernetes RBAC. If general, move to Out of Scope or a shared RBAC test plan.
14. **[MINOR]** Update risk statuses as the STP progresses through review and approval.
15. **[MINOR]** Consolidate or differentiate REQ-CDROM-INJECT-01 and REQ-CDROM-INJECT-03 which have overlapping verification scope.
16. **[MINOR]** Resolve REQ-CDROM-INJECT-01 reference to "VMI status" -- use "VM running state" per review rules mappings.

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| Jira source data available | NO |
| Linked issues fetched | NO |
| PR data referenced in STP | YES (content only, not verified) |
| All STP sections present | YES |
| Template comparison possible | YES |
| Project review rules loaded | YES |

**Confidence rationale:** Confidence is MEDIUM because Jira source data was unavailable for cross-referencing. The review was performed as a content-only assessment, meaning Dimension 2 (Requirement Coverage) could not validate acceptance criteria against the Jira source, and Dimension 4 (Risk & Limitation Accuracy) could not verify limitations against Jira data. All STP sections are present, the template comparison was performed, and project review rules were loaded and applied. The STP content is internally consistent and well-structured, but source data verification would increase confidence to HIGH.
