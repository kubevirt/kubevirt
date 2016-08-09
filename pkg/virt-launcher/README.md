= virt-launcher =

To run virt-launcher, a running libvirt outside of the container is required.
Further a domain.xml needs to be specified.

In its current state the application reads _domain.xml_ file and tells libvirt
to start a VM with this specification. Then virt-launcher is sitting around and
waiting for system signals. If it gets one, it destroys the starte VM and
exits.

== Running virt-launcher ==

Virt-launcher can bestarted through docker with

```
docker run \
    --rm \
    --volume=/var/run/libvirt/libvirt-sock:/var/run/libvirt/libvirt-sock:Z \
    --volume=$PWD/domain.xml:/domain.xml \
    --detach=false \
    kubevirt/virt-launcher:latest
```

where _$PWD/domain.xml_ needs to be replaced with a path to a valid domain
description file.

On bare metal run

```
./virt-launcher --domain-path /my/domain.xml --libvirt-uri qemu:///system
```

== Development ==

=== Build for local usage ===

Checkout the sources and place them in you _$GOPATH_.
Then install _govendor_ with

```
go get -u github.com/kardianos/govendor
make
```

=== Building a Docker image ===

```
go get -u github.com/kardianos/govendor
make docker
```

=== Starting a VM with kubelet ===

After all docker images are built we can use `manifest/manifest-example.yaml`
to start and stop a VM called `testvm` with kubelet.

Assuming you are in the virt-controller repository, type

```
curl -O https://storage.googleapis.com/kubernetes-release/release/v1.3.4/bin/linux/amd64/kubelet
chmod u+x kubelet
sudo mkdir /var/run/vdsm/manifest
sudo cp domain.xml /var/run/vdsm/manifest/testvm.xml
sudo ./kubelet  --config manifest/manifest-example.yaml --allow-privileged=true --docker-only --sync-frequency=10s
```

The kubelet will scan the manifest folder every 10 seconds for pod definitons
and start them. When deleting the pod definition from the manifest folder the
kubelet stops the pod and the VM gets destroyed.
