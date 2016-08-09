= virt-launcher =

To run virt-launcher, a running libvirt outside of the container is required.
Further a domain.xml needs to be specified.

In its current state the application reads _domain.xml_ file and tells libvirt
to start a VM with this specification. Then virt-launcher is sitting around
and waiting for system signals. If it gets one, it destroys the starte VM and
exits.

== Running virt-launcher ==

Virt-launcher can bestarted through docker with

```
docker run \
    --volume=/var/run/libvirt/libvirt-sock:/var/run/libvirt/libvirt-sock:Z \
    --volume=/my/domain/domain.xml:/domain.xml \
    --name virt-launcher \
    --detach=false \
    virt-launcher:latest
```

On bare metal run

```
./virt-launcher --domain-path /my/domain.xml --libvirt-uri qemu:///system
```

== Development ==

Checkout the sources and place them in you _$GOPATH_.
Then install _govendor_ with

```
go get -u github.com/kardianos/govendor

```

Finally build the application with

```
make
```
