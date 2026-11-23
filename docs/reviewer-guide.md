# Reviewer guide

Some tips and tricks to obtain better reviews.

Make a few passes over the code you want to review.

1st pass, make sure the general design makes sense and that the code is structured in a way that's consistent with the project.

2nd pass, detailed look. depending on the complexity of the PR, sometimes it easier check out the code and step through it in your editor.

3rd pass, verify diffs from previous feedback rounds.

4th pass, through the whole PR once you think that it is ready to be merged, try to put yourself in the position of someone who needs that feature desperately and play with the flow several times

## Good to check:

* User input validation
* Reasonable error messages
* Reasonable info messages
* Unit and functional tests to avoid regressions (the core case must be tested)
* The PR implement what is needed (changes out of the PR scope should belong to a separate one)
* Nested loops must be avoided. When matching elements, create an
  index in a hash map on one of the lists and iterate over the other list. This
  trades `O(n^2)` with `O(n)`, which keeps kubevirt robust and scalable.

## Release notes labels

PRs have a ```release-notes``` section of their description and we use GitHub labels to filter release notes when we generate them at release time. For some PRs, kubevirt-bot is able to apply one of more of these labels but in general they need to be manually added.

If a release note is included in the PR, ensure that it has a `kind/` label and, if relevant, a 'sig/' label as well. For more information on these labels, see the [release-procedure.md](release-procedure.md#using-github-labels-for-release-notes) doc in this repo.

## Pull Request structure

* It's preferred that authors rebase on main instead of merging the main branch into their PRs.
* We merge PRs into our branches.
* Commits in a PR should make sense: Ask people to squash commits like "Fix reviewer comments", "wip", "addition", ...

## Common Architecture Flaws to Avoid

* Avoid using api GETs/LISTs to retrieve an object from the api-server when an informer is more appropriate. In general, informers should be used in cluster wide components such as virt-controller and virt-operator. 
* Note that informers should not however be used within virt-api. This is because unlike virt-controller and virt-operator the API can scale freely, does not use any leader election and pre-caching objects with informers can lead to an unexpected high amount of object transfers far beyond context information retrieval via GETs during actual webhook invocations. For this reason using api GETs etc for object retrieval are recommended from within the virt-api. See [this ML thread for more context](https://groups.google.com/g/kubevirt-dev/c/q_hR1tFH4Rk).
* Use a PATCH instead of an UPDATE operation when a controller does not strictly own the object being modified. An example of this is when the live migration controller needs to modify a VMI. The VMI is owned by a different controller, so the migration controller should use a PATCH on the VMI.
* Avoid adding informers to node level components such as virt-handler. This causes api-server pressure at scale.
* Reconcile loops are multithreaded and we must pay attention to thread safety. For example, accessing an external golang map within the reconcile loop must be protected by locks.
* Take a critical look at an new RBAC permissions added to the project and determine if new permissions grant a component permissions that breaks separation of concerns. For example, virt-handler shouldn't need permissions to list all Pods because viewing pods is virt-controller's responsibility.
* Always consider the update path. Most importantly, does a PR cause a change in behavior that impacts previously expected behavior? For example, if virt-handler always expects a file to exist within a certain folder within virt-launcher, and a PR changes that path, then does this impact virt-handler's ability to manage communication with old virt-launcher pods?
* When creating kubernetes events, make sure the code path issuing the event doesn't cause the event to fire every time the object is reconciled. For example, if we want to fire an event when a vmi moves to the running phase, we should compare the old phase with the new phase and only fire the event when the phase transition is occurring. A bad example would be to fire the event every time the reconcile loop sees the vmi's phase is Running. This would cause an unnecessary amount of duplicate events to be sent to the api-server.
* List ordering on CRD APIs matter. If two components need to update a list on the same object, make sure both components do it in a way that preserves the order of the list. For example, both virt-handler and virt-controller need to modify conditions on the VMI status. If both virt-handler and virt-controller are constantly changing the order of the conditions list, that will cause an update storm where both components are competing with one another to write changes.
* Privileged node-level operations should be added to virt-handler and not virt-launcher to keep the privileges on virt-launcher at a minimum.

## In-Depth PATCH/UPDATE considerations

When trying to determine if an UPDATE or a PATCH should be used, the following has to be considered:

### Updating `spec` or `metadata` sections with a controller

Did the controller in question CREATE the object it operates on?
- Yes: `Update` can be used
- No: `Patch` with a narrow focus on the fields of interest needs to be used

Reasoning:

A controller which did not create an object can not have a full picture of the
object which it operates on. Operating on a compatible API does not mean that
the typed rest clients of k8s see all existing fields. Deserializing and
serializing an object can therefore lead to lost information since a the
final `Update` call would just remove the unknown fields.

On the other hand, if the controller created an object, the operator added all
information which mattered itself. Any additional information, since the
controller is the owner, can only be defaulted from the apiserver. Still one has
to be careful to not hot-loop on such defaulted values.

### Updating `status` sections from a single controller

Does the controller own the `status`?
- Yes: `UpdateStatus` can be used
- No: `Patch` needs to be used

Reasoning:

`status` sections are exclusively owned by controllers and are never owned by
end-users. As such, even if an object gets created by users, and the controller
does not own the `spec` and the `metadata` it usually owns the `status`.

Since it owns the status it is solely responsible for it and as such it can do
`UpdateStatus` operations on that section.

### Updating `status` sections from multiple controllers

Does the controller own the `status`?
- Yes: `UpdateStatus` can be used

Does the controller only enrich certain sections of the `status` and does not own it?
- Yes: `Patch` with a narrow focus on the fields of interest needs to be used

Reasoning:

This can be derived from the reasoning of the use-cases above. Such
multi-controller cases very often require special handling during updates (like
which components get updated first).

## When is a PR good enough?

For defining the lowest acceptable standards the project relies on automation.
PRs must pass the required automated checks and provide adequate test coverage
for all features and bug fixes at the appropriate level: unit, integration, or
end-to-end. End-to-end tests are required only when explicitly called for by
the change's testing requirements.
Maintainers are allowed to take in code with varying quality for as long as the
project's maintainability is not at stake and all required criterias are met
(especially the testing and architectural criteria) to be open and inclusive.

The lowest bar for acceptable **coding styles** is enforced via automation:
* [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) to enforce a common coding style for go code.
* [shfmt](https://github.com/mvdan/sh) to enforce a common coding style for bash scripts.

The lowest bar for acceptable golang **coding standards** (anti-patterns, coding errors, ...) is enforce via automation:
* [nogo](https://github.com/bazelbuild/rules_go/blob/master/go/nogo.rst) from
  bazel is used and applies a [huge set](https://github.com/kubevirt/kubevirt/blob/main/nogo_config.json) of code
  analyzers when one builds kubevirt. If a check fails the build fails.

The lowest **testing bar** to pass:
* All features and bug fixes require adequate test coverage of the core behavior.
* Choose unit, integration, or end-to-end tests according to the behavior being
  verified. Prefer the lowest level that reliably exercises the behavior and,
  for bug fixes, prevents the regression.
* Add end-to-end tests when correctness depends on runtime interactions that
  unit or integration fixtures cannot adequately reproduce, such as interactions
  with QEMU/libvirt, the guest, or Kubernetes components. For example, verifying
  that a VM remains operational after live migration requires exercising the
  running virtualization stack; checking the generated migration configuration
  alone cannot establish this.
