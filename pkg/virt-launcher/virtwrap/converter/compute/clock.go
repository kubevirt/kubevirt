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
	"strconv"

	"k8s.io/utils/ptr"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type ClockDomainConfigurator struct{}

func (c ClockDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Clock != nil {
		clock := vmi.Spec.Domain.Clock
		newClock := &api.Clock{}
		err := convertV1ClockToAPIClock(clock, newClock)
		if err != nil {
			return err
		}
		domain.Spec.Clock = newClock
	}

	// Make use of the tsc frequency topology hint
	if topology.IsManualTSCFrequencyRequired(vmi) {
		freq := *vmi.Status.TopologyHints.TSCFrequency
		clock := domain.Spec.Clock
		if clock == nil {
			clock = &api.Clock{}
		}
		clock.Timer = append(clock.Timer, api.Timer{Name: "tsc", Frequency: strconv.FormatInt(freq, 10)})
		domain.Spec.Clock = clock
	}

	return nil
}

func convertV1ClockToAPIClock(source *v1.Clock, clock *api.Clock) error {
	if source.UTC != nil {
		clock.Offset = "utc"
		if source.UTC.OffsetSeconds != nil {
			clock.Adjustment = strconv.Itoa(*source.UTC.OffsetSeconds)
		} else {
			clock.Adjustment = "reset"
		}
	} else if source.Timezone != nil {
		clock.Offset = "timezone"
		clock.Timezone = string(*source.Timezone)
	}

	if source.Timer != nil {
		if source.Timer.RTC != nil {
			newTimer := api.Timer{Name: "rtc"}
			newTimer.Track = string(source.Timer.RTC.Track)
			newTimer.TickPolicy = string(source.Timer.RTC.TickPolicy)
			setPresentField(&newTimer, source.Timer.RTC.Enabled, true)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.PIT != nil {
			newTimer := api.Timer{Name: "pit"}
			setPresentField(&newTimer, source.Timer.PIT.Enabled, true)
			newTimer.TickPolicy = string(source.Timer.PIT.TickPolicy)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.KVM != nil {
			newTimer := api.Timer{Name: "kvmclock"}
			setPresentField(&newTimer, source.Timer.KVM.Enabled, true)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.HPET != nil {
			newTimer := api.Timer{Name: "hpet"}
			setPresentField(&newTimer, source.Timer.HPET.Enabled, true)
			newTimer.TickPolicy = string(source.Timer.HPET.TickPolicy)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.Hyperv != nil {
			newTimer := api.Timer{Name: "hypervclock"}
			setPresentField(&newTimer, source.Timer.Hyperv.Enabled, true)
			clock.Timer = append(clock.Timer, newTimer)
		}
	}

	return nil
}

func setPresentField(timer *api.Timer, value *bool, defVal bool) {
	present := api.YesNoAttr(ptr.Deref(value, defVal))
	timer.Present = ptr.To(present)
}
