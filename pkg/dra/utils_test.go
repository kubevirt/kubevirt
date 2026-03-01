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

var _ = Describe("DownwardAPIAttributes", func() {
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

	// KEP-5304 path: {base}/{claimName}/{requestName}/{driver}-metadata.json
	createMetadataFile := func(claimName, requestName, driver string, md *metadata.DeviceMetadata) {
		dir := filepath.Join(tempDir, claimName, requestName)
		Expect(os.MkdirAll(dir, 0755)).To(Succeed())

		data, err := json.Marshal(md)
		Expect(err).ToNot(HaveOccurred())

		Expect(os.WriteFile(filepath.Join(dir, driver+"-metadata.json"), data, 0644)).To(Succeed())
	}

	Context("NewDownwardAPIAttributesWithBasePath", func() {
		It("should load metadata for pre-existing claims (ResourceClaimName)", func() {
			pciAddr := "0000:02:00.0"
			createMetadataFile("my-gpu-claim", "gpu-request", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "my-gpu-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "gpu-request",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "default",
						Name:   "gpu-0",
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

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())
			Expect(deviceAttrs.resolvedMetadata).To(HaveKey("vmi-claim-ref"))
		})

		It("should load metadata for template-generated claims (ResourceClaimTemplateName)", func() {
			mdevUUID := "123e4567-e89b-12d3-a456-426614174000"
			createMetadataFile("generated-claim-abc123", "vgpu-request", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta:   metav1.ObjectMeta{Name: "generated-claim-abc123"},
				PodClaimName: ptr.To("vmi-template-ref"),
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "vgpu-request",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "default",
						Name:   "vgpu-0",
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

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())
			Expect(deviceAttrs.resolvedMetadata).To(HaveKey("vmi-template-ref"))
		})

		It("should return empty result when no metadata files exist", func() {
			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "missing-claim",
				ResourceClaimName: ptr.To("nonexistent"),
			}}

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())
			Expect(deviceAttrs.resolvedMetadata["missing-claim"]).To(BeNil())
		})
	})

	Context("GetPCIAddressForClaim", func() {
		It("should return the PCI address when present", func() {
			pciAddr := "0000:03:00.0"
			createMetadataFile("pci-claim", "req1", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "pci-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
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

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			addr, err := deviceAttrs.GetPCIAddressForClaim("my-claim", "req1")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return error when claim metadata not found", func() {
			deviceAttrs, _ := NewDownwardAPIAttributesWithBasePath(tempDir, nil)

			_, err := deviceAttrs.GetPCIAddressForClaim("nonexistent", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})

		It("should return error when request not found", func() {
			createMetadataFile("claim1", "placeholder", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "claim1"},
				Requests:   []metadata.DeviceMetadataRequest{},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim1"),
			}}

			deviceAttrs, _ := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)

			_, err := deviceAttrs.GetPCIAddressForClaim("my-claim", "missing-req")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found in metadata"))
		})

		It("should return error when pciBusID attribute not present", func() {
			createMetadataFile("claim1", "req1", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "claim1"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name:    "req1",
					Devices: []metadata.Device{{Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{}}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim1"),
			}}

			deviceAttrs, _ := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)

			_, err := deviceAttrs.GetPCIAddressForClaim("my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pciBusID not found"))
		})

		It("should return error when request has multiple devices (count > 1)", func() {
			pciAddr1 := "0000:03:00.0"
			pciAddr2 := "0000:04:00.0"
			createMetadataFile("claim1", "req1", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "claim1"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "req1",
					Devices: []metadata.Device{
						{Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr1},
						}},
						{Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr2},
						}},
					},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim1"),
			}}

			deviceAttrs, _ := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)

			_, err := deviceAttrs.GetPCIAddressForClaim("my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("count > 1 is not supported"))
		})
	})

	Context("GetMDevUUIDForClaim", func() {
		It("should return the mdev UUID when present", func() {
			uuid := "abcd1234-5678-90ab-cdef-1234567890ab"
			createMetadataFile("mdev-claim", "vgpu-req", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "mdev-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
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

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			result, err := deviceAttrs.GetMDevUUIDForClaim("my-vgpu", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(uuid))
		})

		It("should return error when claim metadata not found", func() {
			deviceAttrs, _ := NewDownwardAPIAttributesWithBasePath(tempDir, nil)

			_, err := deviceAttrs.GetMDevUUIDForClaim("nonexistent", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})

		It("should return error when mdevUUID attribute not present", func() {
			pciAddr := "0000:01:00.0"
			createMetadataFile("pci-only", "req1", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "pci-only"},
				Requests: []metadata.DeviceMetadataRequest{{
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

			deviceAttrs, _ := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)

			_, err := deviceAttrs.GetMDevUUIDForClaim("my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mdevUUID not found"))
		})
	})

	Context("multiple claims and requests", func() {
		It("should handle multiple claims with different device types", func() {
			pciAddr := "0000:04:00.0"
			mdevUUID := "11111111-2222-3333-4444-555555555555"

			createMetadataFile("gpu-claim", "gpu-req", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "gpu-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "gpu-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			createMetadataFile("vgpu-claim", "vgpu-req", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "vgpu-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
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

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			addr, err := deviceAttrs.GetPCIAddressForClaim("claim-gpu", "gpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))

			uuid, err := deviceAttrs.GetMDevUUIDForClaim("claim-vgpu", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})
	})

	Context("ResourceClaimTemplateName claims", func() {
		It("should return PCI address for template-generated claim", func() {
			pciAddr := "0000:05:00.0"
			createMetadataFile("generated-pci-claim-xyz", "pci-req", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta:   metav1.ObjectMeta{Name: "generated-pci-claim-xyz"},
				PodClaimName: ptr.To("template-gpu-claim"),
				Requests: []metadata.DeviceMetadataRequest{{
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

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			addr, err := deviceAttrs.GetPCIAddressForClaim("template-gpu-claim", "pci-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return mdev UUID for template-generated claim", func() {
			mdevUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			createMetadataFile("generated-vgpu-claim-abc", "vgpu-req", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta:   metav1.ObjectMeta{Name: "generated-vgpu-claim-abc"},
				PodClaimName: ptr.To("template-vgpu-claim"),
				Requests: []metadata.DeviceMetadataRequest{{
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

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			uuid, err := deviceAttrs.GetMDevUUIDForClaim("template-vgpu-claim", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})

		It("should handle mixed pre-existing and template-generated claims", func() {
			pciAddr := "0000:06:00.0"
			mdevUUID := "12121212-3434-5656-7878-909090909090"

			createMetadataFile("preexisting-claim", "pci-req", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "preexisting-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "pci-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			createMetadataFile("generated-claim-def456", "vgpu-req", "gpu.example.com", &metadata.DeviceMetadata{
				ObjectMeta:   metav1.ObjectMeta{Name: "generated-claim-def456"},
				PodClaimName: ptr.To("my-template-claim"),
				Requests: []metadata.DeviceMetadataRequest{{
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

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())

			addr, err := deviceAttrs.GetPCIAddressForClaim("existing-ref", "pci-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))

			uuid, err := deviceAttrs.GetMDevUUIDForClaim("my-template-claim", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})

		It("should return error when template claim metadata not found", func() {
			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:                      "missing-template-claim",
				ResourceClaimTemplateName: ptr.To("nonexistent-template"),
			}}

			deviceAttrs, err := NewDownwardAPIAttributesWithBasePath(tempDir, resourceClaims)
			Expect(err).ToNot(HaveOccurred())
			Expect(deviceAttrs.resolvedMetadata["missing-template-claim"]).To(BeNil())

			_, err = deviceAttrs.GetPCIAddressForClaim("missing-template-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})
	})
})
