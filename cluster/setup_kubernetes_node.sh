#/bin/bash -xe

# Example environment variables (set by Vagrantfile)
# export VM_IP=192.168.200.5
# export MASTER_IP=192.168.200.2
bash ./setup_kubernetes_common.sh

set +e

echo 'Trying to register myself...'
kubeadm join --token abcdef.1234567890123456 $MASTER_IP
while [ $? -ne 0 ]
do
  sleep 30
  echo 'Trying to register myself...'
  kubeadm join --token abcdef.1234567890123456 $MASTER_IP
done
