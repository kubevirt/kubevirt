# Release Branch Backporting Policy

Bug fixes are eligible to be backported from the main branch to any previous
release branch. The following criteria must be met before a backport can be
considered. It is the reviewer's and approver's responsibility to uphold this
policy.

- **Bug Fix Only:** The backport must be a [bug fix](https://github.com/kubevirt/kubevirt/blob/main/docs/release-branch-backporting.md#bug-fix-definition) and the bug fix must be
first merged into the main branch. The only exception is when a bug only
exists in a stable branch and does not exist in the main branch.

- **Release Note** The PR description's release-note section must indicate in
a brief one line statement what the backport addresses. This note gets
automatically put into our release notes when a new release is cut from the
stable branch.

- **Detailed Description** The backport pull request's description must either
contain a detailed explanation for what the bug fix addresses or contain a
link to such a explanation present in another PR/Issue.

- **CI Lanes Must Pass** The release branch being backported to must still be
able to execute the unit and functional test lanes in order to validate the bug
fix on that branch. This only impacts very old branches that are no longer
compatible with our CI system.

# Bug Fix Definition

For the purposes of determining what is eligible for a backport we define a bug
fix as any set of patches that loosely fall into one of the following
categories.

- Addresses a logical defect in KubeVirt
- Fixes or improves the stability of our test suite
- Any infrastructure change related testing or releasing
- Logging and debug changes aimed at improving supportability of a stable release.

There is purposefully some ambiguity here. These are meant to be read as
guidelines to help reviewers make judgment calls. The intent here is to keep
our release branches as stable as possible and only backport PRs that meet that
goal.

It's possible the eligibility of a backport will not be clear cut, and require
some debate. Take for instance a PR that introduces new functionality to address
a defect. Would this be considered a bug fix? Maybe, maybe not. The reviewers
will have to weigh whether backporting will improve overall stability or
introduce unacceptable risk.

