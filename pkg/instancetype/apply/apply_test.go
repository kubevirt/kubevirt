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
 */

package apply

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Instancetype spec applier", func() {
	var entries []TableEntry
	for _, applier := range NewSpec().appliers {
		entries = append(entries, Entry(fmt.Sprintf("%T", applier), applier))
	}

	DescribeTable("should ignore nil instancetype specs without modifying the target", func(applier instancetypeApply) {
		preferenceSpec := &v1beta1.VirtualMachinePreferenceSpec{
			PreferredSubdomain: pointer.P("preferred-subdomain"),
			Annotations:        map[string]string{"preferred-annotation": "value"},
		}
		Expect(applier.Apply(nil, nil, preferenceSpec, nil, nil)).To(Succeed())

		vmi := &virtv1.VirtualMachineInstance{}
		expected := vmi.DeepCopy()
		Expect(applier.Apply(nil, nil, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi).To(Equal(expected))
	}, entries)

	It("should return early for a nil instancetype without dereferencing the target", func() {
		preferenceSpec := &v1beta1.VirtualMachinePreferenceSpec{
			PreferredSubdomain: pointer.P("preferred-subdomain"),
		}
		Expect(NewSpec().Apply(nil, nil, preferenceSpec, nil, nil)).To(Succeed())
	})

	It("should collect conflicts in application order", func() {
		instancetypeSpec := &v1beta1.VirtualMachineInstancetypeSpec{
			NodeSelector:  map[string]string{"node": "required"},
			SchedulerName: "required-scheduler",
			CPU:           v1beta1.CPUInstancetype{Guest: 2},
			Memory:        v1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
			Annotations:   map[string]string{"shared": "required"},
		}
		vmi := &virtv1.VirtualMachineInstance{}
		vmi.Spec.NodeSelector = map[string]string{"node": "existing"}
		vmi.Spec.SchedulerName = "existing-scheduler"
		vmi.Spec.Domain.CPU = &virtv1.CPU{Sockets: 1, Cores: 2}
		vmi.Spec.Domain.Memory = &virtv1.Memory{Guest: pointer.P(resource.MustParse("256Mi"))}
		vmi.Annotations = map[string]string{"shared": "existing"}

		conflicts := NewSpec().Apply(k8sfield.NewPath("spec"), instancetypeSpec, nil, &vmi.Spec, &vmi.ObjectMeta)
		paths := make([]string, 0, len(conflicts))
		for _, conflict := range conflicts {
			paths = append(paths, conflict.String())
		}
		Expect(paths).To(Equal([]string{
			"spec.nodeSelector",
			"spec.schedulerName",
			"spec.domain.cpu.sockets",
			"spec.domain.cpu.cores",
			"spec.domain.memory.guest",
			"annotations.shared",
		}))
	})
})
