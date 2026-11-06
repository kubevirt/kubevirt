# Plan: a scenario-shaped API for the virt-handler VMI fixture

Scratch design doc for the `vm-test-fixture` branch. Not meant to be committed with the code;
delete it once the series lands.

Two governing rules:

1. **The fixture assumes nothing.** Every piece of state that can change which branch of
   `execute()` runs is stated by the spec. Nothing is defaulted on the spec's behalf, including the
   things defaulted today.
2. **One entry point, options per part.** The fixture stops being a bag of knobs. A spec states its
   world once, through `f.given(...)`, and every knob becomes a free option function belonging to
   one part of the scenario — not a method on the fixture.

And one constraint on how we get there: **the conversion happens one spec at a time**, with the old
and new vocabularies coexisting in the same file and the same fixture until the last spec moves
(§5).

## 1. The problem

Take the spec the review flagged (`vm_test.go:105`):

```go
It("should delete non-running Domains if no cluster wide equivalent and no grace period info exists", func() {
    domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
    f.seedDomain(domain)

    f.HotplugMounter.EXPECT().UnmountAll(gomock.Any(), f.CgroupManager).Return(nil)
    f.LauncherClient.EXPECT().DeleteDomain(libvmi.New(libvmi.WithName("testvmi"), libvmi.WithUID(vmiTestUUID), libvmi.WithNamespace(metav1.NamespaceDefault)))
    f.executeAllowingDomainRemoval()
    testutils.ExpectEvent(f.Recorder, VMISignalDeletion)
})
```

One line of setup is explicit. Everything else that decides the outcome is inherited silently from
`newHandlerEnv`:

| Fact | Where it is actually decided | Visible in the spec? |
| --- | --- | --- |
| No VMI in the node informer | absence of a `vmiStore.Add` | no |
| No VMI in the global informer | `newFixture` passes **the same** store for both (`fixture_test.go:113-114`) | no, and not expressible |
| No VMI in the API | absence of a `VirtFake` create | no |
| The reconciled VMI is synthesised from the key plus the domain's cached UID (`controller.go:226`, `vm.go:348-363`) | restated by hand in the `DeleteDomain` matcher | only as a magic literal |
| A virt-launcher pod exists on this node, socket present | `environment_test.go:294-315`, unconditional | no |
| Its launcher client is reachable, `Initialized: true` | same | no |
| A mount root with `dev/kvm` exists and `Detect` succeeds for any VMI | `environment_test.go:218-231`, `:283-284` | no |
| No ghost record exists, although a pod virt-handler has talked to would have one | `takeOverGlobals` initialises an empty store | no |

Counting the current call sites, this is not confined to the `seedDomain` specs:

- **57** scenario constructions, of which **37** never state the domain axis — 31 `f.runningVMI()`
  calls inherit "and a Running domain" from the helper (`seed_test.go:161-164`, the default added in
  `f619a1cb61`), 18 `f.scheduledVMI()` calls inherit "and no domain" from its absence.
- **0** state where the VMI object lives. All 57 get node informer + global informer + API.
- **0** state the pod. All 57 get the one healthy pod the env builds whether the spec cares or not.

The second problem is the shape of the escape hatches. When a spec steps off the inherited path it
either calls a one-off fixture method (`f.setLauncherState`, `f.withoutLauncherClient`,
`f.newPodWithoutSocket`) or reaches around the fixture entirely:
`cmdclient.MarkSocketUnresponsive(sockFile)` (`vm_test.go:933`),
`virtcache.GhostRecordGlobalStore.Add(...)` (`:2046`), the hand-rolled store+API+queue update at
`:2086-2094`, `f.Controller.vmiExpectations.SetExpectations(key, 0, 0)` (`:906`). The fixture has
accumulated **24 methods** and five constructor options, two of which (`useClusterConfig`, `setenv`)
have no callers at all. Each new case adds a knob rather than a value in an existing vocabulary.

## 2. What must be stated, and what may be derived

A distinction the plan leans on, so "assume nothing" stays usable.

**State — stated by the spec.** Anything that selects a branch in `execute()`/`sync()`: VMI
existence, placement and phase; what exists on the host for the launcher pod; domain existence and
state.

**Identity — derived by the fixture.** Values that must merely be *consistent* and whose particular
value changes no behaviour: the VMI's UID and resourceVersion, the pod's UID, the domain's
name/namespace and `Spec.Metadata.KubeVirt.UID`, the `NodeNameLabel` / `Status.NodeName` /
`ActivePods` wiring. Deriving these and asserting they agree is the opposite of an assumption — it
is why `assertCoherent` exists, and that part is kept.

Two fixture-wide values stay defaulted and are named here rather than buried: the host name
(`testHost`) and the default `KubeVirtConfiguration`. Both are per-fixture, not per-scenario, both
are already overridable at construction, and every scenario is asserted against the host.

## 3. The API

### 3.1 Entry point

```go
s := f.given(
    withVMI(v1.Running, storedIn(nodeCache, globalCache, api), spec(libvmi.WithTerminationGracePeriod(1))),
    withPod(socketReady(), clientResponsive()),
    withDomain(api.Running, api.ReasonUnknown),
)
```

`f.given` takes `scenarioOption`s, validates, realises and returns the world. Exactly three parts
must be present — VMI, pod, domain — each named once. A missing part fails with a message that says
which option to add and which production branch depends on it; a duplicated part fails too, so a
`DescribeTable` cannot quietly override half a scenario. `given` may be called only once per spec;
later changes go through `then` (§3.3).

`given` reads better than `withScenario` at a call site that is already inside an `It`, and leaves
`with…` to mean "an option" throughout. `f.withScenario(...)` works just as well if you prefer the
explicit noun; it is the same function.

### 3.2 Three parts, three option families

```go
type scenarioOption func(*scenario)   // the three parts, plus incoherent()
type vmiOption      func(*vmiPart)
type podOption      func(*podPart)
type domainOption   func(*api.Domain) // unchanged from today
```

Separate option types mean the compiler keeps the groups apart: a pod option cannot be passed to the
VMI part, and the reader sees three clusters rather than one flat list.

**The parts.**

```go
func withVMI(phase v1.VirtualMachineInstancePhase, opts ...vmiOption) scenarioOption
func noVMI() scenarioOption

func withPod(opts ...podOption) scenarioOption
func noPod() scenarioOption

func withDomain(state api.LifeCycle, reason api.StateChangeReason, opts ...domainOption) scenarioOption
func noDomain() scenarioOption

// incoherent disables the coherence assertions for a scenario whose whole point
// is state the controller should refuse to act on. The reason is printed on failure.
func incoherent(because string) scenarioOption
```

Phase and domain state are positional rather than options because neither has a usable
"not stated" zero value: `v1.VmPhaseUnset` is `""` (`types.go:1112`) and the zero `api.LifeCycle` is
`NoState`, both legitimate. Making them arguments is the only enforcement the compiler can give; the
remaining axes are caught at runtime by `given`.

**VMI options.**

```go
// storedIn lists the places that hold the VMI object. The node informer is what
// getVMIFromCache reads; the global informer is the fallback execute() consults
// when the node informer has not caught up (vm.go:321-334); the API is what the
// controller writes status to (vm.go:1032).
func storedIn(places ...vmiStore) vmiOption   // nodeCache | globalCache | api

func spec(opts ...libvmi.Option) vmiOption    // libvmi and libvmistatus options pass through
func mutate(fn func(*v1.VirtualMachineInstance)) vmiOption  // escape hatch; 25 current callers
```

`storedIn` is a list rather than an enum because the three places are three independent booleans and
all the interesting combinations are real: `storedIn(globalCache, api)` is an informer that has not
caught up, `storedIn(nodeCache, globalCache)` is a VMI already deleted from the API.

This needs a real second store: `handlerEnv` grows `VMIGlobalInformer` and `newFixture` passes
`env.VMIGlobalInformer.GetStore()` as the `vmiGlobalStore` argument instead of reusing
`env.VMIInformer.GetStore()`. Until that lands, `storedIn(globalCache, api)` is a lie and the
`vm.go:321-334` fallback has no test and cannot have one.

**Pod options.** The env stops building a pod: `buildLauncherClient` and the pod half of
`buildFilesystem` move into pod realisation inside `given`.

```go
// what is on the host
func socketReady() podOption         // socket file created under PodsDir
func socketPending() podOption       // socket directory only: launcher has not come up
func socketUnresponsive() podOption  // socket file + cmdclient.MarkSocketUnresponsive
func noSocket() podOption            // nothing on disk at all

// what the launcher client manager reports
func clientResponsive() podOption
func clientUninitialized() podOption
func clientUnresponsive() podOption
func clientAbsent() podOption              // GetVerifiedLauncherClient fails with "Unknown client"
func clientProbeFails(err error) podOption // IsLauncherClientUnresponsive returns err

// overrides for the two derived values, see below
func mountRootReadable() podOption
func mountRootMissing() podOption
func ghostRecord() podOption
func noGhostRecord() podOption
```

Exactly one socket option and exactly one client option are required. They are separate axes on
purpose: `MockLauncherClientManager` answers from flags and never looks at the socket
(`launcher-clients/mock_launcher-clients.go:38-62`), and the existing comment at
`fixture_test.go:234-238` already records that a launcher can be reported responsive and still have
no client to talk to. Collapsing them into one `podRunning`-style enum would make that spec
inexpressible.

The mount root and the ghost record are **derived from the socket** unless overridden: a pod with a
socket gets a readable mount root (the kubelet made the directory) and a ghost record (whoever got a
launcher client wrote one); `noSocket()` gets neither. This is the one place the "state everything"
rule is relaxed, because stating four pod facts at 57 call sites buys very little — but both
derivations have an explicit override, so `noSocket(), ghostRecord()` is how the leaked-record case
(`vm_test.go:2103`) is written, and the override is visible where it matters. Tighten this to four
required options if the relaxation turns out to hide a branch.

**Domain options.** Unchanged type and semantics; `withDomainState` disappears into the positional
arguments, `withGracePeriod` stays.

### 3.3 Transitions use the same vocabulary

Multi-step specs currently open-code the second step. They get the same options through a second
verb, which applies a delta and requires at least one part but not all three:

```go
f.then(withPod(noSocket(), clientAbsent()))   // the pod went away mid-spec
f.then(noDomain())                            // the launcher undefined the domain
f.then(withVMI(v1.Failed, storedIn(nodeCache, globalCache, api)))
```

One transition is not expressible as a delta, because it is about the caches re-reading what the
controller wrote, and that is also what satisfies `vmiExpectations` in production (`vm.go:1031`):

```go
func (f *fixture) informersCatchUp()
```

It replaces both the hand-rolled store+API+enqueue block at `vm_test.go:2086-2094` and the
`vmiExpectations.SetExpectations(key, 0, 0)` poke at `:906`.

### 3.4 What the scenario hands back

```go
s.VMI     // the realised VMI, for gomock matchers
s.Pod     // UID, SockFile, MountRoot
s.Domain  // nil when noDomain()

// reconstructedVMI is the VMI value execute() builds for a VMI it cannot find:
// name and namespace from the key (controller.go:226), UID recovered from the
// domain cache (vm.go:348-353) or the ghost record (vm.go:357-363).
func (s *scenario) reconstructedVMI() *v1.VirtualMachineInstance
```

### 3.5 The exemplar, after

```go
It("should delete non-running Domains when the VMI is gone and the launcher is still up", func() {
    s := f.given(
        noVMI(),
        withPod(socketReady(), clientResponsive()),
        withDomain(api.Shutoff, ""),
    )

    f.HotplugMounter.EXPECT().UnmountAll(gomock.Any(), f.CgroupManager).Return(nil)
    f.LauncherClient.EXPECT().DeleteDomain(s.reconstructedVMI())

    f.executeAllowingDomainRemoval()
    f.expectEvent(VMISignalDeletion)
})
```

the sibling that cannot be written today:

```go
It("should clean up locally when both the VMI and its launcher pod are gone", func() {
    f.given(
        noVMI(),
        withPod(noSocket(), clientAbsent(), ghostRecord()),  // record leaked behind the pod
        withDomain(api.Shutoff, ""),
    )
    ...
})
```

and a typical converted spec:

```go
s := f.given(
    withVMI(v1.Scheduled, storedIn(nodeCache, globalCache, api)),
    withPod(socketPending(), clientUninitialized()),
    noDomain(),
)
```

## 4. The resulting surface

| | today | after |
| --- | --- | --- |
| fixture methods | 24 | 8 |
| scenario-shaping free functions | 3 (`domainOption`s) | ~20, in three typed families |

The eight that remain: `given`, `then`, `informersCatchUp`, `execute`,
`executeAllowingDomainRemoval`, `expectEvent`, `expectNoEvent`, `updateDomain`. Construction options
(`withHostCPUModel`, `withNetConf`, `withClusterConfig`, `withMockContainerDiskMounter`,
`withEnvOptions`) stay on `newFixture` — they configure the controller, not the world.

Deleted outright: `seedDomain`, `seedLegacyVMI`, `removeVMI`, `enqueue`, `setLauncherState`,
`withoutLauncherClient`, `useClusterConfig` (no callers), `setenv` (no callers), `defaultPod`,
`newPod`, `newPodWithoutSocket`, `newAbsentPod`, `vmiInPhase`, `scheduledVMI`, `runningVMI`, `seed`,
`seedRunningVMI`, `seedScheduledVMI`, `seedIncoherent`, and the scenario methods `withDomain`,
`withoutDomain`, `onPod`, `mutate` in their current method form.

## 5. Migrating one spec at a time

### 5.1 Why not a second fixture type

A `fixtureV2` living beside `fixture` is the obvious shape, and it does not work as long as both can
be alive in the same spec. `newHandlerEnv` takes over four pieces of **process-global** state
(`environment_test.go:186-214`): `cmdclient.SetPodsBaseDir`, `diskutils.DefaultOwnershipManager`,
the `getCgroupManager` function variable, and the ghost record cache singleton. The top-level
`BeforeEach` builds a fixture for *every* spec in the Describe (`vm_test.go:85-90`), so a converted
spec that built its own v2 fixture would have two envs alive, the second silently stealing the pods
base dir and the cgroup manager from the first. The failure mode is a socket path computed against a
directory that is no longer current — exactly the kind of thing that produces a flake nobody can
reproduce.

So: **one fixture type, two vocabularies, for the duration.** The new API is additive; the old
helpers keep working until their last caller is gone.

(The one variant that *would* work is moving converted specs into a separate `Describe` with its own
`BeforeEach` — one fixture per spec again. It is listed in §7 and rejected for churn reasons, not
safety.)

### 5.2 The shape during the transition

- **`scenario_test.go`** (new) holds `given`, `then`, the three option families and the realiser.
- **`seed_test.go`** keeps the old helpers, all prefixed `legacy` (`legacyRunningVMI`,
  `legacyScheduledVMI`, `legacySeedDomain`, …). The branch already set this precedent with
  `seedLegacyVMI`. The prefix does three things: it makes every unconverted call site visible, it
  stops new specs reaching for the old vocabulary by habit, and it turns progress into
  `git grep -c legacy`.
- **The default pod becomes lazy.** `newHandlerEnv` stops creating the socket and launcher client
  eagerly; the legacy helpers call an internal `ensureLegacyPod()` on their first use, and `given`
  never does. This is what lets a v2 spec genuinely have no pod while a v1 spec in the next `It`
  still gets one. `f.SockFile` and `f.PodUID` become accessors that trigger the same lazy build (5
  call sites).
- Both vocabularies may appear in the same file, never in the same spec.

End state: delete `seed_test.go`, delete the lazy-pod shim, delete the `legacy` prefix from nothing
because nothing is left.

### 5.3 Phase 0 — enabling commits, before any spec moves

1. **Distinct global VMI store.** `handlerEnv.VMIGlobalInformer`; `newFixture` passes it; the legacy
   seeders write both. No spec changes.
2. **Make the default pod lazy.** No spec changes; `f.SockFile`/`f.PodUID` become accessors.
3. **Prefix the legacy helpers** with `legacy`, mechanically, across all 57 sites. One noisy commit
   that makes every later commit small.
4. **Add `scenario_test.go`**: `given`, `then`, the option families, validation, realisation. No spec
   uses it yet.
5. **Add the translation-table test** (§5.4). This is the commit that makes per-spec conversion
   mechanical rather than a judgement call.
6. **Add two or three specs that only the new API can express** — orphan domain with no pod,
   `storedIn(globalCache, api)`. The API gets exercised before it is imposed on anything.

### 5.4 The translation table

A temporary test, deleted with the last legacy helper, that pins what each legacy helper actually
produces:

```go
// scenario_equivalence_test.go
DescribeTable("a legacy helper realises the same world as its explicit equivalent",
    func(legacy func(*fixture), explicit func(*fixture)) {
        a := newFixture(); legacy(a)
        b := newFixture(); explicit(b)
        Expect(b.world()).To(Equal(a.world()))
    },
    Entry("legacyRunningVMI",
        func(f *fixture) { f.legacySeed(f.legacyRunningVMI()) },
        func(f *fixture) {
            f.given(withVMI(v1.Running, storedIn(nodeCache, globalCache, api)),
                withPod(socketReady(), clientResponsive()),
                withDomain(api.Running, ""))
        }),
    // … legacyScheduledVMI, legacySeedDomain, legacyVMIInPhase
)
```

`f.world()` is a printable snapshot: store keys, API objects, domain store contents, which socket
paths exist, the launcher manager flags, the ghost records. Four entries pin the translation the 63
conversions rely on, and the snapshot is independently useful — a failed conversion prints the two
worlds side by side instead of a bare assertion failure.

### 5.5 The per-spec recipe

For each spec, in order:

1. Run it alone; confirm green.
2. Replace the legacy seeding with `f.given(...)`, stating **exactly** the world the legacy helper
   produced, straight from the translation table. Change nothing else — same assertions, same mock
   expectations, same events.
3. Run it alone again. It must pass for the same reason. **If it fails, stop**: the legacy world was
   not what the table says, and that is a finding about the spec, not a problem with the conversion.
   Record it, do not paper over it.
4. Only in a *separate, later* commit, change the stated world if the spec's name promises something
   different from what it had — for instance the orphan-domain specs that should also exist with no
   pod. A conversion commit never changes behaviour; a behaviour commit never converts.

That discipline is the whole point of going one spec at a time. Without it, 63 rewrites produce 63
chances to silently test something else.

### 5.6 Order and batching

Commit per `Context`, not per spec: 63 seeding specs across ~15 Contexts, so ~15 reviewable commits.
Converting a whole Context at once also keeps its local helpers coherent (`terminatingVMI` at
`vm_test.go:476`, the `DescribeTable` closures) — those become small wrappers over `given` rather
than over the legacy helpers.

Suggested order:

1. `Context("…informed about a Domain change…")` (`:104`) — contains the six `seedDomain` specs that
   prompted the review, and the most hidden state. Validates the API on the hardest case first.
2. `Context("When a Running VMI has no domain")` (`:329`) — five specs, all about launcher state, so
   it exercises the pod options hard.
3. `Context("when VMI has DeletionTimestamp and domain has grace period")` (`:471`) — three specs,
   shows a local helper wrapping `given`.
4. The remaining small Contexts (`:769`, `:957`, `:1142`, `:1202`, `:1293`, `:1532`, `:1683`,
   `:1725`, `:1836`, `:1923`, `:1960`).
5. `Context("decentralized live migration source ghost-record cleanup")` (`:1998`) last — it is the
   only spec that drives ghost records, multi-step transitions and domain deletion at once, so it
   wants `then`, `informersCatchUp` and the ghost record options all finished.
6. Teardown commits: remove the lazy-pod shim and the env's pod fields, delete `seed_test.go`, delete
   the translation-table test, delete the dead knobs (`useClusterConfig`, `setenv`, `enqueue`).

### 5.7 Which specs need a scenario at all

Not all 135 specs in the file. Of the 83 in the main `Describe`, **63 drive `f.execute()`** and need
a world. The other ~20 call a controller method directly (`hasGracePeriodExpired` at `:253`,
`calculateVmPhaseForStatusReason` at `:390`) and build their inputs with `libvmi` — they need no
`given`, and forcing one on them would be ceremony. The rule: **`given` is required for specs that
call `execute`.** The 52 specs in the later `Describe`s (`isGuestAgentSupported`,
`claimDeviceOwnership`, `updateBackupStatus`, `checkVolumesForMigration`, …) are unit tests on
helpers; four of them build a fixture purely to get a controller, and none seeds state. They are out
of scope.

### 5.8 Progress and backsliding

`git grep -c "f\.legacy" pkg/virt-handler/vm_test.go` is the burn-down, 57 → 0. A new spec cannot
reach the old vocabulary by accident because everything in it is prefixed, and the last commit
deletes the file it lives in.

## 6. Risks and things to verify while implementing

- **`noSocket()` and the real container-disk mounter.** `newFixture` wires the production
  `container_disk.NewMounter` unless `withMockContainerDiskMounter()` is used, and `processVmCleanup`
  calls `Unmount(vmi)` first (`vm.go:1517`). With the mount root missing, that may error and change
  the branch under test. Check before wiring `noSocket()` to a failing `Detect`; if it does error,
  either keep the mount root readable for `noSocket()` by default or default those specs to the mock
  mounter.
- **`MockLauncherClientManager` ignores the VMI** (`launcher-clients/mock_launcher-clients.go:38-62`),
  so the `client*` options are fixture-global, not per pod. A scenario with two pods cannot give them
  different liveness, which is what honest coverage of `vm.go:365-388` needs. Keying that mock on VMI
  UID is a follow-up; until then `given` should reject a second pod rather than pretend.
- **`IsolationDetector` is programmed once with `gomock.Any()`** (`environment_test.go:284`). Per-pod
  mount-root behaviour needs the expectation to match on the VMI argument, and ordering against specs
  that set their own `Detect` expectations needs checking.
- **The lazy pod is load-bearing during the transition.** If any legacy path reads `f.SockFile`
  before the accessor exists, or a v2 spec triggers the lazy build through a legacy code path, the
  isolation between the two vocabularies is gone and `noPod()` silently becomes a pod. Worth one
  assertion: `given` fails if the legacy pod was already built in this spec.
- **Verbosity.** Three to five lines at 63 call sites. The temptation will be a `healthyNode()`
  helper in `vm_test.go`, which relocates the assumption instead of removing it. A named combination
  is acceptable only when its name enumerates what it pins and it sits beside the specs that use it —
  `terminatingVMI` at `vm_test.go:476` is the model.
- **The event-leak check in `DeferCleanup`** (`environment_test.go:242-248`) means any new branch a
  converted spec starts taking must have its events asserted. Expect specs to surface events they
  were not previously reaching — that is the point, but it will look like breakage.
- **The teardown commit is the risky one.** Removing the lazy default pod will fail anything that
  still depended on it. By then the failures should be zero; if they are not, they are real findings.

## 7. Alternatives considered

- **A parallel `fixtureV2` type.** Unsafe while both can be alive in one spec: the env owns four
  process globals (§5.1). Rejected.
- **A separate `Describe` (or file) for converted specs,** each with its own `BeforeEach`, specs
  moved across one at a time. This *is* safe — one fixture per spec — and lets v2 be strict from day
  one with no lazy-pod shim. Rejected because every conversion becomes a delete-plus-add diff, the
  Context structure and its local helpers have to be duplicated while both files exist, and the lazy
  pod is a dozen lines that buys the same strictness in place. Reconsider if the shim turns out to
  leak.
- **Flat option namespace** (`vmiRunning()`, `podSocketReady()`, `domainShutoff()` all as
  `scenarioOption`). Fewer types, but the grouping survives only as a naming convention, the compiler
  cannot stop a pod option landing in the VMI part, and every name needs a prefix. Rejected.
- **Scenario at construction** (`f := newFixture(withScenario(...))`). Appealing — there is no
  half-built fixture — but specs set gomock expectations on `f.LauncherClient` between construction
  and seeding, `DescribeTable` entries vary the scenario, and multi-step specs need a second
  statement anyway. `f.given` keeps the `BeforeEach` construction the file already uses.
- **Presets** (`withPod(runningLauncher())` expanding to socket + client + ghost record). Terser, but
  it is exactly the assumption-bundling rule 1 removes, and the bundles would drift. Rejected for
  now; revisit only if the verbosity proves unmanageable, and then only with names that enumerate
  what they pin.
- **A single `podPresence` enum** instead of socket/client options. Fewer illegal states, but it
  cannot express "responsive launcher, no client", which a current spec depends on. Rejected.
- **Struct literal** (`f.given(scenarioSpec{VMI: ..., Pod: ..., Domain: ...})`). Named fields read
  well, but it cannot express a repeated or conditional option, it does not compose with
  `libvmi.Option`, and a missing field is as silent as a missing option while being harder to
  validate per part. Rejected.

## 8. Out of scope

- The migration controllers' tests (`migration_test.go`, `migration-target_test.go`,
  `migration-source_test.go`) do not use the fixture yet. Converting them is separate work, though
  the pod options are where they would benefit most.
- Replacing `MockLauncherClientManager` with a per-VMI fake (see §6).
- `takeOverGlobals`' package-level takeover is unchanged by this plan.
