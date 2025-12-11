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
			clock.Timer = append(clock.Timer, rtcToTimer(source.Timer.RTC))
		}
		if source.Timer.PIT != nil {
			clock.Timer = append(clock.Timer, pitToTimer(source.Timer.PIT))
		}
		if source.Timer.KVM != nil {
			clock.Timer = append(clock.Timer, kvmToTimer(source.Timer.KVM))
		}
		if source.Timer.HPET != nil {
			clock.Timer = append(clock.Timer, hpetToTimer(source.Timer.HPET))
		}
		if source.Timer.Hyperv != nil {
			clock.Timer = append(clock.Timer, hypervToTimer(source.Timer.Hyperv))
		}
	}

	return nil
}

func hypervToTimer(source *v1.HypervTimer) api.Timer {
	newTimer := api.Timer{
		Name:    "hypervclock",
		Present: setPresentField(source.Enabled, true),
	}

	return newTimer
}

func hpetToTimer(source *v1.HPETTimer) api.Timer {
	newTimer := api.Timer{
		Name:    "hpet",
		Present: setPresentField(source.Enabled, true),
	}

	newTimer.TickPolicy = string(source.TickPolicy)
	return newTimer
}

func kvmToTimer(source *v1.KVMTimer) api.Timer {
	newTimer := api.Timer{
		Name:    "kvmclock",
		Present: setPresentField(source.Enabled, true),
	}

	return newTimer
}

func pitToTimer(source *v1.PITTimer) api.Timer {
	newTimer := api.Timer{
		Name:    "pit",
		Present: setPresentField(source.Enabled, true),
	}

	newTimer.TickPolicy = string(source.TickPolicy)
	return newTimer
}

func rtcToTimer(source *v1.RTCTimer) api.Timer {
	newTimer := api.Timer{
		Name:    "rtc",
		Present: setPresentField(source.Enabled, true),
	}
	newTimer.Track = string(source.Track)
	newTimer.TickPolicy = string(source.TickPolicy)

	return newTimer
}

func setPresentField(value *bool, defVal bool) *api.YesNoAttr {
	return ptr.To(api.YesNoAttr(ptr.Deref(value, defVal)))
}
