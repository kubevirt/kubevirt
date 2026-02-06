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

package network

import (
	"errors"
	"regexp"
	"strings"

	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
)

func UpgradeOrdinalNamingScheme(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	ordinalPattern := regexp.MustCompile(`^tap\d+$`)

	hashedPodNamingScheme := namescheme.CreateHashedNetworkNameScheme(vmi.Spec.Networks)

	for i := range domain.Devices.Interfaces {
		iface := &domain.Devices.Interfaces[i]

		if iface.Target == nil || iface.Target.Dev == "" || iface.Target.Dev == "tap0" {
			continue
		}

		if !ordinalPattern.MatchString(iface.Target.Dev) {
			continue
		}

		netName, err := networkNameFromAlias(iface.Alias)
		if err != nil {
			return err
		}

		hashedPodIfaceName, found := hashedPodNamingScheme[netName]
		if !found {
			return errors.New("could not find pod interface name")
		}

		iface.Target.Dev = tapNameFromPodIfaceName(hashedPodIfaceName)
	}

	return nil
}

func networkNameFromAlias(alias *libvirtxml.DomainAlias) (string, error) {
	const aliasPrefix = "ua-"
	if alias == nil {
		return "", errors.New("alias cannot be nil")
	}

	netName, found := strings.CutPrefix(alias.Name, aliasPrefix)
	if !found {
		return "", errors.New("invalid alias format")
	}

	return netName, nil
}

func tapNameFromPodIfaceName(podIfaceName string) string {
	return "tap" + podIfaceName[3:]
}
