# Claim which could be used to back the iscsi-target
#kind: PersistentVolumeClaim
#apiVersion: v1
#metadata:
#  name: my-storage-claim
#spec:
#  accessModes:
#    - ReadWriteOnce
#  resources:
#    requests:
#      storage: 5.1Gi
#  selector:
#    matchLabels:
#      release: "stable"
#---
# This is exposing a file on the claim as a LUN
apiVersion: v1
kind: Pod
metadata:
  labels:
    name: my-iscsi-target-for-my-storage
  name: my-iscsi-target-for-my-storage
spec:
  containers:
    - name: target
      image: kubevirt/iscsi-target-tgtd
      volumeMounts:
        - mountPath: /volume
          name: my-storage-claim
      ports:
        - containerPort: 3260
      env:
      - name: FILE_SIZE
        value: "5G"
  volumes:
    - name: my-storage-claim
      persistentVolumeClaim:
      claimName: my-storage-claim
---
# This is exposing the LUN on the cluster
apiVersion: v1
kind: Service
metadata:
  name: iscsi-target-tgtd
spec:
  ports:
    - name: iscsi
      port: 3260
      targetPort: 3260
  selector:
    name: my-iscsi-target-for-my-storage
