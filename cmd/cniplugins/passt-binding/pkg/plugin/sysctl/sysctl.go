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
 */

package sysctl

import (
	"fmt"
	"strconv"

	"kubevirt.io/kubevirt/pkg/util/sysctl"
)

type sysControl struct{}

var sysCtl = sysctl.New()

func New() sysControl {
	return sysControl{}
}

func (_ sysControl) IPv4SetPingGroupRange(from, to int) error {
	return sysCtl.SetSysctl(sysctl.PingGroupRange, fmt.Sprintf("%d %d", from, to))
}

func (_ sysControl) IPv4SetUnprivilegedPortStart(port int) error {
	return sysCtl.SetSysctl(sysctl.UnprivilegedPortStart, strconv.Itoa(port))
}
