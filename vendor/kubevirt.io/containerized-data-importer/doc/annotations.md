# Sources and contentType
All annotations associated with Containerized Data Importer (CDI) have a prefix of: cdi.kubevirt.io. This denotes that the annotations are meant to be consumed by the CDI controller.

## Source
Source describes the type of data source CDI will be collecting the data from. Based on the value of source, additional annotations may be required to successfully import the data. The full annotation for source is: cdi.kubevirt.io/storage.import.source. The following values are currently available:
* http
* S3
* registry
* none (don't import, but create data based on the contentType annotation)

### http, s3 and registry
The http, s3 and registry sources require an additional annotation to describe the end point CDI needs to connect to. The annotation is cdi.kubevirt.io/storage.import.endpoint. If the end point requires authentication one can add an optional annotation to point to a Kubernetes Secret to get authentication information from. This annotation is: cdi.kubevirt.io/storage.import.secretName. If the source annotation is missing it will default to "http".

#### contentType
There is an additional annotation that determines the content type of the http/s3 source, the content type can be one of the following:
* kubevirt (Virtual Machine image)
* archive (tar archive)
If the contentType is missing, it is defaulted to kubevirt.

#### examples
Creating a PVC that imports data from an http source with kubevirt contentType:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: "example-pvc"
  labels:
    app: containerized-data-importer
  annotations:
    cdi.kubevirt.io/storage.import.source: "http" #defaults to http if missing or invalid
    cdi.kubevirt.io/storage.contentType: "kubevirt" #defaults to kubevirt if missing or invalid.
    cdi.kubevirt.io/storage.import.endpoint: "https://www.source.example/path/of/data" # http or https is supported
    cdi.kubevirt.io/storage.import.secretName: "" # Optional. The name of the secret containing credentials for the end point
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi # Request a size that is large enough to accept the data from the source, including conversion
  # Optional: Set the storage class or omit to accept the default
  # storageClassName: local
``` 

Creating a PVC that imports data from an http source with archive contentType:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: "example-pvc"
  labels:
    app: containerized-data-importer
  annotations:
    cdi.kubevirt.io/storage.import.source: "http" #defaults to http if missing or invalid
    cdi.kubevirt.io/storage.contentType: "archive" #defaults to kubevirt if missing or invalid.
    cdi.kubevirt.io/storage.import.endpoint: "http://www.source.example/path/of/data.tar" # http or https is supported
    cdi.kubevirt.io/storage.import.secretName: "" # Optional. The name of the secret containing credentials for the end point
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi # Request a size that is large enough to accept the data from the source, including conversion
  # Optional: Set the storage class or omit to accept the default
  # storageClassName: local
``` 

Creating a PVC that imports data from an registry source with kubevirt contentType:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: "example-pvc"
  labels:
    app: containerized-data-importer
  annotations:
    cdi.kubevirt.io/storage.import.source: "registry" #defaults to http if missing or invalid
    cdi.kubevirt.io/storage.import.contentType: "kubevirt" #defaults to kubevirt if missing or invalid.
    cdi.kubevirt.io/storage.import.endpoint: "docker://registry:5000/fedora" # docker, oci
    cdi.kubevirt.io/storage.import.secretName: "" # Optional. The name of the secret containing credentials for the end point
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi # Request a size that is large enough to accept the data from the source, including conversion
  # Optional: Set the storage class or omit to accept the default
  # storageClassName: local
``` 

### None
The none source indicates there is no source to get data from and instead a default action should be taken based on the contentType.

#### contentType
There is currently only one contentType that any meaning with a source of None. If the contentType is kubevirt, it will create an empty Virtual Machine image of the size specified in the PersistentVolumeClaim(PVC) request. Specifying a source of none and a contentType of archive will not do anything.

#### example
Creating a PVC that creates an empty virtual machine disk:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: "example-pvc"
  labels:
    app: containerized-data-importer
  annotations:
    cdi.kubevirt.io/storage.import.source: "none"
    cdi.kubevirt.io/storage.contentType: "kubevirt" #defaults to kubevirt if missing or invalid.
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi # Request a size that is large enough to accept the data from the source, including conversion
  # Optional: Set the storage class or omit to accept the default
  # storageClassName: local
``` 


