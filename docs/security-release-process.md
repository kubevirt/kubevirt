# KubeVirt Security Release Process

This document describes how the KubeVirt project handles vulnerabilities
from initial report through a patched release.
It covers the response lifecycle that sits between reporting
(see [SECURITY.md](../SECURITY.md)) and a standard patch release (see
[release-procedure.md](release-procedure.md)).

The goal is to clearly define this process so that it is consistent and reproducible,
and to minimise the gap between CVE embargo lift and an
available upstream patch by (a) developing and testing fixes privately during
the embargo and (b) cutting the release on disclosure day with a pre-checked,
urgency-driven checklist.

## Roles

These roles may be held by the same person for a small incident. All role
holders are drawn from the KubeVirt Security Team, which is defined in
[SECURITY.md](../SECURITY.md#security-team) as the project Maintainers (see
[community/MAINTAINERS.md](https://github.com/kubevirt/community/blob/main/MAINTAINERS.md)),
supported by appropriate members of KubeVirt SIGs, and involved vendor security teams.

- **Security Release Coordinator (SRC)**: Owns the incident end-to-end.
  Appointed per-incident. Any maintainer may serve; by default the maintainer
  carrying security responsibility takes it or explicitly delegates. The SRC
  **must have GitHub `admin` (or Security Manager) permission** on
  `kubevirt/kubevirt`, because creating a draft advisory and a private fork
  requires it (see Section 4). If the appointed coordinator lacks this
  permission, either grant it or co-appoint a maintainer who has it.
- **Fix Developer(s)**: Develop and test the patch in the private fork.
- **Release Cutter**: Runs `hack/release.sh`. May be the SRC or another
  maintainer with release credentials staged (see Section 5).

## Severity and Response Guidelines

Guidance is expressed as priority expectations rather than numeric SLAs. The team
commits to relative urgency, not to hour/day deadlines it cannot consistently
meet.

| Severity | Acknowledgment | Fix development | Release after embargo lift |
| -------- | -------------- | --------------- | -------------------------- |
| Critical | Fastest; acknowledge before other security work | Prioritized above other work | As soon as practicable on disclosure day |
| High     | Prompt | Prioritized above feature work | As soon as practicable on/near disclosure day |
| Medium   | Prompt | Normal cadence, ahead of features | Follows the expedited patch process, not necessarily same-day |
| Low      | Prompt | Normal cadence | May ride the next regular patch release |

### Invoking this process 
The SRC (in consultation with the security team)
decides whether an incident runs the full embargo + private-fork workflow. In
practice:

* Critical/High always do; 
* Medium/Low may use the standard patch process unless there is a reason to embargo. 

Severity is reassessed as understanding evolves, 
i.e. if a Medium is found to be Critical during fix development then it will be escalated.

## Process
### Phase 1: Report Intake and Triage

1. Reports arrive via `cncf-kubevirt-security@lists.cncf.io` or GitHub private
   vulnerability reporting.
2. Following initial review of the report, 1 or more SMEs from applicable KubeVirt SIGs may 
   be added for deeper assessment. Based on this assessment, the team acknowledges
   the reporter promptly (urgency per Section 2) and adds vendor security teams if applicable.
3. Project maintainers appoint the SRC (confirm they hold or can obtain the GitHub `admin` or Security Manager role; otherwise co-appoint).
4. Triage: validate the report, assign a CVSS score/severity, and identify
   affected components and versions.
5. Determine which supported release branches need patches. Consult the current 
   support matrix at
   [sig-release/releases/k8s-support-matrix.md](https://github.com/kubevirt/sig-release/blob/main/releases/k8s-support-matrix.md).
   Only currently supported releases receive security backports.

### Phase 2: GitHub Security Advisory and Private Fork

> **Permissions note**
> Drafting a GHSA, adding collaborators, and creating the temporary private
> fork require `admin` or Security Manager role on `kubevirt/kubevirt`. Confirm
> the SRC (or a co-appointed maintainer) has this before starting.

1. The SRC creates a **draft** GitHub Security Advisory (GHSA) on
   `kubevirt/kubevirt`.
2. Add the relevant security team members and fix developers as advisory
   collaborators. Add only those who need access for this incident.
3. Create the temporary **private fork** from the advisory.
4. In the private fork, create a working branch for `main` plus one per affected
   release branch identified in Phase 1.
5. Fix developers push commits; PRs are reviewed within the private
   fork.
6. **Testing is local-only** (Prow CI cannot run against private forks):
   - `make cluster-up && make cluster-sync` to bring up a cluster with the fix.
   - `make test` for unit tests.
   - `make functest` for targeted functional tests.
   - Reviewers are expected to explicitly confirm local test results in the PR, since CI
     signal is unavailable.
7. Stage the fix so that it is ready to merge on disclosure day.

### Phase 3: Pre-Disclosure Coordination

1. Coordinate the disclosure date with the reporter and involved vendor teams.
2. Request a CVE ID if not yet assigned (GitHub advisory auto-request, or
   MITRE/CNCF).
3. Draft in advance: advisory text, announcement email, and the affected/fixed
   version lists.
4. Release readiness pre-checks (completed during embargo, not on
   disclosure day):
   - Confirm the Release Cutter's credentials are staged: GPG key file, GPG
     passphrase file, and GitHub API token file, with the corresponding env
     vars exported (see [release-procedure.md](release-procedure.md#release-tool-credentials)).
   - **Confirm there are no open `/release-blocker` labels** on `main` or any
     target release branch. An open blocker will cause the release tool to
     refuse to tag (see
     [release-procedure.md](release-procedure.md#handling-release-blockers)).
     Resolve or, if truly not a blocker, cancel it before disclosure day.
   - Pre-pull the release container so a registry hiccup doesn't stall
     disclosure day: `${KUBEVIRT_CRI} pull quay.io/kubevirtci/release-tool:latest`.
5. For multi-repo CVEs: see Section 9.

### Phase 4: Disclosure Day Checklist (Day-0)

`${TAG}` below is the full `v`-prefixed tag (e.g. `v1.9.1`); pass it bare to the release tool.

0. **Timing.** Target a non-Friday weekday, early afternoon UTC (morning
   US Pacific, mid-late afternoon in Central Europe) to maximize maintainer availability and
   avoid weekend on-call exposure.
1. **Merge fixes.** Use the advisory's "merge to main"; then cherry-pick/merge
   to each affected release branch. The `/cherry-pick` bot does not work from a
   private fork. These need to be done manually (see Section 9.)
2. **Verify CI.** Wait for Prow presubmits on the now-public cherry-pick PRs. If
   CI shows only unrelated/flaky failures and the fix was tested locally during
   the embargo, the SRC plus a second maintainer may approve with a documented
   note explaining the override.
3. **Tag releases.** For each affected branch, starting with the most recent:
   `hack/release.sh --new-tag ${TAG} --dry-run=false`
   (branch is auto-detected). Stagger tags across branches to avoid
   overloading postsubmit CI.
4. **Monitor postsubmit.** Watch the `push-release-kubevirt-tag` job at
   prow.ci.kubevirt.io. Verify artifacts on the GitHub release page and images
   on quay.io/kubevirt.
5. **Finalize releases.** In the GitHub UI, uncheck "This is a pre-release" for
   each release.
6. **Publish the GHSA.** Transition it from draft to published.
7. **Announce.** Email `kubevirt-dev@googlegroups.com` with the CVE ID,
   severity, affected/fixed versions, GHSA link, and reporter credit. Post to
   the KubeVirt `#virtualization` channel on Kubernetes Slack. Send subsequent 
   message on the social channels.

## Communication Plan

| Lifecycle stage | Audience | Channel |
| --------------- | -------- | ------- |
| Acknowledgment | Reporter | Reply on intake thread (email or GitHub advisory) |
| Triage / coordination | Security team, vendor security teams | Private advisory + `cncf-kubevirt-security@lists.cncf.io` |
| Disclosure coordination | Reporter, vendors, upstream | Advisory thread / direct |
| Disclosure day | All users | `kubevirt-dev@googlegroups.com`, published GHSA, `#virtualization` on Kubernetes Slack |

## Security Fix Backporting Exceptions

How the standard policy in [release-branch-backporting.md](release-branch-backporting.md)
is adapted for security fixes:

- "Merged to main first" remains the goal, but during embargo the fix is
  prepared against `main` and all affected release branches simultaneously
  in the private fork.
- Local testing substitutes for CI during embargo only. Public post-merge CI
  is still required before tagging.
- Only currently supported releases (per the support matrix) receive security
  backports.

## Limitations & Gotchas

1. Prow CI cannot run on private forks: all pre-disclosure testing is local 
   ([See Phase 3](#phase-2-github-security-advisory-and-private-fork)).
2. The `/cherry-pick` bot does not work from private forks: manual cherry-picks to
   the release branches are required on disclosure day.
3. Creating the GHSA and private fork requires either `admin` or Security Manager
   permission.
4. Release cutter credentials **and** the release container image must be staged
   before disclosure day, not set up during it.
5. An open `/release-blocker` will silently prevent tagging. Clear it during
   the embargo.
6. Stagger tags across branches to avoid overloading CI postsubmit.
7. Branch count is dynamic. Consult the support matrix for affected releases.
8. Dependency-only CVEs (Go module bumps) are lower risk but still follow this
   process when embargoed.
9. **Multi-repo CVEs.** CDI (`containerized-data-importer`) and other org repos
   share the same intake (`cncf-kubevirt-security@lists.cncf.io`) and advisory
   page, so cross-repo CVEs recur. For these: create a **separate GHSA per
   affected repo**, keep a single SRC coordinating across them, align on **one
   shared disclosure date**, and publish/announce the repos together. If a fix
   in one repo depends on another, note the ordering explicitly in the
   coordination thread.
10. The security team roster maps to current Maintainers. Before granting
    advisory/fork access, verify collaborators against the current
    `community/MAINTAINERS.md` file.

## Retrospective

Within 2 weeks of disclosure, the SRC writes a brief retrospective
(what happened, timeline, what to improve) and:

- files it as a tracking issue (or shares it on the maintainer list), so it is
  not lost, and
- opens a PR updating this document if the incident surfaced any process gaps.
