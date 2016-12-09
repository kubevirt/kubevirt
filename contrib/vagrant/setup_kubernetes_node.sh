#/bin/bash -xe

bash ./setup_kubernetes_common.sh

ADVERTISED_MASTER_IP=`dig +short master`

set +e

echo 'Trying to register myself...'
# Skipping preflight checks because of https://github.com/kubernetes/kubeadm/issues/6
kubeadm join --token abcdef.1234567890123456 $ADVERTISED_MASTER_IP --skip-preflight-checks
while [ $? -ne 0 ]; do
  sleep 30
  echo 'Trying to register myself...'
  # Skipping preflight checks because of https://github.com/kubernetes/kubeadm/issues/6
  kubeadm join --token abcdef.1234567890123456 $ADVERTISED_MASTER_IP --skip-preflight-checks
done
