# Container Image for Functional Tests

This repository ships a container image to run functional tests against installed systems. 
The up-to-date image is pushed to `quay.io/kubevirt/hyperconverged-cluster-functest:<csv-version>-unstable` after every merged change.

## How to run

### Locally

* Using `docker run`
    ```shell
     docker run --env KUBECONFIG=/tmp/conf \
      -v $KUBECONFIG:/tmp/conf \
      quay.io/kubevirt/hyperconverged-cluster-functest:1.10.0-unstable 
    ```
    
    > If the cluster is running locally, you may need to pass `--network host` so that clients in the docker image can access the cluster.

* Using `make functest`
    
    If you have created a development cluster using `make cluster-up` (that is, using Kubevirt CI), you can run the 
    functional test with:
    
    ```shell
    KUBECONFIG=_kubevirtci/_ci-configs/k8s-1.26-centos9/.kubeconfig make functest
    ```

### In cluster
```shell
kubectl create clusterrolebinding func-cluster-admin --clusterrole=cluster-admin --serviceaccount=kubevirt-hyperconverged:functest
kubectl create serviceaccount functest
kubectl run functest --image=quay.io/kubevirt/hyperconverged-cluster-functest:1.10.0-unstable --serviceaccount=functest
```


## Requirements
- If you are running locally, you have to set `KUBECONFIG` variable and mount kubeconfig into that path.
- If your systems are running in a namespace different from `kubevirt-hypercongerged`, you have to set the environment variable `INSTALLED_NAMESPACE` 


## Arguments and Configuration

The arguments passed to this container image are directly passed to `func-tests.test` binary in the image. See [the example](#example-command)

To provide test configuration, use `--config-file` flag and mount your configuration file into that path. 
```shell
docker run \
  -v /put-your-configuration-file-here.yaml:/tmp/testconf \
  quay.io/kubevirt/hyperconverged-cluster-functest:1.10.0-unstable \
  --config-file /tmp/testconf    
```

The content of the configuration file must be a valid yaml representation of `TestConfig` in [config.go](../tests/func-tests/config.go)

***Configuration File Example***
```yaml
quickStart:
  testItems:
    - name: test-quick-start
      displayName: Test Quickstart Tour
```

## Example Command

```shell
docker run  --network=host  \
    --env INSTALLED_NAMESPACE=openshift-cnv \
    --env KUBECONFIG=/tmp/kubeconf \
    -v $KUBECONFIG:/tmp/kubeconf \
    -v $MY_TEST_CONF:/tmp/testconf \
    -v /tmp/out:/tmp/out  \
    quay.io/kubevirt/hyperconverged-cluster-functest:1.10.0-unstable   \
    --polarion-execution=true \
    --polarion-project-id=CNV \
    --polarion-report-file=/tmp/out/polarion_results.xml \
    --test-suite-params="env_tier=tier1" \
    --junit-output=/tmp/out/junit.xml \
    --polarion-custom-plannedin=234235 \
    --config-file /tmp/testconf
```

