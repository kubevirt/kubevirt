# Integration tests

Organize integration tests by the SIG responsible for the behavior being tested.
Each SIG directory with tests has its own Go package, Ginkgo suite entry point,
and Bazel test target.

| Directory | Coverage |
| --- | --- |
| `compute/` | VM and VMI lifecycle, controllers, and compute admission |
| `storage/` | Volumes, storage configuration, and storage admission |
| `network/` | Network configuration and network admission |
| `control-plane/` | Operator behavior, cluster configuration, and admission policies |
| `framework/` | Shared integration framework and tests of the framework itself |

Place tests with their owning SIG, including schema and admission tests. Shared
setup belongs in `framework/`; keep test scenarios in the SIG directories.
Create a suite entry point and Bazel target when adding the first tests to a SIG
directory.

The framework configures controller-runtime logging once when first constructed,
so envtest diagnostics reach Ginkgo output without logger setup in each suite.

Run all integration suites with `make test-integration`. To run one SIG's suite
in the builder environment, use, for example:

```sh
hack/dockerized "go test -race ./tests/integration/compute -count=1 -v"
```

Each framework allocates an initial namespace name in `New()`, available through
`f.Namespace()` before startup. `Start()` creates it and registers cleanup for
the same scope as the framework. Use `CreateNamespace(ctx)` for additional
namespaces or when individual specs need their own cleanup scope.

To seed a persisted KubeVirt CR, pass `framework.WithKubeVirtCR(kv)` to `New()`.
The option copies the object and defaults an empty namespace to `f.Namespace()`;
`Start()` creates the CR and any explicit namespace it needs. This option does
not configure the in-process controllers: `WithKubeVirtConfig` continues to
configure those separately.

```go
f := framework.New(framework.WithKubeVirtCR(&v1.KubeVirt{
    ObjectMeta: metav1.ObjectMeta{Name: "kubevirt"},
    Spec: v1.KubeVirtSpec{
        Configuration: v1.KubeVirtConfiguration{
            PermittedHostDevices: permittedHostDevices,
        },
    },
}))
f.Start(context.Background())
// The KubeVirt CR is now available in f.Namespace().
```
