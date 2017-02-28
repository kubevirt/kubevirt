#/bin/bash -xe

export KUBERNETES_MASTER=true
bash ./setup_kubernetes_common.sh

# Cockpit with kubernetes plugin
yum install -y cockpit cockpit-kubernetes
systemctl enable cockpit.socket && systemctl start cockpit.socket

# Create the master
kubeadm init --pod-network-cidr=10.244.0.0/16 --token abcdef.1234567890123456 --use-kubernetes-version v1.4.5

set +e

kubectl -s 127.0.0.1:8080 version
while [ $? -ne 0 ]; do
  sleep 60
  echo 'Waiting for Kubernetes cluster to become functional...'
  kubectl -s 127.0.0.1:8080 version
done

set -e

if [ "$NETWORK_PROVIDER" == "weave" ]; then 
  kubectl apply -s 127.0.0.1:8080 -f https://github.com/weaveworks/weave/releases/download/v1.9.0/weave-daemonset.yaml
else
  kubectl create -s 127.0.0.1:8080 -f kube-$NETWORK_PROVIDER.yaml
fi

# Allow scheduling pods on master
# Ignore retval because it might not be dedicated already
kubectl -s 127.0.0.1:8080 taint nodes --all dedicated- || :

mkdir -p /exports/share1

chmod 0755 /exports/share1
chown 36:36 /exports/share1

echo "/exports/share1  *(rw,anonuid=36,anongid=36,all_squash,sync,no_subtree_check)" > /etc/exports

systemctl enable nfs-server && systemctl start nfs-server
