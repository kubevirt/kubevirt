## Getting Started For Developers

### Download source:

`# in github fork kubevirt/containerized-data-importer to your personal repo`, then:
```
cd $GOPATH/src/
mkdir -p kubevirt.io/containerized-data-importer
go get kubevirt.io/containerized-data-importer
cd kubevirt.io/containerized-data-importer
git remote set-url origin <url-to-your-personal-repo>
git push origin master -f
```

 or

 ```
 cd $GOPATH/src/
 mkdir -p kubevirt.io/kubevirt && cd kubevirt.io/kubevirt
 git clone <your-forked-containerized-data-importer-url>
 cd containerized-data-importer
 git remote add upstream https://kubevirt.io/containerized-data-importer.git
 ```

### Use glide to handle vendoring of dependencies:

Install glide:

`curl https://glide.sh/get | sh`

Then run it from the repo root

`glide install -v`

`glide install` scans imports and resolves missing and unused dependencies. `-v` removes nested vendor and Godeps/_workspace directories.

### Create importer image from source:

```
cd $GOPATH/src/kubevirt.io/containerized-data-importer
make importer
```
which places the binary in _./bin/importer_.
The importer image is pushed to `jcoperh/importer:latest`, and this is where the importer pod pulls the image from.

### Create controller image from source:

```
cd $GOPATH/src/kubevirt.io/containerized-data-importer
make controller
```
which places the binary in _./bin/importer-controller_. The controller image is pushed to `jcoperh/importer-controller:latest`, and this is where the controller pod pulls the image from.

> NOTE: when running the controller in a `local-up-cluster` environment (and in the default namespace) the cluster role binding below was needed to allow the controller pod to list PVCs:
```
kubectl create clusterrolebinding cdi-controller --clusterrole=cluster-admin --serviceaccount=default:default
```

### S3-compatible client setup:

#### AWS S3 cli
$HOME/.aws/credentials
```
[default]
aws_access_key_id = <your-access-key>
aws_secret_access_key = <your-secret>
```

#### Mino cli

$HOME/.mc/config.json:
```
{
        "version": "8",
        "hosts": {
                "s3": {
                        "url": "https://s3.amazonaws.com",
                        "accessKey": "<your-access-key>",
                        "secretKey": "<your-secret>",
                        "api": "S3v4"
                }
        }
}
```
