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

- Having a consistent coding style thought a file is recommended because it
  makes reviews easier.
- In case a cleaner agreed style is suggested by new code, an attempt should
  be made to adjust the existing code as a follow-up contribution.
- When you encounter code that requires a larger cleanup do it in a separate
  commit or PR.

### Bad example

Consider `checkAB` already exists and `checkBC` is added:

```go
func checkAB(in string) bool {
    return in == "A" || in == "B"
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

Using the same style when adding new code:

```go
func checkAB(in string) bool {
    return in == "A" || in == "B"
}

func checkBC(in string) bool {
    return in == "B" || in == "C"
}
```

# Document values meaning through constants or variable names

- Use constants or variables if you would repeat common values otherwise.
- Prefer encapsulation of common values and the operation on them through
  objects and methods.
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

### Another good example (avoiding exposure of constants)

```go
const levelProfessional = "professional"

type Programmer struct {
    level string
}

func (p *Programmer) setProfessional () {
    p.level = levelProfessional
}

func (p *Programmer) isProfessional () bool {
    return p.level == levelProfessional
}
```

# Uniform import order and naming

- Use the following import order with one block per item:
    - Golang standard libraries
    - Ginkgo / Gomega imports (only in test files)
    - Third-party libraries (attempt to group the packages by domain when
      possible)
    - Local packages
- Use the following naming schemes for imports:
    - `v1` for imports of `kubevirt.io/api/core/v1`
    - `metav1` for imports of `k8s.io/apimachinery/pkg/apis/meta/v1`
    - `k8s` prefix for other `k8s.io` imports on collision with imports
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

    v1 "kubevirt.io/api/core/v1"

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
- [Document values meaning through constants or variable names](#document-values-meaning-through-constants-or-variable-names)
- [Uniform import order and naming](#uniform-import-order-and-naming)
- Be cautious when adding new dependencies.
    - New dependencies should be authored by a trusted and well established
      organization. Avoid packages from personal GitHub accounts.
    - Before adding a new dependency pay attention to additional details like:
        - Is it well maintained? Is there activity in the repository?
        - How many users does this package have?
- Prefer to use initialization statements.
    - For example: Inline `err` checks in if-statements.
    - Another example: Inline expression assignment in switch-statements.
- Use switch-cases to avoid long `if`, `else if` and `else` statements.
- Isolate code by using objects and polymorphism (through interfaces).
    - Avoid use of bare structs.
    - Make sure to use interfaces to express behavior and avoid adding only
      similar methods to objects where common behavior is expected.
    - Define interfaces where they are used and do not define them together
      with a concrete implementation. Interfaces are meant to break the
      coupling between behavior and implementation.
- Avoid use of global variables.
    - Use structs and receiver methods to keep state.
- Avoid long files.
    - Avoid adding helpers in a single place. Long files like `tests/utils.go`
      are hard to maintain.
    - Alternatively, add helpers to the places where they are used or group them
      in packages/files with representative names.
- Avoid returning too many values from a function.
- Prefer to define variables in the function body instead of using named
  return values.
    - Avoid naked returns.
    - There are cases where it makes sense to use named return values, e.g.
      to provide documentation when returning two or more values of the same
      type.
    - However, it should be an exception to need named return values and one
      should try to avoid it because it is a smell.
- Use closures with caution, be aware of the risks and use them only when it
  makes sense.
    - Acceptable use cases are:
        - Defined and used inline, not though variable assignment.
        - Used as actual closures, binding to external variables.
        - Grouping a set of instructions to provide common scope functionality
          (e.g. defer).
    - Exceptions may exist, but they need to be well reasoned for, expressing
      what alternatives have been considered.
- Declare empty slices with the var syntax.
    - Pay attention when serializing data,
      see [Declaring Empty Slices](https://go.dev/wiki/CodeReviewComments#declaring-empty-slices).
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
        - For example, `storage.Reader` is better than `storage.
          ReaderInterface`.
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

## Directory and file conventions

- Avoid package sprawl. Find an appropriate subdirectory for new packages.
    - If no appropriate home can be found consult your reviewers.
- Avoid general utility packages.
    - Packages called `common`, `handler`, `general`, `util`, etc. are suspect.
    - Instead, derive a name that describes your desired function.
    - For example, the utility functions dealing with building a VMI are in the
      `libvmi` package.
- All filenames should be lowercase.
- Packages should have a maintainable size (not too many files, not too long
  files) and that functions have clear useful names.
- Go source files and directories use underscores, not dashes.
    - Package directories should generally avoid using separators as much as
      possible.
    - When package names are multiple words, they usually should be in nested
      subdirectories.
- Document directories and filenames should use dashes rather than underscores.

# Additional conventions (for scripts, etc.)

- Avoid relying on Docker Hub.
    - Use [Quay.io](https://quay.io) or the [Google Cloud Container Registry](https://gcr.io) instead.

# Good reads

- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go landmines](https://gist.github.com/lavalamp/4bd23295a9f32706a48f)
- [Go's commenting conventions](http://blog.golang.org/godoc-documenting-go-code)
- [The Go Interface](https://ehaas.net/blog/the-go-interface)
