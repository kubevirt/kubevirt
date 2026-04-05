# Feature Gate Report

Prints a JSON summary of all registered KubeVirt feature gates (excluding GA and Discontinued).

## Usage

```bash
hack/dockerized "bazel build --config=x86_64 //tools/feature-gate-report && \
  ./bazel-bin/tools/feature-gate-report/feature-gate-report_/feature-gate-report"
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
