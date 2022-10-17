= virt-handler =

This process is currently a placeholder

== Running virt-handler ==

Virt-handler can be started through docker with

```
docker run \
    --rm \
    --volume=/var/run/libvirt/virtqemud-sock:/var/run/libvirt/virtqemud-sock:Z \
    --detach=false \
    kubevirt/virt-handler:latest
```

On bare metal run

```
./virt-handler
```
