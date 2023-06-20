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

package link

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
)

// DiscoverByNetwork return the pod interface link of the given network name.
// If link not found, it will try to get the link using the pod interface's ordinal name (net1, net2,...)
// based on the subject network position in the given networks slice.
// If no link is found, a nil link will be returned.
func DiscoverByNetwork(handler driver.NetworkHandler, networks []v1.Network, subjectNetwork v1.Network) (netlink.Link, error) {
	ifaceNames, err := networkInterfaceNames(networks, subjectNetwork)
	if err != nil {
		return nil, err
	}

	return linkByNames(handler, ifaceNames)
}

func networkInterfaceNames(networks []v1.Network, subjectNetwork v1.Network) ([]string, error) {
	ifaceName := namescheme.HashedPodInterfaceName(subjectNetwork)
	ordinalIfaceName := namescheme.OrdinalPodInterfaceName(subjectNetwork.Name, networks)
	if ordinalIfaceName == "" {
		return nil, fmt.Errorf("could not find the pod interface ordinal name for network [%s]", subjectNetwork.Name)
	}

	return []string{ifaceName, ordinalIfaceName}, nil
}

func linkByNames(handler driver.NetworkHandler, names []string) (netlink.Link, error) {
	var errs []string
	for _, name := range names {
		link, err := handler.LinkByName(name)
		if err == nil {
			return link, nil
		}
		var linkNotFoundErr netlink.LinkNotFoundError
		if !errors.As(err, &linkNotFoundErr) {
			errs = append(errs, fmt.Sprintf("could not get link with name %q: %v", name, err))
		}
	}
	if len(errs) == 0 {
		return nil, nil
	}
	return nil, fmt.Errorf(strings.Join(errs, ", "))
}
