# Release Notes

Pull requests in KubeVirt repositories may require a `release-note` block in
the PR description: a note for any user- or operator-visible change, or `NONE`
if the change doesn't need one.

Notes are extracted automatically at release
time and assembled into the changelog,
so a note should be in its final form by the time the PR is approved.

## Does my pull request need a release note?

Any user-visible or operator-visible change qualifies. This could be a:

- Bug fix that affects observable behavior
- New feature or enhancement
- API, CLI flag, configuration schema, default value, or feature gate change
- Behavioral or performance change
- Deprecation or removal
- Security fix (CVE)

No release note (`NONE`) is required for tests, CI/build infrastructure,
documentation, or bugs that were never present in a released version.

## Applying a release note

Fill in the `release-note` block in your pull request description:

````
```release-note
Fixed a bug where live migration failed on nodes with SR-IOV network interfaces.
```
````

> **_NOTE:_**
> Keep the note to a single line. The release tooling extracts only the first
> line after the opening fence.

If the block is missing, Prow adds the `do-not-merge/release-note-label-needed`
label and blocks the merge until a note is provided.

Release notes are categorized in the changelog using GitHub labels: add the
`/kind` and `/sig` commands (for example `/kind enhancement`, `/sig compute`)
in the PR description, in a subsequent comment, or by using the GitHub `Labels`
filter. See the [release procedure](release-procedure.md)
for more on these labels.

### Breaking changes

For changes that require operator action before or after upgrading, start the
note with **"action required"**. Prow then applies the
`release-note-action-required` label:

````
```release-note
action required: The `featureGates.LiveMigration` field has been removed. Enable live migration via `spec.configuration.migrations` in the KubeVirt CR instead.
```
````

> **_NOTE:_**
> Keep the note to a single line. The release tooling extracts only the first
> line; any text after a line break inside the block is silently ignored.

### Changes without a release note

Write `NONE` inside the block, or comment `/release-note-none` on the pull
request. Either way, Prow applies the `release-note-none` label and the PR is
skipped when the changelog is generated.

### Backports

Pull requests against release branches need their own release note: a brief,
one-line statement of what the backport addresses, incorporated into the
changelog when the next version is cut from that branch. See the
[backporting policy](release-branch-backporting.md)
for the full rules.

## Writing good release notes

- Write in the past tense: "Fixed" instead of "Fix", "Added" instead of "Add".
- Write for users and operators, not developers — skip internal implementation
  details.
- Be specific: name the affected feature, workload type, API field, or
  configuration option.
- For action-required notes, include a call to action and link to
  documentation explaining what the operator must do.
- Don't include the PR number or your username — the tooling prepends
  `[PR #N][author]` automatically.

A release note should make sense to someone reading the changelog without
opening the PR. See the release notes of
[recent releases](https://github.com/kubevirt/kubevirt/releases) for reference.
