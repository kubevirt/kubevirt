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

package virtwrap

import (
	"encoding/xml"
	"fmt"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

func hotplugVirtioInterface(vmi *v1.VirtualMachineInstance, converterContext *converter.ConverterContext, dom cli.VirDomain) error {
	ifaceNamingScheme := namescheme.CreateNetworkNameScheme(vmi.Spec.Networks)
	for _, network := range netvmispec.NetworksToHotplugWhosePodIfacesAreReady(vmi) {
		log.Log.Infof("will hot plug %s", network.Name)
		podIfaceName, wasFound := ifaceNamingScheme[network.Name]
		if !wasFound {
			return fmt.Errorf("could not find the pod interface name for network [%s]", network.Name)
		}

		ifaceXML, err := xml.Marshal(converter.VirtIODomainInterfaceSpec(converterContext, network.Name, virtnetlink.GenerateTapDeviceName(podIfaceName)))
		if err != nil {
			log.Log.Warningf("failed to marshall the domain interface spec for hotplugging the %s attachment", network.Name)
			continue
		}

		if err := dom.AttachDevice(strings.ToLower(string(ifaceXML))); err != nil {
			log.Log.Reason(err).Errorf("libvirt failed to attach interface %s: %v", network.Name, err)
			return err
		}
	}
	return nil
}
