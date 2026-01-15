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

package dra

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"kubevirt.io/kubevirt/pkg/dra/metadata"
)

var _ = Describe("DRAFileData", func() {
	var (
		tempDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "dra-metadata-test")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	createMetadataFile := func(driver, claimName string, md *metadata.DeviceMetadata) {
		dir := filepath.Join(tempDir, driver, claimName)
		Expect(os.MkdirAll(dir, 0755)).To(Succeed())

		data, err := json.Marshal(md)
		Expect(err).ToNot(HaveOccurred())

		Expect(os.WriteFile(filepath.Join(dir, MetadataFileName), data, 0644)).To(Succeed())
	}

	Context("NewDRAFileDataWithBasePath", func() {
		It("should load metadata for pre-existing claims (ResourceClaimName)", func() {
			pciAddr := "0000:02:00.0"
			createMetadataFile("gpu.example.com", "my-gpu-claim", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "my-gpu-claim"},
				Requests: []metadata.DeviceRequest{{
					Name: "gpu-request",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "default",
						Device: "gpu-0",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "vmi-claim-ref",
				ResourceClaimName: ptr.To("my-gpu-claim"),
			}}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileData.resolvedMetadata).To(HaveKey("vmi-claim-ref"))
		})

		It("should load metadata for template-generated claims (ResourceClaimTemplateName)", func() {
			mdevUUID := "123e4567-e89b-12d3-a456-426614174000"
			createMetadataFile("gpu.example.com", "generated-claim-abc123", &metadata.DeviceMetadata{
				ObjectMeta:   metav1.ObjectMeta{Name: "generated-claim-abc123"},
				PodClaimName: "vmi-template-ref",
				Requests: []metadata.DeviceRequest{{
					Name: "vgpu-request",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "default",
						Device: "vgpu-0",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &mdevUUID},
						},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:                      "vmi-template-ref",
				ResourceClaimTemplateName: ptr.To("my-template"),
			}}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileData.resolvedMetadata).To(HaveKey("vmi-template-ref"))
		})

		It("should return empty cache when no metadata files exist", func() {
			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "missing-claim",
				ResourceClaimName: ptr.To("nonexistent"),
			}}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileData.resolvedMetadata["missing-claim"]).To(BeNil())
		})
	})

	Context("GetPCIAddressForClaim", func() {
		It("should return the PCI address when present", func() {
			pciAddr := "0000:03:00.0"
			createMetadataFile("gpu.example.com", "pci-claim", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "pci-claim"},
				Requests: []metadata.DeviceRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("pci-claim"),
			}}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			addr, err := fileData.GetPCIAddressForClaim("my-claim", "req1")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return error when claim metadata not found", func() {
			fileData, _ := NewDRAFileDataWithBasePath(tempDir, nil)

			_, err := fileData.GetPCIAddressForClaim("nonexistent", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})

		It("should return error when request not found", func() {
			createMetadataFile("gpu.example.com", "claim1", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "claim1"},
				Requests:   []metadata.DeviceRequest{},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim1"),
			}}

			fileData, _ := NewDRAFileDataWithBasePath(tempDir, resourceClaims)

			_, err := fileData.GetPCIAddressForClaim("my-claim", "missing-req")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found in metadata"))
		})

		It("should return error when pciBusID attribute not present", func() {
			createMetadataFile("gpu.example.com", "claim1", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "claim1"},
				Requests: []metadata.DeviceRequest{{
					Name:    "req1",
					Devices: []metadata.Device{{Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{}}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim1"),
			}}

			fileData, _ := NewDRAFileDataWithBasePath(tempDir, resourceClaims)

			_, err := fileData.GetPCIAddressForClaim("my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pciBusID not found"))
		})
	})

	Context("GetMDevUUIDForClaim", func() {
		It("should return the mdev UUID when present", func() {
			uuid := "abcd1234-5678-90ab-cdef-1234567890ab"
			createMetadataFile("gpu.example.com", "mdev-claim", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "mdev-claim"},
				Requests: []metadata.DeviceRequest{{
					Name: "vgpu-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &uuid},
						},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-vgpu",
				ResourceClaimName: ptr.To("mdev-claim"),
			}}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			result, err := fileData.GetMDevUUIDForClaim("my-vgpu", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(uuid))
		})

		It("should return error when claim metadata not found", func() {
			fileData, _ := NewDRAFileDataWithBasePath(tempDir, nil)

			_, err := fileData.GetMDevUUIDForClaim("nonexistent", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})

		It("should return error when mdevUUID attribute not present", func() {
			pciAddr := "0000:01:00.0"
			createMetadataFile("gpu.example.com", "pci-only", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "pci-only"},
				Requests: []metadata.DeviceRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("pci-only"),
			}}

			fileData, _ := NewDRAFileDataWithBasePath(tempDir, resourceClaims)

			_, err := fileData.GetMDevUUIDForClaim("my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mdevUUID not found"))
		})
	})

	Context("multiple claims and requests", func() {
		It("should handle multiple claims with different device types", func() {
			pciAddr := "0000:04:00.0"
			mdevUUID := "11111111-2222-3333-4444-555555555555"

			createMetadataFile("gpu.example.com", "gpu-claim", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "gpu-claim"},
				Requests: []metadata.DeviceRequest{{
					Name: "gpu-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			createMetadataFile("gpu.example.com", "vgpu-claim", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "vgpu-claim"},
				Requests: []metadata.DeviceRequest{{
					Name: "vgpu-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &mdevUUID},
						},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{
				{Name: "claim-gpu", ResourceClaimName: ptr.To("gpu-claim")},
				{Name: "claim-vgpu", ResourceClaimName: ptr.To("vgpu-claim")},
			}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			addr, err := fileData.GetPCIAddressForClaim("claim-gpu", "gpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))

			uuid, err := fileData.GetMDevUUIDForClaim("claim-vgpu", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})
	})

	Context("ResourceClaimTemplateName claims", func() {
		It("should return PCI address for template-generated claim", func() {
			pciAddr := "0000:05:00.0"
			createMetadataFile("gpu.example.com", "generated-pci-claim-xyz", &metadata.DeviceMetadata{
				ObjectMeta:   metav1.ObjectMeta{Name: "generated-pci-claim-xyz"},
				PodClaimName: "template-gpu-claim",
				Requests: []metadata.DeviceRequest{{
					Name: "pci-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:                      "template-gpu-claim",
				ResourceClaimTemplateName: ptr.To("gpu-template"),
			}}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			addr, err := fileData.GetPCIAddressForClaim("template-gpu-claim", "pci-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return mdev UUID for template-generated claim", func() {
			mdevUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			createMetadataFile("gpu.example.com", "generated-vgpu-claim-abc", &metadata.DeviceMetadata{
				ObjectMeta:   metav1.ObjectMeta{Name: "generated-vgpu-claim-abc"},
				PodClaimName: "template-vgpu-claim",
				Requests: []metadata.DeviceRequest{{
					Name: "vgpu-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &mdevUUID},
						},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:                      "template-vgpu-claim",
				ResourceClaimTemplateName: ptr.To("vgpu-template"),
			}}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			uuid, err := fileData.GetMDevUUIDForClaim("template-vgpu-claim", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})

		It("should handle mixed pre-existing and template-generated claims", func() {
			pciAddr := "0000:06:00.0"
			mdevUUID := "12121212-3434-5656-7878-909090909090"

			createMetadataFile("gpu.example.com", "preexisting-claim", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "preexisting-claim"},
				Requests: []metadata.DeviceRequest{{
					Name: "pci-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			createMetadataFile("gpu.example.com", "generated-claim-def456", &metadata.DeviceMetadata{
				ObjectMeta:   metav1.ObjectMeta{Name: "generated-claim-def456"},
				PodClaimName: "my-template-claim",
				Requests: []metadata.DeviceRequest{{
					Name: "vgpu-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &mdevUUID},
						},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{
				{Name: "existing-ref", ResourceClaimName: ptr.To("preexisting-claim")},
				{Name: "my-template-claim", ResourceClaimTemplateName: ptr.To("some-template")},
			}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			addr, err := fileData.GetPCIAddressForClaim("existing-ref", "pci-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))

			uuid, err := fileData.GetMDevUUIDForClaim("my-template-claim", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})

		It("should return error when template claim metadata not found", func() {
			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:                      "missing-template-claim",
				ResourceClaimTemplateName: ptr.To("nonexistent-template"),
			}}

			fileData, err := NewDRAFileDataWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileData.resolvedMetadata["missing-template-claim"]).To(BeNil())

			_, err = fileData.GetPCIAddressForClaim("missing-template-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})
	})
})
