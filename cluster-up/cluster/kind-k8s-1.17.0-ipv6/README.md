# K8S 1.17.0 in a Kind cluster, with IPv6 only

Provides a pre-deployed k8s cluster with version 1.17.0 that runs using [kind](https://github.com/kubernetes-sigs/kind). The cluster is completely ephemeral and is recreated on every cluster restart.
The KubeVirt containers are built on the local machine and are then pushed to a registry which is exposed at
`localhost:5000`.

cluster is brought up with ipv6 support but without flannel or multi nic support

## Prerequisits
1. kubectl >= 1.16
1. docker network with ipv6.  
    To get that you'll have to add the following section to /etc/docker/daemon.json:  
    ```
    {
      "ipv6": true,
      "fixed-cidr-v6": "2001:db8:1::/64"
    }
    ```  
    and to fully restart docker (systemctl restart docker)  
    if needed, docker can be tested with:
    `docker run --rm busybox ip a`  
    and make sure you get an ipv6 address  
1. With an IPv6-connected host, you may want the pods to be able to reach the rest of the IPv6 world, too:  
    Enable IPv6 NAT:
    ```console
    # ip6tables -t nat -A POSTROUTING -s 2001:db8:1::/64 -j MASQUERADE
    ```
    (Note that the address 2001:db8:1::/64 has to match fixed-cidr-v6 in /etc/docker/daemon.json)

    You will also need the host to be configured with an IPv6-reachable DNS server.  
    If you don't have one, you can use OpenDNS:
    ```console
    # echo "DNS=2620:119:35::35" >> /etc/systemd/resolved.conf
    # systemctl restart systemd-resolved
    ```

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=kind-k8s-1.17.0-ipv6
export KUBEVIRT_NUM_NODES=2 # master + one node
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster-up/kubectl.sh get nodes
NAME                        STATUS   ROLES    AGE    VERSION
kind-1.17.0-control-plane   Ready    master   105s   v1.14.2
kind-1.17.0-worker          Ready    <none>   71s    v1.14.2
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=kind-k8s-1.17.0
make cluster-down
```

This destroys the whole cluster. 

