/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package thp

import (
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hypervisor/common"
)

func processMadviseCollapse(pid int, addr uintptr, length uint64) (int, error) {
	return common.ProcessMadviseCollapse(pid, addr, length)
}

func collapseInfof(msg string, args ...interface{}) {
	log.Log.V(3).Infof(msg, args...)
}

func collapseWarnf(msg string, args ...interface{}) {
	log.Log.Warningf(msg, args...)
}
