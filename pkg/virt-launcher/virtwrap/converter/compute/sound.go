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

package compute

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type SoundDomainConfigurator struct{}

func (s SoundDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	vmiSoundDevice := vmi.Spec.Domain.Devices.Sound
	if vmiSoundDevice == nil {
		return nil
	}

	model := vmiSoundDevice.Model
	switch model {
	case "":
		model = "ich9"
	case "ich9", "ac97":
	default:
		return fmt.Errorf("invalid model: %s", model)
	}

	domain.Spec.Devices.SoundCards = []api.SoundCard{
		{
			Alias: api.NewUserDefinedAlias(vmiSoundDevice.Name),
			Model: model,
		},
	}

	return nil
}
