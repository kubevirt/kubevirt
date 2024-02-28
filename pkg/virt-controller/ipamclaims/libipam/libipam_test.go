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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package libipam_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-controller/ipamclaims/libipam"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

const (
	networkName = "network"
)

var _ = Describe("GetPersistentIPsConf", func() {
	type testCase struct {
		config           string
		expectedAllowIPs bool
		expectedName     string
	}

	DescribeTable("should return expected netConf",
		func(tc testCase) {
			nad := &networkv1.NetworkAttachmentDefinition{
				Spec: networkv1.NetworkAttachmentDefinitionSpec{
					Config: fmt.Sprintf(tc.config, networkName),
				},
			}
			conf, err := libipam.GetPersistentIPsConf(nad)
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.AllowPersistentIPs).To(Equal(tc.expectedAllowIPs))
			Expect(conf.Name).To(Equal(tc.expectedName))
		},
		Entry("when allowPersistentIPs is omitted", testCase{
			config:           `{"name": "%s"}`,
			expectedAllowIPs: false,
			expectedName:     networkName,
		}),
		Entry("when allowPersistentIPs is false", testCase{
			config:           `{"name": "%s", "allowPersistentIPs": false}`,
			expectedAllowIPs: false,
			expectedName:     networkName,
		}),
		Entry("when allowPersistentIPs is true", testCase{
			config:           `{"name": "%s", "allowPersistentIPs": true}`,
			expectedAllowIPs: true,
			expectedName:     networkName,
		}),
	)

	It("when name field is missing, should return error", func() {
		nad := &networkv1.NetworkAttachmentDefinition{
			Spec: networkv1.NetworkAttachmentDefinitionSpec{
				Config: fmt.Sprintf(`{"cniVersion": "1.0.0"}`),
			},
		}
		_, err := libipam.GetPersistentIPsConf(nad)
		Expect(err).To(MatchError(ContainSubstring("failed to obtain network name: missing required field")))
	})
})
