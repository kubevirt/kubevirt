#!/bin/bash
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


echo "Deploying Storage..."
vagrant rsync master # if you do not use NFS

# Gluster
vagrant ssh master -c "sudo /sbin/iptables -A INPUT -p tcp -m state --state NEW -m tcp --dport 24007 -j ACCEPT"
vagrant ssh master -c "sudo /sbin/iptables -A INPUT -p tcp -m state --state NEW -m tcp --dport 24008 -j ACCEPT"
vagrant ssh master -c "sudo /sbin/iptables -A INPUT -p tcp -m state --state NEW -m tcp --dport 2222 -j ACCEPT"
vagrant ssh master -c "sudo /sbin/iptables -A INPUT -p tcp -m state --state NEW -m multiport --dports 49152:49664 -j ACCEPT"
vagrant ssh master -c "sudo /sbin/iptables -A INPUT -p tcp -m state --state NEW -m tcp --dport 24010 -j ACCEPT"
vagrant ssh master -c "sudo /sbin/iptables -A INPUT -p tcp -m state --state NEW -m tcp --dport 3260 -j ACCEPT"
vagrant ssh master -c "sudo /sbin/iptables -A INPUT -p tcp -m state --state NEW -m tcp --dport 111 -j ACCEPT"
vagrant ssh master -c "cd /vagrant && sudo storage/gluster-deploy.sh"
