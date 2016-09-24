# libvirtd

This is a simple container wrapping libvirtd.
Once defined as a daemon set, it will be accessible on any host.
Intended to look like it's installed locally.
So similar like 'system containers'.

You can use virsh on the host to access it.

To access libvirtd via TCP you still need to export
the correct port.

# Try without k8s

Note: Make sure to not run libvirtd on the host.

Start the container

    docker run \
      --name libvirtd \
      --rm \
      --net=host \
      --pid=host \
      --user=root \
      --privileged \
      -v /etc/libvirt:/etc/libvirt:Z \
      -v /var/run/libvirt:/var/run/libvirt:Z \
      -v /var/lib/libvirt:/var/lib/libvirt:Z \
      -v /var/log/libvirt:/var/log/libvirt:Z \
      -v /sys:/sys:Z \
      -v /:/host:Z \
      -it fabiand/libvirtd:latest

Now, to verify, run, on the host:

    virsh capabilities
