# Feature Gate Report

Prints a summary of all registered KubeVirt feature gates (excluding GA and Discontinued).

## Usage

```bash
make feature-gate-report
```

## Options

- `--output-format=json` (default) — JSON array sorted by state then name
- `--output-format=md` — Markdown table sorted by state then name
- `--output-file=PATH` — Write output to a file instead of stdout

## Output

### JSON (default)

```json
[
  {"name": "MyAlphaFeature", "state": "Alpha"},
  {"name": "MyBetaFeature", "state": "Beta"},
  {"name": "MyDeprecatedFeature", "state": "Deprecated"}
]
```

### Markdown

```markdown
# Feature Gate Report

| Feature Gate | State |
|---|---|
| MyAlphaFeature | Alpha |
| MyBetaFeature | Beta |
| MyDeprecatedFeature | Deprecated |
```
