/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package instancetype_test

import (
	"encoding/json"
	goerrors "errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func (s *instancetypeSuite) describeValidation() {
	Context("Required fields", func() {
		var object *unstructured.Unstructured
		BeforeEach(func() {
			object = &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": instancetypev1beta1.SchemeGroupVersion.String(),
				"kind":       "VirtualMachineInstancetype",
				"metadata":   map[string]interface{}{"generateName": "invalid-instancetype-", "namespace": s.namespace},
				"spec": map[string]interface{}{
					"cpu":    map[string]interface{}{"guest": int64(1)},
					"memory": map[string]interface{}{"guest": "128Mi"},
				},
			}}
		})
		DescribeTable(
			"should reject a missing field [test_id:CNV-9083]", func(ctx SpecContext, field string, path []string) {
				unstructured.RemoveNestedField(object.Object, path...)
				payload, err := json.Marshal(object)
				Expect(err).NotTo(HaveOccurred())
				err = s.f.VirtClient().RestClient().Post().AbsPath("/apis/instancetype.kubevirt.io/v1beta1/namespaces/" + s.namespace +
					"/virtualmachineinstancetypes").Body(payload).Do(ctx).Error()
				Expect(errors.IsInvalid(err)).To(BeTrue(), "expected an invalid instancetype, got: %v", err)
				var status errors.APIStatus
				Expect(goerrors.As(err, &status)).To(BeTrue())
				Expect(status.Status().Details.Causes).To(ContainElement(And(
					HaveField("Type", metav1.CauseTypeFieldValueRequired), HaveField("Field", field),
				)))
			},
			Entry("CPU", "spec.cpu", []string{"spec", "cpu"}),
			Entry("CPU guest", "spec.cpu.guest", []string{"spec", "cpu", "guest"}),
			Entry("memory", "spec.memory", []string{"spec", "memory"}),
			Entry("memory guest", "spec.memory.guest", []string{"spec", "memory", "guest"}),
		)
	})
}
