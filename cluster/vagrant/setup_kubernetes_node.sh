#/bin/bash -xe
#
# This file is part of the kubevirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2017 Red Hat, Inc.
#

bash ./setup_kubernetes_common.sh

ADVERTISED_MASTER_IP=`sshpass -p vagrant ssh -oStrictHostKeyChecking=no vagrant@$MASTER_IP hostname -I | cut -d " " -f1`
set +e

echo 'Trying to register myself...'
# Skipping preflight checks because of https://github.com/kubernetes/kubeadm/issues/6
kubeadm join --token abcdef.1234567890123456 $ADVERTISED_MASTER_IP:6443 --skip-preflight-checks
while [ $? -ne 0 ]; do
  sleep 30
  echo 'Trying to register myself...'
  # Skipping preflight checks because of https://github.com/kubernetes/kubeadm/issues/6
  kubeadm join --token abcdef.1234567890123456 $ADVERTISED_MASTER_IP:6443 --skip-preflight-checks
done
