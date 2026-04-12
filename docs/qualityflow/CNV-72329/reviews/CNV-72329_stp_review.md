# STP Review Report: CNV-72329

**Reviewed:** /Users/goron/Downloads/STP_CNV-72329_NAD_Reference_Live_Update (1).md
**Date:** 2026-03-22
**Reviewer:** QualityFlow Automated Review (v1.0)

---

## Verdict: APPROVED_WITH_FINDINGS

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 0 |
| Major findings | 8 |
| Minor findings | 6 |
| Confidence | MEDIUM |

---

## Findings by Dimension

### Dimension 1: Rule Compliance (Rules A-P)

| Rule | Status | Finding |
|:-----|:-------|:--------|
| A -- Abstraction Level | WARN | **[MAJOR]** Several test case titles use internal mechanism language. TC-09 references "RestartRequired condition" (internal Kubernetes API condition name). TC-15 references "restart-requiring changes" (internal controller evaluation logic). TC-23 references "legacy flow." These should use user-observable language. Suggested rewrites: TC-09: "NAD change with feature gate disabled -- VM requires restart"; TC-15: "NAD ref change combined with other changes requiring VM restart"; TC-23: "VM restart applies pending NAD change when feature gate is disabled." Additionally, the Feature Overview (Section 1.2) uses internal terms like "multus annotation" and "RestartRequired condition" -- acceptable in overview context but borderline. |
| A.2 -- Language Precision | WARN | **[MAJOR]** The Jira user story uses anthropomorphizing language: "without the VM noticing." The STP Section 1.3 Motivation repeats "without guest awareness" which is an improvement but still anthropomorphizes the guest OS. Suggested: "transparently to the guest operating system" or "with minimal service disruption." Additionally, TC-02 "Network connectivity verified after NAD swap" uses passive voice -- prefer "Verify network connectivity after NAD reference change." |
| B -- Section I Meta-Checklist | WARN | **[MAJOR]** The STP does not follow the CNV project STP template structure. The template prescribes a four-section structure (I. Motivation and Requirements Review with checklists, II. Software Test Plan with structured strategy/environment/risks tables, III. Test Scenarios and Traceability with Requirements-to-Tests Mapping, IV. Sign-off and Approval). This STP uses a custom 11-section structure (Introduction, Scope, Test Strategy, Test Environment, Test Case Inventory, Acceptance Criteria Mapping, Test Automation Prioritization, Entry/Exit Criteria, Risks, Dependencies, Glossary). While the content is substantive, the following template elements are absent: Section I checklists (Requirement and User Story Review, Technology and Design Review), structured Y/N/N/A strategy matrix (Section II.2), Known Limitations (Section II.6), Requirements-to-Tests Mapping with Tier column (Section III), and Sign-off and Approval (Section IV). |
| C -- Prerequisites vs Scenarios | PASS | All test cases describe testable behaviors rather than configuration prerequisites. Feature gate enablement is tested as behavioral verification (TC-09, TC-10, TC-11), not as a setup step. |
| D -- Dependencies | PASS | Section 10 (Dependencies) correctly lists team deliveries with Jira tracking: VEP approval (CNV-72401, Closed), implementation merge (CNV-76560, Closed), HCO FG enablement (CNV-80604, Testing), DNC compatibility (CNV-80605, New), FG graduation (CNV-82061, In Progress). These are genuine cross-team dependencies. Infrastructure items (Multus CNI, shared storage) correctly appear in Section 4 (Test Environment), not as dependencies -- consistent with `stp_rules.dependencies.infrastructure_not_dependency`. |
| E -- Upgrade Testing | PASS | Upgrade testing is appropriately included (TC-24, TC-25). The feature modifies VM spec behavior and introduces a feature gate that controls runtime behavior. The feature gate state persists across upgrades, and running VMs with NAD references are persistent state. Per `stp_rules.upgrade.persistent_state_indicators`, "running VM with feature-dependent data" applies. |
| F -- Version Derivation | WARN | **[MINOR]** The STP lists "CNV 4.18+" in Test Environment (Section 4.1) but Jira Fix Version is "CNV v4.22.0." The version "4.18+" appears to conflate the upstream KubeVirt version mapping with the downstream CNV product version. Recommend changing to "CNV 4.22" to match the Jira `fix_version` field per `stp_rules.metadata.version_source`. |
| G -- Testing Tools | PASS | No dedicated Testing Tools section is present. This is acceptable -- the STP does not list standard tools unnecessarily. Per `stp_rules.testing_tools.standard_tools`, Ginkgo, pytest, kubectl, oc, and virtctl are all standard and need not be enumerated. |
| G.2 -- Environment Specificity | PASS | Section 4.1 environment requirements are feature-specific: 2+ worker nodes "for live migration," shared storage "for live migration support," VLAN-capable network "for realistic NAD testing," bridge CNI plugin "cnv-bridge or bridge." Each requirement has a feature-specific justification. |
| H -- Risk Deduplication | PASS | Section 9 risks are genuine uncertainties: migration failure on missing NAD, DNC conflict, guest connectivity interruption, flaking upstream tests, user feedback gap (CNV-78912), race conditions, FG persistence across upgrades. None duplicate environment requirements from Section 4. |
| I -- QE Kickoff Timing | WARN | **[MINOR]** No QE Kickoff/Developer Handoff information is present. The template Section I.2 requires documenting when the Dev/Arch walkthrough occurred or should be scheduled. This is absent because the STP does not follow the template structure. |
| J -- One Tier Per Row | WARN | **[MAJOR -- see Dimension 3]** The STP does not include a Tier column in its test case inventory. Test cases are grouped by category (Functional, Feature Gate, Negative, DNC, Regression, Upgrade, Scale) with Priority (P1/P2/P3) and Automation columns, but no Tier 1/Tier 2 classification. The template Section III requires a Tier column per row. |
| K -- Cross-Section Consistency | WARN | **[MINOR]** Section 2.2 lists "SR-IOV or other non-bridge bindings" as out of scope, and TC-14 tests "NAD reference change with non-bridge binding (e.g., SR-IOV)." This is a borderline contradiction -- TC-14 is a negative test verifying the out-of-scope boundary, which is acceptable. However, the STP should explicitly note that TC-14 is a negative validation confirming unsupported binding rejection, not a positive test of SR-IOV support. |
| L -- Section Content Validation | WARN | **[MAJOR]** Section 6 "Acceptance Criteria Mapping" and Section 7 "Test Automation Prioritization" contain content that belongs in Section III (Requirements-to-Tests Mapping) per the template. The acceptance criteria mapping should be integrated into the traceability table. The automation prioritization is operational detail that could be a Comments column in the test inventory rather than a standalone section. |
| M -- Deletion Test | WARN | **[MINOR]** Section 11 (Glossary) defines standard CNV domain terms (NAD, VMI, FG, DNC, CNI, HCO) that do not contribute to Go/No-Go testing decisions. Consider linking to a shared glossary. Section 1 (Introduction) partly repeats Jira/VEP content -- the STP should reference rather than duplicate. |
| N -- Link/Reference Validation | WARN | **[MAJOR]** The VEP Reference link in the header metadata table points to `kubevirt/enhancements/pull/138` but is labeled "VEP #140." The PR number (138) and the VEP tracking issue number (140) differ. Section 1.4 correctly references both the PR (138) and the tracking issue (140) separately, but the header conflates them. The label should read "VEP #140 (Design PR #138)" or link to the tracking issue `kubevirt/enhancements/issues/140` instead. All Jira links verified against fetched data: CNV-80604 (Testing), CNV-80605 (New), CNV-78930 (In Progress), CNV-80573 (New), CNV-82091 (Testing), CNV-72401 (Closed), CNV-76560 (Closed) -- all valid and current. |
| O -- Untestable Aspects | WARN | **[MAJOR]** CNV-78912 ("Design a mitigation for the user feedback issue") describes a known gap: "a user / UI / e2e test cannot tell if the network change was applied." The issue status is "New" -- the mitigation is not yet designed. The STP references this in Section 9 Risks but does not: (1) document it as an untestable aspect with a timeline for resolution, (2) include a placeholder test scenario for when feedback becomes testable, or (3) add it to Known Limitations. This is a testing gap that affects TC-01 and TC-02 verification methods -- how does the test confirm the NAD change was applied without a feedback mechanism? |
| P -- Testing Pyramid Efficiency | PASS | N/A -- CNV-72329 is an Epic (feature ticket), not a Bug/Customer Case. Rule P does not apply. |

### Dimension 2: Requirement Coverage

| Metric | Value |
|:-------|:------|
| Acceptance criteria covered | 2/3 |
| Linked issues reflected | 19/22 |
| Negative scenarios present | YES (7 scenarios: TC-12 through TC-18) |
| Coverage gaps found | 2 |

**Jira acceptance criteria analysis:**

The epic CNV-72329 states the following goal and user story:

- **Goal:** "Allow customers to change the network a VM is connected to, to a different network without the need to reboot the VM."
- **User Story:** "As a VM admin, I want to swap the guests uplink from one network to another without the VM noticing, so they get better/worse link, different VLAN, or isolated segment."

**Coverage mapping:**

| Requirement | Covered By | Status |
|:------------|:-----------|:-------|
| Change network without reboot | TC-01, TC-02, TC-04, TC-05, TC-06, TC-08 | COVERED |
| Different VLAN use case | TC-01, TC-02 (NAD swap implies VLAN change) | COVERED |
| Better/worse link use case | Implicitly covered by NAD swap | COVERED |
| Isolated segment use case | Implicitly covered by NAD swap | COVERED |
| D/S documentation | CNV-76930 (correctly excluded, out of QE test scope) | COVERED |
| Test automation | TC-01 through TC-27 automation column, CNV-80573 | COVERED |
| User feedback visibility (CNV-78912) | Not covered | GAP |

**Linked issue coverage:**

| Linked Issue | Reflected in STP | Notes |
|:-------------|:-----------------|:------|
| CNV-72401 (VEP) | YES | Section 1.4, Section 10 |
| CNV-75778 (STP composition) | YES | Header metadata |
| CNV-76560 (Implementation) | YES | Section 10 |
| CNV-76929 (Release note) | NO | Not relevant to QE testing |
| CNV-76930 (Documentation) | YES | Section 6 acceptance mapping |
| CNV-78912 (User feedback) | PARTIAL | Risks only, no test scenario |
| CNV-78930 (STD) | YES | Section 1.4 |
| CNV-80573 (Automation) | YES | Section 1.4, Section 7 |
| CNV-80604 (HCO FG) | YES | TC-09/10/11, Section 10 |
| CNV-80605 (DNC) | YES | TC-19/20, Section 10 |
| CNV-80620 (Upstream docs) | YES | Section 10 |
| CNV-82061 (FG graduation) | YES | Section 10 |
| CNV-82091 (Flaking tests) | YES | Section 1.4, Section 9 |

**Gaps identified:**

1. **[MAJOR] User feedback/status visibility gap.** CNV-78912 describes a known issue where users cannot tell if a network change was applied. No test scenario covers verifying NAD change completion status. Even as a deferred P3 scenario, a placeholder should exist (e.g., "Verify user can confirm NAD change was applied -- blocked by CNV-78912") with a documented timeline condition ("testable when CNV-78912 mitigation is implemented").

2. **[MINOR] Guest interface name preservation not explicitly tested.** Section 1.3 states the approach "preserves all guest-visible interface properties." TC-08 covers MAC address preservation, but interface name preservation (e.g., eth1 remains eth1) is not explicitly tested. This may be implicitly covered by TC-08 but should be made explicit or noted as included in TC-08 scope.

**Proactive scope completeness observations:**

- **Negative scenario ratio:** 7 negative scenarios among 27 total (26%) -- adequate.
- **Regression coverage:** 3 explicit regression scenarios (TC-21, TC-22, TC-23) covering existing hotplug and migration flows -- good.
- **UI coverage:** UI/Console testing is explicitly excluded in Out of Scope (Section 2.2) -- acceptable with rationale.
- **OS/Platform coverage:** No OS-specific scenarios, which is acceptable for a network-level feature that operates below the guest OS layer.

### Dimension 3: Scenario Quality

| Metric | Value |
|:-------|:------|
| Total scenarios | 27 |
| Tier 1 | N/A (not classified) |
| Tier 2 | N/A (not classified) |
| P1 (per STP, effectively P0) | 10 |
| P2 (per STP, effectively P1) | 11 |
| P3 (per STP, effectively P2) | 6 |
| Positive scenarios | 17 |
| Negative scenarios | 7 |
| Regression scenarios | 3 |

**Scenario-level findings:**

1. **[MAJOR] Missing Tier classification.** The STP does not classify test cases by Tier (Tier 1 Functional vs Tier 2 End-to-End). The CNV template Section III requires a Tier column. This is critical for test planning and execution scheduling. Per linked issue CNV-80573: "tier-2 tests for this feature will very much resemble the existing tests of the interface hot-plug," suggesting most tests will be Tier 2. However, core single-operation validations (TC-01, TC-08, TC-09, TC-12, TC-13) could be Tier 1 candidates. The STP should make tier assignments explicit.

2. **[MINOR] Priority nomenclature mismatch.** The STP uses P1/P2/P3 where the QualityFlow convention uses P0/P1/P2 (P0 being highest). The core scenarios (TC-01, TC-02, TC-08, TC-09) labeled P1 should be P0 under the standard convention.

3. **[MINOR] TC-07 testability uncertain.** TC-07 ("NAD reference change via virtctl (if supported)") includes a qualifying "(if supported)" indicating uncertainty about whether virtctl supports this operation. The STP should confirm virtctl support and remove the qualifier, or move this to Known Limitations/Out of Scope if unsupported.

**Scenario quality assessment:**

- **Specificity:** All scenarios are specific and actionable. TC-01 ("Basic NAD reference change on running VM triggers migration") clearly describes the expected behavior. TC-08 ("MAC address preservation after NAD swap") has a measurable outcome.
- **Uniqueness:** No duplicate scenarios. Each of the 27 test cases tests a distinct behavior or condition.
- **Distribution:** Positive/negative ratio (17:7) with 3 regression scenarios is healthy for a feature of this scope. Scale/stress scenarios (TC-26, TC-27) cover concurrent and rapid-change patterns.

### Dimension 4: Risk and Limitation Accuracy

**Risk assessment (Section 9):**

All 7 risks are genuine uncertainties with impact/likelihood ratings and mitigations:

| Risk | Assessment |
|:-----|:-----------|
| Migration fails due to NAD not available on target node | Valid -- TC-12 covers this with error handling verification |
| DNC conflict causes network plumbing errors | Valid -- TC-19, TC-20 cover DNC compatibility (CNV-80605 status: New) |
| Guest notices network interruption during migration | Valid -- documented as expected behavior in release notes |
| Flaking upstream tests block CI | Valid -- CNV-82091 (Testing) currently blocks CNV-82061 (FG graduation) |
| User feedback issue (CNV-78912) | Valid -- status is "New," mitigation not yet designed |
| Race conditions with rapid NAD changes | Valid -- TC-18, TC-27 cover race conditions |
| Feature gate state not persisted across upgrades | Valid -- TC-24 covers upgrade scenarios |

**Known Limitations gap:**

**[MAJOR]** The STP has no Known Limitations section. The following should be documented as product limitations:

- Bridge binding only -- SR-IOV and other bindings are not supported (currently in Out of Scope but is a product limitation per VEP Beta scope)
- Brief network connectivity interruption during migration is expected (currently in Out of Scope but is a known product behavior)
- No user feedback mechanism for NAD change completion (CNV-78912, status: New)
- No migration retry limiting for missing NAD (VEP non-goal)

Per Rule L, Known Limitations describe constraints that prevent testing or product boundaries, while Out of Scope describes deliberate testing exclusions. The bridge-only limitation and connectivity interruption are product constraints, not testing choices.

### Dimension 5: Scope Boundary Assessment

**Scope alignment with Jira:**

The STP scope (Section 2.1) aligns well with the Jira epic goal. All 10 in-scope areas map to the epic's stated capability:

- NAD reference live update, bridge binding, live migration trigger, feature gate behavior, RestartRequired condition, VM spec sync, multiple networks, DNC compatibility, HCO integration, CLI/API -- all are aspects of "change the network a VM is connected to without rebooting."

**Out of Scope assessment:**

All 8 out-of-scope items are well-justified:

- Migrating between CNI types, changing binding/plugin, non-migratable VMs -- VEP #140 non-goals
- Seamless network connectivity during migration -- documented as expected behavior
- Guest network reconfiguration -- out of product scope
- SR-IOV/non-bridge bindings -- Beta scope limitation
- Migration retry limiting -- VEP non-goal
- UI/Console testing -- separate test effort

**Layered product check (per `stp_rules.scope.layered_product`):**

No scenarios test only platform-level functionality without VM-specific involvement. All scenarios involve VirtualMachine/VirtualMachineInstance resources, which are in-scope per `scope_boundaries.in_scope_resources`. The scope correctly focuses on VM-specific behavior while assuming platform networking (Multus, CNI) works correctly.

No scope boundary issues found.

### Dimension 6: Test Strategy Appropriateness

The STP uses a non-template format for test strategy (Sections 3.1 and 3.2 list test levels and types as tables rather than the Y/N/N/A strategy matrix). Evaluating implied classifications:

| Strategy Row | Implied Classification | Assessment |
|:-------------|:----------------------|:-----------|
| Functional Testing | Y (Section 5.1) | Correct -- per `stp_rules.strategy.always_y` |
| Automation Testing | Y (Automation column) | Correct -- per `stp_rules.strategy.always_y` |
| Performance Testing | Not addressed | Should be N/A with rationale -- no latency/throughput SLA exists |
| Security Testing | Not addressed | Should be N/A -- no RBAC/auth/security boundary changes |
| Usability Testing | Not addressed | Should be N/A -- API-only feature, UI explicitly excluded |
| Compatibility Testing | Y (Section 5.4 DNC) | Correct -- DNC and platform compatibility tested |
| Regression Testing | Y (Section 5.5) | Correct -- hotplug and migration regression covered |
| Upgrade Testing | Y (Section 5.6) | Correct -- FG persistence and rollback tested |
| Backward Compatibility Testing | Not addressed | Should be Y -- FG-disabled behavior must match pre-feature behavior (TC-09, TC-23 cover this) |
| Dependencies | Y (Section 10) | Correct -- cross-team deliveries with Jira tracking |
| Cross Integrations | Implied Y | DNC, HCO, live migration cross-team impact addressed |
| Monitoring Testing | Not addressed | Should be N/A -- no new metrics or alerts introduced |
| Cloud Testing | Not addressed | Should be N/A -- bridge networking is not cloud-specific |

**Finding:** While all relevant test types are addressed in substance, the non-template format means several strategy rows lack explicit N/A justification (Performance, Security, Usability, Monitoring, Cloud). The template requires explicit classification for each row.

### Dimension 7: Metadata Accuracy

| Field | STP Value | Jira Value | Assessment |
|:------|:----------|:-----------|:-----------|
| Epic | CNV-72329 | CNV-72329 | MATCH |
| Parent Feature | VIRTSTRAT-560 | N/A (not in fetched data) | Referenced in STP, plausible |
| Component | CNV Network | CNV Network | MATCH |
| Assignee | Ananya Banerjee | Ananya Banerjee | MATCH |
| QA Contact | Yossi Segev | N/A (not a Jira field) | Consistent with CNV-75778 reporter |
| STP Author | QE (CNV-75778) | CNV-75778 status: Testing | MATCH |
| Fix Version | "CNV 4.18+" (Section 4.1) | CNV v4.22.0 | MISMATCH (see Rule F) |
| Feature Gate | LiveUpdateNADRef (Beta, v1.8) | Consistent with linked issues | PLAUSIBLE |
| VEP Reference | VEP #140 / PR #138 | See Rule N | LABEL MISMATCH |

**Additional metadata findings:**

- **[MINOR] Owning SIG missing.** The STP does not include an Owning SIG field. Based on component "CNV Network" and `sig_mappings` in project config, Owning SIG should be sig-network.
- **[MINOR] Participating SIGs missing.** Given the feature triggers live migration, sig-migration should be listed as a participating SIG.

---

## Recommendations

1. **[MAJOR-1] Adopt the CNV project STP template structure.** Restructure the STP to follow the four-section template at `config/projects/cnv/templates/stp/stp-template.md`. Key additions needed: Section I checklists (Requirement and User Story Review, Technology and Design Review), structured Y/N/N/A strategy matrix in Section II.2, Known Limitations in Section II.6, Requirements-to-Tests Mapping with Tier column in Section III, and Sign-off and Approval in Section IV.

2. **[MAJOR-2] Add Tier classification to all test cases.** Each test case should be classified as Tier 1 (Functional, single-feature) or Tier 2 (End-to-End, multi-step workflow). Per CNV-80573, most tests will resemble existing hot-plug Tier 2 tests, but core single-operation tests may be Tier 1 candidates.

3. **[MAJOR-3] Add Known Limitations section.** Document product constraints: bridge-only binding support, expected brief connectivity interruption during migration, no user feedback mechanism (CNV-78912), no migration retry limiting for missing NAD.

4. **[MAJOR-4] Add a test scenario for user feedback/observability.** Include a deferred P3 scenario (e.g., "Verify user can confirm NAD change was applied") with a note that it is blocked by CNV-78912 resolution. Document the timeline condition: "testable when CNV-78912 mitigation is designed and implemented."

5. **[MAJOR-5] Rewrite test case titles that use internal mechanism language.** TC-09 ("RestartRequired condition"), TC-15 ("restart-requiring changes"), TC-23 ("legacy flow") should use user-observable language.

6. **[MAJOR-6] Fix VEP reference labeling in header.** Clarify the distinction between VEP #140 (tracking issue) and PR #138 (design document), or link to the tracking issue URL.

7. **[MAJOR-7] Merge Sections 6 and 7 into Section III traceability.** Acceptance Criteria Mapping and Test Automation Prioritization should be integrated into the Requirements-to-Tests Mapping table rather than standing as separate sections.

8. **[MAJOR-8] Fix version reference.** Change "CNV 4.18+" in Test Environment to "CNV 4.22" to match Jira Fix Version "CNV v4.22.0."

9. **[MINOR-1] Normalize priority labels to P0/P1/P2.** Align with QualityFlow convention where P0 is highest priority.

10. **[MINOR-2] Resolve TC-07 testability qualifier.** Remove "(if supported)" and confirm or deny virtctl support for NAD reference changes.

11. **[MINOR-3] Add explicit interface name preservation test.** Either expand TC-08 scope description to include interface name, or add a dedicated scenario.

12. **[MINOR-4] Clarify TC-14 as negative boundary test.** Add a note that this tests unsupported binding rejection, not SR-IOV functionality.

13. **[MINOR-5] Add Owning SIG (sig-network) and Participating SIGs (sig-migration) to metadata.** These fields are required by the template.

14. **[MINOR-6] Add QE Kickoff/Developer Handoff documentation.** Record whether a Dev/QE architecture walkthrough has occurred.

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| Jira source data available | YES |
| Linked issues fetched | YES (22 issues) |
| PR data referenced in STP | YES (PR #16412, #16993 referenced) |
| All STP sections present | NO (non-template structure) |
| Template comparison possible | YES |
| Project review rules loaded | YES |

**Confidence rationale:** Confidence is MEDIUM. Jira source data and all 22 linked/child issues were fetched, enabling comprehensive requirement coverage analysis (Dimension 2) and metadata verification (Dimension 7). The Jira Fix Version (CNV v4.22.0) was verified against the STP. PR data is referenced in the STP but PR diffs were not fetched for code-level analysis. The STP does not follow the CNV template structure, which limited Rule B template comparison -- structural gaps were identified but content-level comparison against template fields was not possible for all sections. Project review rules (`review_rules.yaml`) were loaded and applied for enhanced rule precision across all dimensions.
