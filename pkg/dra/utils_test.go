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

package dra

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/dynamic-resource-allocation/api/metadata"
	"k8s.io/dynamic-resource-allocation/devicemetadata"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("DRA device metadata", func() {
	newMetadata := func(requestName string, devices ...metadata.Device) *metadata.DeviceMetadata {
		return &metadata.DeviceMetadata{
			Requests: []metadata.DeviceMetadataRequest{{
				Name:    requestName,
				Devices: devices,
			}},
		}
	}

	directClaim := func(claimRefName, resourceClaimName string) []v1.VirtualMachineInstanceResourceClaim {
		return []v1.VirtualMachineInstanceResourceClaim{{
			Name:              claimRefName,
			ResourceClaimName: ptr.To(resourceClaimName),
		}}
	}

	templateClaim := func(claimRefName string) []v1.VirtualMachineInstanceResourceClaim {
		return []v1.VirtualMachineInstanceResourceClaim{{
			Name:                      claimRefName,
			ResourceClaimTemplateName: ptr.To("gpu-template"),
		}}
	}

	BeforeEach(func() {
		DeferCleanup(func() {
			readResourceClaim = devicemetadata.ReadResourceClaimMetadata
			readResourceClaimTemplate = devicemetadata.ReadResourceClaimTemplateMetadata
		})
	})

	Context("claim routing", func() {
		It("reads a directly referenced claim by its ResourceClaim name", func() {
			readResourceClaim = func(claimName, requestName string) (*metadata.DeviceMetadata, error) {
				Expect(claimName).To(Equal("allocated-claim"))
				Expect(requestName).To(Equal("gpu"))
				return newMetadata("gpu"), nil
			}
			readResourceClaimTemplate = func(string, string) (*metadata.DeviceMetadata, error) {
				return nil, fmt.Errorf("unexpected template reader call")
			}

			_, err := readClaimMetadata(directClaim("claim-ref", "allocated-claim"), "claim-ref", "gpu")
			Expect(err).ToNot(HaveOccurred())
		})

		It("reads a template claim by its pod-local claim name", func() {
			readResourceClaim = func(string, string) (*metadata.DeviceMetadata, error) {
				return nil, fmt.Errorf("unexpected direct reader call")
			}
			readResourceClaimTemplate = func(podClaimName, requestName string) (*metadata.DeviceMetadata, error) {
				Expect(podClaimName).To(Equal("claim-ref"))
				Expect(requestName).To(Equal("gpu"))
				return newMetadata("gpu"), nil
			}

			_, err := readClaimMetadata(templateClaim("claim-ref"), "claim-ref", "gpu")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error when the claim reference is missing", func() {
			_, err := readClaimMetadata(nil, "missing", "gpu")
			Expect(err).To(MatchError(`metadata not found for claim "missing"`))
		})
	})

	Context("attribute extraction", func() {
		BeforeEach(func() {
			readResourceClaim = func(_, _ string) (*metadata.DeviceMetadata, error) {
				return nil, fmt.Errorf("unexpected call")
			}
			readResourceClaimTemplate = func(_, _ string) (*metadata.DeviceMetadata, error) {
				return nil, fmt.Errorf("unexpected call")
			}
		})

		setupReader := func(md *metadata.DeviceMetadata) {
			readResourceClaim = func(_, _ string) (*metadata.DeviceMetadata, error) {
				return md, nil
			}
		}

		It("extracts a PCI address", func() {
			pciAddress := "0000:03:00.0"
			setupReader(newMetadata("gpu", metadata.Device{
				Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
					PCIBusIDAttribute: {StringValue: ptr.To(pciAddress)},
				},
			}))

			addr, err := GetPCIAddressForClaim(directClaim("claim-ref", "claim"), "claim-ref", "gpu")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddress))
		})

		It("extracts an mdev UUID", func() {
			uuid := "123e4567-e89b-12d3-a456-426614174000"
			setupReader(newMetadata("gpu", metadata.Device{
				Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
					MDevUUIDAttribute: {StringValue: ptr.To(uuid)},
				},
			}))

			result, err := GetMDevUUIDForClaim(directClaim("claim-ref", "claim"), "claim-ref", "gpu")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(uuid))
		})

		DescribeTable("rejects unsupported metadata",
			func(md *metadata.DeviceMetadata, expectedError string) {
				setupReader(md)
				_, err := GetPCIAddressForClaim(directClaim("claim-ref", "claim"), "claim-ref", "gpu")
				Expect(err).To(MatchError(ContainSubstring(expectedError)))
			},
			Entry("missing request", newMetadata("other"), `request "gpu" not found`),
			Entry("no devices", newMetadata("gpu"), `request "gpu" has no devices`),
			Entry("multiple devices", newMetadata("gpu", metadata.Device{}, metadata.Device{}), "count > 1 is not supported"),
			Entry("missing PCI attribute", newMetadata("gpu", metadata.Device{}), "pciBusID not found"),
		)

		It("returns error when mdevUUID attribute is missing", func() {
			setupReader(newMetadata("gpu", metadata.Device{}))
			_, err := GetMDevUUIDForClaim(directClaim("claim-ref", "claim"), "claim-ref", "gpu")
			Expect(err).To(MatchError(ContainSubstring("mdevUUID not found")))
		})
	})
})
