#/bin/bash -xe

# Example environment variables (set by Vagrantfile)
# export KUBERNETES_MASTER=true
# export VM_IP=192.168.200.2
# export MASTER_IP=$VM_IP
# export NODE_IPS="192.168.200.5"
bash ./setup_kubernetes_common.sh

cat <<EOT >> /etc/kubernetes/manifests/kubernetes.yaml
apiVersion: v1
kind: Pod
metadata:
  name: kube-master
spec:
  hostNetwork: true
  volumes:
    - name: "etc-kubernetes"
      hostPath:
        path: "/etc/kubernetes"
    - name: "ssl-certs"
      hostPath:
        path: "/usr/share/ca-certificates"
    - name: "var-run-kubernetes"
      hostPath:
        path: "/var/run/kubernetes"
    - name: "etcd-datadir"
      hostPath:
        path: "/var/lib/etcd"
    - name: "usr"
      hostPath:
        path: "/usr"
    - name: "lib64"
      hostPath:
        path: "/lib64"
  containers:
    - name: "etcd"
      image: "b.gcr.io/kuar/etcd:2.1.1"
      args:
        - "--data-dir=/var/lib/etcd"
        - "--advertise-client-urls=http://127.0.0.1:2379"
        - "--listen-client-urls=http://127.0.0.1:2379"
        - "--listen-peer-urls=http://127.0.0.1:2380"
        - "--name=etcd"
      volumeMounts:
        - mountPath: /var/lib/etcd
          name: "etcd-datadir"
    - name: "kube-apiserver"
      image: "b.gcr.io/kuar/kube-apiserver:1.2.0"
      args:
        - "--allow-privileged=true"
        - "--etcd-servers=http://127.0.0.1:2379"
        - "--insecure-bind-address=0.0.0.0"
        - "--service-cluster-ip-range=10.200.20.0/24"
        - "--v=2"
      volumeMounts:
        - mountPath: /etc/kubernetes
          name: "etc-kubernetes"
        - mountPath: /var/run/kubernetes
          name: "var-run-kubernetes"
    - name: "kube-controller-manager"
      image: "b.gcr.io/kuar/kube-controller-manager:1.2.0"
      args:
        - "--master=http://127.0.0.1:8080"
        - "--v=2"
    - name: "kube-scheduler"
      image: "b.gcr.io/kuar/kube-scheduler:1.2.0"
      args:
        - "--master=http://127.0.0.1:8080"
        - "--v=2"
    - name: "kube-proxy"
      image: "b.gcr.io/kuar/kube-proxy:1.2.0"
      args:
        - "--master=http://127.0.0.1:8080"
        - "--v=2"
      securityContext:
        privileged: true
      volumeMounts:
        - mountPath: /etc/kubernetes
          name: "etc-kubernetes"
        - mountPath: /etc/ssl/certs
          name: "ssl-certs"
        - mountPath: /usr
          name: "usr"
        - mountPath: /lib64
          name: "lib64"

EOT

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
