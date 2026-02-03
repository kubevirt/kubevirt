# Updating Dependencies

## Updating golang dependencies

Run `make deps-update` to simply update the golang dependencies to their latest
states. If specific changes are needed, first manipulate our main
[go.mod](../go.mod). Dependencies for
[staging/client-go](../staging/src/kubevirt.io/client-go) are located in a
separate [go.mod](../staging/src/kubevirt.io/client-go/go.mod). Changing in the
staging area will be inherited by the main go.mod when running `make
deps-update`.
To update k8s dependencies please follow [update-k8s-dependencies](update-k8s-dependencies.md)

## Updating RPM dependencies

We can build container images for various architectures using native tools.
RPM dependencies are managed through JSON lock files in [rpm-lockfiles/](../rpm-lockfiles/).

If you need to add new RPMs, modify [hack/rpm-packages.sh](../hack/rpm-packages.sh) 
and run `make rpm-deps` afterwards.

`make rpm-deps` can periodically be run to just update to the latest RPM
packages. The resolved RPMs are saved to JSON lock files in the `rpm-lockfiles/`
directory. These lock files contain the exact package versions and SHA256
checksums for reproducible builds.

To update the RPM repositories in use, change [rpm/repo.yaml](../rpm/repo.yaml).

This is an example entry for Fedora 32 on `aarch64`:

```yaml
- arch: aarch64
  metalink: https://mirrors.fedoraproject.org/metalink?repo=fedora-32&arch=aarch64
  name: 32-aarch64-primary-repo
```

Here the corresponding entry for `x86_64`:

```yaml
- arch: x86_64
  metalink: https://mirrors.fedoraproject.org/metalink?repo=fedora-32&arch=x86_64
  name: 32-x86_64-primary-repo
```

Arbitrary RPM repos can be used too. Demonstrated here by referencing a Fedora
COPR repo:

```yaml
- arch: x86_64
  baseurl: https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-6.6.0-8.el8/fedora-32-x86_64/
  name: kubevirt/libvirt-copr-x86_64
```

## Updating libvirt and libvirt-devel RPM dependencies

Works the same way like for the RPM test dependencies.

## Verifying RPMs

The native RPM freeze tool performs SHA256 verification on all downloaded packages.
These checksums are stored in the JSON lock files and verified during container
image builds.

Local and CI verification can be performed by executing the `make verify-rpm-deps` command.

## Onboarding new architectures

* Create architecture specific entries in [rpm/repo.yaml](../rpm/repo.yaml) and
[hack/rpm-packages.sh](../hack/rpm-packages.sh).
* Add architecture-specific Containerfiles in the [build/](../build/) directory.
* Run `make rpm-deps` to generate lock files for the new architecture.

For x86_64 libvirt-devel dependencies exist for linking and unit-testing.
Updating or adding such targets for other architectures is only necessary if
the unit tests are supposed to be executed on the target platform. Otherwise it
is sufficient to only create images for the target-platform with libvirt
dependencies installed.
