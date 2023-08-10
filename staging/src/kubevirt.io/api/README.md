# KubeVirt API definitions

Go definitions of the [KubeVirt](https://github.com/kubevirt/kubevirt) API.

## How to use it

Add the dependency to your `go.mod`:

```bash
go get kubevirt.io/api
```

Then generate the client
with [client-gen](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/generating-clientset.md).

To download and install the Kubernetes API client code generator `client-gen`, run:

```
go install k8s.io/code-generator/cmd/client-gen@latest
```

The following command creates the client inside an example project called `testapi`:

```bash
client-gen --input-base="kubevirt.io/api/" --input="core/v1" --output-package="testapi/client" --output-base="../" --clientset-name="versioned" --go-header-file boilerplate.go.txt
```

`client-gen` always needs a `boilerplate.go.txt` file. If you don't want to
include a project specific header to the files just create an empty file.

Then run `go get` to fetch any new introduced missing dependencies.

Finally make use of the client:

```golang
cfg, err := clientcmd.BuildConfigFromFlags("", "")
if err != nil {
	panic(err)
}
client := versioned.NewForConfigOrDie(cfg)
client.KubevirtV1().VirtualMachineInstances(v1.NamespaceAll).List(context.Background(), v1.ListOptions{})
```

-----
KubeVirt API is maintained at https://github.com/kubevirt/kubevirt/tree/main/staging/src/kubevirt.io/api.  
The main branch of this repository is updated on every PR merge, release tags are pushed on every release of KubeVirt.

## License

KubeVirt API is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.txt).

    Copyright 2021

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
