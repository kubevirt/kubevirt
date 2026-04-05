/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package nft

import (
	"fmt"
	"os/exec"
)

type NFTBin struct{}

type IPFamily string

const (
	IPv4 IPFamily = "ip"
	IPv6 IPFamily = "ip6"
)

const (
	nftBin = "nft"
)

func (n NFTBin) AddTable(family IPFamily, name string) error {
	cmd := exec.Command(nftBin, "add", "table", string(family), name)
	return execute(cmd)
}

func (n NFTBin) AddChain(family IPFamily, table, name string, chainspec ...string) error {
	args := append([]string{"add", "chain", string(family), table, name}, chainspec...)
	cmd := exec.Command(nftBin, args...)
	return execute(cmd)
}

func (n NFTBin) AddRule(family IPFamily, table, chain string, rulespec ...string) error {
	args := append([]string{"add", "rule", string(family), table, chain}, rulespec...)
	cmd := exec.Command(nftBin, args...)
	return execute(cmd)
}

func execute(cmd *exec.Cmd) error {
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s, error: %v", string(output), err)
	}
	return nil
}
