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

package plugins_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins"
)

var _ = Describe("Plugin store", func() {
	It("should return nil when no plugins are set", func() {
		Expect(plugins.GetPlugins()).To(BeNil())
	})

	It("should store and retrieve plugins", func() {
		pluginList := []pluginv1alpha1.Plugin{
			{ObjectMeta: metav1.ObjectMeta{Name: "stored-plugin"}},
		}
		plugins.SetPlugins(pluginList)
		defer plugins.SetPlugins(nil)

		retrieved := plugins.GetPlugins()
		Expect(retrieved).To(HaveLen(1))
		Expect(retrieved[0].Name).To(Equal("stored-plugin"))
	})
})
