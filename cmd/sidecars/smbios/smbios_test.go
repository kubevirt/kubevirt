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

package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hooksv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	archconverter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"

	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("SMBios sidecar", func() {
	It("should properly alter the libvirt domain", func() {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: k8smeta.ObjectMeta{
				Name:        "testvmi",
				Namespace:   "mynamespace",
				Annotations: renderSidecar(hooksv1alpha1.Version),
			},
		}

		c := &converter.ConverterContext{
			Architecture:   archconverter.NewConverter(runtime.GOARCH),
			VirtualMachine: vmi,
			AllowEmulation: true,
		}

		domain := &api.Domain{}
		err := converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)
		Expect(err).ToNot(HaveOccurred())

		xml, err := xml.Marshal(domain.Spec)
		Expect(err).ToNot(HaveOccurred())
		json, err := json.Marshal(vmi)
		Expect(err).ToNot(HaveOccurred())

		domxml, err := onDefineDomain(json, xml)
		Expect(err).ToNot(HaveOccurred())

		Expect(domxml).To(ContainSubstring(`<sysinfo type="smbios">`))
		Expect(domxml).To(ContainSubstring(`<smbios mode="sysinfo">`))
		Expect(domxml).To(ContainSubstring(`<entry name="manufacturer">Radical Edward</entry>`))
	})
})

func renderSidecar(version string) map[string]string {
	return map[string]string{
		"hooks.kubevirt.io/hookSidecars":              fmt.Sprintf(`[{"args": ["--version", "%s"],"image": "someimage", "imagePullPolicy": "IfNotPresent"}]`, version),
		"smbios.vm.kubevirt.io/baseBoardManufacturer": "Radical Edward",
	}
}
