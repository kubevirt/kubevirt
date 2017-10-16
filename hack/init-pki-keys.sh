#!/bin/bash

set -ex
KUBEDIR=$(pwd)
WORKDIR=$(mktemp -d)
pushd "$WORKDIR"

cat <<EOF | cfssl genkey - | cfssljson -bare server
{
  "hosts": [
    "virt-apiserver-service.default.svc.cluster.local",
    "virt-apiserver-default.pod.cluster.local",
    "virt-apiserver-service.default.svc",
    "192.168.200.2"
  ],
  "CN": "virt-apiserver-service.default.svc",
  "key": {
    "algo": "ecdsa",
    "size": 256
  }
}
EOF

popd

APISERVER_CSR=$(cat "$WORKDIR/server.csr" | base64 | tr -d "\n")

cat <<EOF > "$WORKDIR/csr.yaml"
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: virt-apiserver-service.default
spec:
  groups:
  - system:authenticated
  request: "$APISERVER_CSR"
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

./cluster/kubectl.sh create -f "$WORKDIR/csr.yaml"
./cluster/kubectl.sh certificate approve virt-apiserver-service.default


APISERVER_CRT=$(./cluster/kubectl.sh get csr virt-apiserver-service.default -o jsonpath='{.status.certificate}')
APISERVER_KEY=$(cat "$WORKDIR/server-key.pem" | base64 | tr -d '\n')
REQUESTHEADER_CA_CRT=$(./cluster/kubectl.sh get configmap --namespace kube-system extension-apiserver-authentication -o jsonpath='{.data.requestheader-client-ca-file}' | base64 | tr -d '\n')

cat <<EOF > "$WORKDIR/secret.yaml"
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
EOF

./cluster/kubectl.sh create -f "$WORKDIR/secret.yaml"
echo "$APISERVER_CRT" > cluster/.apiserver.ca.crt

#rm -rf $WORKDIR
