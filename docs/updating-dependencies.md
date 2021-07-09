# Updating Dependencies

## Updating golang dependencies

Run `make deps-update` to simply update the golang dependencies to their latest
states. If specific changes are needed, first manipulate our main
[go.mod](../go.mod). Dependencies for
[staging/client-go](../staging/src/kubevirt.io/client-go) are located in a
separate [go.mod](../staging/src/kubevirt.io/client-go/go.mod). Changing in the
staging area will be inherited by the main go.mod when running `make
deps-update`.

## Updating RPM test dependencies

We can build our own base images for various architectures with bazel without
the need of machines of that architecture. Out test container base images is
defined at this [BUILD.bazel](../images/BUILD.bazel). If you need to add new RPMs
into the  test base image, you can simply add the RPM package to
[hack/rpm-deps.sh](../hack/rpm-deps.sh) and run `make rpm-deps` afterwards.

`make rpm-deps` can periodically be run to just update to the latest RPM
packages. The resolved RPMs are then added to the [WORKSPACE](../WORKSPACE) and
the `rpmtree` targets in [rpm/BUILD.bazel](../rpm/BUILD.bazel) are updated.
Finally no longer needed RPM definitions are removed from the WORKSPACE.  The
updated `rpmtree` dependencies are the base for the test image containers.

To update the RPM repositories in use, change [repo.yaml](../repo.yaml).

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

More information can be found at [bazeldnf](https://github.com/rmohr/bazeldnf). 

## Updating libvirt and libvirt-devel RPM dependencies

Works the same way like for the RPM test dependencies.

## Verifying RPMs

`bazeldnf` does some initial checks based on sha256. Notably the metalink,
repomd.xml and the packages XML are verified.  These checks happen whenever
`make rpm-deps` is run. However, since we have no guarantee to still have the
same RPMs available on subsequent runs, it is hard to check based on this the
validity of the content in CI. RPM repos use therefore GPG signing to verify
the origin of the content.

Therefore, local and CI verification based on gpg keys can be performend by
executing the `make verify-rpm-deps` command.

## Onboarding new architectures

* Create architecture specific entries in [repo.yaml](../repo.yaml) and
[hack/rpm-deps.sh](../hack/rpm-deps.sh).
* Adjust the select clauses on all container entries to choose the right
  target architecture and the right base image.
* Add architecture specific entries to [.bazelrc](../.bazelrc)

For x86_64 libvirt-devel dependencies exist for linking and unit-testing.
Updating or adding such targets for other architectures is only necessary if
the unit tests are supposed to be executed on the target platform. Otherwise it
is sufficient to only create images for the target-platform with libvirt
dependencies installed.
