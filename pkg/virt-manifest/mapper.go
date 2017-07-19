/*
 * This file is part of the kubevirt project
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
	"github.com/jeevatkm/go-model"
	"github.com/libvirt/libvirt-go"
	"github.com/libvirt/libvirt-go-xml"
	kubeapi "k8s.io/client-go/pkg/api"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/designer"
	"kubevirt.io/kubevirt/pkg/inspector"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

// This is a simplified version of the domain creation portion of SyncVM. This is intended primarily
// for mapping the VM spec without starting a domain.
func MapVM(con virtwrap.Connection, vm *v1.VM) (*v1.VM, error) {
	log := logging.DefaultLogger()

	vmCopy := &v1.VM{}
	model.Copy(vmCopy, vm)

	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		panic(err)
	}

	domdesigner := designer.NewDomainDesigner(restClient, kubeapi.NamespaceDefault)
	err = domdesigner.ApplySpec(vm)

	xmlStr, err := domdesigner.Domain.Marshal()
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

	var newcfg libvirtxml.Domain
	err = newcfg.Unmarshal(domXml)
	if err != nil {
		log.Object(vm).Error().Reason(err).Msg("Error parsing domain XML.")
		return nil, err
	}

	err = inspector.ApplyConfig(&newcfg, vm.Spec.Domain)
	if err != nil {
		log.Object(vm).Error().Reason(err).Msg("Error inspecting domain XML.")
		return nil, err
	}

	return vmCopy, nil
}
