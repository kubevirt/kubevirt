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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package components

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	secv1 "github.com/openshift/api/security/v1"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discoveryFake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

var _ = Describe("Deployments", func() {

	Describe("Virt-Handler Daemonset", func() {

		var ctrl *gomock.Controller
		var virtClient *kubecli.MockKubevirtClient
		var discoveryClient *discoveryFake.FakeDiscovery

		BeforeEach(func() {

			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			discoveryClient = &discoveryFake.FakeDiscovery{
				Fake: &fake.NewSimpleClientset().Fake,
			}
			virtClient.EXPECT().DiscoveryClient().Return(discoveryClient).AnyTimes()

		})

		getServerResources := func(onOpenShift bool) []*metav1.APIResourceList {
			list := []*metav1.APIResourceList{
				{
					GroupVersion: v1.GroupVersion.String(),
					APIResources: []metav1.APIResource{
						{
							Name: "kubevirts",
						},
					},
				},
			}
			if onOpenShift {
				list = append(list, &metav1.APIResourceList{
					GroupVersion: secv1.GroupVersion.String(),
					APIResources: []metav1.APIResource{
						{
							Name: "securitycontextconstraints",
						},
					},
				})
			}
			return list
		}

		table.DescribeTable("Testing for NodeSelector", func(onOpenShift bool) {

			discoveryClient.Fake.Resources = getServerResources(onOpenShift)

			handler, err := newHandlerDaemonSetWithOpenshiftCheck("ns", "kubevirt", "latest", k8sv1.PullPolicy("Always"), "2", virtClient)
			Expect(err).ToNot(HaveOccurred(), "should not return an error")

			if onOpenShift {
				Expect(handler.Spec.Template.Spec.NodeSelector["node-role.kubernetes.io/compute"]).To(Equal("true"))
			} else {
				Expect(handler.Spec.Template.Spec.NodeSelector).To(BeNil())
			}

		},
			table.Entry("on Kubernetes", false),
			table.Entry("on OpenShift", true),
		)

	})

})
