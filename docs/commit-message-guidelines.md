# Commit message guidelines

KubeVirt follows commit message guidelines based on the [Kubernetes Pull Request
Process guide](https://www.kubernetes.dev/docs/guide/pull-requests/#commit-message-guidelines).
For a general commit message convention, see [Conventional Commits](https://www.conventionalcommits.org/).

## Subject line

- **Length**: Try to keep the subject line to 50 characters or less; do not exceed 72 characters
- **Mood**: Use imperative mood (e.g., "Add", "Fix", "Update", "Remove", "Refactor")
- **Capitalization**: The first word should be capitalized unless it starts with a lowercase symbol or other identifier
- **Punctuation**: Do not end the subject line with a period

### Commit prefix conventions

Use a hybrid approach: start with the **intent** (type), followed by an **optional
scope** (the SIG or area it belongs to) in parentheses. Format: `type(scope): description`.

**Intent (required):**

- `docs` - documentation changes
- `fix` - bug fixes
- `feat` - new features
- `test` or `tests` - test changes
- `build` - build system changes
- `refactor` - code refactoring

**Scope (optional):** Add the SIG or area in parentheses, e.g. `network`, `storage`,
`virt-operator`, `backup`.

**Examples:**

- `fix(network): resolve SR-IOV interface binding issue`
- `feat(storage): add pull mode support`
- `docs: add commit message guidelines`

## Commit message body

- **Blank line**: Add a single blank line before the commit message body
- **Length**: Wrap the commit message body at 72 characters
- **Content**: Use the commit message body to explain the what and why of the commit
- **GitHub keywords**: Do not use GitHub keywords (like "Fixes #xxxx") or (@)mentions
  within your commit message. Place these in the **PR title or description**
  instead, where they will properly trigger GitHub's issue linking and automation.

## DCO compliance

All commits must include a `Signed-off-by` line for [DCO compliance](../CONTRIBUTING.md#contributor-compliance-with-developer-certificate-of-origin-dco).
See the [Contributing guide](../CONTRIBUTING.md#contributor-compliance-with-developer-certificate-of-origin-dco) for details.

## Examples

### Good example

```
feat(network): Add support for SR-IOV interfaces

This change introduces support for SR-IOV network interfaces in
KubeVirt, allowing VMs to directly access SR-IOV virtual functions
for improved network performance.

Signed-off-by: John Doe <jdoe@example.org>
```

### Bad example

```
Fixing network stuff

I updated the network code to add SR-IOV support. This fixes
the issue we were having.

Fixes #123
```

Issues with the bad example:

- Subject uses past tense ("Fixing") instead of imperative mood
- Subject is too vague ("network stuff")
- Body uses GitHub keywords ("Fixes #123") — use the PR title or description instead
- Body doesn't explain the what and why clearly
- Missing Signed-off-by line (required for DCO compliance)

## Additional resources

The [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) spec
provides a similar structure with type prefixes (feat, fix, docs, etc.) and can
enhance commit display in tools like Refined GitHub. While not required for
KubeVirt, you may find it helpful for categorization.
