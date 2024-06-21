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

package components_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const testNamespace = "test-namespace"

var _ = Describe("Routes", func() {
	It("should successfully create virt export route", func() {
		route := components.NewExportProxyRoute(testNamespace)
		Expect(route).ToNot(BeNil())
		Expect(route.Namespace).To(Equal(testNamespace))
		Expect(route.Name).To(Equal(components.VirtExportProxyName))
		Expect(route.Spec.TLS).ToNot(BeNil())
		Expect(route.Spec.TLS.Termination).To(Equal(routev1.TLSTerminationReencrypt))
		Expect(route.Spec.TLS.InsecureEdgeTerminationPolicy).To(Equal(routev1.InsecureEdgeTerminationPolicyRedirect))
	})
})
