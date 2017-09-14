# Overview
In order to provide proper authentication for client requests, the KubeVirt
addon apiserver requires a CA certificate and keypair be registered with
Kubernetes. This is a one-time process that is required when first setting up
KubeVirt. An attempt to script this process has been included in
`hack/init-pki-keys.sh`, but this document exists for completeness and clarity.

# Requirements

## Download CFSSL
CFSSL is Cloudflare's PKI/TLS "swiss army knife".

It can be obtained [Here](https://github.com/cloudflare/cfssl) or
[Here](https://pkg.cfssl.org/)

The two commands needed are cfssl, and cfssljson.

# Create a CSR

* Prepare a CSR configuration `cfssl.json` for CFSSL:
```json
{
  "hosts": [
    "virt-apiserver-service.default.svc.cluster.local",
    "virt-apiserver-default.pod.cluster.local",
    "virt-apiserver-service.default.svc",
    "192.168.200.2",
  ],
  "CN": "virt-apiserver-service.default.svc",
  "key": {
    "algo": "ecdsa",
    "size": 256
  }
}
```

* Execute: `cat cfssl.json | cfssl genkey - | cfssljson -bare server`
This will create two files in your current working directory: `server-key.pem`
and `server.csr`

* Execute: `cat server.csr | base64 | tr -d "\n"`
This will format the certificate signing request into a format appropriate for
Kubernetes.

* Create a manifest `csr.yaml` to request this CSR be signed by Kuberenetes:

```yaml
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: virt-apiserver-service.default
spec:
  groups:
  - system:authenticated
  request: "{{ CSR.CRT }}"
  usages:
  - digital signature
  - key encipherment
  - server auth
```

Replace `{{ CSR.CRT }}` with the output from the previous command.

* Execute: `./cluster/kubectl.sh create -f csr.yaml`
This will create a CSR object in the pending state.

* Execute: `./cluster/kubectl.sh certificate approve virt-apiserver-service.default`
This will approve the CSR to authorize it so that it can be used.

## Create a Kubernetes Secret
The previous section created a TLS Certificate, but the KubeVirt apiserver will not
be able to access that data, so we need to create a secret containing the correct
values.

### Apiserver Cert
APISERVER_CRT=$(./cluster/kubectl.sh get csr virt-apiserver-service.default -o jsonpath='{.status.certificate}')

### Apiserver Key
APISERVER_KEY=$(cat server-key.pem | base64 | tr -d '\n')

### RequestHeader CA Cert
REQUESTHEADER_CA_CRT=$(./cluster/kubectl.sh get configmap --namespace kube-system extension-apiserver-authentication -o jsonpath='{.data.requestheader-client-ca-file}' | base64 | tr -d '\n')

* Create a manifest `secret.yaml` using the 3 values we just looked up.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: virt-apiserver-cert
  labels:
    app: virt-apiserver
type: Opaque
data:
  tls.crt: "$APISERVER_CRT"
  tls.key: "$APISERVER_KEY"
  requestheader-ca.crt: "$REQUESTHEADER_CA_CRT"
```

* Execute: `./cluster/kubectl.sh create -f secret.yaml`

* Finally, save the Apiserver Certificate in a known location
`echo "$APISERVER_CRT" > cluster/.apiserver.ca.crt`

