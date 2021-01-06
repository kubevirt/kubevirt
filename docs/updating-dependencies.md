# Updating Dependencies

## Updating golang dependencies

Run `make deps-update` to simply update the golang dependencies to their latest
states. If specific changes are needed, first manipulate our main
[go.mod](go.mod). Dependencies for
[staging/client-go](staging/src/kubevirt.io/client-go) are located in a
separate [go.mod](staging/src/kubevirt.io/client-go/go.mod). Changing in the
staging area will be inherited by the main go.mod when running `make
deps-update`.

## Updating RPM test dependencies

We can build our own base images for various architectures with bazel without
the need of machines of that architecture. Out test container base images is
defined at this [BUILD.bazel](images/BUILD.bazel). If you need to add new RPMs
into the  test base image, you can simply add the RPM package to
[hack/rpm-deps.sh](hack/rpm-deps.sh) and run `make rpm-deps` afterwards.

`make rpm-deps` can periodically be run to just update to the latest RPM
packages. The resolved RPMs are then added to the [WORKSPACE](WORKSPACE) and
the `rpmtree` targets in [rpm/BUILD.bazel](rpm/BUILD.bazel) are updated.
Finally no longer needed RPM definitions are removed from the WORKSPACE.  The
updated `rpmtree` dependencies are the base for the test image containers.

To update the RPM repositories in use, change [repo.yaml](repo.yaml).

This is an example entry for Fedora 32 on `ppc64le`:

```yaml
- arch: ppc64le
  metalink: https://mirrors.fedoraproject.org/metalink?repo=fedora-32&arch=ppc64le
  name: 32-ppc64le-primary-repo
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

## Updating libvirt-devel RPM dependencies

This is at this moment still a manual task where one manually updates the
dependencies defined in the [WORKSPACE](WORKSPACE).
