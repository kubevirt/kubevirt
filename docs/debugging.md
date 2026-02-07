# Debugging

```bash
kubevirtci/cluster-up/kubectl.sh version
```

will try to connect to the apiserver.

## Retrieving Logs

To investigate the logs of a pod, you can view the logs via
`kubevirtci/cluster-up/kubectl.sh logs`. To view the logs of `virt-api`, type

```bash
kubevirtci/cluster-up/kubectl.sh logs virt-api -f
```

Sometimes a container in a pod is crashlooping because of an application error
inside it. In this case, you normally can't see any logs, because the container
is already gone, and so are the logs. To get the logs from the last run
attempt, the `--previous` flag can be used. To view the logs of the container
`virt-api` in the pod `virt-api` from the previous run, type

```bash
kubevirtci/cluster-up/kubectl.sh logs virt-api -f -c virt-api -p
```

Note that you always have to select a container inside a pod for fetching old
logs with the `--previous` flag.

## Watching Events

Both, Kubernetes and KubeVirt are creating events, which can be viewed via

```bash
kubevirtci/cluster-up/kubectl.sh get events --all-namespaces --watch
```

This way it is pretty easy to detect if a Pod or a VMI got started.

## Entering Containers

It can be very valuable to enter a container and do some investigations there,
to see what is going wrong. In this case the kubectl `exec` command can be
used. To enter `virt-api` with an interactive shell, type

```bash
kubevirtci/cluster-up/kubectl.sh exec virt-api -c virt-api -i -t -- sh
```

## Kubelet Logs

After all you might not see errors in the logs provided by Kubernetes. In that case
you can take a look at the logs of the `kubelet` on the host where the issue is
appearing. Depending on the error it is getting logged to either the system logs or
to the kubelet logs, you can use the following commands to view them:

```bash
journalctl
# or
journalctl -u kubelet
```

## References

 - [kubectl overview](https://kubernetes.io/docs/reference/kubectl/overview/)
 - [kubectl reference](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands)

# Using a Debugger (delve)

This shows the basic principle on how remote debugging can be done.

 - Add delve to the container
 - Start delve on a specific port ( `dlv attach <pid> --headless=true --api-version=2 --listen=:1234`)
 - Use kube-proxy to forward the port to your machine

A go debugger can be attached to kubevirt processes in the following manners:
## Local execution of the kubevirt process
Not all processes can easily be executed locally, since some of the processes require specific 
dependencies that exist in the runtime pod and node
For processes that are easily executed locally, such as `virt-controler`, the following program arguments
can be passed to virt-controller `--kubeconfig /path/to/kubeconfig --leader-elect-lease-duration 99h` in order 
to successfully debug. Since virt-controller and other control plane components rely on leader election to achieve
consensus among multiple replicas, and we only want our local replica to be the leader, even in the absence of other 
replicas, we can temporarily grant control to our local replica by specifying the `--leader-elect-lease-duration` 
argument, as shown above, with an example value of `99h`, granting the lease for 99 hours.    
> **Note** `/path/to/kubeconfig` must point to a running kubernetes cluster with kubevirt installed
> 
> **Note** It is recommended to scale down both `virt-controller` and also `virt-operator` deployments to
0 replicas, because otherwise it will override the `virt-controller` deployment manifest.
## Remote debugging running pods
Remote debugging may be more complex to setup, but has the advantage of executing the procesess in its
real runtime environment, where all dependencies are satisfied, therefore it can work for any go process 
### Step 1 - recompile go code, build and publish container image
Compile the required go [cmd](https://github.com/kubevirt/kubevirt/tree/main/cmd) with `-gcflags='all=-N -l'` to produce 
a debuggable binary.  
Validate that it is in fact debuggable by running `file <binary file>` and expect to see `with debug_info, not stripped`
in the output.
For `Bazel` based build, add `--@io_bazel_rules_go//go/config:gc_goopts=-N,-l --strip=never` to the bazel 
build command.
Build a container image, with the compiled binary and publish to a registry that is accessible to the cluster
consider adding `dlv` executable to the image for step 2
### Step 2 - modify workload manifest to consume debug container image and execute delve
Having scaled down virt-operator, patch or edit the respective k8s workload manifest 
(e.g. `virt-controller` Deployment, `virt-handler` Daemonset ) to:
- Consume the debug image
- Execute `dlv`, either in a separate container, and attach to the debugged process as shown above.
  [kubectl debug](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_debug/) can be used, to easily
  add another container to the pod. 
  Note that `shareProcessNamespace: true` is necessary, for dlv to be able to debug a process running in another container.
  The other alternative is to modify the app container `cmd` to have `dlv` `exec` the process in the same container:
  `dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec <binary>`
- Turn off probes, tweak securityContext and anything else that could interfere with slow execution

Here's a working example of a `virt-controller` deployment patch 

```bash
cluster-up/kubectl.sh --namespace kubevirt patch deployment virt-controller --type='json' -p '[
   {
      "op":"add",
      "path":"/spec/template/spec/containers/-",
      "value":{
         "name":"dlv-debugger",
         "image":"golang:1.23-alpine",
         "command":[
            "sh",
            "-c",
            "apk add --no-cache git bash && go install github.com/go-delve/delve/cmd/dlv@latest && \\
dlv attach $(pgrep virt-controller) --headless --accept-multiclient --api-version 2 --listen=:2345"
         ],
         "securityContext":{
            "seccompProfile":{
               "type":"Unconfined"
            },
            "runAsUser":0
         }
      }
   },
   {
      "op":"add",
      "path":"/spec/template/spec/shareProcessNamespace",
      "value":true
   },
   {
      "op":"replace",
      "path":"/spec/template/spec/containers/0/image",
      "value":"registry:5000/kubevirt/virt-controller:debug"
   },
   {
      "op":"replace",
      "path":"/spec/replicas",
      "value":1
   },
   {
      "op":"replace",
      "path":"/spec/template/spec/containers/0/imagePullPolicy",
      "value":"Always"
   },
   {
      "op":"replace",
      "path":"/spec/template/spec/containers/0/securityContext",
      "value":{
         "runAsUser":0
      }
   },
   {
      "op":"replace",
      "path":"/spec/template/spec/securityContext",
      "value":{
         
      }
   },
   {
      "op":"remove",
      "path":"/spec/template/spec/containers/0/readinessProbe"
   },
   {
      "op":"remove",
      "path":"/spec/template/spec/containers/0/livenessProbe"
   }
]'
```

### Step 3 - port-forward a local port to the target pod designated debug port
`kubectl port-forward <pod> 2345`

### Step 4 - connect to localhost:2345 with your debugger

