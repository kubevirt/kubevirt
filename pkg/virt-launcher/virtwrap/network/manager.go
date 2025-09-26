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
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

func Sync(
	domain *api.Domain,
	oldSpec *api.DomainSpec,
	dom cli.VirDomain,
	vmi *v1.VirtualMachineInstance,
	domainAttachments map[string]string,
) error {
	if !vmi.IsRunning() {
		return nil
	}

	networkConfigurator := netsetup.NewVMNetworkConfigurator(vmi, cache.CacheCreator{}, netsetup.WithDomainAttachments(domainAttachments))
	networkInterfaceManager := newVirtIOInterfaceManager(dom, networkConfigurator)
	if err := networkInterfaceManager.hotplugVirtioInterface(vmi, &api.Domain{Spec: *oldSpec}, domain); err != nil {
		return err
	}
	if err := networkInterfaceManager.hotUnplugVirtioInterface(vmi, &api.Domain{Spec: *oldSpec}); err != nil {
		return err
	}
	if err := networkInterfaceManager.updateDomainLinkState(&api.Domain{Spec: *oldSpec}, domain); err != nil {
		return err
	}

	return nil
}
