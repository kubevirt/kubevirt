# Perscale Load Generator

The load generator is a tool aimed at stressing kubernetes and kubevirt control plane by creating several VMIs in parallel. The main functionallity it provides can be summarized as follows:
- Create/delete the VMIs declared in a scenario.
- Create VMIs in one namespaces, or one VMI per namespace.
- Wait VMIs be created in each iteration.
- Wait VMIs be deleted in each iteration.

This tool introduces load into the system and the relevant metrics and results can be collected using the [perfscale-audit tool](https://github.com/kubevirt/kubevirt/tree/main/tools/perfscale-audit).

## CLI
```
Usage of perfscale-load-generator:
  -burst int
    	maximum burst for throttle the VMI creation (default 20)
  -img-prefix string
    	Set the repository prefix for all images (default "quay.io/kubevirt")
  -img-tag string
    	Set the image tag or digest to use (default "latest")
  -iteration-cleanup
    	clean up old tests, delete all created VMIs and namespaces before moving forward to the next iteration (default true)
  -iteration-count int
    	how many times to execute the scenario (default 1)
  -iteration-interval duration
    	how much time to wait between each scenario iteration
  -iteration-vmi-wait
    	wait for all vmis to be running before moving forward to the next iteration (default true)
  -iteration-wait-for-deletion
    	wait for VMIs to be deleted and all objects disapear in each iteration (default true)
  -kubeconfig string
    	absolute path to the kubeconfig file
  -master string
    	kubernetes master url
  -max-wait-timeout duration
    	maximum wait period (default 5m0s)
  -name string
    	scenario name (default "kubevirt-test-default")
  -namespace string
    	namespace base name to use (default "kubevirt-test-default")
  -namespaced-iterations
    	create a namespace per scenario iteration
  -qps float
    	number of queries per second for VMI creation (default 20)
  -uuid string
    	scenario uuid (default "26a02ef8-f5d8-11eb-ac46-061526969e47")
  -v int
    	log level for V logs (default 2)
  -vmi-count int
    	total number of VMs to be created (default 100)
  -vmi-cpu-limit string
    	vmi CPU request and limit (1 CPU = 1000m) (default "100m")
  -vmi-img string
    	vmi image name (cirros, alpine, fedora-cloud) (default "cirros")
  -vmi-mem-limit string
    	vmi memory request and limit (MEM overhead ~ +170Mi) (default "90Mi")
  -wait-when-finished duration
    	delays the termination of the scenario (default 30s)
```

## Comments

It is needed to tweak QPS/Burst and maxWaitTimeout parameters according to the cluster size and VMICount.
> If you set QPS/Burst=0 the kubeAPI and virtAPI get too overloaded and quay.io pull requests exceedes the max QPS

All VMIs are created with the Kubernetes emphemeral disk [emptyDir](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir).