#/bin/bash -xe

# Example environment variables (set by Vagrantfile)
# export VM_IP=192.168.200.5
# export MASTER_IP=192.168.200.2
bash ./setup_kubernetes_common.sh

# TODO yaml
cat <<EOT >> /etc/kubernetes/manifests/node.yaml
apiVersion: v1
kind: Pod
metadata:
  name: kube-node
spec:
  hostNetwork: true
  volumes:
    - name: "etc-kubernetes"
      hostPath:
        path: "/etc/kubernetes"
    - name: "ssl-certs"
      hostPath:
        path: "/usr/share/ca-certificates"
    - name: "usr"
      hostPath:
        path: "/usr"
    - name: "lib64"
      hostPath:
        path: "/lib64"
  containers:
    - name: "kube-proxy"
      image: "b.gcr.io/kuar/kube-proxy:1.2.0"
      args:
        - "--master=http://${MASTER_IP}:8080"
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
