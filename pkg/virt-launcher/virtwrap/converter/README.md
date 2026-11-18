# converter

This package converts a `v1.VirtualMachineInstance` into an `api.Domain` (the
libvirt domain representation that virt-launcher feeds to libvirt).

## Entry point

`Convert_v1_VirtualMachineInstance_To_api_Domain` (in `converter.go`) is the
single entry point of the package. Its core job is to:

1. Assemble the list of configurators that build the domain.
2. Run them through the `DomainBuilder`.

It then runs a small set of whole-domain passes over the assembled result (see
[Whole-domain post-processing](#whole-domain-post-processing)).

**Please keep this function thin.** Wiring a new configurator into the list is
expected; adding per-device or per-feature conversion logic to the function body
is not. Expand it only when there is no other option.

## How conversion is structured

Conversion is split into small, focused units that each implement the
`Configurator` interface (see `types/builder.go`):

```go
type Configurator interface {
    Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error
}
```

Each implementation (named `<Concern>DomainConfigurator`, e.g.
`HostDeviceDomainConfigurator`) reads from the VMI and mutates the `api.Domain`
for one concern: networking, storage disks, CPU, host devices, and so on. The
`DomainBuilder` simply runs them in order.

Configurators must be orthogonal: each owns a distinct part of the domain and
must not depend on the order in which the builder runs them, nor on another
configurator having run first. The builder's ordering is an implementation
detail, not a contract. Logic that genuinely needs the fully assembled domain or
a cross-device view does not belong in a configurator (see
[Whole-domain post-processing](#whole-domain-post-processing)).

These configurators live in subpackages grouped by concern, for example:

- `compute/` - compute devices and domain features
- `storage/` - storage disks and related devices
- `network/` - network interfaces

Where a subpackage defines an `OWNERS` file (e.g. `network/`, `storage/`), it
names the SIG responsible for reviewing changes there.

## Making a change

To change or add domain conversion behavior, modify the relevant
`<Concern>DomainConfigurator`, or add a new one in the appropriate concern's
subpackage and wire it into the list in
`Convert_v1_VirtualMachineInstance_To_api_Domain`.

This keeps the entry point readable, keeps each concern reviewable by its owning
SIG, and keeps conversion logic testable in isolation.

## Whole-domain post-processing

After the builder runs, `Convert_v1_VirtualMachineInstance_To_api_Domain`
performs a few passes that operate on the fully assembled domain, for example
CPU/topology pinning and PCI device placement (NUMA alignment, root complex).

These are deliberately not configurators: they need to see the complete set of
devices at once, which would violate the orthogonality and order-independence
rules above. This is the one case where logic legitimately lives outside a
configurator. Keep it to transforms that truly require the whole domain;
single-concern logic still belongs in a configurator.

## Testing

Each `DomainConfigurator` is unit tested next to its implementation, in its own
subpackage. That is where single-configurator behavior belongs.

Add a test to `converter_test.go` only when it exercises a combination of two or
more configurators together. A test that only exercises a single concern is a
better fit in that concern's configurator subpackage, and existing ones should be
moved there over time.

Note that `converter_test.go` currently lives in the production `converter`
package rather than a `converter_test` package, so it can reach unexported
internals. Do not use that access to test implementation details: keep these
tests to observable behavior of configurator combinations.
