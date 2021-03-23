# Profiling

HCO<sup>[1](#hco-footnote)</sup> has [pprof][1] instrumentation build in, so cpu and memory usage of a running HCO installation can be profiled.

The profiling information can be accessed by setting the environment variable `HCO_PPROF_ADDR` on either the `hco-operator` or `hco-webhook` deployments, which will startup an extra [http endpoint][2] to grap heap and allocation profiles and to run CPU profiling.

> Can I profile my production services?
>
> Yes. It is safe to profile programs in production, but enabling some profiles (e.g. the CPU profile) adds cost. You should expect to see performance downgrade. The performance penalty can be estimated by measuring the overhead of the profiler before turning it on in production.
>
> You may want to periodically profile your production services. Especially in a system with many replicas of a single process, selecting a random replica periodically is a safe option. Select a production process, profile it for X seconds for every Y seconds and save the results for visualization and analysis; then repeat periodically. Results may be manually and/or automatically reviewed to find problems. Collection of profiles can interfere with each other, so it is recommended to collect only a single profile at a time.
> https://golang.org/doc/diagnostics

## Enabling the pprof endpoint

The pprof endpoints can be enabled by setting the environment variable `HCO_PPROF_ADDR` to a valid port + optional address to limit the endpoints exposure.

Example: `HCO_PROF_ADDR=":8070"`

### With OLM

To set the `HCO_PROF_ADDR` environment variable when HCO<sup>[1](#hco-footnote)</sup> is deployed with OLM<sup>[2](#olm-footnote)</sup>, locate the `Subscription` object in the `kubevirt-hyperconverged` Namespace.

```
$ kubectl get subscription -n kubevirt-hyperconverged
NAME                                PACKAGE                             SOURCE                CHANNEL
community-kubevirt-hyperconverged   community-kubevirt-hyperconverged   community-operators   stable
```

Now edit the `Subscription` to add the environment variable:

```
$ kubectl edit subscription -n kubevirt-hyperconverged community-kubevirt-hyperconverged
```

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: community-kubevirt-hyperconverged
  namespace: kubevirt-hyperconverged
spec:
  channel: stable
  config:
    env:                   # <---------------
    - name: HCO_PPROF_ADDR # Add this section
      value: :8070         # <---------------
  installPlanApproval: Manual
  name: community-kubevirt-hyperconverged
  source: community-operators
  sourceNamespace: openshift-marketplace
  startingCSV: kubevirt-hyperconverged-operator.v1.3.0
```

OLM<sup>[2](#olm-footnote)</sup> should now reconcile the `hco-operator` Deployment in the `kubevirt-hyperconverged` Namespace to add this new environment variable.

## Accessing the profiling data

First we have to find out the name of the HCO Pod that we want to profile.
```
$ kubectl get po -n kubevirt-hyperconverged | grep hco-
hco-operator-d8dc95b89-tx5x6                       1/1     Running   0          19m
hco-webhook-cf8b4d457-csqvs                        1/1     Running   0          19m
```

Now we have to make the pprof endpoint available locally so we can access it.  
I want to profile the operator itself, but the hco-webhook also supports profiling, just use a different `Pod` name in the command below.

```
$ kubectl port-forward -n kubevirt-hyperconverged pod/hco-operator-d8dc95b89-tx5x6 8070:8070
```

## Analysing Data

When running `go tool pprof` it drops you into a pprof command prompt.  
The most useful is `top` to show the highest values of the current profile.

Please refer to the [pprof blog post][1] or the [pprof package documentation][3] for details.

### CPU Profile

CPU profiling can be started by supplying the http endpoint directly to pprof.  
By default it will profile for 30s and report the results back to analyse.

```
$ go tool pprof http://localhost:8070
```

### Heap Profile

```
$ curl -sK -v http://localhost:8070/debug/pprof/heap > heap.out
$ go tool pprof heap.out
File: hyperconverged-cluster-operator
Type: inuse_space
Time: Mar 17, 2021 at 11:49am (CET)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) 
```

### Allocs

```
$ curl -sK -v http://localhost:8070/debug/pprof/allocs > allocs.out
$ go tool pprof allocs.out
File: hyperconverged-cluster-operator
Type: alloc_space
Time: Mar 17, 2021 at 11:49am (CET)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) 
```

## Footnotes

<dl>
  <dt id="hco-footnote">HCO</dt>
  <dd>Hyperconverged Cluster Operator</dd>
  <dt id="olm-footnote">OLM</dt>
  <dd>Operator Lifecycle Manager</dd>
</dl>

[1]: https://blog.golang.org/pprof
[2]: https://golang.org/pkg/net/http/pprof/
[3]: https://golang.org/pkg/runtime/pprof/
