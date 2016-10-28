#/bin/bash -xe

# Example environment variables (set by Vagrantfile)
# export VM_IP=192.168.200.5
# export MASTER_IP=192.168.200.2
bash ./setup_kubernetes_common.sh

sed -i s@YOUR_IP_HERE@${MASTER_IP}@ kubernetes/kubernetes-node.yaml
cp kubernetes/kubernetes-node.yaml /etc/kubernetes/manifests/kubernetes-node.yaml

systemctl start kubelet
systemctl enable kubelet

set +e

kubectl -s ${MASTER_IP}:8080 version 2>&1
while [ $? -ne 0 ]
do
  sleep 60
  echo 'Waiting for Kubernetes cluster to become functional...'
  kubectl -s ${MASTER_IP}:8080 version 2>&1
done

kubectl -s ${MASTER_IP}:8080 get node $(hostname) -o json | jq '.status.conditions[] | select(.reason == "KubeletReady")' -e
while [ $? -ne 0 ]
do
  sleep 10
  echo 'Waiting for myself to become an operational node in kubernetes...'
  kubectl -s ${MASTER_IP}:8080 get node $(hostname) -o json | jq '.status.conditions[] | select(.reason == "KubeletReady")' -e
done
