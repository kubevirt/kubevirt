#/bin/bash -xe
#
# This file is part of the KubeVirt project
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
master_ip=$1

bash /vagrant/cluster/vagrant-kubernetes/setup_common.sh

ADVERTISED_MASTER_IP=$(sshpass -p vagrant ssh -oStrictHostKeyChecking=no vagrant@$master_ip hostname -I | cut -d " " -f1)
set +e

echo 'Trying to register myself...'
# Skipping preflight checks because of https://github.com/kubernetes/kubeadm/issues/6
kubeadm join --token abcdef.1234567890123456 $ADVERTISED_MASTER_IP:6443 --ignore-preflight-errors=all --discovery-token-unsafe-skip-ca-verification=true
while [ $? -ne 0 ]; do
    sleep 30
    echo 'Trying to register myself...'
    # Skipping preflight checks because of https://github.com/kubernetes/kubeadm/issues/6
    kubeadm join --token abcdef.1234567890123456 $ADVERTISED_MASTER_IP:6443 --ignore-preflight-errors=all --discovery-token-unsafe-skip-ca-verification=true
done

echo -e "\033[0;32m Deployment was successful!\033[0m"
