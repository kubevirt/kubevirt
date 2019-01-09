# Local PV Allocation
The purpose of this document is to show how to provision a Virtual Machine(VM) disk image on a specific node to be able 
to run the VM on the same node. This is important if you are using local-volume storage and you want a disk image to be 
provisioned on a particular node.

### Prerequisites
You have a Kubernetes cluster up and running with CDI installed.

### Create local-volume based PersistentVolume manifest
First create a PersistentVolume (PV) yaml file with NodeAffinity field to allocate the PV on the specific required node. 
Verify that the label you specify in the NodeAffinity exists on the node you want the PV to be allocated.
In addition, add a specific label to be matched by a PersistenVolumeClaim during the binding process.
An example of such PersistentVolume is:

```yaml
kind: PersistentVolume
apiVersion: v1
metadata:
  name: local-pv-allocation
  labels:
    node: node02 #This is the label used by the PVC during the binding process.
spec:
  storageClassName: manual #This should be the name of your local-volume storage class.
  persistentVolumeReclaimPolicy: Delete
  capacity:
    storage: 10Gi #This should be the size you want the PV to be.
  local:
    path: /mnt/local-storage/local/disk2
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - node02 #This is the node label of the node you want this PV allocated to.
  accessModes:
    - ReadWriteOnce
  volumeMode: Filesystem

```
### Deploy the PersistentVolume to the cluster:
```bash
$ kubectl create -f manifests/example/local-pv-allocation.yaml
```

### Create your PersistenVolumeClaim manifest
Create a PersistentVolumeClaim yaml file with a selector field that matches the label you specified in the PersistentVolume.
In Addition add the source, the contentType and the endpoint annotations, to start importing Virtual Machine (VM) disk image by CDI.
An example of such PersistentVolumeClaim is:

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: local-pv-allocation-claim
  annotations:
    cdi.kubevirt.io/storage.import.source: "http" #defaults to http if missing or invalid
    cdi.kubevirt.io/storage.contentType: "kubevirt" #defaults to kubevirt if missing or invalid.
    cdi.kubevirt.io/storage.import.endpoint: "http://distro.ibiblio.org/tinycorelinux/9.x/x86/release/Core-current.iso" # http or https is supported
    cdi.kubevirt.io/storage.import.secretName: "" # Optional. The name of the secret containing credentials for the end point	    
spec:
  storageClassName: manual #This should be the name of your local-volume storage class.
  accessModes:
    - ReadWriteOnce
  selector:
    matchLabels:
      node: node02
  resources:
    requests:
      storage: 3Gi

```

### Deploy the PersistentVolumeClaim to the cluster:
```bash
$ kubectl create -f manifests/example/local-pv-allocation-claim.yaml
```

Verify that the PVC was deployed and is bound to your PV:
```bash
$ kubectl.sh get pvc local-pv-allocation-claim

NAME                        STATUS    VOLUME                CAPACITY   ACCESS MODES   STORAGECLASS   AGE
local-pv-allocation-claim   Bound     local-pv-allocation   10Gi       RWO            manual         12m
```

Verify that an importer pod has been created and is running on the node you specified in the NodeAffinity 
field in the PersistentVolume.
When the importer pod is completed and the disk image was imported to the node, you can keep going and run this VM
on the same Node.
