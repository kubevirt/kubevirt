/*
Copyright 2015 The Kubernetes Authors.
Copyright 2020 The KubeVirt Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Originally copied from https://github.com/kubernetes/kubernetes/blob/d8695d06b7191db56ebbbc0340da263833c9bb6f/pkg/util/sysctl/sysctl.go
*/

package sysctl

import (
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/pkg/util"
)

const (
	sysctlBase        = "/proc/sys"
	NetIPv6Forwarding = "net/ipv6/conf/all/forwarding"
	NetIPv4Forwarding = "net/ipv4/ip_forward"
	Ipv4ArpIgnoreAll  = "net/ipv4/conf/all/arp_ignore"
)

// Interface is an injectable interface for running sysctl commands.
type Interface interface {
	// GetSysctl returns the value for the specified sysctl setting
	GetSysctl(sysctl string) (int, error)
	// SetSysctl modifies the specified sysctl flag to the new value
	SetSysctl(sysctl string, newVal int) error
}

// New returns a new Interface for accessing sysctl
func New() Interface {
	return &procSysctl{}
}

// procSysctl implements Interface by reading and writing files under /proc/sys
type procSysctl struct {
}

// GetSysctl returns the value for the specified sysctl setting
func (*procSysctl) GetSysctl(sysctl string) (int, error) {
	data, err := ioutil.ReadFile(path.Join(sysctlBase, sysctl))
	if err != nil {
		return -1, err
	}
	val, err := strconv.Atoi(strings.Trim(string(data), " \n"))
	if err != nil {
		return -1, err
	}
	return val, nil
}

// SetSysctl modifies the specified sysctl flag to the new value
func (*procSysctl) SetSysctl(sysctl string, newVal int) error {
	return util.WriteFileWithNosec(path.Join(sysctlBase, sysctl), []byte(strconv.Itoa(newVal)))
}
