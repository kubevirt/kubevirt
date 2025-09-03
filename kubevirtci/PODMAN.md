# Use kubevirtci with podman instead of docker

Install podman 3.1+, then run it in docker compatible mode:

## Rootless podman

```
systemctl start --user podman.socket
```

Currently rootless podman is **not** working with the `make cluster-sync`
command, essentially because incoming traffic is coming from the loopback device
instead of eth0.

The current rules - [ssh](https://github.com/kubevirt/kubevirtci/blob/962d90cead28fc2aadcc07388b18d2479b2b6714/cluster-provision/centos8/scripts/vm.sh#L73), [restricted ports](https://github.com/kubevirt/kubevirtci/blob/962d90cead28fc2aadcc07388b18d2479b2b6714/cluster-provision/centos8/scripts/vm.sh#L83) - allow `make cluster-up` to run successfully, but
unfortunately they break the cluster's network connectivity in a subtle way:
image pulling fails because outgoing traffic to ports 22 6443 8443 80 443 30007
30008 31001 30085 is redirected to the VM in the respective node container (i.e.
itself) instead of going to the specified host (e.g. quay.io).

This will use `fuse-overlayfs` as storage layer. If the performance is not
satisfactory, consider running podman as root to use plain `overlayfs2`:

## Rootful podman

In order to use rootful podman by a non root user, we will need to bind podman
to a socket, accessible by the user (as docker does).

Assuming the user is in `wheel` group please do the following (one time):

As root, create a Drop-In file `/etc/systemd/system/podman.socket.d/10-socketgroup.conf`
with the following content:
```
[Socket]
SocketGroup=wheel
ExecStartPost=/usr/bin/chmod 755 /run/podman
```

The 1st line is needed in order to create the socket accessible by the `wheel` group.
2nd line because systemd-tmpfiles recreates the folder as root:root without group reading rights.

Stop `podman.socket` if it is running,
reload the daemon `systemctl daemon-reload` since we changed the systemd settings
and restart it again `systemctl enable --now podman.socket`

As the user add the following to your ~/.bashrc
```
alias podman="podman --remote"
export CONTAINER_HOST=unix:///run/podman/podman.sock
```

Validate it by running `podman run hello-world` as the non root user
and see that as root `podman ps -a` shows the same exited container (or vice versa).

In case you wish to use a custom socket path, change the values of `CONTAINER_HOST`
and `KUBEVIRTCI_PODMAN_SOCKET` accordingly,
i.e `export KUBEVIRTCI_PODMAN_SOCKET="${XDG_RUNTIME_DIR}/podman/podman.sock"`

Tested on fedora 35.

## Resource Adjustments

When working with Podman, you might encounter PID resource constraints. To resolve this issue:

1. Locate and edit your `containers.conf` file (typically in `/usr/share/containers`)
2. Add or modify the PID limit setting:
    ```toml
    [containers]
    # Configure the process ID (PID) limit for containers
    # Options:
    #   Numeric value: Sets specific PID limit (e.g., 2048)
    #   -1: Removes PID limitations entirely
    pids_limit = 2048
    ```
