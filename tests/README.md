= Integration tests =

Integration tests require a running Kubevirt cluster.  Once you have a running
Kubevirt cluster, you can use the `-master` and the `-kubeconfig` flags to
point the tests to the cluster.

== Run them on Vagrant ==

The vagrant environment has an unprotected haproxy in front of the apiserver,
so only `-master` needs to be set to run the tests.

```
cd tests # from the git repo root folder
go test -master=http://192.168.200.2:8184
```

There is a helper script to run this:


```
# from the git repo root folder
hack/build-go.sh functest
```

Or simply

```
make functest
```
