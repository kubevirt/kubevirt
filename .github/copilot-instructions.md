## KubeVirt – AI Assistant Project Instructions

Purpose: Give an AI agent just enough KubeVirt context to be productive fast and avoid common pitfalls. Keep answers and code changes aligned with established build + packaging + controller patterns.

### 0. Quick Start for AI Assistants
- **Primary Language**: Go (with Bazel build system)
- **Key Commands**: `make generate`, `make bazel-build`, `make bazel-test`, `hack/dockerized`
- **Never** manually edit generated files (look for `// Code generated` headers)
- **Always** use `hack/dockerized` for reproducible builds
- **Security**: Follow established patterns, never introduce vulnerabilities

### 1. Big Picture
- KubeVirt adds VM primitives to Kubernetes via CRDs (core object: `VirtualMachineInstance` plus higher-level `VirtualMachine`, `VirtualMachineInstanceReplicaSet`, etc.).
- Control-plane components run as pods: `virt-api` (admission/defaulting/validation), `virt-controller` (cluster‑wide orchestration), `virt-handler` (node agent), per‑VMI `virt-launcher` pods (run libvirt + qemu inside container cgroups), optional sidecars/tools.
- Choreography pattern: controllers & handlers react to CR changes rather than a single central reconciler.
- Images are reproducibly built via Bazel + RPM dependency trees (qemu, libvirt, firmware) produced by `make rpm-deps` + `hack/rpm-deps.sh`.

### 2. Repository Layout (core areas you’ll touch)
- `cmd/virt-*` – Go entrypoints for each component (API, controller, handler, launcher, tools). Follow existing flag & logging patterns.
- `pkg/` – Shared libraries (API helpers, device/network/storage logic, validation, informers). Prefer reusing existing utilities vs. new abstractions.
- `api/` – Generated + handwritten API type metadata (CRDs). Changes usually require: adjust types -> run `make generate` -> commit updated generated files + OpenAPI.
- `rpm/` + `hack/rpm-deps.sh` – Definition of container rootfs package sets (qemu/libvirt/etc.). Touch only when altering runtime dependency versions or adding capabilities.
- `hack/` – Build, test, generation, cluster-up scripts. Always invoke via `hack/dockerized` for reproducibility unless intentionally doing a raw local experiment.
- `tests/` – Functional & E2E Ginkgo-based tests (avoid adding ad‑hoc test frameworks elsewhere).

### 3. Build & Gen Workflow (ALWAYS use these)
- Full image build: `make bazel-build-images` (calls Bazel through containerized environment).
- Push images (multi-arch manifest): `make bazel-push-images`.
- Update generated code after API/spec edits: `make generate` then `make generate-verify` (ensures nothing missing). Never hand-edit generated files.
- RPM dependency refresh (e.g., custom qemu/libvirt): `make rpm-deps QEMU_VERSION=17:... LIBVIRT_VERSION=... CUSTOM_REPO=custom.yaml` then rebuild images.
- Lint & basic verification: `make bazel-build-verify` or targeted `make lint`.
- Unit tests (Bazel): `make bazel-test`; pure Go (no Bazel) path: `make go-test WHAT=./pkg/some/subpkg`.

### 4. Conventions & Patterns
- Logging: Use existing structured logging helpers (follow component examples) – stay consistent with verbosity flags.
- Error handling: Prefer returning errors upward; controllers use event recording + status updates rather than panics.
- Feature gating: Check for existing feature flag or config CR fields before introducing new environment variables; add to config APIs where appropriate.
- API changes: Must be backward compatible—add new optional fields, avoid renames/removals. Document in `docs/` and update CRD schema.
- Image content: Do NOT apt/yum install inside Dockerfiles—rootfs is constructed from rpmtrees. Add packages by extending lists in `hack/rpm-deps.sh` & regenerating.
- Vendoring: Use `make deps-update` / `deps-update-patch`; don’t manually edit `go.mod` then forget Bazel sync.
- Multi-arch: Avoid architecture conditionals in Go unless absolutely required; check existing patterns (e.g., build tags) before adding new ones.
- **Code Style**: Follow Go standards + `.golangci.yml` linter config. Run `make lint` before submitting.
- **Security**: Never introduce security vulnerabilities. Follow security guidelines in `SECURITY.md`. Use `gosec` linter findings seriously.

### 5. Typical Change Flows (Examples)
- Add a new field to VMI spec:
  1. Edit type in `api/` (e.g., `api/core/v1/types.go`).
  2. Run `make generate` (regenerates deepcopy, OpenAPI, clients, manifests) + `make generate-verify`.
  3. Add validation/defaulting logic (likely under `pkg/virt-api/webhooks/` or related packages).
  4. Add functional test in `tests/` verifying field behavior.
- Introduce new runtime binary dependency:
  1. Add RPM name/version in appropriate group in `hack/rpm-deps.sh`.
  2. `make rpm-deps` then rebuild images.
  3. Reference binary by absolute path if needed (usually `/usr/bin/...`).

### 6. Testing Strategy
- Prefer adding/adjusting Ginkgo functional tests for behavioral changes (look at existing tests for style/labels). Keep unit tests small and focused on pure logic packages.
- Ensure reproducibility: if a change impacts image content or API surfaces, include generated file updates in same PR.

### 7. Performance & Footprint Awareness
- Avoid adding heavy libraries or broad dependency chains—every added RPM inflates multiple images. Reuse existing helper packages; refactor shared logic rather than duplicating.
- For hot paths (e.g., launch flows, monitoring loops), check similar code for established caching/timing patterns before introducing new ones.

### 8. Troubleshooting Build Issues
- Mismatch after editing APIs: run `make generate` again; verify no leftover uncommitted diffs with `git diff`.
- Bazel stale cache: run inside dockerized shell `bazel clean --expunge` (provided via scripts, not ad‑hoc local host environment).
- Missing binary in image: confirm RPM present in rpmtree (`rpm/BUILD.bazel` diff) before inspecting Docker layers.

### 9. Safe AI Change Checklist
Before proposing edits, mentally confirm:
- Is there an existing helper or pattern? (Search `pkg/`.)
- Does change require regenerating code or updating rpm-deps? Run required Make targets.
- Are tests or docs needed to accompany change? Provide minimal but complete coverage.

### 10. Contribution Guidelines
- **DCO Requirement**: All commits must be signed off with `git commit --signoff` for Developer Certificate of Origin compliance.
- **PR Guidelines**: Follow `CONTRIBUTING.md`. Consider opening draft PRs for work-in-progress changes.
- **Testing**: "Untested features do not exist" – always include appropriate unit and functional tests.
- **Review Process**: PRs require maintainer approval. New contributors need `/ok-to-test` comment for CI.

### 11. Provide Responses This Way
- Cite concrete files (`hack/rpm-deps.sh`, `Makefile`, `cmd/virt-controller/...`) rather than generic advice.
- Offer exact Make targets instead of raw `bazel`/`go build` commands.
- For multi-step edits, outline sequence (edit -> generate -> test) succinctly.

---
Questions / gaps? Ask whether more detail is needed on API generation, image build internals, or packaging flow.