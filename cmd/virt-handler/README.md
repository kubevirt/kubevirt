= virt-handler =

This process is currently a placeholder

== Running virt-handler ==

Virt-handler can be started through docker with

```
docker run \
    --rm \
    --volume=/var/run/libvirt/libvirt-sock:/var/run/libvirt/libvirt-sock:Z \
    --detach=false \
    kubevirt/virt-handler:latest
```

On bare metal run

```
./virt-handler
```
