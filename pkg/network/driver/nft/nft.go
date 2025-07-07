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
 * Copyright 2022 Red Hat, Inc.
 *
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
