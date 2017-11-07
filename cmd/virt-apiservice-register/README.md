# API service registration tool

This file implements a tool that simplifies registration of API
services. The tool takes as parameters the API group and version, and a
selector for the pod where the API server is running. It does the
following:

* Checks if a secret named `apiservice` followed by the API group and
  version exists. If it doesn't the tool creates it. The name of the
  secret can be changed with the `-secret-name` command line option.

* Checks if the secret contains a `ca.crt` key, and if that key contains
  the CA certificate tht Kubernetes uses to sign certificates for API
  services. If the secret doesn't contain that key, then the tool finds
  the CA certificate and updates the secret.

* Checks if the secret contains a `tls.key` key, and if that key
  contains a private key. If the secret doesn't contain that key, then
  the tool generates a new one and updates the secret.

* Checks if the secret contains a `tls.crt` key, and if that key
  contains a certificate. If the secret doesn`t contain that
  certificate, then the tool checks if there is a certificate signing
  request for the private key, creates it if needed, and waits till it
  is approved. When the certificate signing request is approved the tool
  updates the secret.

* Checks if the service that handles the traffic to the API server
  exists. If it doesn't exist then the tool creates it. By default the
  name of this service is `apiservice` followed by the API group and
  version, but it can be changed using the optional `-service-name`
  command line option.

* Checks if the the API service is registered. If it isn't registered
  then the tool registers it. The name used to register the API service
  is the API version followed by a dot and the API group. It can't be
  changed, as this is required by Kubernetes.

For example, to create register the service for API group `myapi.io` and
version `v1alpha` the command could be like this:

```shell
virt-apiservice-register \
-api-group=myapi.io \
-api-version=v1alpha1 \
-secret-name=mysecret \
-service-name=myservice \
-target-selector=app=myapiservice
```

It is important to make sure that the target, the pod where the actual
API service is running, has labels that matches the selector given in the
'-target-selector' option.

## How to use it

The tool is intended for use in an initialization container inside the
pod of the API server itself, so that it is always executed before the
API server container.

The tool can create the secret if it doesn't exist, but that doesn't
work if the secret is going to be mounted by the pod, as Kubernetes will
not start the pod till the secret exists. In this case the secret needs
to be created manually, ideally as part of the same manifests that
creates the pod where the tool will be executed. For example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
---
apiVersion: v1
kind: ReplicationController
metadata:
  name: myserver
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: myserver
    spec:
      serviceAccountName: myaccount
      volumes:
      - name: pkifiles
        secret:
          secretName: mysecret
      initContainers:
      - name: register
        image: virt-apiservice-register:latest
        imagePullPolicy: IfNotPresent
        args:
        - -api-group=myapi.io
        - -api-version=v1alpha1
        - -secret-name=mysecret
        - -service-name=myservice
        - -target-selector=app=myserver
      containers:
      - name: server
        image: myimage:latest
        imagePullPolicy: IfNotPresent
        args:
        - --etcd-servers=http://localhost:2379
        - --tls-ca-file=/mypkifiles/ca.crt
        - --tls-cert-file=/mypkifiles/tls.crt
        - --tls-private-key-file=/mypkifiles/tls.key
        volumeMounts:
        - name: pkifiles
          mountPath: "/mypkifiles"
          readOnly: true
```

By default the tool doesn't automatically approve the requested
certificate, it will just wait for the administrator (or some other
tool) to approve or deny it:

```shell
$ kubctl certificate approve myservice.default.svc.cluster.local
```

To enable automatic approval of the certificate add the
`-auto-approve=true` command line option. Note that this requires
additional permissions, as described in the next section.

## Permissions required

In order to run successfully the tool needs permissions. They can be
created with a role like this:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: virt-apiserver-register
rules:
- apiGroups:
  - apiregistration.k8s.io
  resources:
  - apiservices
  verbs:
  - create
  - get
- apiGroups:
  - ""
  resources:
  - configmaps
  verbs:
  - get
  resourceNames:
  - extension-apiserver-authentication
- apiGroups:
  - ""
  resources:
  - services
  verbs:
  - create
  - get
- apiGroups:
  - ""
  resources:
  - secrets
  verbs:
  - create
  - get
  - put
- apiGroups:
  - certificates.k8s.io
  resources:
  - certificatesigningrequests
  verbs:
  - create
  - delete
  - get
  - watch
```

The tool can also automatically approve the certificate signing request
that it generates, if the `-auto-approve=true` command line option is
used, but to do so it will need extra permissions that aren't usually
granted to services. Use this capability carefully. The additional
permissions required are the following:

```yaml
- apiGroups:
  - certificates.k8s.io
  resources:
  - certificatesigningrequests/approval
  verbs:
  - put
`
```

