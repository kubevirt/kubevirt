/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package netns

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
)

type NetNS struct {
	nspath string
}

func New(pid int) NetNS {
	return NetNS{nspath: fmt.Sprintf("/proc/%d/ns/net", pid)}
}

func (n NetNS) Do(f func() error) error {
	netns, err := ns.GetNS(n.nspath)
	if err != nil {
		return fmt.Errorf("failed to fetch network namespace object: %v", err)
	}

	return netns.Do(func(_ ns.NetNS) error {
		return f()
	})
}
