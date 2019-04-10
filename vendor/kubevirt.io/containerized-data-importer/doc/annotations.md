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
Creating a Datavolume that imports data from an http source with kubevirt(the default) contentType:
```yaml
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
metadata:
  name: my-data-volume
spec:
  source:
      http:
         url: "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img"
  contentType: kubevirt
  pvc:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 500Mi
``` 

Creating a Datavolume that imports data from an http source with archive contentType:
```yaml
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
metadata:
  name: import-archive-datavolume
spec:
  source:
      http:
         url: "http://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz"
  contentType: archive
  pvc:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 500Mi
``` 

Creating a Datavolume that imports data from an registry source with kubevirt contentType:
```yaml
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
metadata:
  name: registry-image-datavolume
spec:
  source:
      registry: "docker://kubevirt/fedora-cloud-registry-disk-demo"
  pvc:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
``` 

### None
The none source indicates there is no source to get data from and instead the default action for the contentType should be taken.

#### contentType
There is currently only one contentType that any meaning with a source of None. If the contentType is kubevirt, it will create an empty Virtual Machine image of the size specified in the Datavolume(DV) request. Specifying a source of none and a contentType of archive will not do anything.

#### example
Creating a Datavolume that creates an empty virtual image:
```yaml
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
metadata:
  name: blank-image-datavolume
spec:
  source:
      blank: {}
  pvc:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 500Mi
``` 