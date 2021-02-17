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
 * Copyright 2018 Red Hat, Inc.
 *
 */

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package network

import (
	"fmt"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type SlirpBindMechanism struct {
	vmi       *v1.VirtualMachineInstance
	iface     *v1.Interface
	virtIface *api.Interface
	domain    *api.Domain
}

func (s *SlirpBindMechanism) discoverPodNetworkInterface() error {
	return nil
}

func (s *SlirpBindMechanism) preparePodNetworkInterfaces(queueNumber uint32, launcherPID int) error {
	return nil
}

func (s *SlirpBindMechanism) startDHCP(vmi *v1.VirtualMachineInstance) error {
	return nil
}

func (s *SlirpBindMechanism) decorateConfig() error {
	// remove slirp interface from domain spec devices interfaces
	var foundIfaceModelType string
	ifaces := s.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == s.iface.Name {
			s.domain.Spec.Devices.Interfaces = append(ifaces[:i], ifaces[i+1:]...)
			foundIfaceModelType = iface.Model.Type
			break
		}
	}

	if foundIfaceModelType == "" {
		return fmt.Errorf("failed to find interface %s in vmi spec", s.iface.Name)
	}

	qemuArg := fmt.Sprintf("%s,netdev=%s,id=%s", foundIfaceModelType, s.iface.Name, s.iface.Name)
	if s.iface.MacAddress != "" {
		// We assume address was already validated in API layer so just pass it to libvirt as-is.
		qemuArg += fmt.Sprintf(",mac=%s", s.iface.MacAddress)
	}
	// Add interface configuration to qemuArgs
	s.domain.Spec.QEMUCmd.QEMUArg = append(s.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-device"})
	s.domain.Spec.QEMUCmd.QEMUArg = append(s.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: qemuArg})

	return nil
}

func (s *SlirpBindMechanism) loadCachedInterface(pid, name string) (bool, error) {
	return true, nil
}

func (s *SlirpBindMechanism) loadCachedVIF(pid, name string) (bool, error) {
	return true, nil
}

func (b *SlirpBindMechanism) setCachedVIF(pid, name string) error {
	return nil
}

func (s *SlirpBindMechanism) setCachedInterface(pid, name string) error {
	return nil
}
