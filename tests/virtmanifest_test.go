/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-manifest"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Virtmanifest", func() {
	Context("Manifest Service", func() {
		flag.Parse()

		var manifestClient *rest.RESTClient

		BeforeEach(func() {
			tests.BeforeTestCleanup()

			var err error
			var masterUrl *url.URL
			masterUrl, err = url.Parse(flag.Lookup("master").Value.String())
			Expect(err).ToNot(HaveOccurred())
			hostParts := strings.Split(masterUrl.Host, ":")
			Expect(len(hostParts)).To(Equal(2))

			manifestClient, err = kubecli.GetRESTClientFromFlags(fmt.Sprintf("http://%s:8186", hostParts[0]), "")
			Expect(err).ToNot(HaveOccurred())

		})

		It("Should report server status", func() {
			ref := map[string]string{"status": "ok"}
			data := map[string]string{}

			res, err := manifestClient.Get().RequestURI("/api/v1/status").DoRaw()
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(res, &data)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal(ref))
		})

		It("Should return YAML if requested", func() {
			ref := "status: ok\n"
			res, err := manifestClient.Get().RequestURI("/api/v1/status").SetHeader("Accept", "application/yaml").DoRaw()
			Expect(err).ToNot(HaveOccurred())

			Expect(string(res)).To(Equal(ref))
		})

		It("Should map a VM manifest", func() {
			vm := tests.NewRandomVM()
			vmName := vm.ObjectMeta.Name
			mappedVm := v1.VM{}

			request, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			res, err := manifestClient.Post().SetHeader("Content-type", "application/json").Resource("manifest").Body(request).DoRaw()
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(res, &mappedVm)
			Expect(mappedVm.ObjectMeta.Name).To(Equal(vmName))
			Expect(mappedVm.Spec.Domain.Type).To(Equal("qemu"))
		})

		It("Should map PersistentVolumeClaims", func() {
			vm := tests.NewRandomVMWithPVC("disk-alpine")
			mappedVm := v1.VM{}

			request, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			res, err := manifestClient.Post().SetHeader("Content-type", "application/json").Resource("manifest").Body(request).DoRaw()
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(res, &mappedVm)
			Expect(len(mappedVm.Spec.Domain.Devices.Disks)).To(Equal(1))
			Expect(mappedVm.Spec.Domain.Devices.Disks[0].Type).To(Equal(virt_manifest.Type_PersistentVolumeClaim))
		})
	})
})
