# Integration tests

Kubevirt integration tests are a suite of end-to-end tests which run
against an active Kubevirt cluster.

The tests may execute against various clusters, from integrated kubevirtci
deployments and up to external kubernetes or OpenShift ones.

## Kubevirtci

[Kubevirtci](https://github.com/kubevirt/kubevirtci/)
provides containerized Kubernetes clusters which can be used
to test KubeVirt and similar projects.

KubeVirt integration tests are being run against kubevirtci clusters on CI
and is the simplest way to run the tests locally in a development environment.

To run the test locally:
- Configure the desired cluster setup (provider, number of nodes and nics).

   > **Note:** Some tests may run only on specific configuration and hardware.

   > **Note:** For detailed instructions on how to work with a specific provider,
        please explore kubevirtci providers, e.g.
  [k8s-1.18](https://github.com/kubevirt/kubevirtci/tree/master/cluster-up/cluster/k8s-1.18)
  
  The following configuration is of a k8s-1.18 provider with two nodes, each
  having two additional interfaces (3 in total).
```
export KUBEVIRT_PROVIDER=k8s-1.18
export KUBEVIRT_NUM_NODES=2
export KUBEVIRT_NUM_SECONDARY_NICS=2
```

- Raise a cluster based on the configuration.
  At this stage, it will not contain kubevirt.
  ```
  make cluster-up
  ```
- Build KubeVirt from sources.
  ```
  make
  ```
- Deploy KubeVirt onto the cluster.
  ```
  make cluster-sync
  ```
- Run the tests.
  ```
  make functest
  ```
- Take down and delete the cluster.
  ```
  make cluster-down
  ```

On a regular workflow of development or debugging, rebuilding KubeVirt is
needed only when the production code is changed.
In case only test code changes, `make functest` will rebuild the test code
implicitly.

## Existing (external) Kubevirt Cluster

The tests may be executed against an already deployed (and active) Kubevirt
cluster.

The environment needs to be adjusted to allow the tests to operate against
such a cluster.
The following high level steps are required to accomplish this:
- Clone kubevirt repository on the machine that has access to the cluster.
- Identify the deployed kubevirt version.
- From the git repo, checkout to the commit that correlates to the running
kubevirt cluster.
- Run the tests by executing the
  [./tests/run-tests-on-external-cluster.sh](./run-tests-on-external-cluster.sh)
  script.

> **Note:** In case the external cluster uses dynamic storage provisioner
such as hostpath, then there is a need to provide a specific custom storage
configuration through a configuration file.
This is currently not covered by this document and the script helper.

### Identify the commit on which the deployed kubevirt cluster is based on

In order to identify on which commit the current kubevirt cluster is based on,
there is a need to query a kubevirt infra pod and then inspect its base
container.

- Lookup for the `virt-operator` pod:
  `kubectl get pods --all-namespaces | grep "virt-operator"`
- Use the full pod name and its namespace to inspect the container it uses:
  `kubectl describe <virt-operator full name> -n <virt-operator namespace>`
- Identify and record the pod image:
  ```
  Containers:
  virt-operator:
    Image:         <reference to image on the registry>
  ```
- Inspect the image to retrieve the commit:
  `skopeo inspect docker://<image path>`
  The commit ID is set on the `upstream-vcs-ref`.
- Use git-checkout to sync the sources to the relevant commit ID.
  `git checkout <commit id>`

At this stage, the tests sources should be in sync with the running cluster
and it is safe to setup the configuration and run the tests.

### Run the tests

The test environment needs to be configured to work with the running cluster
and the test can then be build and executed against it.

These steps have been automated by the
[run-tests-on-external-cluster.sh](./run-tests-on-external-cluster.sh) script.

Note that the namespace in which kubevirt components run on the cluster
needs to be specified.

```
KUBEVIRTNAMESPACE="openshift-cnv" ./tests/run-tests-on-external-cluster.sh
```

## Run them on an arbitrary KubeVirt installation

```
cd tests # from the git repo root folder
go test -kubeconfig=path/to/my/config -config=default-config.json
```

## Run them anywhere inside of container

```
# Create directory for data / results / kubectl binary
mkdir -p /tmp/kubevirt-tests-data
# Make sure that eveybody can write there
setfacl -m d:o:rwx /tmp/kubevirt-tests-data
setfacl -m o:rwx /tmp/kubevirt-tests-data

docker run \
    -v /tmp/kubevirt-tests-data:/home/kubevirt-tests/data:rw,z --rm \
    kubevirt/tests:latest \
        --kubeconfig=data/openshift-master.kubeconfig \
        --container-tag=latest \
        --container-prefix=docker.io/kubevirt \
        --test.timeout 180m \
        --junit-output=data/results/junit.xml \
        --deploy-testing-infra \
        --path-to-testing-infra-manifests=data/manifests
```
