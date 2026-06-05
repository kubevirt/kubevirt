# Commit message guidelines

KubeVirt follows the [Commit Message Guidelines](https://github.com/kubernetes/community/blob/main/contributors/guide/pull-requests.md#commit-message-guidelines) in the Kubernetes contributor guide. Use the links below for the full explanation of each rule.

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

## Examples

### Prefer

```
net, vmi: Add support for SR-IOV interfaces

This change introduces support for SR-IOV network interfaces in
KubeVirt, allowing VMs to directly access SR-IOV virtual functions
for improved network performance.

Signed-off-by: John Doe <jdoe@example.org>
```

### Avoid

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
