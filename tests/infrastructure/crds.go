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

package infrastructure

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1ext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crds "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

var _ = Describe(SIG("CRDs", func() {
	It("[test_id:5177]Should have structural schema", func() {
		ourCRDs := []string{
			crds.VIRTUALMACHINE, crds.VIRTUALMACHINEINSTANCE,
			crds.VIRTUALMACHINEINSTANCEREPLICASET, crds.VIRTUALMACHINEINSTANCEMIGRATION, crds.KUBEVIRT,
			crds.VIRTUALMACHINESNAPSHOT, crds.VIRTUALMACHINESNAPSHOTCONTENT,
			crds.VIRTUALMACHINECLONE,
			crds.VIRTUALMACHINEEXPORT,
		}

		for _, name := range ourCRDs {
			crd, err := kubevirt.Client().ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(
				context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(crd).To(matcher.HaveConditionMissingOrFalse(v1ext.NonStructuralSchema))
		}
	})
}))
