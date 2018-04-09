#!/usr/bin/sh.orig

args="$@"
if [ "$args" = "-i -c TERM=xterm /bin/sh" ] ; then
  namespace="$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)"
  name="$(ls /var/run/kubevirt-private/${namespace}/)"
  exec /usr/bin/sh.orig -c "/sock-connector /var/run/kubevirt-private/${namespace}/${name}/virt-serial0"
else
  exec /usr/bin/sh.orig "$@"
fi
