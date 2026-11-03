# CEL Validation Cost Guide

This document explains how the Kubernetes apiserver estimates the cost of
`x-kubernetes-validations` (CEL) rules on a CustomResourceDefinition, why
certain intuitive-looking rules get rejected at CRD-install time, and how to
decide between CEL, a native OpenAPI schema keyword, or a non-CEL structural
mechanism (`listType=map`) when migrating a validation check. It complements
[design_guidelines.md](design_guidelines.md) §4.1–4.2, which covers *when* a
constraint belongs in the schema versus a webhook and the upgrade-safety rules
for introducing one; this document covers the cost-estimation mechanics that
determine whether a CEL rule you've already decided to write is actually
affordable, and cites the authoritative sources behind it.

All claims below are cited to a Kubernetes source file, an official doc, or
the cel-go cost-model specification. Findings carried over from empirical
testing against this repository's own envtest harness during the sig-network
admission-webhook-to-CEL migration ([#19701](https://github.com/kubevirt/kubevirt/issues/19701))
are marked **[Empirical]**.

## 1. How CEL Cost Estimation Actually Works

### 1.1 The two budgets

Source: `k8s.io/apiserver/pkg/apis/cel/config.go` (`kubernetes/kubernetes`
staging repo):

```go
PerCallLimit                         = 1_000_000   // ~0.1s per single CEL validation call
RuntimeCELCostBudget                 = 10_000_000  // ~1s total, per CR validation (all rules summed)
RuntimeCELCostBudgetMatchConditions  = 2_500_000   // ~0.25s, webhook/VAP matchConditions
MaxRequestSizeBytes                  = 3*1024*1024 // 3 MiB — the apiserver's max request size
```

There are two separate checks, both using a *static, worst-case estimate*
computed at CRD-install time, not a runtime measurement:

1. **Per-rule check**: each individual `x-kubernetes-validations` rule's
   estimated cost is checked against `PerCallLimit`.
2. **Per-CRD aggregate check**: the sum of every rule's estimated max cost on
   the whole CRD is checked against the CRD-level budget.

Kubernetes docs: *"Each CustomResourceDefinition is also allowed a certain
amount of resources to finish executing all of its validation rules. If the
sum total of its rules are estimated at creation time to go over that limit,
then a validation error will also occur."* — [Validation Rules reference](https://kubernetes.io/docs/reference/using-api/cel/#resource-constraints)

At runtime, the *actual* (not estimated) cost is tracked per-call against
`PerCallLimit` and per-CR against `RuntimeCELCostBudget`; if actual execution
exceeds it, evaluation halts and the request is rejected. But the gate that
blocks you from installing the CRD at all is the static estimate, which is
deliberately pessimistic (worst case, not expected case).

Source on regex cost specifically: *"Many of CEL's core operations have fixed
costs... Calls to functions implemented in native code approximate cost based
on time complexity... operations that use regular expressions, such as match
and find, are estimated using an approximated cost of
`length(regexString)*length(inputString)`."* — same reference as above.

### 1.2 "Unbounded" means "up to ~3 MiB", not infinite

This is the mechanical fact behind the most commonly hit failure mode: any
string/list/map field with no `maxLength`/`maxItems`/`maxProperties` declared
is treated as effectively unbounded for cost-estimation purposes, regardless
of what the CEL rule's own logic does with it.

Source: `k8s.io/apiserver/pkg/cel/common/schemas.go` (`SchemaDeclType`):

```go
// k8s.io/apiserver/pkg/cel/common/schemas.go
case "string":
    ...
    strWithMaxLength := apiservercel.NewSimpleTypeWithMinSize("string", cel.StringType, types.String(""), apiservercel.MinStringSize)
    if s.MaxLength() != nil {
        strWithMaxLength.MaxElements = estimateMaxElementsFromMaxLength(s)   // MaxLength * 4
    } else {
        if len(s.Enum()) > 0 {
            strWithMaxLength.MaxElements = estimateMaxStringEnumLength(s)   // length of longest enum value
        } else {
            strWithMaxLength.MaxElements = estimateMaxStringLengthPerRequest(s) // ~3 MiB
        }
    }
```

- **String with `MaxLength` declared** → estimated max size = `MaxLength * 4`
  bytes (the ×4 accounts for UTF-8 code points up to 4 bytes each).
- **String with no `MaxLength` but a native OpenAPI `enum`** → estimated max
  size = the length of the *longest enum string value*. This is a real,
  source-confirmed escape hatch: a native `enum` gives the cost estimator a
  tight bound *without* a `maxLength` annotation. See §3.2 below for why this
  usually doesn't help KubeVirt's open-string-set fields anyway.
- **String with neither** → estimated max size = `MaxRequestSizeBytes - 2` ≈
  **3,145,726 bytes**. Not literally unbounded, but large enough that any
  non-trivial per-character cost (e.g. a regex match) reliably blows the
  per-rule budget by 10–100×+.
- **Array with no `maxItems`**: estimated max item count = `(MaxRequestSizeBytes
  - 2) / (minSerializedSizeOfOneItem + 1)` — "however many of the
  smallest-possible item could theoretically fit in one request." Still huge
  for small items.
- **Map/object with no `maxProperties`**: analogous fallback via
  `estimateMaxAdditionalPropertiesFromMinSize`.

**[Empirical]** A `self.matches(regex)` call on a single unbounded string
field, evaluated once per item in a `maxItems:256` list (no cross-list logic
at all), was rejected with *"estimated rule cost exceeds budget by factor of
32.2x"*. The list itself was bounded; the string inside each item was not,
and that alone was enough.

**[Empirical]** A list-uniqueness idiom,
`self.all(x, self.exists_one(y, y.name == x.name))` (the Kubernetes-documented
pattern for enforcing unique values across a list), over two unbounded-string
lists was rejected by *"more than 100x"* over budget — O(n²) comparisons
against an effectively-3MiB string estimate compounds fast.

Kubernetes' own docs give the canonical minimal example of this exact failure:

```yaml
foo:
  type: array
  items:
    type: string
  x-kubernetes-validations:
    - rule: "self.all(x, x.contains('a string'))"
```

rejected with:

```
CEL rule exceeded budget by more than 100x (try simplifying the rule, or adding maxItems, maxProperties, and
maxLength where arrays, maps, and strings are used)
```

The only fix shown in the official docs is adding `maxItems`/`maxLength` to
the schema — there is no CEL-side fix.

### 1.3 Why an in-expression `self.size() <= N && ...` guard does not help

**[Empirical]** Tested directly: `self.size() <= 12 && self.matches(...)`
produced a byte-for-byte identical cost-budget error to the guard-less
version. This is not a quirk of this one case — it's how the cost model is
fundamentally defined, confirmed by two independent pieces of evidence:

1. **The Kubernetes-specific cost estimator does no data-flow narrowing.**
   `k8s.io/apiserver/pkg/cel/library/cost.go` has a `knownUnhandledFunctions`
   map that explicitly includes `_&&_`, `_||_`, `_<=_`, `_?_:_`, etc. — these
   core operators are costed by CEL's own generic operator-combinator logic,
   which only ever combines the *independently-estimated* costs/sizes of
   their operands. There is no code path that takes a comparison like
   `self.size() <= 12` and uses it to shrink the size interval used to cost a
   *different* reference to `self` later in the same expression.

2. **cel-go's own cost-model spec says so explicitly.** From
   [cel-go's `docs/cost-model.md`](https://github.com/google/cel-go) §3.10
   ("Overloads"): for short-circuiting operators, an operand that may not run
   counts toward **Max** but not **Min** — `&&` and `||` add only the first
   operand to Min; `?:` adds the condition and the cheaper branch to Min and
   the costlier branch to Max. The apiserver's admission-time rejection is
   driven by the **Max** of the estimated interval (the worst case), not the
   Min. For `A && B`, Max is *always* `cost(A) + cost(B)` — the estimator must
   assume `B` could run, so it always prices in the full cost of
   `self.matches(...)` against the unbounded size of `self`, regardless of
   what `A` claims to guarantee. The spec also documents (§3.5, "Interval
   dependency"): *"an operand that appears twice is treated as two
   independent quantities... This over-approximates, so it is sound."* —
   `self` appearing once in `.size()` and again in `.matches()` are priced
   completely independently; the second use gets no benefit from the first's
   comparison.

**Consequence: no syntactic reordering or restructuring of a same-expression
guard changes this.** Putting the guard after instead of before
(`self.matches(...) && self.size() <= 12`) does not help either, nor would
wrapping it differently.

### 1.4 Splitting one rule into several also does not help

Official Kubernetes docs test this exact idea directly (moving the same
comprehension from the array-level schema down to the per-item schema,
superficially "splitting" the rule per item) and confirm it changes nothing:

> "the following rule will have the same estimated cost as the previous
> example (despite the rule now being defined on the individual array
> items)... The cost estimation system takes into account **how many times
> the rule will be executed** in addition to the estimated cost of the rule
> itself."

Total estimated cost is `(cost of one evaluation) × (number of times it's
estimated to run)`. Relocating or splitting the rule doesn't change that
product, it just moves where the multiplication happens.

### 1.5 Embedding the bound inside the regex itself does not help

Hypothesis: could writing the length bound into the pattern (e.g.
`^.{0,12}...$`) change the cost estimate differently than a separate
`self.size() <= N &&` guard? No — the regex cost formula is
`length(regexPattern) * length(inputString) * RegexStringLengthCostFactor`.
The pattern's own content (including any `{0,12}` written into it) only
affects `length(regexPattern)` — a handful of extra characters, trivial. It
has zero effect on `length(inputString)`, which is `self`'s estimated size
and comes exclusively from the schema-path lookup in §1.2. The estimator
never parses or interprets the regex's semantics (e.g. that it can only match
short strings); it only measures the regex *source string's* character
length.

### 1.6 `cel.bind()` solves a different problem

`cel.bind(var, expr, body)` avoids *recomputing* an expensive derived value
that's referenced multiple times in one rule (it's priced once instead of
once per reference). It does nothing for a single `.matches()` call that's
already evaluated once, and it has no mechanism to inject a tighter size
estimate for an unbounded field — the bound expression's own estimated size
is still derived the same schema-path way described in §1.2.

### 1.7 Is there any annotation that supplies a cost hint without an enforced bound?

No. Traced the full chain: `apiextensions-apiserver/pkg/apiserver/schema/cel/compilation.go`
(`sizeEstimator.EstimateSize`) walks the CEL expression's field path against
the structural schema's `DeclType` (built by
`k8s.io/apiserver/pkg/cel/common.SchemaDeclType`), and that `DeclType`'s size
bound is populated *exclusively* from `MaxLength` / `MaxItems` /
`MaxProperties` / `Enum` on the OpenAPI schema itself (§1.2). There is no
separate `x-kubernetes-*` annotation, no `+kubebuilder` marker, and no CEL
syntax that feeds a size hint into the estimator independently of an actual,
enforced, structural-schema constraint.

### 1.8 What is cheap

Worth stating explicitly, since it's the shape that works without any bound:
`has()`/presence checks and arithmetic/boolean combinations over struct
fields cost O(1) regardless of field sizes — they never touch string length
or list/map size at all. Simple scalar comparisons (`==`, `!=` between
fixed-size values) are similarly cheap. A CEL `in` membership test against a
literal list of short constant strings (`self in ['a', 'b', 'c']`) costs
`ListCreateBaseCost` (≈40) plus O(n) comparisons for n literal values — its
cost is proportional to the *list literal's* size, not to `self`'s
(potentially unbounded) length. See the worked examples in §3.

## 2. Decision Checklist

Apply in this order when migrating an admission-webhook check to the CRD
schema, or writing a new validation rule:

1. **Is the check expressible as `has()`/presence/arithmetic over struct
   fields, with no string-length- or list/map-size-dependent operation
   (`matches`, `contains`, `.all`/`.exists`/`.exists_one` over unbounded
   collections, concatenation of unbounded fields)?**
   → Use CEL directly. No schema bound needed. (§3.1)

2. **Does the check require comparing values across list elements by a
   natural key (uniqueness, "at most one of X")?**
   → Prefer `+listType=map` / `+listMapKey=<key>` (native structural-schema
   uniqueness) over a CEL cross-list idiom
   (`self.all(x, self.exists_one(y, ...))`). The apiserver enforces key
   uniqueness natively, at zero CEL cost, with no length bound required on
   the key field. This is the single most cost-dangerous common CEL pattern
   (O(n²) over two unbounded-string lists) and usually has a cheaper native
   equivalent. (§3.2)
   - Caveat: `listType=map` changes server-side-apply merge semantics
     (per-element merge-by-key instead of whole-list replace) — call this out
     to reviewers.
   - This only solves *single-list* key uniqueness. Cross-list
     existence/pairing checks (e.g. "every interface name must have a
     matching network name") still need either CEL (if the fields involved
     are short/bounded) or must stay as webhook code if neither mechanism is
     affordable or safe.

3. **Does the check require a string-length-, list-size-, or map-size-
   dependent CEL operation on a field with no declared
   `maxLength`/`maxItems`/`maxProperties`?**
   → This cannot be written affordably without a schema-level bound (§1).
   The question becomes "can I safely add that bound?", governed by
   [design_guidelines.md](design_guidelines.md) §4.1.1 and §4.2.3:
   - **The field is brand new** (first release it can appear in) → safe to
     add the bound alongside the CEL rule from day one. No existing client
     can already be writing values that violate it.
   - **The field is existing/previously unbounded, but the exact length
     bound is mathematically implied by a format rule being added in the
     very same change** (e.g. a fixed-width hardware address format) →
     safe. The bound excludes nothing beyond what the new format rule
     already excludes, and if the format check itself isn't new (it was
     already enforced by a webhook), no already-stored object could already
     violate either. (§3.4)
   - **The field is existing/previously unbounded, and the bound is *not*
     implied by anything already enforced** → unsafe per §4.2.3. CRD
     Validation Ratcheting is not sufficient cover: it's GA only from
     Kubernetes 1.33, and even then it protects only an unchanged,
     already-stored value from being re-rejected on an unrelated update — it
     does not protect a client actively writing a new/changed value for that
     field. Options, in order of preference:
     a. If it's a list/map key-uniqueness check → use `listType=map` instead
        (step 2); sidesteps CEL/length entirely.
     b. If the field is new or the bound is genuinely a provisional
        guess rather than a retrofit onto old data → see
        [design_guidelines.md §4.2.4](design_guidelines.md#424-when-uncertain-start-restrictive)
        ("When Uncertain, Start Restrictive") — note that section still
        requires the bound to go through CEL, not a native OpenAPI
        constraint, precisely so it stays easy to loosen later.
     c. Otherwise → leave the check as webhook-enforced; an explicit,
        sanctioned fallback per
        [design_guidelines.md §4.2.1](design_guidelines.md#421-prefer-cel-over-webhooks)
        when CEL genuinely can't express the constraint affordably or
        safely.

4. **Is the field a closed, fixed, non-extensible set of values?**
   → A native OpenAPI `enum` is viable, and per §1.2 it also gives the cost
   estimator a free size bound derived from the longest enum literal, with no
   `maxLength` declaration needed — but only if the set is genuinely closed.
   If it's an **extensible list of choices** (disk bus types, binding modes,
   hypervisor features, interface models, etc.),
   [design_guidelines.md §3.2](design_guidelines.md#32-open-string-set-avoiding-enums)
   forbids a native `enum` regardless of the cost-estimation convenience,
   because it breaks rolling upgrades for a different reason: an older
   client or control-plane component holding the old schema permanently
   rejects a value a newer release considers valid. In that case, use a CEL
   `self in [...]` allow-list instead (§1.8, §3.3):
   - A CEL `self in [...]` list-literal membership check is not a
     published/bundleable OpenAPI schema keyword the way `enum` is — it
     stays a server-side-only concern, not something third-party tooling can
     independently bundle/enforce.
   - Its cost is proportional to the list-literal size, not to `self`'s
     length, so **no `maxLength` is needed** even on an old, unbounded
     field.

5. In all cases, before merging: is the check itself new, or just relocated
   from an existing webhook that already ran on every create/update? If it's
   a relocation of an already-universally-enforced check, there is no
   backward-compatibility risk regardless of which mechanism it lands on —
   state this explicitly in the PR description; it is the strongest argument
   for safety when converting an old field.

## 3. Worked Examples

All from the sig-network admission-webhook-to-CEL migration
([#19701](https://github.com/kubevirt/kubevirt/issues/19701)).

### 3.1 Cheap CEL, no bound needed — exactly one network source

Replacing a webhook check with pure `has()`/arithmetic over struct presence —
no string or collection size involved anywhere:

```go
// +kubebuilder:validation:XValidation:rule="(has(self.pod)?1:0)+(has(self.multus)?1:0)+(has(self.resourceClaim)?1:0)==1",message="network source must have exactly one of pod, multus or resourceClaim"
```

Compiled and passed cost estimation immediately — the ideal CEL migration
shape (checklist step 1).

### 3.2 List-uniqueness via `listType=map`, not CEL — network/interface name uniqueness

A straightforward `self.all(x, self.exists_one(y, y.name == x.name))`
translation of "every network name is unique" blew the budget by >100×
(`Network.Name`/`Interface.Name` have no `MaxLength`). Adding `MaxLength`
purely to fix the cost problem was rejected as unsafe (§4.2.3 — an old,
previously-unbounded field, with no ratcheting cover for actively-written
values). Instead:

```go
// +listType=map
// +listMapKey=name
Networks []Network `json:"networks,omitempty"`
```

(same pattern on `Devices.Interfaces`). Duplicate-key rejection is now
enforced by the apiserver's native merge-key logic — zero CEL cost, no
length bound required anywhere. This also happened to resolve two
pre-existing `list_type_missing` entries already flagged in
`api/api-rule-violations-known.list`, independent corroboration that this
was the intended native mechanism for these fields all along.

This is checklist step 2's canonical example, and the strongest illustration
of "when a CEL cross-list idiom is too expensive, look for a native
structural-schema mechanism first."

### 3.3 CEL `self in [...]` instead of a native `enum` — interface model allow-list

```go
// +kubebuilder:validation:XValidation:rule="self in ['e1000','e1000e','igb','ne2k_pci','pcnet','rtl8139','virtio','']",message="interface uses a model that is not supported"
```

replacing a webhook allow-list check. Deliberately not a native `enum`,
because `Interface.Model` is an extensible list of choices (§3.2 of
design_guidelines.md) — a native enum would permanently block new model
names for any client holding a stale copy of the schema. The CEL `in` check
costs O(list-literal-size), not O(field-length), so it required no
`maxLength` addition despite `Model` being an old, unbounded field — a
cost-cheap, upgrade-safe, and extensibility-preserving outcome, all without
touching the schema's length constraints.

### 3.4 A `MaxLength` addition judged safe because it's implied by a same-change format rule — PCI address

```go
// +kubebuilder:validation:MaxLength:=12
// +kubebuilder:validation:XValidation:rule="self == '' || self.matches('^[0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\\.[0-7]$')",message="interface has a malformed PCI address"
PciAddress string `json:"pciAddress,omitempty"`
```

`PciAddress` is an old, previously-unbounded field — ordinarily a
`MaxLength` addition here would hit the same §4.2.3 objection as the
name-uniqueness case (§3.2). The distinguishing fact, stated explicitly in
the commit introducing this rule, is that a value *matching the new format
regex is always exactly 12 characters* — so `MaxLength:=12` excludes
nothing beyond what the (not-new — previously webhook-enforced) format rule
already excluded. The length bound is mathematically implied by the format,
in the same change, not an independent arbitrary restriction. Note
`design_guidelines.md` §4.1.2 independently lists "Fixed Cryptographic/IEEE
Formats... where the string layout is governed by immutable external RFC
standards" as a legitimate category for a structural-schema constraint — PCI
bus:device.function addressing is exactly such a fixed, externally-governed
format.

This does **not** generalize to "any field where you're also adding a format
check" — only to formats with a truly fixed, invariant length (unlike, say,
a free-form name field with a regex that still allows variable length, or a
format with more than one valid length).

#### CEL regex escaping gotcha

A literal `\.` inside a CEL regex, written inside a
`+kubebuilder:validation:XValidation:rule="..."` marker, needs **four**
backslashes in the Go source comment, not two. Two separate layers each
consume one level of backslash-doubling:

1. controller-gen's own marker-value parsing (extracting the `rule="..."`
   argument from the Go comment) unescapes `\\` → `\` once.
2. CEL's own string-literal lexer unescapes `\\` → `\` again when parsing the
   single-quoted string inside the rule.

So to get a literal `\.` into the final compiled CEL regex, the Go source
must contain `\\\\.`  (four backslashes then a dot). Verify by inspecting the
generated CRD YAML's `rule:` line after `make generate` — it should show
exactly `\.` (one backslash) there, which is what CEL's lexer will accept.
Getting this wrong produces a CEL *compile* error (not a cost-budget error):
`token recognition error at: ''...\.''` — a lexer failure, distinct from the
cost-estimation failures described in §1.

### 3.5 Pre-existing CEL rule found with zero test coverage — `ContainerPathVolumeSource`

Not a cost-estimation lesson, but a process one worth keeping alongside
these: this was the only CEL rule in the codebase before this migration
effort began, and it had no envtest coverage. Establishing envtest coverage
for it set the pattern now used for every rule in this effort — each new
`XValidation` rule gets coverage in `tests/integration/network/*_test.go`
(via the `tests/integration/framework` envtest harness) asserting both
accept and reject cases, including, where a `MaxLength` is also involved,
separate cases for "rejected by length" versus "rejected by the CEL rule
itself" — these two failure modes produce different top-level apiserver
error text (`Too long: may not be more than N bytes` vs. the CEL rule's own
`message`), so a single shared assertion across both would be wrong for
whichever case doesn't match.

## 4. Summary: Can You Avoid a Published Schema-Level Length Declaration?

No — not through any CEL syntax or annotation available today. The only
inputs the apiserver's cost estimator accepts for bounding a string/list/map
field's assumed size are, exclusively:

- `maxLength` (strings) — gives `MaxLength × 4` bytes,
- a native OpenAPI `enum` on a string with no `maxLength` — gives the
  longest enum literal's length (usable only where a native enum is
  otherwise acceptable, i.e. not on an extensible-choice field, §2 step 4),
- `maxItems` (arrays),
- `maxProperties` (maps/objects with `additionalProperties`).

An in-expression `size() <= N` guard (§1.3), embedding the bound inside the
regex pattern itself (§1.5), `cel.bind()` (§1.6), and splitting one rule into
several (§1.4) do not change the estimated size of an unbounded field — each
is ruled out by a distinct, specific mechanism in the cost estimator or in
CEL's own cost-model specification, not just by this repository's own
empirical trial and error. The only way to reduce an unbounded field's
contribution to estimated cost is a genuine, enforced structural-schema
bound — which must then separately clear the upgrade-safety bar in
[design_guidelines.md §4.2.3](design_guidelines.md#423-upgrade-safety) /
[§4.2.4](design_guidelines.md#424-when-uncertain-start-restrictive) — or be
sidestepped entirely via a non-CEL mechanism like `listType=map`, or a CEL
idiom like `in [...]` whose cost doesn't depend on the unbounded field's
length in the first place.

## Sources

- [Kubernetes docs: CEL resource constraints](https://kubernetes.io/docs/reference/using-api/cel/#resource-constraints) —
  cost units, budgets, the worked 100×-over-budget example and why relocating
  the rule doesn't help.
- [Kubernetes docs: Validation Rules](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules) —
  validation rules authoring reference; Validation Ratcheting (GA v1.33)
  scope and exceptions.
- `kubernetes/kubernetes` staging source:
  - `staging/src/k8s.io/apiserver/pkg/apis/cel/config.go` — exact budget
    constants.
  - `staging/src/k8s.io/apiserver/pkg/cel/library/cost.go` — `CostEstimator`,
    `knownUnhandledFunctions`, the `matches`/`find` regex cost formula.
  - `staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel/compilation.go` —
    `sizeEstimator.EstimateSize`, the schema-path-only size lookup.
  - `staging/src/k8s.io/apiserver/pkg/cel/common/schemas.go` —
    `SchemaDeclType`, the exact `MaxLength`/`enum`/fallback size-derivation
    logic quoted in §1.2.
- [cel-go `docs/cost-model.md`](https://github.com/google/cel-go) §3.5
  ("Interval dependency"), §3.10 ("Overloads" — short-circuit Min/Max
  accounting for `&&`/`||`/`?:`).
- [design_guidelines.md](design_guidelines.md) §3.2 ("Open String Set"),
  §4.1.1–§4.1.2 ("Only Absolute Invariants", "Legitimate Structural
  Invariants"), §4.2.1–§4.2.4 ("CEL Validation", "Upgrade Safety", "When
  Uncertain, Start Restrictive").
- This repository's own sig-network admission-webhook-to-CEL migration
  ([#19701](https://github.com/kubevirt/kubevirt/issues/19701)) and its
  commits on branches `network-cel-single-network-source` and
  `network-iface-cel-checks`.
