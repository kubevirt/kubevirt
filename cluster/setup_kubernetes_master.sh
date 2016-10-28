#/bin/bash -xe

# Example environment variables (set by Vagrantfile)
# export KUBERNETES_MASTER=true
# export VM_IP=192.168.200.2
# export MASTER_IP=$VM_IP
# export NODE_IPS="192.168.200.5"
bash ./setup_kubernetes_common.sh

sed -i s@YOUR_IP_HERE@${MASTER_IP}@ kubernetes/kubernetes-master.yaml
cp kubernetes/kubernetes-master.yaml /etc/kubernetes/manifests/kubernetes-master.yaml

{
yum install -y cockpit cockpit-kubernetes
systemctl start cockpit.socket
systemctl enable cockpit.socket
} &

# Wait for all async jobs, like pulls
wait

systemctl start kubelet
systemctl enable kubelet

set +e

kubectl -s ${MASTER_IP}:8080 version > /dev/null 2>&1
while [ $? -ne 0 ]
do
sleep 60
echo 'Waiting for Kubernetes cluster to become functional...'
kubectl -s ${MASTER_IP}:8080 version > /dev/null 2>&1
done

NFSHOST=192.168.200.3
if ${WITH_LOCAL_NFS:-false}; then
mkdir -p /exports/nfs_clean/share1

chmod 0755 /exports/nfs_clean/share1
chown 36:36 /exports/nfs_clean/share1

echo "/exports/nfs_clean/share1  *(rw,anonuid=36,anongid=36,all_squash,sync,no_subtree_check)" > /etc/exports

systemctl enable nfs-server
systemctl restart nfs-server

fi
