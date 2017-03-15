Live migration is the act of moving a virtual machine (VM) from one
hypervisor to another while maintaining the connectivity and liveness
of the applications running on the virtual machine.  It is one of the
primary advantages of using an external VM management service.

Triggering a live migration requires that the user perform an API
call.

To Create (start migration):

```bash

kubectl create -f migration.yaml
```

Here is an example of a  migration request in YAML format:

```yaml

apiVersion: kubevirt.io/v1alpha1
kind: Migration
metadata:
  name: testvm-migration
spec:
  selector:
    name: testvm
```


The selector section indicates which VM objects are to be
migrated. The name field shown here would match a VM with the `name`
value of `testvm.`


To list all migrations:

```bash

kubectl get migrations
```

To query a specific migration named `testvm_migration`:

```bash

kubectl get migrations testvm_migration -o yaml
```

Which produces:

```yaml

apiVersion: kubevirt.io/v1alpha1
kind: Migration
metadata:
  creationTimestamp: 2017-03-15T11:25:47Z
  name: testvm-migration
  namespace: default
  resourceVersion: "96512"
  selfLink: /apis/kubevirt.io/v1alpha1/namespaces/default/migrations/testvm-migration
  uid: 230cb349-0972-11e7-b9cf-525400b9ab10
spec:
  selector:
    name: testvm
status:
  phase: Succeeded

```

To cancel `testvm_migration`:

```bash

kubectl delete migrations testvm_migration
```


 Each successfully running virtual machine object has an
 associated Pod that contains the VM as a process. When a Pod is
 scheduled onto a node, it stays there until it is deleted. A Pod is
 completely immutable after creation. A VM is mutable regarding to
 scheduling (cluster) after creation, but those changes will not take
 effect until a migration takes place.  For example, to pin a VM to a
 specific node, the VM might be launched with the node selection
 criteria.

```yaml
nodeSelector:
  kubernetes.io/hostname: node0
```

Which would pin it to node0. In order to migrate to another node, the
user would first have to remove (via put or patch) the key and value
`kubernetes.io/hostname: node0`. This *would not* force a
migration, it would only  *allow* a migration to take place.

The migration and the vm can both have node selectors.  The two sets
of node selectors are merged and used for scheduling the destination
of the migration.  If there is a contradiction between the node
selection criteria, the migration will fail.

A migration also has an associated pod that runs the migration
process. When the migration completes, the process running in the
pod's container exits. The return code of the process indicates the
status of the migration. This is translated into the status field of
the migration object.
