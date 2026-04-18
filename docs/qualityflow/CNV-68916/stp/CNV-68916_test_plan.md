# Openshift-virtualization-tests Test plan

## **GA: Eject/Inject CD-ROM Support via Declarative Hotplug Volumes - Quality Engineering Plan**

### **Metadata & Tracking**

| Field                  | Details                                                           |
|:-----------------------|:------------------------------------------------------------------|
| **Enhancement(s)**     | [kubevirt/enhancements#31](https://github.com/kubevirt/enhancements/issues/31) |
| **Feature in Jira**    | [CNV-68916](https://issues.redhat.com/browse/CNV-68916)           |
| **Jira Tracking**      | [CNV-68916](https://issues.redhat.com/browse/CNV-68916) (Epic), [CNV-79690](https://issues.redhat.com/browse/CNV-79690) (Declarative Hotplug on by default), [CNV-77383](https://issues.redhat.com/browse/CNV-77383) (UI: GA: Eject/Inject CD-ROM Support), [CNV-64402](https://issues.redhat.com/browse/CNV-64402) (GA: Adopt declarative hotplug volumes API) |
| **QE Owner(s)**        | Yan Du                                                            |
| **Owning SIG**         | sig-storage                                                       |
| **Participating SIGs** | sig-compute, sig-network                                          |
| **Current Status**     | GA (off by default; fully-supported when enabled)                 |

**Document Conventions (if applicable):** This STP covers the GA promotion of the Eject/Inject CD-ROM feature via declarative hotplug volumes. The feature is GA but remains off by default until telemetry confirms safety for broad enablement (per CNV-79690). Ephemeral hotplug volumes are not supported when the `DeclarativeHotplugVolumes` feature gate is enabled. The `--persist` flag in virtctl is deprecated; persist-by-default is the new behavior.

### **Feature Overview**

The Eject/Inject CD-ROM Support feature allows VM owners to change CD-ROM media on a running VirtualMachine without requiring a restart. It is implemented via the `DeclarativeHotplugVolumes` feature gate, which replaces the imperative sub-resource based hotplug API with a declarative approach where volume changes are made directly in the VM spec and automatically propagated to the running VMI.

Key capabilities:

- **CD-ROM Inject:** Attach a CD-ROM volume (DataVolume, PVC) to a running VM by adding a volume reference to the VM spec for an existing empty CD-ROM disk
- **CD-ROM Eject:** Remove a CD-ROM volume from a running VM by removing the volume reference from the VM spec; the empty CD-ROM drive remains in the guest
- **CD-ROM Swap:** Replace the CD-ROM media source by updating the volume reference (e.g., changing the DataVolume name) in the VM spec
- **Empty CD-ROM Support:** VMs can be created with an empty CD-ROM disk (disk defined without a volume reference); the guest sees a drive with "No medium found"
- **Declarative Volume Hotplug:** Volume changes are persisted in the VM spec (GitOps-friendly), replacing the ephemeral VMI sub-resource API
- **Virtio Bus Support:** Hotplugged disks can now use the virtio bus in addition to SCSI and SATA (PR [#14907](https://github.com/kubevirt/kubevirt/pull/14907))
- **PCI Port Allocation:** Automatic allocation of PCI ports for hotplug capacity: 8 ports (at least 3 free) for VMs with 2G or less memory, 16 ports (at least 6 free) for VMs with more than 2G memory (PR [#14754](https://github.com/kubevirt/kubevirt/pull/14754))
- **RestartRequired Condition:** Removing a CD-ROM disk entry from the VM spec triggers RestartRequired instead of attempting a live SATA detach, which would fail at the libvirt level (PR [#15969](https://github.com/kubevirt/kubevirt/pull/15969))
- **virtctl Persist-by-Default:** `virtctl addvolume` and `virtctl removevolume` now persist changes to the VM spec by default; the `--persist` flag is deprecated (PR [#16280](https://github.com/kubevirt/kubevirt/pull/16280))
- **Ephemeral Hotplug Volume Metric:** New `kubevirt_vmi_contains_ephemeral_hotplug_volume` metric and alert to identify VMIs with ephemeral hotplug volumes ahead of `HotplugVolumes` deprecation (PR [#15815](https://github.com/kubevirt/kubevirt/pull/15815))

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

| Check                                  | Done | Details/Notes                                                                                                                                                                           | Comments |
|:---------------------------------------|:-----|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:---------|
| **Review Requirements**                | [x]  | Reviewed the relevant requirements.                                                                                                                                                     | Requirements defined across CNV-68916 (epic), linked issues CNV-79690, CNV-77383, CNV-64402, and upstream PRs kubevirt/kubevirt#13847, #15969, #14907, #14754, #16280, #15815, #14998, #15788. Feature gate `DeclarativeHotplugVolumes` replaces `HotplugVolumes`. |
| **Understand Value**                   | [x]  | Confirmed clear user stories and understood. <br/>Understand the difference between U/S and D/S requirements<br/> **What is the value of the feature for RH customers**.               | User story: "As a VM owner I want to change the CD-ROM media without needing to restart my VM." This eliminates service disruption for media changes, enables install-from-ISO workflows, and supports GitOps-friendly declarative configuration. Declarative approach is more aligned with Kubernetes-native workflows than the imperative sub-resource API. |
| **Customer Use Cases**                 | [x]  | Ensured requirements contain relevant **customer use cases**.                                                                                                                           | Primary use cases: OS installation from ISO without restart, driver/tool CD swap on running VMs, declarative volume management for GitOps pipelines, media rotation for content delivery, and automated provisioning workflows that require mounting different media at different lifecycle stages. |
| **Testability**                        | [x]  | Confirmed requirements are **testable and unambiguous**.                                                                                                                                | Feature is testable via VM spec manipulation and verification of running VMI state. Upstream e2e tests exist in `tests/storage/declarative-hotplug.go` and cover inject, swap, eject, and RestartRequired scenarios. Downstream tests exist in `tests/storage/test_hotplug.py`. |
| **Acceptance Criteria**                | [x]  | Ensured acceptance criteria are **defined clearly** (clear user stories; D/S requirements clearly defined in Jira).                                                                     | Acceptance criteria derived from upstream PR behavior and Jira epic: (1) inject, eject, swap CD-ROM operations work on running VMs, (2) RestartRequired condition set on CD-ROM disk removal, (3) virtctl persist-by-default behavior, (4) ephemeral hotplug restriction enforced, (5) feature is GA but off by default until telemetry confirms safe enablement. |
| **Non-Functional Requirements (NFRs)** | [x]  | Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs. | Performance: PCI port allocation scheme ensures sufficient hotplug capacity without excessive resource overhead. Security: RBAC enforcement for VM spec modifications. Monitoring: Ephemeral hotplug volume metric `kubevirt_vmi_contains_ephemeral_hotplug_volume` and associated alert added (PR #15815). Docs: downstream doc story tracked. |

#### **2. Technology and Design Review**

| Check                            | Done | Details/Notes                                                                                                                                           | Comments |
|:---------------------------------|:-----|:--------------------------------------------------------------------------------------------------------------------------------------------------------|:---------|
| **Developer Handoff/QE Kickoff** | [x]  | A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.** | Feature designed by Michael Henriksen (mhenriks). VEP tracked at kubevirt/enhancements#31. Architecture uses VM controller to reconcile VM spec changes to VMI, and virt-launcher to handle CD-ROM operations via libvirt. Core hotplug logic centralized in `pkg/storage/hotplug/hotplug.go`. Danny Sanatar (Dsanatar) contributed CD-ROM detach fix, virtctl persist-by-default, and ephemeral hotplug metric. |
| **Technology Challenges**        | [x]  | Identified potential testing challenges related to the underlying technology.                                                                            | SATA CD-ROM drives cannot be hot-detached from QEMU/libvirt; removing a CD-ROM disk from VM spec triggers RestartRequired instead of live removal. PCI address stability across upgrades is an open concern (PR #17029 in progress). Ephemeral hotplugged disks are not compatible with DeclarativeHotplugVolumes. Volume unplug ordering was a source of flaky tests; fix in PR #14998 reorders unplug to: unplug from VM, unmount from virt-launcher, delete hotplug pod, remove volume status. |
| **Test Environment Needs**       | [x]  | Determined necessary **test environment setups and tools**.                                                                                              | Requires cluster with `DeclarativeHotplugVolumes` feature gate enabled via HyperConverged CR or KubeVirt CR configuration. CDI must be available for DataVolume creation. Storage class supporting ReadWriteOnce PVCs with dynamic provisioning is required. |
| **API Extensions**               | [x]  | Reviewed new or modified APIs and their impact on testing.                                                                                               | VM spec now supports empty CD-ROM disks (disk defined without volume reference). New `hotpluggable: true` field on DataVolumeSource entries. `DeclarativeHotplugVolumes` feature gate added to `developerConfiguration.featureGates`. `--persist` flag deprecated in virtctl `addvolume`/`removevolume`. VMI update admitter modified to allow virtio bus for hotplugged disks. New ephemeral hotplug annotation on VMIs. |
| **Topology Considerations**      | [x]  | Evaluated multi-cluster, network topology, and architectural impacts.                                                                                    | No multi-cluster considerations. Feature operates within single-cluster scope. Storage backend must support dynamic provisioning for DataVolume creation. Migration compatibility verified (hotplugged volumes work correctly during live migration). |

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

This test plan covers the GA validation of the CD-ROM Eject/Inject feature via declarative hotplug volumes in OpenShift Virtualization. Testing scope includes the core CD-ROM operations (inject, eject, swap), empty CD-ROM support, declarative volume hotplug behavior, feature gate interactions, bus type support (SATA, virtio), PCI port allocation, RestartRequired condition handling, virtctl persist-by-default behavior, ephemeral hotplug restriction, monitoring (metric and alert), RBAC enforcement, integration with existing features (migration, snapshots, restore), and regression coverage for areas impacted by the code changes.

**Testing Goals**

- Validate CD-ROM inject, eject, and swap operations on running VMs without restart
- Verify declarative hotplug volumes correctly reconcile VM spec changes to running VMIs
- Confirm empty CD-ROM drive can be defined in VM spec and reports "No medium found" in guest
- Confirm RestartRequired condition is set when CD-ROM disk entries are removed from VM spec
- Validate feature gate behavior (DeclarativeHotplugVolumes enable/disable, interaction with HotplugVolumes)
- Verify virtio bus support for hotplugged disks alongside existing SATA/SCSI support
- Validate PCI port allocation for hotplug capacity (8 ports for small VMs, 16 for large VMs)
- Verify virtctl addvolume/removevolume persist-by-default behavior and --persist flag deprecation
- Confirm ephemeral hotplug volume restriction and associated monitoring metric/alert
- Validate no regressions in existing hotplug, migration, snapshot, and restore workflows
- Validate error handling for invalid operations (e.g., operations with feature gate disabled, unsupported bus types)
- Verify RBAC enforcement for VM spec volume modifications

**Out of Scope (Testing Scope Exclusions)**

| Out-of-Scope Item | Rationale | PM/ Lead Agreement |
|:-------------------|:----------|:-------------------|
| UI testing of CD-ROM eject/inject | Covered by separate epic CNV-77383 with dedicated QE (Guohua Ouyang) | Agreed |
| Kubernetes storage provisioner internals | Platform-level testing; CSI driver validation is the responsibility of the storage team | Agreed |
| QEMU/libvirt CD-ROM device emulation | Upstream component; validated by libvirt/QEMU upstream testing | Agreed |
| Multi-cluster federation scenarios | Feature operates within single-cluster scope; no federation requirements | Agreed |
| Performance benchmarking of hotplug latency | Not a GA requirement; may be pursued separately if needed | Agreed |

#### **2. Test Strategy**

| Item                           | Description                                                                                                                                                  | Applicable (Y/N or N/A) | Comments |
|:-------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------|:------------------------|:---------|
| Functional Testing             | Validates that the feature works according to specified requirements and user stories                                                                        | Y | Core CD-ROM inject/eject/swap operations; empty CD-ROM; feature gate behavior; API validation; RestartRequired conditions; bus type support; PCI port allocation; virtctl persist-by-default |
| Automation Testing             | Ensures test cases are automated for continuous integration and regression coverage                                                                          | Y | Upstream tests in `tests/storage/declarative-hotplug.go` and `tests/hotplug/pci-ports.go`; downstream automation in `tests/storage/test_hotplug.py` |
| Performance Testing            | Validates feature performance meets requirements (latency, throughput, resource usage)                                                                       | Y | Validate PCI port allocation does not degrade VM startup time; verify hotplug operations complete within acceptable time windows |
| Security Testing               | Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning                                                              | Y | RBAC enforcement for VM spec modifications; validate non-privileged users cannot bypass feature gate restrictions |
| Usability Testing              | Validates user experience, UI/UX consistency, and accessibility requirements. Does the feature require UI? If so, ensure the UI aligns with the requirements | N/A | UI testing covered by CNV-77383 epic |
| Compatibility Testing          | Ensures feature works across supported platforms, versions, and configurations                                                                               | Y | Test with various storage backends (OCS/ODF, hostpath-provisioner); test with different VM configurations and memory sizes |
| Regression Testing             | Verifies that new changes do not break existing functionality                                                                                                | Y | Validate existing hotplug volume workflows; verify snapshot/restore with hotplugged volumes; verify migration compatibility; verify volume unplug ordering |
| Upgrade Testing                | Validates upgrade paths from previous versions, data migration, and configuration preservation                                                               | Y | Verify feature gate state preserved across upgrade; verify VM with hotplugged volumes survives upgrade; PCI address stability across upgrades |
| Backward Compatibility Testing | Ensures feature maintains compatibility with previous API versions and configurations                                                                        | Y | When both HotplugVolumes and DeclarativeHotplugVolumes are enabled, HotplugVolumes takes precedence; virtctl --persist flag deprecated but still functional |
| Dependencies                   | Dependent on deliverables from other components/products? Identify what is tested by which team.                                                             | Y | CDI (DataVolume provisioning); virt-controller (VM reconciliation); virt-launcher (libvirt operations); virt-handler (volume mount/unmount). UI team tests CNV-77383 separately. |
| Cross Integrations             | Does the feature affect other features/require testing by other components? Identify what is tested by which team.                                           | Y | Memory dump (uses volume API); kubevirt-csi; virtctl volume operations; snapshot/restore with hotplugged volumes; live migration with hotplugged volumes |
| Monitoring                     | Does the feature require metrics and/or alerts?                                                                                                              | Y | `kubevirt_vmi_contains_ephemeral_hotplug_volume` metric and associated alert added (PR #15815); verify metric exposed correctly and alert fires when ephemeral hotplugs detected |
| Cloud Testing                  | Does the feature require multi-cloud platform testing? Consider cloud-specific features.                                                                     | N | Feature is storage-backend agnostic; no cloud-specific behavior |

#### **3. Test Environment**

| Environment Component                         | Configuration | Specification Examples |
|:----------------------------------------------|:--------------|:-----------------------|
| **Cluster Topology**                          | Standard HA cluster | 3 control-plane + 2 worker nodes minimum |
| **OCP & OpenShift Virtualization Version(s)** | OCP 4.18+, CNV 4.22+ | OCP 4.18 with CNV 4.22.0 |
| **CPU Virtualization**                        | Hardware virtualization enabled | Intel VT-x or AMD-V |
| **Compute Resources**                         | Adequate for running multiple VMs | Workers with 16 vCPU, 64Gi RAM minimum |
| **Special Hardware**                          | None required | N/A |
| **Storage**                                   | Dynamic provisioning with RWO support | OCS/ODF, hostpath-provisioner, or NFS-backed StorageClass |
| **Network**                                   | Standard cluster networking | OVN-Kubernetes or OpenShiftSDN |
| **Required Operators**                        | OpenShift Virtualization, CDI | HyperConverged operator with DeclarativeHotplugVolumes feature gate enabled |
| **Platform**                                  | Bare metal or virtualized | Bare metal preferred for production-like validation |
| **Special Configurations**                    | Feature gate configuration | `DeclarativeHotplugVolumes` enabled via HyperConverged CR: `spec.featureGates.declarativeHotplugVolumes: true` |

#### **3.1. Testing Tools & Frameworks**

| Category           | Tools/Frameworks |
|:-------------------|:-----------------|
| **Test Framework** | Ginkgo v2 + Gomega (Tier 1), pytest (Tier 2) |
| **CI/CD**          | OpenShift CI (Prow), Jenkins |
| **Other Tools**    | virtctl, oc CLI, kubectl |

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [x] Requirements and design documents are **approved and merged**
- [x] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] `DeclarativeHotplugVolumes` feature gate is available in the target CNV version
- [ ] CDI operator is deployed and functional for DataVolume creation
- [ ] At least one StorageClass with dynamic provisioning is available
- [ ] Upstream e2e tests from `tests/storage/declarative-hotplug.go` pass in CI

#### **5. Risks**

| Risk Category        | Specific Risk for This Feature | Mitigation Strategy | Status |
|:---------------------|:-------------------------------|:--------------------|:-------|
| Timeline/Schedule    | Feature is GA but off by default; may delay full rollout until telemetry confirms safety | Track CNV-79690 (Declarative Hotplug on by default) for timeline alignment | [ ] |
| Test Coverage        | Upstream test flakiness in declarative hotplug path (volume unplug ordering race conditions) | Monitor PR #14998 fix in CI; quarantine and investigate any remaining flakes | [ ] |
| Test Environment     | Storage backend variability may mask issues | Test with at least two storage backends (OCS/ODF and hostpath-provisioner) | [ ] |
| Untestable Aspects   | PCI address stability across upgrades (PR #17029 still in progress) | Monitor upstream PR progress; add upgrade-specific PCI address tests when fix is available | [ ] |
| Resource Constraints | Multiple VMs and DataVolumes needed for hotplug testing consume significant cluster resources | Pre-provision DataVolumes to reduce test setup time; use small VM images (Cirros) where possible | [ ] |
| Dependencies         | CDI must be functional for DataVolume-based tests | Include CDI health check as test precondition; have PVC-based fallback tests | [ ] |
| Other                | Interaction between HotplugVolumes and DeclarativeHotplugVolumes feature gates creates four state combinations | Test all four combinations of feature gate states (both on, both off, each independently) | [ ] |

#### **6. Known Limitations**

- **Ephemeral hotplug volumes are not supported** when `DeclarativeHotplugVolumes` is enabled. If both `DeclarativeHotplugVolumes` and `HotplugVolumes` are enabled, `HotplugVolumes` takes precedence to maintain backward compatibility. Ephemeral volumes in a VMI that are not present in the owner VM spec are automatically removed by the VM controller.
- **SATA CD-ROM drives cannot be hot-detached** from a running VM. Removing a CD-ROM disk entry from the VM spec triggers a `RestartRequired` condition instead of attempting a live removal (which would cause a libvirt error). The CD-ROM media can be ejected without restart, but the disk device itself requires a restart to be removed.
- **PCI address stability across upgrades** is an open concern. Upstream PR kubevirt/kubevirt#17029 is addressing this with a v3 hotplug port topology, but it has not yet merged.
- **Feature is GA but off by default.** The `DeclarativeHotplugVolumes` feature gate must be explicitly enabled. It will remain off by default until telemetry confirms safety for broad enablement (tracked in CNV-79690).
- **VM must be running** for hotplug operations to take effect on the VMI. Changes to a stopped VM spec are applied on next boot without requiring the feature gate.
- **Volume ordering changes do not trigger RestartRequired.** Adding a hotplug volume at any position in the volumes list (beginning, end) should not produce a RestartRequired condition (PR #15788 fix).

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

| Requirement ID | Requirement Summary | Test Scenario(s) | Tier | Priority |
|:---------------|:--------------------|:-----------------|:-----|:---------|
| REQ-CDROM-INJECT-01 | CD-ROM can be injected into a running VM via declarative hotplug | Verify CD-ROM inject with DataVolume source on running VM with empty CD-ROM; confirm volume appears in VMI status and CD-ROM is mountable/readable in guest | Tier 1 | P0 |
| REQ-CDROM-INJECT-02 | CD-ROM inject works with PVC volume source | Verify CD-ROM inject with PVC source on running VM; confirm CD-ROM content is accessible in guest | Tier 1 | P1 |
| REQ-CDROM-INJECT-03 | Injected CD-ROM reflects correct content in guest | Verify injected CD-ROM is mountable, readable, and shows expected file count and content inside guest OS | Tier 1 | P0 |
| REQ-CDROM-INJECT-04 | CD-ROM inject fails gracefully when feature gate is disabled | Verify that adding a volume to an empty CD-ROM disk does not hot-inject when DeclarativeHotplugVolumes gate is disabled (requires restart) | Tier 1 | P1 |
| REQ-CDROM-EJECT-01 | CD-ROM can be ejected from a running VM by removing volume reference | Verify CD-ROM eject by removing volume reference from VM spec while keeping the CD-ROM disk entry; confirm volume is unplugged and guest reports "No medium found" | Tier 1 | P0 |
| REQ-CDROM-EJECT-02 | Ejected CD-ROM drive remains as empty device in guest | Verify that after ejecting CD-ROM media, the /dev/sr0 device still exists in guest but reports "No medium found" on mount attempt | Tier 1 | P1 |
| REQ-CDROM-EJECT-03 | CD-ROM eject fails gracefully when feature gate is disabled | Verify that removing a volume reference with feature gate disabled does not hot-eject (change queued for next restart) | Tier 1 | P1 |
| REQ-CDROM-SWAP-01 | CD-ROM media can be swapped on a running VM without restart | Verify CD-ROM swap by replacing the DataVolume name in the volume reference; confirm new CD-ROM content is accessible in guest and old content is no longer present | Tier 1 | P0 |
| REQ-CDROM-SWAP-02 | CD-ROM swap between different volume types | Verify CD-ROM swap from DataVolume source to PVC source and vice versa | Tier 1 | P1 |
| REQ-CDROM-SWAP-03 | CD-ROM swap preserves VM operation without restart | Verify VM continues running without interruption during CD-ROM swap; verify no RestartRequired condition after swap | Tier 1 | P0 |
| REQ-EMPTY-CDROM-01 | Empty CD-ROM drive can be defined in VM spec | Verify VM starts successfully with an empty CD-ROM disk (disk defined without volume reference); confirm guest OS boots and /dev/sr0 exists | Tier 1 | P0 |
| REQ-EMPTY-CDROM-02 | Empty CD-ROM reports "No medium found" in guest | Verify that attempting to mount an empty CD-ROM in guest returns "No medium found" error | Tier 1 | P1 |
| REQ-FGATE-01 | DeclarativeHotplugVolumes feature gate enables CD-ROM hotplug | Verify CD-ROM inject/eject/swap operations succeed when DeclarativeHotplugVolumes feature gate is enabled | Tier 1 | P0 |
| REQ-FGATE-02 | CD-ROM hotplug blocked when feature gate is disabled | Verify CD-ROM eject/inject operations are not hot-applied when DeclarativeHotplugVolumes feature gate is disabled | Tier 1 | P0 |
| REQ-FGATE-03 | HotplugVolumes takes precedence when both gates enabled | Verify that when both HotplugVolumes and DeclarativeHotplugVolumes are enabled, the legacy HotplugVolumes behavior takes precedence | Tier 1 | P1 |
| REQ-FGATE-04 | Both feature gates disabled | Verify that with both feature gates disabled, volume changes in VM spec require a restart to take effect | Tier 1 | P2 |
| REQ-RESTART-01 | Removing CD-ROM disk from VM spec triggers RestartRequired | Verify RestartRequired condition is set when a CD-ROM disk entry (not just the volume reference) is removed from VM spec while the VM is running | Tier 1 | P0 |
| REQ-RESTART-02 | VM continues running after RestartRequired is set | Verify VM remains operational after RestartRequired condition; guest workloads continue running; the removed CD-ROM is still accessible until restart | Tier 1 | P1 |
| REQ-RESTART-03 | CD-ROM disk removal takes effect after restart | Verify that after restarting a VM with RestartRequired condition from CD-ROM disk removal, the CD-ROM device is no longer present in the guest | Tier 1 | P1 |
| REQ-RESTART-04 | Volume ordering changes do not trigger RestartRequired | Verify that adding a hotplug volume at the beginning of the volumes list does not trigger RestartRequired condition | Tier 1 | P1 |
| REQ-BUS-01 | Hotplugged disks support virtio bus type | Verify disk hotplug with virtio bus type succeeds; confirm disk is accessible in guest with virtio driver | Tier 1 | P1 |
| REQ-BUS-02 | CD-ROM hotplug works with SATA bus | Verify CD-ROM hotplug with SATA bus succeeds (default CD-ROM bus type); confirm CD-ROM accessible in guest | Tier 1 | P1 |
| REQ-BUS-03 | SCSI bus supported for hotplugged disks | Verify disk hotplug with SCSI bus type succeeds | Tier 1 | P2 |
| REQ-PCI-01 | VMs with 2G or less memory get 8 PCI ports | Verify a VM configured with 2G or less guest memory has 8 total PCI ports allocated for hotplug; at least 3 ports are free for use | Tier 1 | P1 |
| REQ-PCI-02 | VMs with more than 2G memory get 16 PCI ports | Verify a VM configured with more than 2G guest memory has 16 total PCI ports allocated for hotplug; at least 6 ports are free for use | Tier 1 | P1 |
| REQ-PCI-03 | Hotplug operations respect PCI port limits | Verify that hotplugging volumes up to the free port limit succeeds; verify appropriate error when exceeding available ports | Tier 1 | P2 |
| REQ-VIRTCTL-01 | virtctl addvolume persists to VM by default | Verify `virtctl addvolume` persists the volume in both VM and VMI specs without requiring the `--persist` flag | Tier 1 | P1 |
| REQ-VIRTCTL-02 | virtctl removevolume persists to VM by default | Verify `virtctl removevolume` removes the volume from both VM and VMI specs without requiring the `--persist` flag | Tier 1 | P1 |
| REQ-VIRTCTL-03 | --persist flag shows deprecation warning | Verify that using the `--persist` flag with `virtctl addvolume` or `virtctl removevolume` produces a deprecation warning but still functions correctly | Tier 1 | P2 |
| REQ-VIRTCTL-04 | Standalone VMI behavior unaffected | Verify that `virtctl addvolume`/`removevolume` behavior for standalone VMIs (not owned by a VM) remains unchanged | Tier 1 | P2 |
| REQ-EPHEMERAL-01 | Ephemeral hotplug volumes removed by VM controller | Verify that when DeclarativeHotplugVolumes is enabled and a volume exists only in the VMI (not the VM spec), the VM controller removes it | Tier 1 | P1 |
| REQ-EPHEMERAL-02 | Ephemeral hotplug volume metric exposed | Verify `kubevirt_vmi_contains_ephemeral_hotplug_volume` metric is exposed when a VMI contains ephemeral hotplug volumes; verify the associated alert fires | Tier 1 | P2 |
| REQ-HOTPLUG-DISK-01 | Non-CD-ROM disk can be hotplugged declaratively | Verify adding a hotplug disk (not CD-ROM) by updating VM spec; disk appears in VMI and is usable in guest | Tier 1 | P1 |
| REQ-HOTPLUG-DISK-02 | Non-CD-ROM disk can be hot-unplugged declaratively | Verify removing a hotplug disk and volume from VM spec; disk is detached from VMI and inaccessible in guest | Tier 1 | P1 |
| REQ-E2E-LIFECYCLE-01 | Full CD-ROM inject, swap, and eject lifecycle | Verify complete lifecycle: create VM with empty CD-ROM, inject media, verify content, swap media, verify new content, eject media, verify "No medium found", re-inject and verify again | Tier 2 | P0 |
| REQ-E2E-PERSIST-01 | CD-ROM hotplug state persists through VM restart | Verify injected CD-ROM volume survives VM stop/start cycle; content remains accessible after restart | Tier 2 | P0 |
| REQ-E2E-PERSIST-02 | Ejected CD-ROM state persists after VM restart | Verify that an ejected CD-ROM (empty drive) persists correctly after VM restart; guest still sees empty CD-ROM device | Tier 2 | P1 |
| REQ-E2E-MIGRATION-01 | VM with hotplugged CD-ROM migrates successfully | Verify VM with an injected CD-ROM can be live-migrated; CD-ROM remains accessible on target node after migration | Tier 2 | P1 |
| REQ-E2E-MIGRATION-02 | Hotplugged disk data accessible after migration | Verify hotplugged data disk content is intact and accessible after live migration | Tier 2 | P1 |
| REQ-E2E-SNAPSHOT-01 | Snapshot of VM with hotplugged volumes | Verify snapshot creation succeeds for a VM with declaratively hotplugged volumes | Tier 2 | P1 |
| REQ-E2E-SNAPSHOT-02 | Restore of VM with hotplugged volumes | Verify restore from snapshot preserves hotplugged volume configuration and data | Tier 2 | P1 |
| REQ-E2E-UPGRADE-01 | Feature gate state preserved across upgrade | Verify DeclarativeHotplugVolumes feature gate configuration survives OCP/CNV upgrade | Tier 2 | P1 |
| REQ-E2E-UPGRADE-02 | VM with hotplugged volumes operational after upgrade | Verify VM with hotplugged CD-ROM and disk volumes continues to function correctly after cluster upgrade | Tier 2 | P1 |
| REQ-RBAC-01 | Non-admin user cannot modify VM volumes without permission | Verify that a user without VM edit permissions cannot add or remove volumes from a VM spec | Tier 1 | P1 |
| REQ-RBAC-02 | Admin can grant volume management permissions | Verify that a cluster admin can grant a user permission to modify VM volumes via RBAC roles | Tier 1 | P2 |
| REQ-NEGATIVE-01 | Hotplug of non-hotpluggable volume is rejected | Verify that attempting to hotplug a volume without `hotpluggable: true` is rejected or triggers RestartRequired | Tier 1 | P2 |
| REQ-NEGATIVE-02 | Invalid volume reference is handled gracefully | Verify that referencing a non-existent DataVolume or PVC in a CD-ROM volume update is handled with appropriate error/event | Tier 1 | P2 |

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - Yan Du
  - [Name / @github-username]
* **Approvers:**
  - Adam Litke
  - [Name / @github-username]
