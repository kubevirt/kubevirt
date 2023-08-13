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
	"os"
	"path"
	"strings"

	"kubevirt.io/kubevirt/pkg/util"
)

const (
	sysctlBase            = "/proc/sys"
	NetIPv6Forwarding     = "net/ipv6/conf/all/forwarding"
	NetIPv4Forwarding     = "net/ipv4/ip_forward"
	Ipv4ArpIgnoreAll      = "net/ipv4/conf/all/arp_ignore"
	Ipv4ArpIgnore         = "net/ipv4/conf/%s/arp_ignore"
	PingGroupRange        = "net/ipv4/ping_group_range"
	IPv4RouteLocalNet     = "net/ipv4/conf/%s/route_localnet"
	UnprivilegedPortStart = "net/ipv4/ip_unprivileged_port_start"
)

// Interface is an injectable interface for running sysctl commands.
type Interface interface {
	// GetSysctl returns the value for the specified sysctl setting
	GetSysctl(sysctl string) (string, error)
	// SetSysctl modifies the specified sysctl flag to the new value
	SetSysctl(sysctl string, newVal string) error
}

// New returns a new Interface for accessing sysctl
func New() Interface {
	return &procSysctl{}
}

// procSysctl implements Interface by reading and writing files under /proc/sys
type procSysctl struct {
}

// GetSysctl returns the value for the specified sysctl setting
func (*procSysctl) GetSysctl(sysctl string) (string, error) {
	data, err := os.ReadFile(path.Join(sysctlBase, sysctl))
	if err != nil {
		return "-1", err
	}
	val := strings.Trim(string(data), " \n")
	return val, nil
}

// SetSysctl modifies the specified sysctl flag to the new value
func (*procSysctl) SetSysctl(sysctl string, newVal string) error {
	return util.WriteFileWithNosec(path.Join(sysctlBase, sysctl), []byte(newVal))
}
