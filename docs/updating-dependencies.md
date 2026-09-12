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

## Reproducible and Idempotent RPM Resolution

To ensure reproducible RPM dependency resolution across developers, CI runs, and mirror caches:

1. **CentOS Stream Compose Snapshots**:
   By default, `hack/rpm-deps.sh` uses repository metadata from CentOS Stream. To avoid non-deterministic drift caused by rolling mirror changes, you can pin repository definitions to an immutable production compose snapshot:
   ```bash
   # Pin both CS9 and CS10 to specific composes:
   make CENTOS_STREAM_COMPOSE="latest-CentOS-Stream" rpm-deps-all
   # Or pin a specific compose release:
   make CENTOS_STREAM_9_COMPOSE="CentOS-Stream-9-20260908.0" rpm-deps-cs9
   ```

2. **Solver Best-Candidate Selection**:
   By default, `hack/rpm-deps.sh` omits `--nobest` during `bazeldnf rpmtree` calls, enforcing that libsolv chooses the highest available candidate version instead of non-deterministic permutations. If loose fallback resolution is needed, set `BAZELDNF_NOBEST=true`.

3. **Anchored Transitive Base Libraries**:
   Shared base libraries (`glib2`, `gnutls`, `systemd-libs`, `libcap-ng`) are included in `centos_extra` so they are anchored across all container images. You can also override their versions explicitly:
   ```bash
   make GLIB2_VERSION="0:2.68.4-29.el9" rpm-deps-cs9
   ```

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
* Running `make rpm-deps` requires a sandbox enviroment, which is also updated from the previous command. You need to either run the command on an already onboarded architecture or updating the sandbox manually, by sourcing the variables from `hack/rpm-deps.sh` and running the bazeldnf command.
```
bazeldnf rpmtree \
        --public --nobest \
        --name sandboxroot_s390x --arch s390x \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $sandboxroot_main
```

For x86_64 libvirt-devel dependencies exist for linking and unit-testing.
Updating or adding such targets for other architectures is only necessary if
the unit tests are supposed to be executed on the target platform. Otherwise it
is sufficient to only create images for the target-platform with libvirt
dependencies installed.
