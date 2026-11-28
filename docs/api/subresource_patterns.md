# Kubernetes Subresource Design Patterns

This document is a detailed survey of how Kubernetes itself uses subresources,
and how that maps onto KubeVirt's own runtime-action subresources (`pause`,
`unpause`, `freeze`, `unfreeze`, `reset`, `softreboot`). It is the extended
reference behind the compressed rules in [§1.3](design_guidelines.md#13-declarative-intent-not-imperative-action)
and [§2.4](design_guidelines.md#24-subresources-for-actions) of the [KubeVirt
API Design Guidelines](design_guidelines.md); those sections state the rule,
this document shows the evidence.

## Why subresources exist

A subresource is a URI path rooted below a specific resource
(`/apis/GROUP/VERSION/.../RESOURCETYPE/NAME/SUBRESOURCE`) that exposes an
alternative action or view "closely coupled to a single resource" ([Kubernetes
API
Conventions](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api-conventions.md#types-kinds-and-resources)).
The recurring justification across every subresource in Kubernetes is **not**
"this is an action" — it is one of:

- a need for **RBAC distinct from the main object's `spec`/`status`**,
- a need for a **representation or protocol different from a singular JSON
  object** (a byte stream, a generic cross-type schema), or
- a need to **carve a narrow, scoped exception into an object whose `spec` is
  otherwise immutable or whose `status` is otherwise system-owned**.

None of these is "the operation can't be expressed as a field," even though
that is often true as well. The sections below work through every
non-trivial subresource pattern Kubernetes actually uses (excluding plain
`/status`, whose single-writer rationale is already well documented) and
what specifically justifies each one.

## Taxonomy

| Category | Examples | What's written, and where | Why a subresource |
|---|---|---|---|
| Streaming/connection | Pod `exec`, `attach`, `log`, `portforward` | Nothing — a live byte stream, not JSON | Different wire protocol (WebSocket/SPDY), different RBAC verb per stream type |
| One-shot admission decision | Pod `binding`, Pod `eviction` | A synchronous, final decision returned in the HTTP response itself | Decision requires evaluating a policy or relationship external to the object (PDB, scheduler placement); needs RBAC distinct from editing `spec` |
| Declarative window into immutable spec | Pod `resize`, Pod `ephemeralcontainers` | Ordinary declarative data, reconciled asynchronously, reported via ordinary `status` | The rest of `spec` is immutable post-creation; the subresource is a narrow, separately-authorized exception, not an imperative action |
| Schema-generic, cross-type | `scale` (Deployment, ReplicaSet, StatefulSet, ReplicationController) | A `Scale` object (just a replica count), reconciled asynchronously like any spec field | Lets a single generic controller (HorizontalPodAutoscaler) operate on many unrelated resource schemas without knowing their full shape |
| Virtual, non-persisted | ServiceAccount `token` (TokenRequest) | Nothing is stored on the object; a short-lived credential is computed and returned synchronously | The "resource" has no durable existence at all — it's a request/response computation scoped to the parent object, not a mutation of it |
| Request-as-Condition | CertificateSigningRequest `approval` | A `Condition` (`Approved`/`Denied`) appended to `status.conditions`, later completed by a different actor | Needs RBAC distinct from both plain object edits and from `/status` (which a *different* controller owns); reuses the standard Condition shape instead of a one-off field |

Each is detailed below, followed by a historical counter-example of an
imperative action subresource Kubernetes **removed**.

### Streaming/connection subresources

`exec`, `attach`, `log`, and `portforward` are WebSocket/SPDY upgrades that
open a live binary channel between the client and the running container
([API Conventions, "WebSockets and
SPDY"](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api-conventions.md#websockets-and-spdy)).
They do not mutate the Pod object, and there is no result to evaluate
afterward — the stream itself, observed in real time, is the result. This
pattern only applies to workloads that expose a live, addressable process;
it has no analogue in KubeVirt's guest-level actions, since none of
`pause`/`unpause`/`freeze`/`unfreeze`/`reset`/`softreboot` open a channel
into the guest.

### One-shot admission-decision subresources

`binding` and `eviction` are `POST`s where the HTTP response **is** the
authoritative, final answer:

- **`eviction`** evaluates the target's PodDisruptionBudget synchronously and
  returns `200` (the pod will be deleted) or `429 Too Many Requests`
  (rejected) within that same call. The caller never polls `status` to learn
  whether the request was accepted — the response code is the complete
  answer, even though the actual pod termination that follows is ordinary
  asynchronous deletion.
- **`binding`** synchronously commits `spec.nodeName` (for a Pod) or an
  equivalent claim→volume assignment (for a PersistentVolumeClaim) and
  succeeds or fails on the spot; it cannot be partially applied or retried
  into a different outcome.

Both fold a decision that depends on state *outside* the target object (a
PDB, the scheduler's placement choice) into a single call, and both are
effectively irreversible — there is no "un-bind" or "un-evict."

### Declarative-window subresources

`resize` (in-place Pod resource resize) and `ephemeralcontainers` (Pod debug
containers) are, underneath, fully declarative: `PATCH .../resize` writes new
container resource requests/limits, `PATCH .../ephemeralcontainers` appends
an ephemeral container spec, and in both cases the kubelet reconciles the
change asynchronously and reports progress through completely ordinary
`status` fields (a resize-progress indicator alongside per-container
resource status; `status.ephemeralContainerStatuses`) — the exact shape of
the resize status field has evolved across Kubernetes releases, but the
mechanism has stayed declarative throughout. The only reason either is a
subresource is that Pod's `spec` is otherwise immutable after creation, and
each needs its own RBAC verb (`pods/resize`, `pods/ephemeralcontainers`) so
that granting "debug a pod" or "resize a pod" doesn't also grant "rewrite a
pod's containers." This is the cleanest existing precedent for KubeVirt's
*level-based* actions (see the VMI mapping table below): the mechanism is
declarative end-to-end, split out purely for authorization scoping.

### Schema-generic subresources

`/scale` exists on Deployment, ReplicaSet, StatefulSet, and
ReplicationController. Its payload, the `Scale` type, carries nothing but a
desired/observed replica count — deliberately independent of each resource's
actual schema ([API
Conventions](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api-conventions.md#metadata)
notes `Scale` is one of the few "simple kinds" shared across API groups,
alongside `Status`, `List`, and `WatchEvent`). This lets the
HorizontalPodAutoscaler controller scale *any* of these workload types
through one generic interface without a type-specific integration for each.
The motivation here is reusability across unrelated schemas, not RBAC or
imperativeness — a distinct justification from every other row in the
taxonomy, and one that has no real KubeVirt parallel today (there is no
family of unrelated KubeVirt resources that all need a uniform "quantity"
interface).

### Virtual, non-persisted subresources

`serviceaccounts/token` (the TokenRequest API) accepts a `TokenRequest` and
synchronously returns a short-lived, audience-bound token — nothing is ever
written to the ServiceAccount object itself. The "resource" exists only for
the duration of the request/response; there is no object to read back later.
This is the extreme end of "not persisted, not reconciled, pure computation,"
and it is distinct from `authorization.k8s.io`'s `SubjectAccessReview` and
`TokenReview`, which are not subresources at all but standalone virtual
*resources* with the same non-persisted character — useful context for the
general pattern, but not evidence for "when should an operation live under
an existing object's path."

### Request-as-Condition: CertificateSigningRequest's `/approval`

This is the pattern most relevant to edge-triggered KubeVirt actions. A
`CertificateSigningRequest` goes through two independently-authorized
subresources:

| Step | Subresource | Field written | Who | Extra authorization |
|---|---|---|---|---|
| Approve/Deny | `certificatesigningrequests/approval` | `Approved`/`Denied` Condition in `status.conditions` | An approver (human or controller) | `update` on `approval`, plus `approve` verb on `signers`, scoped by signer name |
| Sign | `certificatesigningrequests/status` | `status.certificate` (PEM cert) | The signing controller | `update` on `status`, plus `sign` verb on `signers`, scoped by signer name |

(Source: [Kubernetes CertificateSigningRequest Authorization
docs](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/).)

The full lifecycle: a client creates the CSR object (`spec.request`,
`spec.signerName`); an approver `PATCH`es an `Approved` or `Denied`
`metav1.Condition` through `/approval`; if approved, the signing controller
watches for that condition, performs the signing, and writes the certificate
through the *separate* `/status` endpoint. Two things make this different
from a generic status write:

1. **The RBAC split is three-way**, not two-way: approving and signing are
   gated by different verbs on a third, unrelated resource (`signers`,
   scoped per signer name), on top of the subresource-level `update`
   permission. A client with `update` on `approval` but no `approve` grant
   for the relevant signer still cannot approve.
2. **The request and its outcome share one vehicle** — the `Condition` type
   (`status`, `reason`, `message`, `lastTransitionTime`) — rather than a
   bespoke field invented for this one use case. A watcher learns both "was
   this requested" and "what happened" from the same structured value.

Applied to an edge-triggered KubeVirt action (using `reset` as a stand-in):
a dedicated subresource could set `Type: Reset, Status: "True", Reason:
Requested, LastTransitionTime: T0` on acceptance; the controller performing
the actual hypervisor call would later flip it to `Status: "False", Reason:
Completed` (or `Failed`, with a `Message`) at `LastTransitionTime: T1`. A
caller records the time it issued the request and polls for
`lastTransitionTime > T0` — the same idiom already used for
`status.observedGeneration`.

### A counter-example: Deployment's removed `/rollback` subresource

Kubernetes once had exactly the kind of bare imperative action subresource
being discussed here, and removed it. `extensions/v1beta1` Deployments
exposed a `/rollback` subresource: `POST`ing a `DeploymentRollback` would
imperatively revert the Deployment to a prior revision. When `apps/v1`
stabilized the Deployment API, this subresource was dropped —
[kubernetes/kubernetes#70039](https://github.com/kubernetes/kubernetes/pull/70039)
("Switch kubectl rollout to directly rolling back deployments") is the
change that moved `kubectl` off of it — and rollback was reimplemented
entirely client-side: `kubectl rollout undo` reads the target revision from
the `deployment.kubernetes.io/revision` annotations kept on old
ReplicaSets, and `PATCH`es the Deployment's `spec.template` back to match
that ReplicaSet's pod template. The existing Deployment controller's normal
rolling-update reconciliation then does the work, with rollout progress
reported through the Deployment's ordinary `status` fields — no bespoke
action endpoint, no separate completion signal to invent. The
`extensions/v1beta1` Deployment (along with its `/rollback` subresource)
stopped being served when that API group's workload resources were removed
in Kubernetes 1.16 — a scoped removal of the Deployment-family resources,
not of `extensions/v1beta1` as a whole (its `Ingress` resource lived on
under that group until 1.22).

This is direct historical evidence for the preference ordering in
[§2.4](design_guidelines.md#24-subresources-for-actions): even
Kubernetes' own core workload API shipped an imperative action subresource
for an edge-triggered operation, found it added a parallel, special-cased
completion story that the ordinary reconciliation loop didn't need, and
replaced it with a declarative field patch plus the pre-existing status
machinery.

## How KubeVirt's existing runtime subresources actually behave

Traced end to end in the codebase (current as of this writing):

### `reset` / `softreboot`: a single synchronous call chain

1. **virt-api** (`pkg/virt-api/rest/lifecycle.go`, `ResetVMIRequestHandler` /
   `SoftRebootVMIRequestHandler`) calls into
   `pkg/virt-api/rest/subresource.go`'s `putRequestHandler` /
   `putRequestHandlerWithErrorPostProcessing`, which issues a blocking `PUT`
   to virt-handler via `virtHandlerConn.Put` (defined in
   `staging/src/kubevirt.io/client-go/kubecli/handler.go`). This HTTP client
   has a **10-second timeout**, set in `pkg/virt-api/rest/subresource.go`.
2. **virt-handler**'s HTTP server (`pkg/virt-handler/rest/lifecycle.go`,
   `ResetHandler` / `SoftRebootHandler`) calls
   `client.ResetVirtualMachine(vmi)` / `SoftRebootVirtualMachine(vmi)`
   directly and writes its HTTP response from that call's returned error —
   no workqueue, no deferred processing.
3. That client (`pkg/virt-handler/cmd-client/client.go`) makes a blocking
   unary gRPC call to virt-launcher via `genericSendVMICmd`, under a
   **20-second timeout** (`longTimeout`, same file).
4. **virt-launcher**'s gRPC server
   (`pkg/virt-launcher/virtwrap/cmd-server/server.go`, `ResetVirtualMachine`
   / `SoftRebootVirtualMachine`) calls
   `domainManager.ResetVMI`/`SoftRebootVMI` synchronously and sets the gRPC
   response's success/error fields directly from the result.
5. **`pkg/virt-launcher/virtwrap/manager.go`** calls libvirt directly —
   `dom.Reset(0)` for reset, `dom.Reboot(...)` for soft reboot — and returns
   whatever libvirt returns.

No goroutine, channel, or queue appears anywhere in this chain. The client's
original HTTP response genuinely reflects the outcome of the full
virt-api→virt-handler→virt-launcher→libvirt round trip — this is
structurally the same "response is the answer" shape as `binding`/`eviction`,
not a fire-and-forget trigger. There is, however, **no condition or other
status field recording that a reset/soft-reboot ever happened**; a watcher
other than the original caller has no way to find out.

> **Timeout-budget defect:** step 1's 10s timeout is shorter than step 3's
> 20s timeout. A hypervisor call that takes 10–20 seconds can time out the
> virt-api→virt-handler hop and surface as a client-visible failure, while
> the gRPC call to virt-launcher — and the underlying libvirt action — is
> still in flight and may still succeed. This is a defect in the current
> implementation, not a property of the synchronous-proxy pattern itself; any
> new subresource following this pattern must keep the outer timeout at
> least as large as the sum of every inner timeout it depends on.

### `pause` / `unpause`: the same trigger, plus an independently-verified condition

The request path for `pause`/`unpause`
(`pkg/virt-api/rest/subresource.go` → `pkg/virt-handler/rest/` →
`pkg/virt-launcher/virtwrap/manager.go`'s `PauseVMI`/`UnpauseVMI` → libvirt
`dom.Suspend()`/`dom.Resume()`) is the identical synchronous proxy chain
described above. What's different is that virt-handler *also* runs a
continuous domain-state reconciliation loop
(`pkg/virt-handler/vm.go`'s `updatePausedConditions` and
`calculatePausedCondition`) that independently observes the domain's
already-translated status (`domain.Status.Status == api.Paused` — the
translation from the raw libvirt domain state happens earlier, in
virt-launcher) and adds or removes the `Paused` condition through the
generic condition-manager helpers in `pkg/controller/conditions.go` —
entirely decoupled from whether a `pause` request was ever made, based
solely on what is actually true right now. This is why `pause` already fits
closer to a declarative, level-based field (per
[§1.3](design_guidelines.md#13-declarative-intent-not-imperative-action)):
there is a steady state ("paused") that a controller continuously confirms,
independent of the triggering call.

## Mapping KubeVirt's runtime actions onto the taxonomy

| KubeVirt subresource | Closest Kubernetes pattern | Why |
|---|---|---|
| `pause` / `unpause` | Declarative-window / level-based state | "Paused" is a steady state, independently reconciled and confirmed via the `Paused` condition — the subresource is really an RBAC-scoped trigger for something that is otherwise shaped like a declarative field |
| `freeze` / `unfreeze` | Declarative-window / level-based state | Same reasoning as pause: filesystem-freeze is a steady state a controller could, in principle, reconcile toward |
| `reset` / `softreboot` | One-shot admission decision (mechanically), but missing its defining property | The call chain is synchronous like `binding`/`eviction`, but unlike them, nothing durable records the outcome for a third-party watcher, and there is no cross-resource policy decision (PDB, scheduling) being made — it is closer to a bare RPC proxy than a policy gate |

## References

- [Kubernetes API Conventions — resource/subresource anatomy, declarative vs.
  imperative
  fields](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api-conventions.md)
- [Kubernetes API Concepts — virtual resource types
  (`eviction`, `TokenRequest`-style request/response
  resources)](https://kubernetes.io/docs/reference/using-api/api-concepts/)
- [Kubernetes CertificateSigningRequest Authorization
  docs](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/)
- [TokenRequest API
  (`serviceaccounts/token`)](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/)
- [Pod Resize
  (in-place resource resize)](https://kubernetes.io/docs/concepts/workloads/pods/pod-resize/)
- [Ephemeral Containers](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/)
- [API-Initiated Eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/api-eviction/)
- [kubernetes/kubernetes#70039 — Switch kubectl rollout to directly rolling
  back deployments](https://github.com/kubernetes/kubernetes/pull/70039) and
  the [1.16 deprecated-API removal
  announcement](https://kubernetes.io/blog/2019/07/18/api-deprecations-in-1-16/)
  (removal of Deployment's `/rollback` subresource)
