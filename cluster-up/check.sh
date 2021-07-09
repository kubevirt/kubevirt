#!/usr/bin/env bash
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
# Copyright 2019 Red Hat, Inc.
#

set -e
if [ ! -c /dev/kvm ]; then
	echo "[ERR ] missing /dev/kvm"
else
	echo "[ OK ] found /dev/kvm"
fi

KVM_ARCH=""
KVM_NESTED="unknown"
if [ -f "/sys/module/kvm_intel/parameters/nested" ]; then
	KVM_NESTED=$( cat /sys/module/kvm_intel/parameters/nested )
	KVM_ARCH="intel"
elif [ -f "/sys/module/kvm_amd/parameters/nested" ]; then
	KVM_NESTED=$( cat /sys/module/kvm_amd/parameters/nested )
	KVM_ARCH="amd"
fi

function is_enabled() {
	if [ "$1" == "1" ]; then
		return 0
	fi
	if [ "$1" == "Y" ] || [ "$1" == "y"]; then
		return 0
	fi
	return 1
}

if is_enabled "$KVM_NESTED"; then
	echo "[ OK ] $KVM_ARCH nested virtualization enabled"
else
	echo "[ERR ] $KVM_ARCH nested virtualization not enabled"
fi
