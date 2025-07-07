# PerfScale Load Generator

The load generator is a tool aimed at stressing the Kubernetes and KubeVirt control plane by creating several objects (e.g., VM, VMI, and VMIReplicaSet). The main functionality it provides can be summarized as follows:
- Create the objects declared in a workload's description
- Watch for the cluster to have the expected number of object
- Wait for objects to reach their desired state

This tool introduces load into the system and the relevant metrics and results can be collected using the [perfscale-audit tool](https://github.com/kubevirt/kubevirt/tree/main/tools/perfscale-audit).

## CLI
When running the benchmark, you must configure `KUBECONFIG` environment variable of providing the absolute path to the kubeconfig file in the parameter `-kubeconfig`.

The tool runs the default workload example, but you should provide them to the file containing the workload configuration in the parameter `-workload`.

Deleting a workload requires an explicit request by the user.  Run the same command used to create the workload but with the `-delete` flag.
```
  -delete
        Delete a workload
  -kubeconfig string
        absolute path to the kubeconfig file
  -master string
        kubernetes master url
  -verbose int
        log level for V logs (default 2)
  -workload string
        path to the file containing the worload configuration (default "tools/perfscale-load-generator/examples/workload/kubevirt-density/kubevirt-density.yaml")
```

## Workload
There is an example of the workload configuration in [kubevirt-density.yaml](tools/perfscale-load-generator/examples/workload/kubevirt-density/kubevirt-density.yaml)

The workload configuration parameters are defined as follows:
```
name: kubevirt-burst-test
// How long a job will run.  steady-state jobs it will always run until the timesout
timeout: 5m
// Number of objects to manage
count: 5
// Test type - burst or steady-state
type: "burst"
// Object to manage
object:
  templateFile: vmi-ephemeral.yaml
  inputVars:
    containerPrefix: quay.io/kubevirt
    containerImg: cirros-container-disk-demo
    containerTag: ""
    namespace: default
```

#### Workload Types
The **Burst Test** tests how quickly a system can move between different capacities.
A good example of a Burst workload is a sudden spike in demand for compute resources
to handle a massive increase of users.

The **Steady State Test** tests how quickly a system can move between
on how well the system can maintain max capacity or near max capacity.
For example, a system may create a certain number of warm resources
