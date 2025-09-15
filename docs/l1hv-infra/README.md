# L1HV CI/CD Infrastructure

> Concise documentation of the CI/CD pipeline, environments, feature gates, and test strategy for introducing the L1HV feature.

## 1. Purpose
Provide a clear view of how L1HV-related changes are validated through layered environments (Dev -> Test variants) with selective test execution to accelerate feedback while safeguarding stability.

## 2. High-Level Flow

```mermaid
graph LR
  A[PR Open / Update] --> B[Quality Checks (pr.yaml)
  - Format / Lint
  - Unit Tests
  - Static Analysis]
  B --> C[Build & Push Images]
  C --> D[Deploy to Dev]
  D --> E[Run Dev Tests
   - New L1HV tests only
   - FeatureGate: L1HV=Enabled]
  E --> F[Promote Artifacts]
  F --> G{Parallel Test Environments}
  G --> H1[ test-kvm
   - Existing e2e only
   - Excludes label: mshv
   - L1HV Gate: Disabled]
  G --> H2[test-emulation
   - All e2e (existing + new)
   - Emulation mode
   - L1HV Gate: Disabled]
  G --> H3[test-mshv
   - All e2e (existing + new)
   - Hypervisor: MSHV
   - L1HV Gate: Enabled]
```

## 3. Environments (GitHub Environments)
Each environment is a GitHub Environment holding secrets/vars for deploy & test.

| Environment | Purpose | Feature Gate L1HV | Test Scope | Hypervisor Mode |
|-------------|---------|-------------------|------------|-----------------|
| `dev` | Fast feedback for new feature coverage | Enabled | New L1HV tests only | Host default (KVM) |
| `test-kvm` | Regression of existing stable e2e set | Disabled | Existing e2e excluding `mshv`-labeled | KVM |
| `test-emulation` | Full logical coverage under emulation | Disabled | All e2e (existing + new) | QEMU Emulation |
| `test-mshv` | Full suite on Microsoft Hyper-V stack | Enabled | All e2e (existing + new) | MSHV |

> Note: If a variant named `test-lvm` was referenced earlier, it is assumed to be `test-kvm` (please adjust if different).

## 4. Test Selection Logic

```mermaid
flowchart TD
  subgraph DEV
    D1[Select tests tagged: new-L1HV]
    D2[Enable L1HV gate]
  end

  subgraph TEST-KVM
    K1[Select tests: existing-e2e]
    K2[Exclude label: mshv]
    K3[Disable L1HV gate]
  end

  subgraph TEST-EMULATION
    E1[Select all e2e]
    E2[Force Emulation Mode]
    E3[Disable L1HV gate]
  end

  subgraph TEST-MSHV
    M1[Select all e2e]
    M2[Hypervisor=mshv]
    M3[Enable L1HV gate]
  end
```

### Label / Filter Reference (proposed)
| Purpose | Mechanism |
|---------|-----------|
| New L1HV tests | Label: `l1hv-new` or build tag/regex focus |
| Exclude MSHV-only from KVM regression | Label: `mshv` |
| Emulation mode activation | Env var / flag (e.g. `KUBEVIRT_EMULATION=true`) |
| Feature gate control | KubeVirt CR patch / deployment overlay |

## 5. Promotion & Artifacts
1. Images built once after Quality Checks pass.
2. Dev deployment uses same image digest as Test stages (immutability).
3. On successful Dev test run, same artifacts are promoted (no rebuild) to parallel test environments.
4. Each test environment publishes JUnit + logs (path: `_out/artifacts/...`).

## 6. Parallel Test Stage Orchestration

```mermaid
sequenceDiagram
    participant PR
    participant Dev
    participant KVM as test-kvm
    participant Emu as test-emulation
    participant MSHV as test-mshv

    PR->>Dev: Deploy & run new L1HV tests
    Dev-->>PR: JUnit (new feature signal)
    PR->>KVM: Deploy artifacts
    PR->>Emu: Deploy artifacts
    PR->>MSHV: Deploy artifacts
    par Parallel Exec
      KVM->>KVM: Existing e2e (exclude mshv)
      Emu->>Emu: All e2e (emulation)
      MSHV->>MSHV: All e2e (mshv, L1HV enabled)
    end
    KVM-->>PR: Stable regression results
    Emu-->>PR: Emulation coverage results
    MSHV-->>PR: MSHV + L1HV results
```

## 7. Failure Handling (Suggested)
| Stage | Action on Failure |
|-------|-------------------|
| Dev | Block promotion; annotate PR with failure summary |
| test-kvm | Mark regression risk; require review |
| test-emulation | Flag environment-specific issues |
| test-mshv | Flag hypervisor/feature-specific issues |

## 8. Future Enhancements
- Flake classification & auto-rerun for known flaky tests.
- Test focus/skip lists generated dynamically from labels.
- Dashboard aggregating pass rate per environment.
- Tighten artifact retention policies with pruning automation.
- Add mutation/fuzz tests gated behind opt-in label.

## 9. Quick Reference
| Item | Value |
|------|-------|
| Primary workflow file | `.github/workflows/pr.yaml` |
| Dev-only test focus | New L1HV labeled tests |
| Parallel environments | `test-kvm`, `test-emulation`, `test-mshv` |
| L1HV Gate Enabled In | `dev`, `test-mshv` |
| Artifact test report | `_out/artifacts/junit/junit.unittests.xml` |

---
_Last updated: TBD_
