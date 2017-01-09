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

    virsh -c "qemu+tcp://127.0.0.1/system" capabilities


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
      -v /:/host:Z \
      -it kubevirt/libvirtd:latest

Now, to verify, run, on the host:

    virsh capabilities

# Environment Variables

These environment variables can be passed into the container

* LIBVIRTD_DEFAULT_NETWORK_DEVICE: Set it to an existing device
  to let the default network point to it.

# Notes

Considerations that need to be taken into account:

* The D-Bus socket is not exposed inside the container
  so firewalld cannot be notified of changes (also
  not every host system uses firewalld) so the following
  ports might need to be allowed in if iptables is not
  accepting input by default:
  - TCP 16509
  - TCP 5900->590X (depending on Spice/VNC settings of guest)
