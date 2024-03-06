# Coding conventions

This document outlines a collection of guidelines, style suggestions, and tips
for writing code in the KubeVirt project. It is partially based on [Kubernetes
Coding Conventions](https://github.com/kubernetes/community/blob/master/contributors/guide/coding-conventions.md).

The coding conventions in this document are mainly focused on Golang,
the language used primarily in the KubeVirt project.

Consider each convention a broad and opinionated statement. That means
maintainers are allowed to make well-motivated exceptions, but it should not
be the norm.

As a developer you should at least be familiar with the [short coding
conventions (TLDR)](#short-coding-conventions-tldr) of the KubeVirt project.

# Overview of the most important conventions

When contributing to the KubeVirt project, pay special attention to the
following:

## Write elegant, cohesive and easily readable code

- If reviewers ask questions about why the code is the way it is, that's a
  sign that your code is not clear enough.
- Try improving your code on the expense of writing comments. A properly-named
  function is better than a comment on a code stanza.
- Add comments where code-documentation is not enough.

## Avoid nesting and complexity by using early returns

- Deeply nested if/else statements make it harder to understand code.
- By using early returns you can avoid nesting of if/else statements.
- Code that is easier to understand will be easier to review and less
  likely to contain hidden bugs (e.g. control flow issues).

### Bad example

```go
val, err := doSomething()
if err == nil {
    if val {
        return doSomethingElse()
    } else {
        return doAnotherThing()
    }
} else {
    return err
}
```

### Good example

```go
val, err := doSomething()
if err != nil {
    return err
}

if val {
    return doSomethingElse()
}

return doAnotherThing()
```

## Use the same coding style throughout a file

- When adding to a file, stick to the existing coding style to make reviews
  easier.
- When you encounter code that requires a larger cleanup do it in a separate
  commit or PR.

### Bad example

```go
func checkAB(in string) bool {
    var bool ok
    if in == "A" || in == "B" {
        ok = true
    } else if in == "C" {
        ok = false
    }
    return ok
}

func checkBC(in string) bool {
    switch in {
    case "B", "C":
        return true
    }

    return false
}
```

### Good example

```go
func checkAB(in string) bool {
    return in == "A" || in == "B"
}

func checkBC(in string) bool {
    return in == "B" || in == "C"
}
```

# Avoid repetition of hardcoded values

- Use constants if you would repeat hardcoded values otherwise.
- Keep them private if they are used only in a single package.
- Carefully consider if you should make a constant exported.

### Bad example

```go
func getImportantAnnotation(obj metav1.ObjectMeta) string {
    return obj.GetAnnotations()["kubevirt.io/my-annotation"]
}

func setImportantAnnotation(obj metav1.ObjectMeta, val string) {
    obj.GetAnnotations()["kubevirt.io/my-annotation"] = val
}
```

### Good example

```go
const annotationKey = "kubevirt.io/my-annotation"

func getImportantAnnotation(obj metav1.ObjectMeta) string {
    return obj.GetAnnotations()[annotationKey]
}

func setImportantAnnotation(obj metav1.ObjectMeta, val string) {
    obj.GetAnnotations()[annotationKey] = val
}
```

# Uniform import order and naming

- Use the following import order with one block per item:
    - Golang standard libraries
    - Ginkgo / Gomega imports (only in test files)
    - Third-party libraries
    - Local packages
- Use the following naming schemes for imports:
    - `virtv1` for imports of `kubevirt.io/api`
    - `metav1` for imports of `k8s.io/apimachinery/pkg/apis/meta/v1`
    - `k8s` prefix for `k8s.io` imports on collision with other imports
    - TODO: Add more
- `make format` is able to help you with this.

### Bad example

```go
import (
    "context"
    k8sv1 "k8s.io/api/core/v1"
    "time"

    ."github.com/onsi/ginkgo/v2"
    ."github.com/onsi/gomega"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    v1 "kubevirt.io/api/core/v1"
    "kubevirt.io/kubevirt/tests/clientcmd"

    "kubevirt.io/kubevirt/tests/testsuite"
)
```

### Good example

```go
import (
    "context"
    "time"

    ."github.com/onsi/ginkgo/v2"
    ."github.com/onsi/gomega"

    k8sv1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    virtv1 "kubevirt.io/api/core/v1"

    "kubevirt.io/kubevirt/tests/clientcmd"
    "kubevirt.io/kubevirt/tests/testsuite"
)
```

## Short coding conventions (TLDR)

Here are some important but short conventions. If you want to learn more
about conventions in detail take a look at the [detailed
coding conventions](#detailed-coding-conventions).

- [Write elegant, cohesive and easily readable code](#write-elegant-cohesive-and-easily-readable-code)
- [Avoid nesting and complexity by using early returns](#avoid-nesting-and-complexity-by-using-early-returns)
- [Use the same coding style throughout a file](#use-the-same-coding-style-throughout-a-file)
- [Avoid repetition of hardcoded values](#avoid-repetition-of-hardcoded-values)
- [Uniform import order and naming](#uniform-import-order-and-naming)
- Prefert to inline variable assignments.
    - For example: Inline `err` checks in if-statements.
- Use switch-cases to avoid long if/else if/else statements.
- Isolate code.
    - Use interfaces instead of bare structs.
- Avoid use of global variables.
    - Use structs and receiver methods to keep state.
- Avoid long files.
    - Avoid adding helpers in a single place. Long files like `tests/utils.go`
      are hard to maintain.
    - Alternatively, add helpers to the places where they are used or group them
      in packages/files with representative names.
- Prefer to define variables in the function body
    - Avoid returning too many values from a function.
    - Avoid naked returns.
- Use closures with caution, be aware of the risks and use them only when it
  makes sense.
    - They can still be used if there is a good reason for it.
    - A useful use case of closures is to define a function that is relevant
      only for a very specific scope.
- Declare empty slices with the var syntax.
    - Pay attention when serializing data, see [Declaring Empty Slices](https://go.dev/wiki/CodeReviewComments#declaring-empty-slices).
- Avoid use of `fmt.Sprintf` for manual construction of complex objects
  or operations (e.g. paths or patches).
    - Use helpers or builders when available.
        - Build patches with the `PatchSet` interface.
        - Construct paths with the `path` package.
- Use the `kubevirt.io/kubevirt/pkg/pointer` package when pointers are
  required.
- Keep function signatures lean.
    - E.g. use `kubevirt.Client()` in test functions instead of passing the
      client as an additional function call argument.
    - Again: Use structs and receiver methods to keep state.
- Table-driven tests are preferred for testing matrices of scenarios/inputs.
    - Use Gingko's `DescribeTable` to construct test tables.
- Do not expect an asynchronous thing to happen immediately.
    - For example do not wait for one second and expect a VM to be running.
    - Wait and retry instead, in test code use `Eventually`.
- Avoid `Skip` in tests.
    - Use `decorators` to control in which lanes tests run.
- Namings (packages, interfaces, etc.)
    - Consider the package name when selecting an interface name and avoid
      redundancy.
      - For example, `storage.Interface` is better than `storage.
        StorageInterface`.
    - Consider the parent directory name when choosing a package name.
      - For example, `pkg/controllers/autoscaler/foo.go` should
        say `package autoscaler` not `package autoscalercontroller`.
    - Unless there's a good reason, the `package foo` line should match the
      name of the directory in which the `.go` file exists.
    - Do not use uppercase characters, underscores, or dashes in package names.
    - Command-line flags should use dashes, not underscores.
    - Importers can use a different name if they need to disambiguate.
    - Locks should be called `lock` and should never be embedded (
      always `lock sync.Mutex`).
    - When multiple locks are present, give each lock a distinct name following
      Go conventions: `stateLock`, `mapLock` etc.
- Avoid relying on Docker Hub.
    - Use the [Google Cloud Container Registry](https://gcr.io) instead.

## Directory and file conventions

- Avoid package sprawl. Find an appropriate subdirectory for new packages.
    - Libraries with no appropriate home belong in new package subdirectories
      of `pkg/util`.
- Avoid general utility packages.
    - Packages called `util` are suspect.
    - Instead, derive a name that describes your desired function.
    - For example, the utility functions dealing with VMI creation are in the
      `libvmi` package and include functionality like `New`. The full name is
      `libvmi.New`.
- All filenames should be lowercase.
- Packages should have a maintainable size (not too many files, not too long
  files) and that functions have clear useful names.
- Go source files and directories use underscores, not dashes.
    - Package directories should generally avoid using separators as much as
      possible.
    - When package names are multiple words, they usually should be in nested
      subdirectories.
- Document directories and filenames should use dashes rather than underscores.

# Detailed coding conventions

TODO

# Additional conventions for scripts

TODO

# Good reads

- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go landmines](https://gist.github.com/lavalamp/4bd23295a9f32706a48f)
- [Go's commenting conventions](http://blog.golang.org/godoc-documenting-go-code)
