## Getting Started For Developers

* [Download CDI](#download-cdi)
    * [Lint, Test, Build](#lint-test-build)
        * [Make Targets](#make-targets)
        * [Make Variables](#make-variables)
        * [Execute Standard Environment Functional Tests](#execute-standard-environment-functional-tests)
        * [Execute Alternative Environment Functional Tests](#execute-alternative-environment-functional-tests)
        * [Submit PRs](#submit-prs)
        * [Releases](#releases)
        * [Vendoring Dependencies](#vendoring-dependencies)
        * [S3 Compatible Client Setup](#s3-compatible-client-setup)
            * [AWS S3 CLI](#aws-s3-cli)
            * [Minio CLI](#minio-cli)

### Download CDI

To download the source directly, simply

`$ go get -u kubevirt.io/containerized-data-importer`

### Lint, Test, Build

GnuMake is used to drive a set of scripts that handle linting, testing, compiling, and containerizing.  Executing the scripts directly is not supported at present.

    NOTE: Standard builds require a running Docker daemon!

The standard workflow is performed inside a helper container to normalize the build and test environment for all devs.  Building in the host environment is supported by the Makefile, but is not recommended.

    Docker builds may be disabled by setting DOCKER=0; e.g.
    $ make all DOCKER=0

`$ make all` executes the full workflow.  For granular control of the workflow, several Make targets are defined:

#### Make Targets

- `all`: cleans up previous build artifacts, compiles all CDI packages and builds containers
- `clean`: cleans up previous build artifacts
- `build`: compile all CDI binary artifacts and generate controller manifest
    - `build-controller`: compile cdi-controller binary
    - `build-importer`: compile cdi-importer binary
    - No `build-cloner` target exists as the code is written in bash
- `test`: execute all tests (_NOTE:_ `WHAT` is expected to match the go cli pattern for paths e.g. `./pkg/...`.  This differs slightly from rest of the `make` targets)
    - `test-unit`: execute all tests under `./pkg/...`
    - `test-functional`: execute functional tests under `./tests/...`. Additional test flags can be passed to the test binary via the TEST_ARGS variable, see below for an example and restrictions.
    - `test-lint` runs `gofmt` and `golint` tests against src files
- `build-functest-image-init`: build the init container for the testing file server. (NOTE: the http and s3 components contain no CDI code, so do no require a build)
- `build-functest-image-http` build the http container for the testing file server
- `docker`: compile all binaries and build all containerized
    - `docker-controller`: compile cdi-controller and build cdi-controller image
    - `docker-importer`: compile cdi-importer and build cdi-importer image
    - `docker-cloner`: build the cdi-cloner image (cloner is driven by a shell script, not a binary)
    - `docker-functest-image`: Compile and build the file host image for functional tests
        - `docker-functest-image-init`: Compile and build the file host init image for functional tests
        - `docker-functest-image-http`: Only build the file host http container for functional tests
        - Note: there is no target for the S3 container, an offical Minio container is used instead
- `manifests`: Generate a cdi-controller manifest in `manifests/generated/`.  Accepts [make variables](#make-variables) DOCKER_TAG, DOCKER_REPO, VERBOSITY, and PULL_POLICY
- `push`: compiles, builds, and pushes to the repo passed in `DOCKER_REPO=<my repo>`
    - `push-controller`: compile, build, and push cdi-controller
    - `push-importer`: compile, build, and push cdi-importer
    - `push-cloner`: compile, build, and push cdi-cloner
- `vet`: lint all CDI packages
- `format`: Execute `shfmt`, `goimports`, and `go vet` on all CDI packages.  Writes back to the source files.
- `publish`: CI ONLY - this recipe is not intended for use by developers
- `cluster-up`: Start a default Kubernetes or Open Shift cluster. set KUBEVIRT_PROVIDER environment variable to either 'k8s-1.11.0' or 'os-3.11.0' to select the type of cluster. set KUBEVIRT_NUM_NODES to something higher than 1 to have more than one node.
- `cluster-down`: Stop the cluster, doing a make cluster-down && make cluster-up will basically restart the cluster into an empty fresh state.
- `cluster-sync`: Builds the controller/importer/cloner, and pushes it into a running cluster. The cluster must be up before running a cluster sync. Also generates a manifest and applies it to the running cluster after pushing the images to it.
- `release-description`: Generate a release announcement detailing changes between 2 commits (typically tags).  Expects `RELREF` and `PREREF` to be set
    -  e.g. `$ make release-description RELREF=v1.1.1 PREREF=v1.1.1-alpha.1`

#### Make Variables

Several variables are provided to alter the targets of the above `Makefile` recipes.

These may be passed to a target as `$ make VARIABLE=value target`

- `WHAT`:  The path from the repository root to a target directory (e.g. `make test WHAT=pkg/importer`)
- `DOCKER_REPO`: (default: kubevirt) Set repo globally for image and manifest creation
- `DOCKER_TAG`: (default: latest) Set global version tags for image and manifest creation
- `VERBOSITY`: (default: 1) Set global log level verbosity
- `PULL_POLICY`: (default: IfNotPresent) Set global CDI pull policy
- `TEST_ARGS`: A variable containing a list of additional ginkgo flags to be passed to functional tests. The string "--test-args=" must prefix the variable value. For example:

             `make TEST_ARGS="--test-args=-ginkgo.noColor=true" test-functional >& foo`.

  Note: the following extra flags are not supported in TEST_ARGS: -master, -cdi-namespace, -kubeconfig, -kubectl-path
since these flags are overridden by the _hack/build/run-functional-tests.sh_ script.
To change the default settings for these values the KUBE_MASTER_URL, CDI_NAMESPACE, KUBECONFIG, and KUBECTL variables, respectively, must be set.
- `RELREF`: Required by `release-description`. Must be a commit or tag.  Should be the more recent than `PREREF`
- `PREREF`: Required by `release-description`. Must also be a commit or tag.  Should be the later than `RELREF`

#### Execute Standard Environment Functional Tests

If using a standard bare-metal/local laptop rhel/kvm environment where nested
virtualization is supported then the standard *kubevirtci framework* can be used.

Environment Variables and Supported Values

| Env Variable       | Default       | Additional Values  |
|--------------------|---------------|--------------------|
|KUBEVIRT_PROVIDER   | k8s-1.11.0    | os-3.11.0          |
|NUM_NODES           | 1             | 2-5                |

To Run Standard *cluster-up/kubevirtci* Tests
```
 # make cluster-up
 # make cluster-sync
 # make test-functional
```

To run specific functional tests, you can leverage ginkgo command line options as follows:
```
# make TEST_ARGS="--test-args=-ginkgo.focus=<test_suite_name>" test-functional
```
E.g. to run the tests in transport_test.go:
```
# make TEST_ARGS="--test-args=-ginkgo.focus=Transport" test-functional
```

Clean Up
```
 # make cluster-down
```

#### Execute Alternative Environment Functional Tests

If running in a non-standard environment such as Mac or Cloud where the *kubevirtci framework* is
not supported, then you can use the following example to run Functional Tests.

1. Stand-up a Kubernetes cluster (local-up-cluster.sh/kubeadm/minikube/etc...)

2. Clone or get the kubevirt/containerized-data-importer repo

3. Run the CDI controller manifests

   - To generate latest manifests
   ```
   # make manifests
   ```
   *To customize environment variables see [make targets](#make-targets)*

   - Run the generated latest manfifest
   ```
   # kubectl create -f manifests/generated/cdi-controller.yaml

     serviceaccount/cdi-sa created
     clusterrole.rbac.authorization.k8s.io/cdi created
     clusterrolebinding.rbac.authorization.k8s.io/cdi-sa created
     deployment.apps/cdi-deployment created
     customresourcedefinition.apiextensions.k8s.io/datavolumes.cdi.kubevirt.io created
   ```

4. Run the host-file-server

   - host-file-server is required by the functional tests and provides an
     endpoint server for image files and s3 buckets
   ```
   # make docker-functest-image
   ```

5. Run the tests
```
 # make test-functional
```

6. If you encounter test errors and are following the above steps try:
```
 # make clean && make docker
```
redeploy the manifests above, and re-run the tests.

### Submit PRs

All PRs should originate from forks of kubevirt.io/containerized-data-importer.  Work should not be done directly in the upstream repository.  Open new working branches from master/HEAD of your forked repository and push them to your remote repo.  Then submit PRs of the working branch against the upstream master branch.

### Releases

Release practices are described in the [release doc](/doc/releases.md).

### Vendoring Dependencies

This project uses `glide` as it's dependency manager.  At present, all project dependencies are vendored; using `glide` is unnecessary in the normal work flow.

Install glide:

`curl https://glide.sh/get | sh`

Then run it from the repo root

`glide install -v`

`glide install` scans imports and resolves missing and unused dependencies. `-v` removes nested vendor and Godeps/_workspace directories.

### S3-compatible client setup:

#### AWS S3 cli
$HOME/.aws/credentials
```
[default]
aws_access_key_id = <your-access-key>
aws_secret_access_key = <your-secret>
```

#### Minio cli

$HOME/.mc/config.json:
```
{
        "version": "8",
        "hosts": {
                "s3": {
                        "url": "https://s3.amazonaws.com",
                        "accessKey": "<your-access-key>",
                        "secretKey": "<your-secret>",
                        "api": "S3v4"
                }
        }
}
```
