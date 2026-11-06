# E2E (functional-test) code coverage

This describes the optional Go code-coverage collection for KubeVirt's
functional (integration) tests. It is **opt-in** and has **no effect on
production builds**.

Coverage here is not a merge-blocking metric — good line coverage does not mean
good integration tests. It is a *reference* signal: it exposes code paths that
no functional test ever exercises, so maintainers (or tooling) can decide which
gaps are worth covering. See VIRTCNV-434.

## Why it is non-trivial

`go build -cover` (Go 1.20+) can instrument a binary so it writes coverage
profiles to `$GOCOVERDIR` — but only when the process exits normally. KubeVirt
is different in two ways:

- It is **multi-binary**: virt-api, virt-controller, virt-handler,
  virt-operator, virt-launcher, etc., each in its own container.
- Its components are **long-running** (control-plane daemons never exit during a
  test) or **ephemeral** (a virt-launcher per VM, created and destroyed many
  times, often killed rather than exiting cleanly — so exit-time flushing loses
  data).

So the work is orchestration: collect coverage from every component before it
disappears, then merge it into one report.

## How it works

### 1. Build (`--build-cover`)

`hack/bazel-build-images.sh --build-cover` builds the component images with
coverage instrumentation:

- `--collect_code_coverage` with `--instrumentation_filter` scoped to
  `//cmd/...`, `//pkg/...` and client-go.
- `cover_format=go_cover` and the `coverage_e2e` build tag.

Covermode is **atomic** automatically: rules_go forces atomic mode whenever
coverage is enabled (`go/private/actions/compilepkg.bzl`), which is what the
`runtime/coverage` APIs — in particular `ClearCounters` — require.

### 2. Control-plane daemons (`pkg/coveragehttp`)

Long-running daemons cannot rely on process exit. Under the `coverage_e2e` tag,
each daemon's `main` calls `coveragehttp.Start(<component>)`, which serves on
`:6061`:

| Endpoint | Action |
| --- | --- |
| `POST /coverage/reset` | `runtime/coverage.ClearCounters()` (per-test reset) |
| `POST /coverage/flush` | write meta (first flush only) + counters, upload to the collector |

It also flushes on `SIGTERM`, so coverage survives daemons that are terminated
(for example on operator uninstall) rather than exiting. In non-coverage builds
`Start` is a no-op, so production images carry no coverage server.

Tests reach `:6061` through the Kubernetes pod proxy.

### 3. Collector (`kv-coverage-collector`)

`cmd/kv-coverage-collector` (package `pkg/coveragecollector`) receives the
uploaded `covmeta`/`covcounters` files and stores them per run-id as a covdata
directory:

| Endpoint | Action |
| --- | --- |
| `POST /coverage` | store one covmeta/covcounters file (labelled via `X-Coverage-*` headers) |
| `GET /coverage/report` | `go tool covdata percent` summary |
| `GET /coverage/raw` | merged covdata directory as a tar |
| `GET /healthz` | readiness |

Config via env: `LISTEN_ADDR` (default `:8080`), `COVERAGE_DIR`
(default `/coverage-data`), `GO_BINARY` (default `go`).

### 4. virt-launcher (planned)

virt-launcher pods are ephemeral and created dynamically, so a
MutatingAdmissionPolicy injects a `GOCOVERDIR` volume and an uploader sidecar
that ships the coverage files to the collector before the pod is deleted. (Not
yet implemented.)

## Usage (once fully wired)

```sh
hack/bazel-build-images.sh --build-cover
make push                 # DOCKER_PREFIX / DOCKER_TAG
make && make manifests
hack/functests.sh --cov-report
```

Coverage artifacts are written under `$ARTIFACTS/coverage/`
(`coverage.txt`, `coverage.html`, merged covdata).

## Status

- Done: `--build-cover`; `pkg/coveragehttp` wired into virt-api, virt-controller,
  virt-handler, virt-operator; `kv-coverage-collector`.
- Pending: collector/uploader image targets, virt-launcher sidecar + MAP,
  `--cov-report` suite orchestration, CI lane.

## Open questions

- The collector shells out to `go tool covdata`, so its image needs the Go
  toolchain. Alternative: make the collector store-only and merge/report in the
  test runner's `ReportAfterSuite` (which already has Go), keeping the collector
  image minimal.
