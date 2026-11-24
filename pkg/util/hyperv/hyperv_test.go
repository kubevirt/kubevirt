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

package hyperv

import (
	"reflect"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Hyperv host-model", func() {
	enabled := v1.FeatureState{Enabled: pointer.P(true)}
	disabled := v1.FeatureState{Enabled: pointer.P(false)}

	withTDX := func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{TDX: &v1.TDX{}}
	}
	withHypervClock := func(present bool) libvmi.Option {
		return libvmi.WithClock(v1.Clock{Timer: &v1.Timer{Hyperv: &v1.HypervTimer{Enabled: pointer.P(present)}}})
	}

	DescribeTable("should detect host-model candidates", func(arch string, features *v1.Features, opts []libvmi.Option, expected bool) {
		vmi := libvmi.New(opts...)
		vmi.Spec.Architecture = arch
		vmi.Spec.Domain.Features = features
		Expect(IsHostModelCandidate(vmi)).To(Equal(expected))
	},
		Entry("amd64 with hyperv features", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{}}, nil, true),
		Entry("arm64 with hyperv features", "arm64", &v1.Features{Hyperv: &v1.FeatureHyperv{}}, nil, false),
		Entry("unset architecture", "", &v1.Features{Hyperv: &v1.FeatureHyperv{}}, nil, false),
		Entry("no features", "amd64", nil, nil, false),
		Entry("no hyperv features", "amd64", &v1.Features{}, nil, false),
		Entry("SEV", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{}}, []libvmi.Option{libvmi.WithSEV(false, false)}, false),
		Entry("SEV-SNP", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{}}, []libvmi.Option{libvmi.WithSEV(false, true)}, false),
		Entry("TDX", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{}}, []libvmi.Option{withTDX}, false),
		Entry("enabled features", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{
			Relaxed:    &enabled,
			SyNICTimer: &v1.SyNICTimer{FeatureState: enabled, Direct: &enabled},
		}}, nil, true),
		Entry("a disabled feature", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{Reenlightenment: &disabled}}, nil, false),
		Entry("disabled spinlocks", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{
			Spinlocks: &v1.FeatureSpinlocks{FeatureState: disabled},
		}}, nil, false),
		Entry("a disabled sub-feature", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{
			TLBFlush: &v1.TLBFlush{Extended: &disabled},
		}}, nil, false),
		Entry("an enabled hypervclock timer", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{}}, []libvmi.Option{withHypervClock(true)}, true),
		Entry("a disabled hypervclock timer", "amd64", &v1.Features{Hyperv: &v1.FeatureHyperv{}}, []libvmi.Option{withHypervClock(false)}, false),
	)

	// Guards hasDisabledFeature against v1.FeatureHyperv fields it does not know.
	It("should not detect a candidate when any hyperv feature state is disabled", func() {
		paths := featureStatePaths(reflect.TypeFor[v1.FeatureHyperv](), nil)
		Expect(paths).ToNot(BeEmpty())

		for _, path := range paths {
			hyperv := &v1.FeatureHyperv{}
			disableFeatureStateAt(reflect.ValueOf(hyperv).Elem(), path)
			vmi := libvmi.New(libvmi.WithArchitecture("amd64"))
			vmi.Spec.Domain.Features = &v1.Features{Hyperv: hyperv}

			Expect(IsHostModelCandidate(vmi)).To(BeFalse(), "field index path %v", path)
		}
	})

	Context("HostModelOverrides", func() {
		It("should turn unset reenlightenment and vendor_id off", func() {
			Expect(HostModelOverrides(&v1.FeatureHyperv{Relaxed: &enabled})).To(Equal(&v1.FeatureHyperv{
				Relaxed:         &enabled,
				Reenlightenment: &disabled,
				VendorID:        &v1.FeatureVendorID{FeatureState: disabled},
			}))
		})

		It("should keep requested reenlightenment and vendor_id", func() {
			hyperv := &v1.FeatureHyperv{
				Reenlightenment: &enabled,
				VendorID:        &v1.FeatureVendorID{FeatureState: enabled, VendorID: "myvendor"},
			}
			Expect(HostModelOverrides(hyperv)).To(Equal(hyperv))
		})

		It("should not mutate its input", func() {
			hyperv := &v1.FeatureHyperv{}
			_ = HostModelOverrides(hyperv)
			Expect(hyperv).To(Equal(&v1.FeatureHyperv{}))
		})
	})
})

// featureStatePaths returns the field index paths of every FeatureState
// reachable from struct type t, following pointers to structs.
func featureStatePaths(t reflect.Type, prefix []int) [][]int {
	var paths [][]int
	for i := range t.NumField() {
		path := append(slices.Clone(prefix), i)
		fieldType := t.Field(i).Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		switch {
		case fieldType == reflect.TypeFor[v1.FeatureState]():
			paths = append(paths, path)
		case fieldType.Kind() == reflect.Struct:
			paths = append(paths, featureStatePaths(fieldType, path)...)
		}
	}
	return paths
}

// disableFeatureStateAt allocates the pointers along path and disables the
// FeatureState it leads to.
func disableFeatureStateAt(v reflect.Value, path []int) {
	for _, i := range path {
		v = v.Field(i)
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
	}
	v.Addr().Interface().(*v1.FeatureState).Enabled = pointer.P(false)
}
