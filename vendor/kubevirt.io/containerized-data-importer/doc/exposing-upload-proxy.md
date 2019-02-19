# Exposing CDI Upload Proxy
The purpose of this document is to show how to expose CDI Upload Proxy service in a production environment.

## Prerequesites
You have a Kubernetes cluster up and running with CDI installed.

In order to upload data to your cluster, the cdi-uploadproxy service must be accessible from outside the cluster.
This can be achieved using Ingress (Kubernetes) or Route (Openshift).


### Kubernetes

Before starting to work with Ingress resource, you will need to setup an Ingress Controller. Simply creating the resource will take no affect.
There are number of [Ingress controllers](https://kubernetes.io/docs/concepts/services-networking/ingress/#ingress-controllers) you can choose from.

Create Ingress for the upload proxy:


```bash
cat <<EOF | kubectl apply -f -
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: cdi-uploadproxy-ingress
  namespace: cdi
  annotations:
    nginx.org/ssl-services: "cdi-uploadproxy"
    ingress.kubernetes.io/ssl-passthrough: "true"
    nginx.ingress.kubernetes.io/secure-backends: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "0"
spec:
  rules:
    # change to a valid FQDN in your organization
  - host: cdi-uploadproxy.example.com
    http:
      paths:
      - backend:
          serviceName: cdi-uploadproxy
          servicePort: 443
  tls:
  - hosts:
    # change to a valid FQDN in your organization
    - cdi-uploadproxy.example.com
EOF
```


### Openshift

Create Route for the upload proxy:

```bash
cat <<EOF | oc apply -f -
apiVersion: v1
kind: Route
metadata:
  name: cdi-uploadproxy-route
  namespace: cdi
spec:
  host: cdi-uploadproxy.example.com
  to:
    kind: Service
    name: cdi-uploadproxy
  tls:
    termination: passthrough
EOF
```

### Upload an Image

Assuming you completed the steps in [Upload document](upload.md) execute the following to upload the image:

```bash
curl -v --insecure -H "Authorization: Bearer $TOKEN" --data-binary @tests/images/cirros-qcow2.img https://cdi-uploadproxy.example.com/v1alpha1/upload
```