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
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type SoundDomainConfigurator struct{}

func (s SoundDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	sound := vmi.Spec.Domain.Devices.Sound

	// Default is to not have any Sound device
	if sound == nil {
		return nil
	}

	model := "ich9"
	if sound.Model == "ac97" {
		model = "ac97"
	}

	soundCards := make([]api.SoundCard, 1)
	soundCards[0] = api.SoundCard{
		Alias: api.NewUserDefinedAlias(sound.Name),
		Model: model,
	}

	domain.Spec.Devices.SoundCards = soundCards
	return nil
}
