#!/bin/sh
set -e

# rpcbind requires the 'rpc' user to drop privileges.
# Bazel RPM layers install packages without running scriptlets,
# so the user that rpcbind's package normally creates is missing.
id rpc >/dev/null 2>&1 || useradd -r -s /sbin/nologin rpc

mkdir -p /exports /var/lib/nfs
touch /var/lib/nfs/etab
echo '/exports *(rw,no_subtree_check,no_root_squash,fsid=0)' >/etc/exports

rpcbind
exportfs -a
rpc.nfsd 8
exec rpc.mountd -F
