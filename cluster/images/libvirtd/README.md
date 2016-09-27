# libvirtd

This is a simple container wrapping libvirtd.


# Purpose
Instead of delivering libvirtd in a package, libvirtd is
now delivered in a container.
When running the container it will _look_ like libvirtd is
running on the host's namespace. But in fact it's running
in a container.

This is similar to Atomic's 'system containers'.

You can use virsh on the host to access it.

To access libvirtd via TCP you still need to export
the correct port.


# Try with k8s

For convenience there is a daemon set definition which
can be used with kubernetes.

Note: make sure that libvirtd is not running on the hosts
of the k8s cluster.

Define the daemon set:

    kubectl create -f libvirtd-ds.yaml

Now test on any host of the cluster:

    virsh capabilities


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
