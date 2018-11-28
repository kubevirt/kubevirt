The registrydisk can be used ina  different context as well:
It can be used in a standalone pod to expose the disk it carries as an iSCSI target.

Two things you need:
1. A pod to run the iSCSI target and a servcie exporting it
2. A PV and PVC to make it usable on the clutser

Creating the pod and service:

```
$ kubectl create -f - <<<EOY
apiVersion: v1
kind: Pod
metadata:
  name: my-iscsi-target
  labels:
    app: my-iscsi-target
spec:
  containers:
  - name: my-iscsi-target
    image: kubevirt/cirros-registry-disk-demo:latest
    env:
    - name: AS_ISCSI
      value: yes
---
kind: Service
apiVersion: v1
metadata:
  name: my-iscsi-target
spec:
  selector:
    app: my-iscsi-target
  ports:
  - protocol: TCP
    port: 3260
    targetPort: 3260
EOY
```

Creating a PV and PVC pointing to it:

```
$ kubectl create -f - <<<EOY
apiVersion: "v1"
kind: "PersistentVolume"
metadata:
  name: my-iscsi-lun
spec:
  capacity:
    storage: "10G"
  accessModes:
    - "ReadWriteMany"
  claimRef:
    namespace: default
    name: my-iscsi-lun-claim
  volumeMode: block
  iscsi:
    targetPortal: my-iscsi-target.svc:3260
    iqn: iqn.2018-01.io.kubevirt:wrapper
    lun: 1
    readOnly: false
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: my-iscsi-lun
spec:
  accessModes:
    - "ReadWriteMany"
  resources:
    requests:
      storage: "10G"
EOY
```
