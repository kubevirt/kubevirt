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
* Copyright 2023 Red Hat, Inc.
*
 */

package network

import (
	"errors"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"

	"github.com/vishvananda/netlink"
	k8serrors "k8s.io/apimachinery/pkg/util/errors"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

type Unpluggedpodnic struct {
	network v1.Network
	handler netdriver.NetworkHandler
}

func NewUnpluggedpodnic(network v1.Network, handler netdriver.NetworkHandler) Unpluggedpodnic {
	return Unpluggedpodnic{network: network, handler: handler}
}

func (c Unpluggedpodnic) UnplugPhase1() error {
	var unplugErrors []error

	podInterfaceName := namescheme.HashedPodInterfaceName(c.network)
	bridgeName := virtnetlink.GenerateBridgeName(podInterfaceName)
	err := c.delLinkIfExists(bridgeName)
	if err != nil {
		unplugErrors = append(unplugErrors, err)
	}

	// remove extra nic
	dummyIfaceName := virtnetlink.GenerateNewBridgedVmiInterfaceName(podInterfaceName)
	err = c.delLinkIfExists(dummyIfaceName)
	if err != nil {
		unplugErrors = append(unplugErrors, err)
	}

	// remove tap if exists
	tapDeviceName := virtnetlink.GenerateTapDeviceName(podInterfaceName)
	err = c.delLinkIfExists(tapDeviceName)
	if err != nil {
		unplugErrors = append(unplugErrors, err)
	}

	// clean caches
	// TODO remove all three cache files

	return k8serrors.NewAggregate(unplugErrors)
}

func (c Unpluggedpodnic) delLinkIfExists(linkName string) error {
	link, err := c.handler.LinkByName(linkName)
	if err != nil {
		var linkNotFoundErr netlink.LinkNotFoundError
		if errors.As(err, &linkNotFoundErr) {
			return nil
		}
		return err
	}
	return c.handler.LinkDel(link)
}
