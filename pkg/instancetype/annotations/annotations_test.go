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

package annotations_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"

	"kubevirt.io/kubevirt/pkg/instancetype/annotations"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Annotations", func() {
	var (
		vm   *v1.VirtualMachine
		meta *metav1.ObjectMeta
	)

	const instancetypeName = "instancetype-name"

	BeforeEach(func() {
		vm = libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithInstancetype(instancetypeName))

		meta = &metav1.ObjectMeta{}
	})

	It("should add instancetype name annotation", func() {
		vm.Spec.Instancetype.Kind = apiinstancetype.SingularResourceName

		annotations.Set(vm, meta)

		Expect(meta.Annotations).To(HaveKeyWithValue(v1.InstancetypeAnnotation, instancetypeName))
		Expect(meta.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
	})

	It("should add cluster instancetype name annotation", func() {
		vm.Spec.Instancetype.Kind = apiinstancetype.ClusterSingularResourceName

		annotations.Set(vm, meta)

		Expect(meta.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
		Expect(meta.Annotations).To(HaveKeyWithValue(v1.ClusterInstancetypeAnnotation, instancetypeName))
	})

	It("should add cluster name annotation, if instancetype.kind is empty", func() {
		vm.Spec.Instancetype.Kind = ""

		annotations.Set(vm, meta)

		Expect(meta.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
		Expect(meta.Annotations).To(HaveKeyWithValue(v1.ClusterInstancetypeAnnotation, instancetypeName))
	})
})
