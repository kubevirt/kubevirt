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
	"strconv"

	"kubevirt.io/kubevirt/pkg/network/cache"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"

	"github.com/vishvananda/netlink"
	k8serrors "k8s.io/apimachinery/pkg/util/errors"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

type Unpluggedpodnic struct {
	network      v1.Network
	handler      netdriver.NetworkHandler
	launcherPID  int
	cacheCreator cacheCreator
	vmId         string
}

func NewUnpluggedpodnic(vmId string, network v1.Network, handler netdriver.NetworkHandler, launcherPID int, cacheCreator cacheCreator) Unpluggedpodnic {
	return Unpluggedpodnic{vmId: vmId, network: network, handler: handler, launcherPID: launcherPID, cacheCreator: cacheCreator}
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
	err = cache.DeleteDomainInterfaceCache(c.cacheCreator, strconv.Itoa(c.launcherPID), c.network.Name)
	if err != nil {
		unplugErrors = append(unplugErrors, err)
	}

	err = cache.DeleteDHCPInterfaceCache(c.cacheCreator, strconv.Itoa(c.launcherPID), podInterfaceName)
	if err != nil {
		unplugErrors = append(unplugErrors, err)
	}

	// the PodInterface cache should be the last one to be cleaned.
	// It should be cleaned as the last step of the cleanup, since it is the indicator the cleanup should be done/not over yet.
	if len(unplugErrors) == 0 {
		err = cache.DeletePodInterfaceCache(c.cacheCreator, c.vmId, c.network.Name)
		if err != nil {
			unplugErrors = append(unplugErrors, err)
		}
	}

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
