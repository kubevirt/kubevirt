# CDI Upload User Guide
The purpose of this document is to show how to upload a VM disk image on your local system to a PersistentVolumeClaim in Kubernetes.

## Prerequesites
You have a Kubernetes cluster up and running with CDI installed and at least one PersistentVolume is available.

Commands/manifests below will be run from the root of the CDI repo against a Minikube cluster.

If you are using Minikube with the `storage-provisioner` addon enabled.  You can create a PersistentVolume like so:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv0001
spec:
  accessModes:
    - ReadWriteOnce
  capacity:
    storage: 5Gi
  hostPath:
    path: /data/pv0001/
EOF
```

## Expose cdi-uploadproxy service
In order to upload data to your cluster, the cdi-uploadproxy service must be accessible from outside the cluster.  In a production environment, this probably involves setting up a Ingress or a LoadBalancer Service.

### Minikube

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: cdi-uploadproxy-nodeport
  namespace: cdi
  labels:
    cdi.kubevirt.io: "cdi-uploadproxy"
spec:
  type: NodePort
  ports:
    - port: 443
      targetPort: 8443
      nodePort: 31001
      protocol: TCP
  selector:
    cdi.kubevirt.io: cdi-uploadproxy
EOF
```

### Minishift

```bash
oc get secret -n cdi cdi-upload-proxy-ca-key -o=jsonpath="{.data['tls\.crt']}" | base64 -d > tls.crt && \
oc create route reencrypt -n cdi --service=cdi-uploadproxy --dest-ca-cert=tls.crt && \
rm tls.crt
```

### Port forwarding via the API server

`kubectl port-forward -n cdi service/cdi-uploadproxy 8443:443`

(Make sure port 8443 on your system isn't occupied.)

## Create a Data Volume
Specifying an 'upload' source will mark the data volume as a target for upload.

To create an upload datavolume use the following [example](../manifests/example/upload-datavolume.yaml).
```yaml
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
metadata:
  name: upload-datavolume
spec:
  source:
      upload: {}
  pvc:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 500Mi
```

```bash
kubectl apply -f manifests/example/upload-datavolume.yaml
```

## Request an Upload Token
Before sending data to the Upload Proxy, and Upload Token must be requested.  The CDI API Server validatees that the user has permissions to `post` to `uploadtokenrequest` resources.

Take a look at at `manifests/example/upload-token.yaml` for an example.
```yaml
apiVersion: upload.cdi.kubevirt.io/v1alpha1
kind: UploadTokenRequest
metadata:
  name: upload-test
  namespace: default
spec:
  pvcName: upload-test

```
```bash
kubectl apply -f manifests/example/upload-datavolume-token.yaml -o yaml
apiVersion: upload.cdi.kubevirt.io/v1alpha1
kind: UploadTokenRequest
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"upload.cdi.kubevirt.io/v1alpha1","kind":"UploadTokenRequest","metadata":{"annotations":{},"name":"upload-datavolume-token","namespace":"default"},"spec":{"pvcName":"upload-datavolume"}}
  creationTimestamp: null
  name: upload-datavolume-token
  namespace: default
spec:
  pvcName: upload-datavolume
status:
  token: eyJhbGciOiJQUzUxMiIsImtpZCI6IiJ9.eyJwdmNOYW1lIjoidXBsb2FkLXRlc3QiLCJuYW1lc3BhY2UiOiJkZWZhdWx0IiwiY3JlYXRpb25UaW1lc3RhbXAiOiIyMDE4LTA5LTIxVDE4OjEyOjE5LjQwODI1MDQ4NFoifQ.JWk1VyvzSse3eFiBROKgGoLnOPCiYW9JdDWKXFROEL6XY0O5lFb1R0rwdfWwC3BBOtEA9mC9x3ZGYPnYWO-5G_r1fWKHjF-zifrCX_3Dhp3vfSq6Zfpu-vV0Qn0A3YkSCCmiC_nONAhVjEDuQsRFIKwYcxBoEOpye92ggH2u5FxQE7FwxxH6-RHun9tc_lIFX-ZFKnq7n5tWbjsTmAZI_4rDNgYkVFhFtENU6e-5_Ncokxs3YVzkbSrXweZpRmmaYQOmZhjXSLjKED_2FVq7tYeVueEEhKC_zJ-AEivstALPwPjiwyWXJyfE3dCmbA1sBKuNUrAaDlBvSAp1uPV9eQ
```

Save the `token` field of the response status.  It will be used to authorize our CDI Upload request. Tokens are good for 5 minutes.

You can capture the token in an environment variable by doing this:
```bash
TOKEN=$(kubectl apply -f manifests/example/upload-datavolume-token.yaml -o="jsonpath={.status.token}")
``` 

## Upload an Image
We will be using [curl](https://github.com/curl/curl) to upload `tests/images/cirros-qcow2.img` to the datavolume.

Assuming that the environment variable `TOKEN` contains a valid UploadToken, execute the following to upload the image:

### Minikube
```bash
curl -v --insecure -H "Authorization: Bearer $TOKEN" --data-binary @tests/images/cirros-qcow2.img https://$(minikube ip):31001/v1alpha1/upload
```

### Minishift

```bash
curl -v --insecure -H "Authorization: Bearer $TOKEN" --data-binary @tests/images/cirros-qcow2.img https://cdi-uploadproxy-cdi.$(minishift ip).nip.io/v1alpha1/upload
```

Assuming you did not get an error, the Datavolume `upload-datavolume` should now contain a bootable VM image.

