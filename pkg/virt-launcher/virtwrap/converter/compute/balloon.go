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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
)

type BalloonDomainConfigurator struct {
	architecture          string
	useVirtioTransitional bool
	useLaunchSecuritySEV  bool
	useLaunchSecurityPV   bool
	freePageReporting     bool
	memBalloonStatsPeriod uint
}

type balloonOption func(*BalloonDomainConfigurator)

func NewBalloonDomainConfigurator(options ...balloonOption) BalloonDomainConfigurator {
	var configurator BalloonDomainConfigurator

	for _, f := range options {
		f(&configurator)
	}

	return configurator
}

func (b BalloonDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	domain.Spec.Devices.Ballooning = &api.MemBalloon{}
	convertV1ToAPIBalloning(
		&vmi.Spec.Domain.Devices,
		domain.Spec.Devices.Ballooning,
		b.architecture,
		b.useVirtioTransitional,
		b.useLaunchSecuritySEV,
		b.useLaunchSecurityPV,
		b.freePageReporting,
		b.memBalloonStatsPeriod,
	)

	return nil
}

func BalloonWithArchitecture(architecture string) balloonOption {
	return func(b *BalloonDomainConfigurator) {
		b.architecture = architecture
	}
}

func BalloonWithUseVirtioTransitional(useVirtioTranslation bool) balloonOption {
	return func(b *BalloonDomainConfigurator) {
		b.useVirtioTransitional = useVirtioTranslation
	}
}

func BalloonWithUseLaunchSecuritySEV(useLaunchSecuritySEV bool) balloonOption {
	return func(b *BalloonDomainConfigurator) {
		b.useLaunchSecuritySEV = useLaunchSecuritySEV
	}
}

func BalloonWithUseLaunchSecurityPV(useLaunchSecurityPV bool) balloonOption {
	return func(b *BalloonDomainConfigurator) {
		b.useLaunchSecurityPV = useLaunchSecurityPV
	}
}

func BalloonWithFreePageReporting(freePageReporting bool) balloonOption {
	return func(b *BalloonDomainConfigurator) {
		b.freePageReporting = freePageReporting
	}
}

func BalloonWithMemBalloonStatsPeriod(memBalloonStatsPeriod uint) balloonOption {
	return func(b *BalloonDomainConfigurator) {
		b.memBalloonStatsPeriod = memBalloonStatsPeriod
	}
}

func convertV1ToAPIBalloning(
	source *v1.Devices,
	ballooning *api.MemBalloon,
	architecture string,
	useVirtioTransitional,
	useLaunchSecuritySEV,
	useLaunchSecurityPV,
	freePageReporting bool,
	memBalloonStatsPeriod uint,
) {
	if source != nil && source.AutoattachMemBalloon != nil && !*source.AutoattachMemBalloon {
		ballooning.Model = "none"
		ballooning.Stats = nil
	} else {
		ballooning.Model = virtio.InterpretTransitionalModelType(&useVirtioTransitional, architecture)
		if memBalloonStatsPeriod != 0 {
			ballooning.Stats = &api.Stats{Period: memBalloonStatsPeriod}
		}
		if useLaunchSecuritySEV || useLaunchSecurityPV {
			ballooning.Driver = &api.MemBalloonDriver{
				IOMMU: "on",
			}
		}
		ballooning.FreePageReporting = boolToOnOff(&freePageReporting, false)
	}
}

func boolToOnOff(value *bool, defaultOn bool) string {
	return boolToString(value, defaultOn, "on", "off")
}
