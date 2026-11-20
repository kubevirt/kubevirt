# Writing integration tests

KubeVirt integration tests exercise controller interactions and API behavior
against a real Kubernetes API server and etcd, without deploying a cluster or
booting a guest. The shared framework uses controller-runtime's envtest to start
those processes. The tests live under `tests/integration/`; envtest is the
underlying mechanism, rather than the name of this test layer.

This guide covers the framework available in this checkout.

## Choose the test layer

Use the smallest test environment that includes the behavior you need to verify.
A feature can need coverage at more than one layer.

| Layer | Use it to verify | Examples |
| --- | --- | --- |
| Unit | A function, reconciliation decision, or component with controlled dependencies | Domain XML conversion, validation helpers, error handling, individual instancetype appliers |
| Integration | Interactions with a real API server, persisted objects, watches, and the controllers started by the framework | VM/VMI generation tracking, ControllerRevision ownership, controller-created pods, CRD schema and CEL rules |
| Functional / E2E | Behavior requiring a deployed KubeVirt installation or working node and guest components | Guest boot and console, real scheduling, live migration, storage/network connectivity, operator installation, pod-bound service-account tokens |

For example, checking that a VM spec change increments its generation and is
observed by a controller fits an integration test. Checking that a guest sees a
hotplugged device needs E2E coverage. Checking the generated libvirt XML can
usually remain a unit test.

Before moving a test, check that the relevant implementation actually runs in
the framework. A real API server does not make every KubeVirt component real.

## What the framework runs

[`framework.go`](../../tests/integration/framework/framework.go) starts:

- An isolated API server and etcd, with KubeVirt CRDs built from the repository's
  production CRD generators. CRD validation, status subresources, resource
  versions, generation changes, and watches use the real API server.
- The VM and VMI controllers with informer caches synchronized before their
  workers start.
- A pod simulator that binds virt-launcher pods to a seeded node and gives them
  running/ready status, including synthetic pod IPs and container statuses.
- Seed namespaces and a ready node named `functional-node` with allocatable
  CPU, memory, and pod capacity.

The framework always starts its own control plane, including when
`USE_EXISTING_CLUSTER` is inherited from the caller. Use `f.VirtClient()` for
KubeVirt resources and `f.K8sClient()` for Kubernetes resources; they point to
this control plane. Avoid the functional suite's global cluster client.

The environment has these boundaries:

- There is no kubelet, scheduler, virt-handler, virt-launcher process, libvirt,
  QEMU, or guest. A pod marked `Running` is simulated. The VMI lifecycle example
  reaches `Scheduled`; waiting for a working guest or `Running` VMI needs
  additional real components.
- The framework does not deploy virt-operator, virt-api's aggregated API server,
  CDI controllers, or the Kubernetes controller manager. Creating a resource
  does not start its controller. Migration informers and dummy CDI informers do
  not supply migration or import execution.
- VM synchronization, instancetype handling, topology, network/storage status,
  and other dependencies include stubs. Inspect
  [`stubs.go`](../../tests/integration/framework/stubs.go),
  `buildVMController`, and `buildVMIController` before relying on them. Controller
  events use a fake recorder that discards events.
- The framework does not install KubeVirt admission webhooks or
  ValidatingAdmissionPolicies automatically. CRD schema and embedded CEL rules
  still run. Webhooks and policies require explicit setup.
- Namespace deletion and owner-reference garbage collection have no controller
  processing them. Framework cleanup removes resources and finalizers explicitly;
  it does not demonstrate production garbage-collection behavior.

If a test needs a missing component, extend the shared framework with a specific
dependency or keep the scenario at its existing test layer. Do not simulate the
behavior that the test is meant to verify.

## Run the tests

From the repository root:

```sh
make test-integration
```

This runs `go test -race ./tests/integration/... -count=1 -v` inside the repository
builder. It does not require `make cluster-up`, deployed images, nested
virtualization, or access to `/dev/kvm`. The container runtime and rsync setup
described in [Getting Started](../getting-started.md#building) is still required.
`make integ-test` is a separate network integration-test target.

For a single SIG suite or scenario:

```sh
hack/dockerized "go test -race ./tests/integration/compute -count=1 -v"
hack/dockerized "go test -race ./tests/integration/compute -count=1 -v -ginkgo.focus='VM Lifecycle'"
```

The builder includes the API server and etcd binaries and sets
`KUBEBUILDER_ASSETS`. To run directly on a compatible Linux host, use the Go
toolchain required by this checkout and the envtest binary version pinned in
[`hack/builder/Dockerfile`](../../hack/builder/Dockerfile):

```sh
KUBEBUILDER_ASSETS=/path/to/compatible/envtest/bin \
  go test -race ./tests/integration/compute -count=1 -v
```

Prefer the builder when reproducing CI. Check the
[`test-integration` target](../../Makefile) for the command used by this checkout.

## Place and register a new test

Use the owning SIG's directory as described in the
[integration README](../../tests/integration/README.md). Shared setup belongs in
`framework/`; scenarios belong in their SIG's package.

For an existing suite, add a descriptive `*_test.go` file using the existing
package, such as `compute_test`. Include the repository's
[Go license header](../../hack/boilerplate/boilerplate.go.txt). Update its
`BUILD.bazel` sources and dependencies as well as adding the Go file; successful
`go test` execution alone does not check Bazel registration.

For a SIG directory receiving its first tests, add a Ginkgo suite entry point
and a `go_test` target, following
[`compute_suite_test.go`](../../tests/integration/compute/compute_suite_test.go)
and [`compute/BUILD.bazel`](../../tests/integration/compute/BUILD.bazel). Use that
directory's SIG ownership. The recursive integration target will discover the
new Go package. The framework configures logging once, so suite entry points do
not need to set up controller-runtime logging again.

## Write a controller integration test

This example can be added to the compute package. Include the license header
when creating the file.

```go
package compute_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

var _ = Describe("VM startup through controllers", func() {
	var f *framework.Framework

	BeforeEach(func() {
		f = framework.New()
		f.Start(context.Background())
	})

	It("schedules a VMI for an always-running VM", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)
		vm, err := f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(ctx, matcher.ThisVMIWith(ns, vm.Name), 30*time.Second, 100*time.Millisecond).
			Should(matcher.BeInPhase(virtv1.Scheduled))
	}, SpecTimeout(time.Minute))
})
```

Construct and start the framework inside a setup node, where Ginkgo can register
cleanup. `Start` registers `Stop` with `DeferCleanup`, including for partial
startup failures. `CreateNamespace` registers cleanup of that namespace and its
resources. Avoid adding another routine call to `Stop`.

Use a context whose lifetime covers the framework's use for `Start`. The example
uses `context.Background()` because the framework cancels its own child context
during cleanup. A `BeforeEach` or `BeforeAll` `SpecContext` ends with that setup
node; using it as the framework lifetime would stop controllers before the test
body runs. Use each spec's `SpecContext` for client calls and bounded polling.

### Isolate fixtures and share setup deliberately

Start with a fresh framework in `BeforeEach` when specs need different controller
configuration or cluster-scoped state. For several specs using identical setup,
an `Ordered, ContinueOnFailure` container can start one framework in `BeforeAll`
and create a fresh namespace in each `It`. See the
[generation tracking tests](../../tests/integration/compute/vm_generation_test.go).

Namespace isolation does not reset cluster-scoped resources, configuration, or
controller state. Register explicit cleanup for cluster-scoped fixtures, and
keep specs independent even when they share a framework. Avoid concurrently
starting multiple frameworks in one process: shared matcher helpers use a
process-wide client.

### Configure the framework before startup

`framework.New(opts ...framework.Option)` applies options before startup.
`WithKubeVirtConfig` copies the supplied configuration into the configuration
used by both controllers. A nil argument leaves the current configuration
unchanged. For example, replace the example's setup with:

```go
BeforeEach(func() {
	strategy := virtv1.VMRolloutStrategyStage
	f = framework.New(framework.WithKubeVirtConfig(&virtv1.KubeVirtConfiguration{
		VMRolloutStrategy: &strategy,
	}))
	f.Start(context.Background())
})
```

Use this option for feature gates and controller configuration rather than
updating a live cluster. The controllers use the framework's test configuration;
creating or patching a KubeVirt CR is not a substitute for this option. Separate
frameworks can exercise different configurations without modifying shared
production-style configuration through functional test helpers.

### Assert persisted behavior

Controllers reconcile asynchronously. Poll a fresh GET or a client-backed
matcher, rather than repeatedly checking the object returned by `Create` or
`Patch`. Keep object names stable and check errors inside the polling function.

Use bounded `Eventually` to wait for a positive signal that reconciliation has
processed the change. For a negative assertion, follow that signal with bounded
`Consistently` checks. For example, wait for `DesiredGeneration` to catch up
before asserting that a non-live-updatable change leaves `ObservedGeneration`
unchanged. An immediate assertion could pass before reconciliation starts.

Use field-specific JSON patches for spec changes, following `patchVM` in the
generation tests. Preserve unrelated controller-owned fields. Verify object
UIDs, owner references, and revision data when a test claims an object was
preserved or reused. Use table entries for independent rejection cases so one
failure does not hide the remaining cases.

## Adapt an E2E test

Identify the behavior each existing assertion proves before changing its setup.
Keep node, guest, deployment, and real-identity checks in the functional suite
when an API-only scenario cannot replace them.

| Existing E2E pattern | Integration adaptation |
| --- | --- |
| Shared cluster client and namespace | Use the framework's clients and `CreateNamespace(ctx)` |
| Update global KubeVirt configuration | Pass `WithKubeVirtConfig` before starting the framework |
| `virtctl start` or VM start subresource | Patch `spec.runStrategy` when the behavior being tested is controller startup; this does not test the aggregated start API |
| Wait for guest login or a running VMI | Assert the relevant API/controller effect, such as a created VMI or `Scheduled` phase, if that is the original test's purpose |
| Wait for a PVC import or actual pod scheduling | Retain E2E coverage unless the required controller or scheduler is explicitly added |
| Delete an owner and wait for dependents to disappear | Retain garbage-collection coverage in E2E; framework teardown removes resources explicitly |
| Create an object expected to fail admission | Determine whether the responsible validator is the CRD, webhook, or policy; exercise that validator and assert its specific error |
| Use a real handler token to patch a node | Impersonation can exercise policy logic, but retain E2E coverage for pod-bound token identity and deployed policy installation |

For required-field validation, typed Go serialization may still emit a zero-valued
field. Marshal the object, remove the precise nested field from decoded JSON,
then submit the raw JSON. For CEL cases, choose an input that reaches the intended
rule. Check the API error category and the field or rule's error text; a generic
failure can come from another layer.

Port one scenario at a time, verify it against the real component under test,
and document which remaining assertions still need E2E coverage. When removing
an E2E test, include that coverage comparison in the PR. Add reusable setup to
the shared framework instead of duplicating controller wiring in each suite.

## Troubleshoot failures

| Symptom | Check |
| --- | --- |
| API server or etcd executable is missing | Use the repository builder, or point `KUBEBUILDER_ASSETS` at compatible envtest binaries |
| Control-plane startup fails | Read Ginkgo output for envtest diagnostics; check binary compatibility, process permissions, and free ports |
| A VMI never reaches `Running` | Check whether the expectation requires virt-handler or a guest; the simulator only supplies pod scheduling/readiness |
| A watch-dependent assertion times out | Confirm the relevant controller and informer run in this framework, keep the framework context alive, and poll fresh API results |
| An invalid object is accepted | Identify its validator; CRDs are installed, while webhooks and policies require explicit setup |
| An expected deletion never finishes | Check finalizers and absent garbage-collection or namespace controllers; do not use teardown as evidence of production deletion behavior |
| Tests pass with `go test` but are missing from Bazel | Register the source file and its imports in the suite's `BUILD.bazel` |
| Build tooling reports unrelated worktree files | Follow the [worktree build guidance](../getting-started.md#git-worktrees-and-build-tooling) and keep auxiliary checkouts outside the repository |

Run the relevant suite with race detection and `-count=1` before sending the PR.
Review changes for focused or skipped specs, unbounded waits, fixture leaks,
missing build registration, and assertions that depend on simulated behavior.
Check both positive behavior and the intended rejection path when adding an
admission scenario.
