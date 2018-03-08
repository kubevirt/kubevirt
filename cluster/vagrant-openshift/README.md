# OpenShift 3.9.0-alpha.4 in vagrant VM

Start vagrant VM and deploy OpenShift Origin with version 3.9.0-alpha.4 on it.
It will deploy OpenShift only first time when you start a VM.

## Bringing the cluster up

```bash
export PROVIDER=vagrant-openshift
export VAGRANT_NUM_NODES=1
make cluster-up
```

If you want to get access to OpenShift web console you will need to add line to `/etc/hosts`
```bash
echo "127.0.0.1 node01" >> /etc/hosts
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME      STATUS    ROLES     AGE       VERSION
master    Ready     master    8m        v1.9.1+a0ce1bc657
node0     Ready     <none>    6m        v1.9.1+a0ce1bc657
```

## Bringing the cluster down

```bash
export PROVIDER=vagrant-openshift
make cluster-down
```

It will shutdown vagrant VM without destroy it.
