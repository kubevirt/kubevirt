# PerfScale Load Generator

The load generator is a tool aimed at stressing the Kubernetes and KubeVirt control plane by creating several objects (e.g., VM, VMI, and VMIReplicaSet). The main functionality it provides can be summarized as follows:
- Create the objects declared in a workload description.
- Create VMIs in one namespace or one VMI per namespace.
- Wait for VMIs to be created in each iteration.
- Wait for VMIs to be deleted, after deleting the namespaces in each iteration (i.e., clean up the iteration).

This tool introduces load into the system and the relevant metrics and results can be collected using the [perfscale-audit tool](https://github.com/kubevirt/kubevirt/tree/main/tools/perfscale-audit).

## CLI
When running the benchmark, you must configure `KUBECONFIG` environment variable of providing the absolute path to the kubeconfig file in the parameter `-kubeconfig`.

The tool runs the default workload example, but you should provide them to the file containing the workload configuration in the parameter `-workload`.

## Workload
There is an example of the workload configuration in [kubevirt-density.yaml](tools/perfscale-load-generator/examples/workload/kubevirt-density/kubevirt-density.yaml)

The workload configuration parameters are defined as follows:
```
globalConfig:
    // qps and bust are used to configure the kubernetes API client set, which is used to List, Watch, Create and Delete objects
    // it is needed to tweak QPS/Burst and maxWaitTimeout parameters according to the cluster size and number of created objects
    qps: 0
    burst: 0
// workloads define a list of workloads to be executed
workloads:
  - name: kubevirt-density-10
    // iterationCount defined how many times to execute the workload
    iterationCount: 1
    // iterationInterval defines how much time to wait between each workload iteration
    iterationInterval: 0
    // iterationCreationWait wait for all objects to be running before moving forward to the next iteration
    iterationCreationWait: true
    // iterationCleanup clean up old tests, e.g., namespaces, nodes, configurations, before moving forward to the next iteration
    iterationCleanup: true
    // iterationDeletionWait wait for objects to be deleted in each iteration
    iterationDeletionWait: true
    // create a namespace per workload iteratio
    namespacedIterations: false   
    // maximum wait period for all iterations
    maxWaitTimeout: 30m
    // qps is the max number of queries per second to control the job creation rate
    qps: 20
    // burst is the maximum burst for throttle to control the job creation rate
    burst: 20
    // waitWhenFinished delays the termination of the workload
    waitWhenFinished: 30s
    // objects defines a list of object spec to be created
    objects:
        // templateFile is the relative path to a valid YAML definition of a kubevirt resource
      - templateFile: templates/vmi-ephemeral.yaml
        // replicas is the number of replicas to create of the given object
        replicas: 1
        // InputVars contains a map of arbitrary user-define input variables that can be introduced in the template by users
        // All the variables in the inputVars depend on the dynamic user-define variables in the templateFile
        inputVars:
          // containerPrefix defines the repository prefix for all images
          // if the value is empty (i.e., "") the default value defined in the cmd flag will be used
          containerPrefix: registry:5000/kubevirt/
          // containerImg defines the VMI image in the 
          containerImg: cirros-container-disk-demo
          // containerTag defines the image tag
          // if the value is empty (i.e., "") the default value defined in the cmd flag will be used
          containerTag: devel
          // namespace defines the prefix of the namespace to create the object
          namespace: kubevirt-density
```

## Workflow
- The load generator ranges the list of workloads and sequentially executes each workload.
- Each workload might run one or more times in different iterations
- Each workload iteration might create a list of different objects
- Each workload iteration might wait for all objects to be created
- Each workload iteration might clean up the iteration, deleting all created namespaces and consequently deleting all created objects
- Each workload iteration might sleep X times before each iteration to avoid being affected by a previous execution

## Comments
Note that the containerPrefix can also be defined via command line to make it easier to run the workload in different environments. For example, when creating a cluster using the kubevirtci, the containerPrefix = `registry:5000/kubevirt/`, but then creating the cluster using kubespray the containerPrefix = `localhost:5000/kubevirt/`.
