# Containerized Data Importer
[![Go Report Card](https://goreportcard.com/badge/github.com/kubevirt/containerized-data-importer)](https://goreportcard.com/report/github.com/kubevirt/containerized-data-importer)
[![Coverage Status](https://img.shields.io/coveralls/kubevirt/containerized-data-importer/master.svg)](https://coveralls.io/github/kubevirt/containerized-data-importer?branch=master)

A declarative Kubernetes utility to import Virtual Machine images for use with [Kubevirt](https://github.com/kubevirt/kubevirt). At a high level, a persistent volume claim (PVC), which defines VM-suitable storage via a storage class, is created. A custom controller watches for importer specific claims, and when discovered, starts an import/copy process. The status of the import process is reflected in the same claim, and when the copy completes Kubevirt can create the VM based on the just-imported image.
In Addition, the Containerized Data Importer gives the option to clone the imported VM image from one PVC to another one across two different namespaces (A.K.A Host-Assisted cloning). 

1. [Purpose](#purpose)
1. [Versions](#versions)
1. [Design](/doc/design.md#design)
1. [Running the CDI Controller](#deploying-cdi)
1. [Cloning VM Images](#cloning-vm-images)
1. [Hacking (WIP)](hack/README.md#getting-started-for-developers)
1. [Security Configurations](#security-configurations)

## Overview

### Purpose

This project is designed with Kubevirt in mind and provides a declarative method for importing VM images into a Kuberenetes cluster.
Kubevirt detects when the VM image copy is complete and, using the same PVC that triggered the import process, creates the VM.

This approach supports two main use-cases:
-  a cluster administrator can build an abstract registry of immutable images (referred to as "Golden Images") which can be cloned and later consumed by Kubevirt, or
-  an ad-hoc user (granted access) can import a VM image into their own namespace and feed this image directly to Kubevirt, bypassing the cloning step.


For an in depth look at the system and workflow, see the [Design](/doc/design.md#design) documentation.

### Versions

CDI follows the common semantic version scheme defined at semver.org in the "vMajor.Minor.Patch" pattern.  These are defined as:

- Major: API or other changes that will break existing CDI deployments.  These changes are expected to require users to alter the way they interact with CDI.  Major versions are released at the end a development cycle (every 2 weeks) in leiu of a Minor version.

- Minor: Backwards compatible changes within the current Major version.  These changes represent the products of a 2 week developement cycle and contain bug fixes and new features.  By releasing merged work at the end of the cycle, users are able to closely track the project's progress and report issues or bugs soon after they are introduced.  At the same time, it should be easy to roll back to the previous Minor version if the release blocks the user's workflow.

- Patch: Mid development cycle critical bug fix. In the case that a Minor release has a bug that must be fixed for users before the next release cycle, a Patch may be published.  Patches should be small in scope and only alter / introduce code related to the bug fix.

See [releases.md](/doc/releases.md) for more information on versioning.

### Data Format

The importer is capable of performing certain functions that streamline its use with Kubevirt.  It automatically decompresses **gzip** and **xz** files, and un-tar's **tar** archives. Also, **qcow2** images are converted into a raw image files needed by Kubevirt, resulting in the final file being a simple _.img_ file.

Supported file formats are:

- .tar
- .gz
- .xz
- .img
- .iso
- .qcow2

## Deploying CDI

### Assumptions
- A running Kubernetes cluster with roles and role bindings implementing security necesary for the CDI controller to watch PVCs and pods across all namespaces.
- A storage class and provisioner.
- An HTTP or S3 file server hosting VM images

- An optional "golden" namespace acting as the image registry. The `default` namespace is fine for tire kicking.

### Download CDI

`$ go get kubevirt.io/containerized-data-importer`

*Or*

`$ git clone https://github.com/kubevirt/containerized-data-importer.git $GOPATH/src/kubevirt.io/containerized-data-importer`

*Or download only the yamls:*

```shell
$ mkdir cdi-manifests && cd cdi-manifests
$ wget https://raw.githubusercontent.com/kubevirt/containerized-data-importer/kubevirt-centric-readme/manifests/example/golden-pvc.yaml
$ wget https://raw.githubusercontent.com/kubevirt/containerized-data-importer/kubevirt-centric-readme/manifests/example/endpoint-secret.yaml
```

### Deploy CDI from a release

Deploying the CDI controller is straight forward. In this document the _default_ namespace is used, but in a production setup a [protected namespace](#protecting-the-golden-image-namespace) that is inaccessible to regular users should be used instead.

1. Ensure that the cdi-sa service account has proper authority to run privileged containers, typically in a kube environment this is true by default.
   If you are running an openshift variation of kubernetes you may need to enable privileged containers in the security context:

```
$ oc adm policy add-scc-to-user privileged -z cdi-sa
```

2. Deploy the controller from the release manifest:

```
$ VERSION=<cdi version>
$ kubectl create -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-controller.yaml
```

### Deploy CDI using a template

By default when using `manifests/generated/cdi-controller.yaml` CDI will deploy into the kube-system namespace using default settings.  You can customize the deployment by using the generated `manifests/generated/cdi-controller.yaml.j2` jinja2 template.  This allows you to alter the install namespace, docker image repo, docker image tags, etc.  To deploy using the template follow these steps:

1. Install j2cli:

    `$ pip install j2cli`

1. Install CDI:

    ```
    $ cdi_namespace=default \
      docker_prefix=kubevirt \
      docker_tag=v1.1.1 \
      pull_policy=IfNotPresent \
      verbosity=1 \
      j2 manifests/generated/cdi-controller.yaml.j2 | kubectl create -f -
    ```
    Check the template file and make sure to supply values for all variables.

> Note: the default verbosity level is set to 1 in the controller deployment file, which is minimal logging. If greater details are desired increase the `-v` number to 2 or 3.

> Note: the importer pod uses the same logging verbosity as the controller. If a different level of logging is required after the controller has been started, the deployment can be edited and applied by using `kubectl apply -f <CDI-MANIFEST>`. This will not alter the running controller's logging level but will affect importer pods created after the change. To change the running controller's log level requires it to be restarted after the deployment has been edited.


### Start Importing Images

> Note: The CDI controller is a required part of this work flow.

Make copies of the [example manifests](./manifests/example) for editing. The necessary files are:
- golden-pvc.yaml
- endpoint-secret.yaml

###### Edit golden-pvc.yaml:
1.  `storageClassName:` The default StorageClass will be used if not set.  Otherwise, set to a desired StorageClass.

1.  `cdi.kubevirt.io/storage.import.endpoint:` The full URL to the VM image in the format of: `http://www.myUrl.com/path/of/data` or `s3://bucketName/fileName`.

1.  `cdi.kubevirt.io/storage.import.secretName:` (Optional) The name of the secret containing the authentication credentials required by the file server.

###### Edit endpoint-secret.yaml (Optional):

> Note: Only set these values if the file server requires authentication credentials.

1. `metadata.name:` Arbitrary name of the secret. Must match the PVC's `cdi.kubevirt.io/storage.import.secretName:`

1.  `accessKeyId:` Contains the endpoint's key and/or user name. This value **must be base64 encoded** with no extraneous linefeeds. Use `echo -n "xyzzy" | base64` or `printf "xyzzy" | base64` to avoid a trailing linefeed

1.  `secretKey:` the endpoint's secret or password, again **base64 encoded** with no extraneous linefeeds.

### Deploy the API Objects

1. (Optional) Create the endpoint secret in the PVC's namespace:

   `$ kubectl -n <NAMESPACE> create -f endpoint-secret.yaml`

1. Create the persistent volume claim to trigger the import process;

   `$ kubectl -n <NAMESPACE> create -f golden-pvc.yaml`

1. Monitor the cdi-controller:

   `$ kubectl -n <CDI-NAMESPACE> logs cdi-deployment-<RANDOM>`

1. Monitor the importer pod:

   `$ kubectl -n <NAMESPACE> logs importer-<PVC-NAME>` # pvc name is shown in controller log

     _or_

   `kubectl get -n <NAMESPACE> pvc <PVC-NAME> -o yaml | grep "storage.import.pod.phase:"` # to see the status of the importer pod triggered by the pvc


### Cloning VM Images

Cloning is achieved by creating a new PVC with the 'k8s.io/CloneRequest' annotation indicating the name of the PVC the image is copied from.
Once the controller detects the PVC, it starts two pods (source and target pods) which are responsible for the cloning of the image from one PVC to another using a unix socket that is created on the host itself. 
When the cloning is completed, the PVC which the image was copied to, is assigned with the 'k8s.io/CloneOf' annotation to indicate cloning completion.
The copied VM image can be used by a new pod only after the cloning process is completed.

The two cloning pods must execute on the same node. Pod affinity is used to enforce this requirement; however, the cluster also needs to be configured to delay volume binding until pod scheduling has completed.

In Kubernetes 1.9 and older export KUBE_FEATURE_GATES before bringing up the cluster:
`$ export KUBE_FEATURE_GATES="PersistentLocalVolumes=true,VolumeScheduling=true,MountPropagation=true"`

These features default to true in Kubernetes 1.10 and later and thus do not need to be set.

Regardless of the Kubernetes version, a storage class with _volumeBindingMode_ set to "WaitForFirstConsumer" needs to be created. Eg:
```
   kind: StorageClass
   apiVersion: storage.k8s.io/v1
   metadata:
     name: <local-storage-name>
   provisioner: kubernetes.io/no-provisioner
   volumeBindingMode: WaitForFirstConsumer
```


### Start Cloning Images

From the [example manifests](./manifests/example) you copied earlier, the necessary file is:
- target-pvc.yaml

###### Edit target-pvc.yaml:
1.  `k8s.io/CloneRequest:` The name of the PVC we copy the image from (including its namespace). For example: "source-ns/golden-pvc".
2. add the name of the storage class which defines `volumeBindingMode` per above. Note, this is not required in Kubernetes 1.10 and later.

### Deploy the API Object
1. (Optional) Create the namespace where the target PVC will be deployed:

   `$ kubectl create ns <TARGET-NAMESPACE>`

1. Deploy the target PVC:

   `$ kubectl -n <TARGET-NAMESPACE> create -f target-pvc.yaml`

1. Monitor the cloning pods:

   `$ kubectl -n <SOURCE-NAMESPACE> logs <clone-source-pod-name>`

   `$ kubectl -n <TARGET-NAMESPACE> logs <clone-target-pod-name>`

1. Check the target PVC for 'k8s.io/CloneOf' annotation:

   `$ kubectl -n <TARGET-NAMESPACE> get pvc <target-pvc-name> -o yaml`


### Security Configurations

#### RBAC Roles

CDI runs under a custom ServiceAccount (cdi-sa) and uses the [Kubernetes RBAC model](https://kubernetes.io/docs/admin/authorization/rbac/) to apply an application specific custom ClusterRole with rules to properly access needed resources such as PersistentVolumeClaims and Pods.


#### Protecting VM Image Namespaces

Currently there is no support for automatically implementing [Kubernetes ResourceQuotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/) and Limits on desired namespaces and resources, therefore administrators need to manually lock down all new namespaces from being able to use the StorageClass associated with CDI/Kubevirt and cloning capabilities. This capability of automatically restricting resources is planned for future releases. Below are some examples of how one might achieve this level of resource protection:

- Lock Down StorageClass Usage for Namespace:

```
apiVersion: v1
kind: ResourceQuota
metadata:
  name: protect-mynamespace
spec:
  hard:
    <STORAGE-CLASS-NAME>.storageclass.storage.k8s.io/requests.storage: "0"
```

> NOTE: <STORAGE-CLASS-NAME>.storageclass.storage.k8s.io/persistentvolumeclaims: "0" would also accomplish the same affect by not allowing any pvc requests against the storageclass for this namespace.


- Open Up StorageClass Usage for Namespace:

```
apiVersion: v1
kind: ResourceQuota
metadata:
  name: protect-mynamespace
spec:
  hard:
    <STORAGE-CLASS-NAME>.storageclass.storage.k8s.io/requests.storage: "500Gi"
```

> NOTE: <STORAGE-CLASS-NAME>.storageclass.storage.k8s.io/persistentvolumeclaims: "4" could be used and this would only allow for 4 pvc requests in this namespace, anything over that would be denied.


