#!/bin/sh
set -e
export TOKEN=`cat /var/run/secrets/kubernetes.io/serviceaccount/token`
export DNS=`cat /etc/resolv.conf |grep -i nameserver|head -n1|cut -d ' ' -f2`
/sbin/syslogd -O /dev/stdout && /docker-entrypoint-orig.sh $@
