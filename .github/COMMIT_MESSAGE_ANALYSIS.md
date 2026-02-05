# Commit Message Analysis Configuration

This repository uses Source AI to analyze commit messages in pull requests for format compliance and alignment with code changes.

## Configuration File

- **`.github/sourceai.yaml`** - Source AI configuration file that defines:
  - Format patterns and their expected distribution
  - Common characteristics to check
  - Alignment validation settings
  - Feedback configuration

Source AI will automatically read this configuration file when analyzing pull requests.

## Format Patterns

The analyzer checks commit messages against these patterns (in order of preference):

1. **Component prefix (~80%)**: `component: description` or `component1, component2: description`
   - Example: `network: Add support for SR-IOV interfaces`

2. **Capitalized component prefix (~10%)**: `Component: Description`
   - Example: `Network: Add support for SR-IOV interfaces`

3. **Conventional commits (~5%)**: `type(scope): description` or `type: description`
   - Example: `feat(network): Add support for SR-IOV interfaces`

4. **No prefix (~5%)**: Direct description starting with capital letter
   - Example: `Add support for SR-IOV interfaces`

## Common Characteristics Checked

- ✅ Imperative mood (Add, Fix, Refactor, Remove, etc.)
- ✅ First word capitalized
- ✅ First line under 72 characters (recommended: ≤50)
- ✅ Descriptive and clear (avoids generic words)
- ✅ Alignment with code changes

## Usage

Source AI will automatically read `.github/sourceai.yaml` when it runs code reviews on PRs. Since Source AI is already configured and running code reviews automatically, it will pick up this configuration file and perform commit message analysis in addition to its normal code review.

**Testing:** To verify Source AI is using this configuration, create a test PR and check if Source AI's review includes commit message analysis feedback.

## Feedback Format

Source AI will post feedback as a PR comment with:
- Summary statistics
- List of commits with format issues
- List of commits with alignment issues
- Recommendations and examples

The feedback will be automatically included in Source AI's regular PR review comments.
