# Integration tests

Organize integration tests by the SIG responsible for the behavior being tested.
Each suite has its own Go package, Ginkgo suite entry point, and Bazel test
target. SIG directories can contain area-specific suites with their own OWNERS.

| Directory | Coverage |
| --- | --- |
| `compute/` | VM and VMI lifecycle, controllers, and compute admission |
| `compute/instancetype/` | Instancetype and preference application, revisions, and schema validation |
| `storage/` | Volumes, storage configuration, and storage admission |
| `network/` | Network configuration and network admission |
| `control-plane/` | Operator behavior, cluster configuration, and admission policies |
| `framework/` | Shared integration framework and tests of the framework itself |

Place tests with their owning SIG, including schema and admission tests. Shared
setup belongs in `framework/`; keep test scenarios in the SIG directories.
Create a suite entry point and Bazel target when adding the first tests to a SIG
directory.

Run all integration suites with `make test-integration`. To run one SIG's suite
in the builder environment, use, for example:

```sh
hack/dockerized "go test -race ./tests/integration/compute -count=1 -v"
```
