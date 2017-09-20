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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virt_manifest

import (
	"encoding/xml"

	"github.com/jeevatkm/go-model"
	"github.com/libvirt/libvirt-go"
	"k8s.io/apimachinery/pkg/util/errors"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
)

const (
	Type_PersistentVolumeClaim = "PersistentVolumeClaim"
	Type_Network               = "network"
)

type savedDisk struct {
	idx  int
	disk v1.Disk
}

func ExtractPvc(dom *v1.DomainSpec) (*v1.DomainSpec, []savedDisk) {
	specCopy := &v1.DomainSpec{}
	model.Copy(specCopy, dom)

	pvcDisks := []savedDisk{}
	allDisks := []v1.Disk{}

	for idx, disk := range specCopy.Devices.Disks {
		if disk.Type == Type_PersistentVolumeClaim {
			// Save the disk so we can fix it later
			diskCopy := v1.Disk{}
			model.Copy(&diskCopy, disk)
			pvcDisks = append(pvcDisks, savedDisk{disk: diskCopy, idx: idx})

			// Alter the disk record so that libvirt will accept it
			disk.Type = Type_Network
			disk.Source.Protocol = "iscsi"
		}
		allDisks = append(allDisks, disk)
	}
	// Replace the Domain's disks with modified records
	specCopy.Devices.Disks = allDisks

	return specCopy, pvcDisks
}

// This is a simplified version of the domain creation portion of SyncVM. This is intended primarily
// for mapping the VM spec without starting a domain.
func MapVM(con cli.Connection, vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	log := logging.DefaultLogger()

	vmCopy := &v1.VirtualMachine{}
	model.Copy(vmCopy, vm)

	specCopy, pvcs := ExtractPvc(vm.Spec.Domain)

	var wantedSpec api.DomainSpec
	mappingErrs := model.Copy(&wantedSpec, specCopy)

	if len(mappingErrs) > 0 {
		return nil, errors.NewAggregate(mappingErrs)
	}

	wantedSpec.Name = vmCopy.GetObjectMeta().GetName()
	wantedSpec.UUID = string(vmCopy.GetObjectMeta().GetUID())
	xmlStr, err := xml.Marshal(&wantedSpec)
	if err != nil {
		log.Object(vm).Error().Reason(err).Msg("Generating the domain XML failed.")
		return nil, err
	}

	log.Object(vm).Info().V(3).Msg("Domain XML generated.")
	dom, err := con.DomainDefineXML(string(xmlStr))
	if err != nil {
		log.Object(vm).Error().Reason(err).Msg("Defining the VM failed.")
		return nil, err
	}
	log.Object(vm).Info().Msg("Domain defined.")

	defer func() {
		err = dom.Undefine()
		if err != nil {
			log.Object(vm).Warning().Reason(err).Msg("Undefining the domain failed.")
		} else {
			log.Object(vm).Info().Msg("Domain defined.")
		}
	}()

	domXml, err := dom.GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE)
	if err != nil {
		log.Object(vm).Error().Reason(err).Msg("Error retrieving domain XML.")
		return nil, err
	}

	// api.DomainSpec has xml struct tags.
	mappedDom := api.DomainSpec{}
	xml.Unmarshal([]byte(domXml), &mappedDom)
	model.Copy(vmCopy.Spec.Domain, mappedDom)

	// Re-add the PersistentVolumeClaims that were stripped earlier
	for _, pvc := range pvcs {
		vmCopy.Spec.Domain.Devices.Disks[pvc.idx].Type = Type_PersistentVolumeClaim
		vmCopy.Spec.Domain.Devices.Disks[pvc.idx].Source.Protocol = pvc.disk.Source.Protocol
	}

	return vmCopy, nil
}
