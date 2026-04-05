# Feature Gate Report

Prints a JSON summary of all registered KubeVirt feature gates (excluding GA and Discontinued).

## Usage

```bash
make feature-gate-report
```

## Output

JSON array sorted by state then name:

```json
[
  {"name": "MyAlphaFeature", "state": "Alpha"},
  {"name": "MyBetaFeature", "state": "Beta"},
  {"name": "MyDeprecatedFeature", "state": "Deprecated"}
]
```
