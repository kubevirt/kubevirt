# NetworkAttachmentDefinition CRD Client

Based on https://github.com/openshift-evangelists/crd-code-generation

**Note:** You have to clone/import this repository with all lower-case letters:

```
github.com/k8snetworkplumbingwg/network-attachment-definition-client
```

## Getting Started

First register the custom resource definition:

```
kubectl apply -f artifacts/network-crd.yaml
```

Then add an example of the `NetworkAttachmentDefinition` kind:

```
kubectl apply -f artifacts/my-network.yaml
```

Finally build and run the example:

```
go build
./example -kubeconfig ~/.kube/config
```
