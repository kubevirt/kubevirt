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

package compute_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("Clock Domain Configurator", func() {
	It("should not set clock when clock is unspecified on the VMI", func() {
		vmi := libvmi.New()

		var domain api.Domain

		Expect(compute.ClockDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	DescribeTable("should configure the domain clock from the VMI spec",
		func(vmi *v1.VirtualMachineInstance, expectedClock *api.Clock) {
			var domain api.Domain

			Expect(compute.ClockDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithClock(expectedClock)
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("timezone offset",
			libvmi.New(libvmi.WithClock(v1.Clock{
				ClockOffset: v1.ClockOffset{
					Timezone: pointer.P(v1.ClockOffsetTimezone("America/New_York")),
				},
			})),
			&api.Clock{
				Offset:   "timezone",
				Timezone: "America/New_York",
			},
		),
		Entry("utc offset with adjustment=reset when no offset seconds",
			libvmi.New(libvmi.WithClock(v1.Clock{
				ClockOffset: v1.ClockOffset{
					UTC: &v1.ClockOffsetUTC{},
				},
			})),
			&api.Clock{
				Offset:     "utc",
				Adjustment: "reset",
			},
		),
		Entry("utc offset with seconds adjustment",
			libvmi.New(libvmi.WithClock(v1.Clock{
				ClockOffset: v1.ClockOffset{
					UTC: &v1.ClockOffsetUTC{
						OffsetSeconds: pointer.P(3600),
					},
				},
			})),
			&api.Clock{
				Offset:     "utc",
				Adjustment: "3600",
			},
		),
		Entry("RTC timer",
			libvmi.New(libvmi.WithClock(v1.Clock{
				Timer: &v1.Timer{
					RTC: &v1.RTCTimer{
						Track:      v1.TrackGuest,
						TickPolicy: v1.RTCTickPolicyCatchup,
						Enabled:    pointer.P(true),
					},
				},
			})),
			&api.Clock{
				Timer: []api.Timer{
					{Name: "rtc", Track: "guest", TickPolicy: "catchup", Present: "yes"},
				},
			},
		),
		Entry("PIT timer with disabled state",
			libvmi.New(libvmi.WithClock(v1.Clock{
				Timer: &v1.Timer{
					PIT: &v1.PITTimer{
						TickPolicy: v1.PITTickPolicyDelay,
						Enabled:    pointer.P(false),
					},
				},
			})),
			&api.Clock{
				Timer: []api.Timer{
					{Name: "pit", TickPolicy: "delay", Present: "no"},
				},
			},
		),
		Entry("KVM timer",
			libvmi.New(libvmi.WithClock(v1.Clock{
				Timer: &v1.Timer{
					KVM: &v1.KVMTimer{},
				},
			})),
			&api.Clock{
				Timer: []api.Timer{
					{Name: "kvmclock", Present: "yes"},
				},
			},
		),
		Entry("HPET timer",
			libvmi.New(libvmi.WithClock(v1.Clock{
				Timer: &v1.Timer{
					HPET: &v1.HPETTimer{
						TickPolicy: v1.HPETTickPolicyDelay,
						Enabled:    pointer.P(true),
					},
				},
			})),
			&api.Clock{
				Timer: []api.Timer{
					{Name: "hpet", TickPolicy: "delay", Present: "yes"},
				},
			},
		),
		Entry("Hyperv timer",
			libvmi.New(libvmi.WithClock(v1.Clock{
				Timer: &v1.Timer{
					Hyperv: &v1.HypervTimer{
						Enabled: pointer.P(true),
					},
				},
			})),
			&api.Clock{
				Timer: []api.Timer{
					{Name: "hypervclock", Present: "yes"},
				},
			},
		),
		Entry("all timers together",
			libvmi.New(libvmi.WithClock(v1.Clock{
				ClockOffset: v1.ClockOffset{
					UTC: &v1.ClockOffsetUTC{},
				},
				Timer: &v1.Timer{
					RTC:    &v1.RTCTimer{Track: v1.TrackWall, TickPolicy: v1.RTCTickPolicyDelay},
					PIT:    &v1.PITTimer{TickPolicy: v1.PITTickPolicyCatchup},
					KVM:    &v1.KVMTimer{Enabled: pointer.P(true)},
					HPET:   &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyCatchup},
					Hyperv: &v1.HypervTimer{},
				},
			})),
			&api.Clock{
				Offset:     "utc",
				Adjustment: "reset",
				Timer: []api.Timer{
					{Name: "rtc", Track: "wall", TickPolicy: "delay", Present: "yes"},
					{Name: "pit", TickPolicy: "catchup", Present: "yes"},
					{Name: "kvmclock", Present: "yes"},
					{Name: "hpet", TickPolicy: "catchup", Present: "yes"},
					{Name: "hypervclock", Present: "yes"},
				},
			},
		),
		Entry("TSC timer without existing clock",
			libvmi.New(
				libvmi.WithCPUFeature("invtsc", "require"),
				libvmistatus.WithStatus(libvmistatus.New(withTSCFrequency(1234567890))),
			),
			&api.Clock{
				Timer: []api.Timer{
					{Name: "tsc", Frequency: "1234567890"},
				},
			},
		),
		Entry("TSC timer appended to existing clock timers",
			libvmi.New(
				libvmi.WithClock(v1.Clock{
					ClockOffset: v1.ClockOffset{
						UTC: &v1.ClockOffsetUTC{},
					},
					Timer: &v1.Timer{
						KVM: &v1.KVMTimer{},
					},
				}),
				libvmi.WithCPUFeature("invtsc", "require"),
				libvmistatus.WithStatus(libvmistatus.New(withTSCFrequency(9999))),
			),
			&api.Clock{
				Offset:     "utc",
				Adjustment: "reset",
				Timer: []api.Timer{
					{Name: "kvmclock", Present: "yes"},
					{Name: "tsc", Frequency: "9999"},
				},
			},
		),
	)
})

func newDomainWithClock(clock *api.Clock) api.Domain {
	return api.Domain{
		Spec: api.DomainSpec{
			Clock: clock,
		},
	}
}

func withTSCFrequency(freq int64) libvmistatus.Option {
	return func(vmiStatus *v1.VirtualMachineInstanceStatus) {
		vmiStatus.TopologyHints = &v1.TopologyHints{TSCFrequency: &freq}
	}
}
