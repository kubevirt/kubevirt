# Pull request guidelines

This document describes how to shape contributions so others can review them
effectively. A change that is correct and useful still needs to be
**reviewable**: reviewers must be able to understand intent, scan the diff,
and reason about risk in a reasonable amount of time.

This document is not a substitute
to [`CONTRIBUTING.md`](../CONTRIBUTING.md) and the
[`Reviewer guide`](reviewer-guide.md). [`CONTRIBUTING.md`](../CONTRIBUTING.md) describes the contribution process, [`Reviewer guide`](reviewer-guide.md) highlights project-specific conventions and guidelines, and this document describes how best to structure your commit to minimize the time to merge.

## Structuring commits and pull requests

Follow [Smaller is better: Small commits, small pull
requests](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#smaller-is-better-small-commits-small-pull-requests).
Prefer a series of logical commits or smaller PRs over one large unreadable
change. Large diffs fatigue reviewers and stall more often than focused ones.

[Do not open pull requests that span the whole
repository](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#dont-open-pull-requests-that-span-the-whole-repository)
unless you deliberately split work by OWNERS boundaries and approvals as
upstream describes. The same idea applies across KubeVirt packages: unrelated
cleanup or sweeping edits deserve their own PRs.

## Comments in code

Upstream encourages commenting when behavior would otherwise be unclear. See
[Comments
matter](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#comments-matter).

KubeVirt reviewers also ask for balance: avoid **over-commenting** and long,
noisy explanations in the codebase. Comments that restate what the code
already expresses add maintenance cost. Sometimes a comment is genuinely needed to
explain a constraint, API quirk, or invariant that cannot be conveyed cleanly in
code alone; often, however, if you feel you must explain something at length, it
can signal that the structure or naming should change instead.

## Tests

Nothing in [Test](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#test)
is optional for substantive behavior changes: **new features need tests**.

When fixing a bug, **consider adding or extending tests** so the regression does
not return.

## Before you submit

- Run **`make fmt`** so Go and project formatting matches what CI expects.
- Run **`make generate`** after edits that touch generated artifacts (API
  machinery, mocks, protobuf, etc.), so checked-in generated files stay in sync
  with sources.

## Getting eyes on your change

KubeVirt receives many PRs, and legitimate work can slip down the queue. Beyond
comments and Slack, use **SIG meetings and KubeVirt community meetings** as an
opportunity to surface a stalled or high-importance PR. The public community
calendar can be found [here](https://calendar.google.com/calendar/u/0/embed?src=kubevirt@cncf.io).

## Squashing and fixup commits

> **TODO:** Clarify at community meeting. From experience, our policy on
>  [Squashing](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#squashing) seems to
> differ from Kubernetes.

## Commit message guidelines

KubeVirt follows the [Commit message
guidelines](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#commit-message-guidelines)
in the Kubernetes contributor guide. Use the links below for the full
explanation of each rule.

- [Try to keep the subject line to 50 characters or less; do not exceed 72 characters](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#try-to-keep-the-subject-line-to-50-characters-or-less-do-not-exceed-72-characters)
- [The first word in the commit message subject should be capitalized unless it starts with a lowercase symbol or other identifier](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#the-first-word-in-the-commit-message-subject-should-be-capitalized-unless-it-starts-with-a-lowercase-symbol-or-other-identifier)
- [Do not end the commit message subject with a period](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#do-not-end-the-commit-message-subject-with-a-period)
- [Use imperative mood in your commit message subject](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#use-imperative-mood-in-your-commit-message-subject)
- [Add a single blank line before the commit message body](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#add-a-single-blank-line-before-the-commit-message-body)
- [Wrap the commit message body at 72 characters](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#wrap-the-commit-message-body-at-72-characters)
- [Do not use GitHub keywords or (@)mentions within your commit message](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#do-not-use-github-keywords-or-mentions-within-your-commit-message)
- [Use the commit message body to explain the what and why of the commit](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#use-the-commit-message-body-to-explain-the-what-and-why-of-the-commit)

Optional subject prefixes (area) are described in [Providing additional context](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#providing-additional-context).

All commits need a `Signed-off-by` line for [DCO compliance](../CONTRIBUTING.md#contributor-compliance-with-developer-certificate-of-origin-dco).

### Examples

#### Prefer

```
net, vmi: Add support for SR-IOV interfaces

This change introduces support for SR-IOV network interfaces in
KubeVirt, allowing VMs to directly access SR-IOV virtual functions
for improved network performance.

Signed-off-by: John Doe <jdoe@example.org>
```

#### Avoid

```
Fixing network stuff

I updated the network code to add SR-IOV support. This fixes
the issue we were having.

Fixes #123
```

Issues with this example:

- Subject uses past tense ("Fixing") instead of [imperative mood](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#use-imperative-mood-in-your-commit-message-subject)
- Subject is too vague ("network stuff")
- Body does not clearly explain the what and why of the change
- Body uses GitHub keyword `Fixes #123`—put issue links in the [PR title or description](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#do-not-use-github-keywords-or-mentions-within-your-commit-message) instead
- Missing `Signed-off-by` line ([DCO](../CONTRIBUTING.md#contributor-compliance-with-developer-certificate-of-origin-dco))
