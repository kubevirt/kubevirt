# Deploy HCO using kustomize
The KubeVirt Hyperconverged Cluster Operator (HCO) is delivered and deployed on a running OCP/OKD cluster using the kustomize method. 

# Kustomize Manifests
In order to install HCO on your cluster, two necessary steps to be performed:
1. **Delivery** - Make HCO recognized and available for the operator-lifecycle-manager (OLM).
2. **Deployment** - Use OLM provided resources and APIs to deploy HCO on the cluster.

The directory tree consists of kustomize-based manifests with default values, supporting various deployment configurations.

## Delivery
There are two distinct options to deliver HCO operator to OLM - Marketplace and Image Registry.

### Marketplace
This method is taking advantage of CatalogSourceConfig, pointing to an OperatorSource, which makes the operator available on OLM OperatorHub.
To manually deliver HCO using marketplace, edit `spec.registryNamespace` of `marketplace/operator_source.yaml` to the desired value (default is "kubevirt-hyperconverged"), and run:
```
oc apply -k marketplace
```
Which will create the HCO catalog source with default configuration. After processing is complete, the package will be available in OperatorHub.

#### Private Repo
If the operator source is located in a private Quay.io registry, you should provide the OperatorSource resource with a secret, which can be extracted by:
```
curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
  {
      "user": {
          "username": "'"${QUAY_USERNAME}"'",
          "password": "'"${QUAY_PASSWORD}"'"
      }
  }' | jq -r '.token'
```
The token should be inserted in `spec.authorizationToken.secretName` of `private_repo/operator_source.patch.yaml`, then run:
```
oc apply -k private_repo
```

### Image Registry
This method is delivering the operator's bundle image via a grpc protocol from an image registry.
To manually deliver HCO using image registry, edit `spec.image` of `image_registry/catalog_source.yaml` to the desired image bundle URL, and run:
```
oc apply -k image_registry
```

### Automation
The shell script `deploy_kustomize.sh` can be used to automate delivery of HCO to OLM.

#### Content-Only flag
To make HCO available for deployment in the cluster, without actually deploy it, set "CONTENT_ONLY" to "true". That will stop script execution before entering the deployment phase.

#### Marketplace
Set environment variable "MARKETPLACE_MODE" to "true".

##### Private Repo
Set "PRIVATE_REPO" to "true" and provide credentials using "QUAY_USERNAME" and "QUAY_PASSWORD" environment variables.

#### Image Registry
Set environment variable "MARKETPLACE_MODE" to "false".

## Deployment
The deployment phase is consisting of 3 resources, located in `base` directory:
* OperatorGroup
* Subscription
* HyperConverged Custom Resource

In addition, a namespace must be deployed prior to the deployment of resources above. the namespace resource can be found in `namespace.yaml`.
To deploy HCO with default settings, run:
```
cat <<EOF >kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
  - base
resources:
  - namespace.yaml
EOF

oc apply -k .
```

### KVM Emulation
If KVM emulation is required on your environment, use the following overlay to add the Subscription resource with relevant KVM config:
```
oc apply -k kvm_emulation
```

### Automation
To automate the process of delivery **and** deployment, set the environment variable "CONTENT_ONLY" to "false", then run `./deploy_kustomize.sh`.
To use the script in conjunction with KVM_EMULATION property, set "KVM_EMULATION" env var to "true" prior to running the script. 

## Customizations
Existing manifests in this repository are representing an HCO deployment with default settings.
In order to make customizations to your deployment, you need to set up other environment variables and create kustomize overlays to override default settings.

### Change Deployment Namespace
The default namespace is `kubevirt-hyperconverged`.
In order to change that to a custom value, you should edit `namespace.yaml` and update its `metadata.name` value, and run:
```
cat <<EOF >kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: ${DESIRED_NAMESPACE}
bases:
  - base
resources:
  - namespace.yaml
EOF

oc apply -k .
```

### Modify HCO Channel and Version
Create a Subscription patch to reflect the desired version and channel.
```
cat > subscription.patch.yaml << EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: hco-operatorhub
spec:
  startingCSV: kubevirt-hyperconverged-operator.v${HCO_VERSION}
  channel: "${HCO_CHANNEL}"
```

#### Deploy
When customizations are ready, run `./deploy_kustomize.sh`.
The script will prepare and submit kustimized manifests directories to the cluster. It will also check whenever deployment is complete (HCO CR reports Condition "Available" True), and finish successfully.