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

	// Direct claims:   {base}/resourceclaims/{claimName}/{requestName}/{driverName}-metadata.json
	// Template claims: {base}/resourceclaimtemplates/{podClaimName}/{requestName}/{driverName}-metadata.json
	writeMetadataJSON := func(dir, driverName string, md *metadata.DeviceMetadata) {
		Expect(os.MkdirAll(dir, 0755)).To(Succeed())
		md.APIVersion = "metadata.resource.k8s.io/v1alpha1"
		md.Kind = "DeviceMetadata"
		data, err := json.Marshal(md)
		Expect(err).ToNot(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(dir, driverName+metadataFileSuffix), data, 0644)).To(Succeed())
	}

	createMetadataFile := func(claimName, requestName string, md *metadata.DeviceMetadata) {
		dir := filepath.Join(tempDir, resourceClaimsSubdir, claimName, requestName)
		writeMetadataJSON(dir, "gpu.example.com", md)
	}

	createTemplateMetadataFile := func(podClaimName, requestName string, md *metadata.DeviceMetadata) {
		dir := filepath.Join(tempDir, resourceClaimTemplatesSubdir, podClaimName, requestName)
		writeMetadataJSON(dir, "gpu.example.com", md)
	}

	Context("lazy resolution", func() {
		It("should resolve metadata for pre-existing claims (ResourceClaimName)", func() {
			pciAddr := "0000:02:00.0"
			createMetadataFile("my-gpu-claim", "gpu-request", &metadata.DeviceMetadata{
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

			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, "vmi-claim-ref", "gpu-request")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should resolve metadata for template-generated claims (ResourceClaimTemplateName)", func() {
			mdevUUID := "123e4567-e89b-12d3-a456-426614174000"
			createTemplateMetadataFile("vmi-template-ref", "vgpu-request", &metadata.DeviceMetadata{
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

			uuid, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "vmi-template-ref", "vgpu-request")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})

		It("should return error when no metadata files exist", func() {
			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "missing-claim",
				ResourceClaimName: ptr.To("nonexistent"),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "missing-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read metadata"))
		})
	})

	Context("GetPCIAddressForClaim", func() {
		It("should return the PCI address when present", func() {
			pciAddr := "0000:03:00.0"
			createMetadataFile("pci-claim", "req1", &metadata.DeviceMetadata{
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

			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, "my-claim", "req1")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return error when claim ref not found", func() {
			_, err := GetPCIAddressForClaim(tempDir, nil, "nonexistent", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})

		It("should return error when request not found in metadata file", func() {
			createMetadataFile("claim1", "other-req", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "claim1"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "other-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim1"),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "my-claim", "missing-req")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read metadata for claim"))
		})

		It("should return error when pciBusID attribute not present", func() {
			createMetadataFile("claim1", "req1", &metadata.DeviceMetadata{
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

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pciBusID not found"))
		})

		It("should return error when request has multiple devices (count > 1)", func() {
			pciAddr1 := "0000:03:00.0"
			pciAddr2 := "0000:04:00.0"
			createMetadataFile("claim1", "req1", &metadata.DeviceMetadata{
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

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("count > 1 is not supported"))
		})
	})

	Context("GetMDevUUIDForClaim", func() {
		It("should return the mdev UUID when present", func() {
			uuid := "abcd1234-5678-90ab-cdef-1234567890ab"
			createMetadataFile("mdev-claim", "vgpu-req", &metadata.DeviceMetadata{
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

			result, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "my-vgpu", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(uuid))
		})

		It("should return error when claim ref not found", func() {
			_, err := GetMDevUUIDForClaim(tempDir, nil, "nonexistent", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})

		It("should return error when mdevUUID attribute not present", func() {
			pciAddr := "0000:01:00.0"
			createMetadataFile("pci-only", "req1", &metadata.DeviceMetadata{
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

			_, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mdevUUID not found"))
		})
	})

	Context("multiple claims and requests", func() {
		It("should handle multiple claims with different device types", func() {
			pciAddr := "0000:04:00.0"
			mdevUUID := "11111111-2222-3333-4444-555555555555"

			createMetadataFile("gpu-claim", "gpu-req", &metadata.DeviceMetadata{
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

			createMetadataFile("vgpu-claim", "vgpu-req", &metadata.DeviceMetadata{
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

			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, "claim-gpu", "gpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))

			uuid, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "claim-vgpu", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})
	})

	Context("ResourceClaimTemplateName claims", func() {
		It("should return PCI address for template-generated claim", func() {
			pciAddr := "0000:05:00.0"
			createTemplateMetadataFile("template-gpu-claim", "pci-req", &metadata.DeviceMetadata{
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

			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, "template-gpu-claim", "pci-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return mdev UUID for template-generated claim", func() {
			mdevUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			createTemplateMetadataFile("template-vgpu-claim", "vgpu-req", &metadata.DeviceMetadata{
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

			uuid, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "template-vgpu-claim", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})

		It("should handle mixed pre-existing and template-generated claims", func() {
			pciAddr := "0000:06:00.0"
			mdevUUID := "12121212-3434-5656-7878-909090909090"

			createMetadataFile("preexisting-claim", "pci-req", &metadata.DeviceMetadata{
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

			createTemplateMetadataFile("my-template-claim", "vgpu-req", &metadata.DeviceMetadata{
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

			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, "existing-ref", "pci-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))

			uuid, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "my-template-claim", "vgpu-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})

		It("should return error when template claim metadata not found", func() {
			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:                      "missing-template-claim",
				ResourceClaimTemplateName: ptr.To("nonexistent-template"),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "missing-template-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read metadata"))
		})
	})

	Context("request name mismatch in metadata content", func() {
		It("should return error with available requests when request not found in metadata JSON", func() {
			createMetadataFile("claim1", "req1", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "claim1"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "actual-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{},
					}},
				}},
			})

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim1"),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found in metadata for claim"))
			Expect(err.Error()).To(ContainSubstring("claim1"))
			Expect(err.Error()).To(ContainSubstring("available requests: [actual-req]"))
		})
	})

	Context("JSON stream version negotiation", func() {
		writeRawStreamFile := func(claimName, requestName, driverName string, objects ...string) {
			dir := filepath.Join(tempDir, resourceClaimsSubdir, claimName, requestName)
			Expect(os.MkdirAll(dir, 0755)).To(Succeed())
			var content []byte
			for _, obj := range objects {
				content = append(content, []byte(obj+"\n")...)
			}
			Expect(os.WriteFile(filepath.Join(dir, driverName+metadataFileSuffix), content, 0644)).To(Succeed())
		}

		It("should skip unknown apiVersion and decode v1alpha1 from stream", func() {
			pciAddr := "0000:07:00.0"
			v2Obj := `{"apiVersion":"metadata.resource.k8s.io/v2beta1","kind":"DeviceMetadata","metadata":{"name":"claim1"},"newField":"ignored"}`
			v1Obj, err := json.Marshal(&metadata.DeviceMetadata{
				TypeMeta:   metav1.TypeMeta{APIVersion: "metadata.resource.k8s.io/v1alpha1", Kind: "DeviceMetadata"},
				ObjectMeta: metav1.ObjectMeta{Name: "claim1"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})
			Expect(err).ToNot(HaveOccurred())

			writeRawStreamFile("claim1", "req1", "gpu.example.com", v2Obj, string(v1Obj))

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim1"),
			}}
			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, "my-claim", "req1")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return error when stream contains only unsupported versions", func() {
			writeRawStreamFile("claim2", "req1", "gpu.example.com",
				`{"apiVersion":"metadata.resource.k8s.io/v99","kind":"DeviceMetadata"}`,
			)

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim2"),
			}}
			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no compatible metadata version"))
		})

		It("should return error on empty stream", func() {
			writeRawStreamFile("claim3", "req1", "gpu.example.com")

			resourceClaims := []k8sv1.PodResourceClaim{{
				Name:              "my-claim",
				ResourceClaimName: ptr.To("claim3"),
			}}
			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "my-claim", "req1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no metadata objects"))
		})
	})
})
