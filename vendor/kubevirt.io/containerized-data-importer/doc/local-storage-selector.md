# Local storage
Sometimes when using local volume storage it can be beneficial to be able to specify which Persistent Volume (PV) to use on a particular node. Maybe that node is better suited for running a particular work load, but because of the nature of how local volume storage works, the pod scheduler may or may not pick the node you want to start the POD and begin the Persistent Volume Claim(PVC) to PV binding process.

You cannot specify a node selector in a PVC because PVCs are more general than a PV and the storage class used might be a shared storage, where it makes no sense to be able to specify which node you want the PVC to be bound on. Luckily like most objects in Kubernetes PVCs allow you to use a labelSelector, which we can use to select which PV we want the PVC bound to, and thus force a POD that uses that PVC on the node you want.

## Example
For this example we will be using a 3 node cluster, the nodes are named node01/02/03 for simplicity.

```
NAME      STATUS    ROLES     AGE       VERSION
node01    Ready     master    1h        v1.11.0
node02    Ready     <none>    1h        v1.11.0
node03    Ready     <none>    1h        v1.11.0
```

On each node we have specified 3 local volume PVs ahead of time, giving us a total of 9 available PVs, 3 on each node.

```
NAME                         CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS      CLAIM     STORAGECLASS   REASON    AGE
local-pv-182992f7            37Gi       RWO            Delete           Available             local                    1h
local-pv-25a396e8            37Gi       RWO            Delete           Available             local                    1h
local-pv-2df7bcf             37Gi       RWO            Delete           Available             local                    1h
local-pv-38f4ffe             37Gi       RWO            Delete           Available             local                    1h
local-pv-3fb65ef2            37Gi       RWO            Delete           Available             local                    1h
local-pv-62a1c3c8            37Gi       RWO            Delete           Available             local                    1h
local-pv-6b6380e2            37Gi       RWO            Delete           Available             local                    1h
local-pv-76a7717             37Gi       RWO            Delete           Available             local                    1h
local-pv-7ae73fde            37Gi       RWO            Delete           Available             local                    1h
```

In order to determine which node a particular PV lives on we can look at the definition of the PV

```bash
$ kubectl get pv local-pv-182992f7 -o yaml
```

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  annotations:
    pv.kubernetes.io/provisioned-by: local-volume-provisioner-node01-05534b27-45c1-11e9-8819-525500d15501
  creationTimestamp: 2019-03-13T18:52:23Z
  finalizers:
  - kubernetes.io/pv-protection
  name: local-pv-182992f7
  resourceVersion: "649"
  selfLink: /api/v1/persistentvolumes/local-pv-182992f7
  uid: 23433212-45c1-11e9-8819-525500d15501
spec:
  accessModes:
  - ReadWriteOnce
  capacity:
    storage: 37Gi
  local:
    path: /mnt/local-storage/local/disk9
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - node01
  persistentVolumeReclaimPolicy: Delete
  storageClassName: local
  volumeMode: Filesystem
status:
  phase: Available
```
As you can see by looking at the nodeAffinity of the PV, this PV lives on node01. Lets say node01 has some special meaning and we want to use Containerized Data Importer (CDI) to import a disk image on that particular node. Normally we would create a Data Volume (DV) and have the CDI controller start an importer POD which is then scheduled on a node that the scheduler picks. But we want to force it onto node01. A DV will create a PVC to hold the data.

First we will need to label the PV with a label we can use in the DV specification as a labelSelector.

```
$ kubectl label pv local-pv-182992f7 node=node01
persistentvolume/local-pv-182992f7 labeled
```

Here I picked the label name to be node, but it can be anything you want. Now create a DV like normal, but with a label selector.

```yaml
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
metadata:
  name: "example-import-dv"
spec:
  source:
      http:
         url: "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img"
  pvc:
    accessModes:
      - ReadWriteOnce
    selector:
      matchLabels:
        node: node01
    resources:
      requests:
        storage: "64Mi"
```

As you can see compared to the normal example, there is an extra matchLabels selector, to match against the label we just added to the PV. Now create the DV like normal and it will create a PVC that will get bound to the PV with that label, which is on node01 like we want.

If you are using a local volume provisioner and have the reclaim policy set to 'Delete' it will delete and re-create the PV after you delete a PVC bound to that PV. This will remove the labeling on the PV.