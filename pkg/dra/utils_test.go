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
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/dra/metadata"
)

const (
	claim1Name         = "claim1"
	deviceMetadataKind = "DeviceMetadata"
	gpuRequestName     = "gpu-req"
	pciAddr0300        = "0000:03:00.0"
	pciAddr0400        = "0000:04:00.0"
	request1Name       = "req1"
	testClaimName      = "my-claim"
	vgpuRequestName    = "vgpu-req"
)

var _ = Describe("DownwardAPIAttributes", func() {
	const (
		gpuDriver        = "gpu.example.com"
		gpuPool          = "default"
		gpuDeviceName    = "gpu-0"
		multiDriverClaim = "multi-driver-claim"
		worker0Pool      = "worker-0"
	)
	var tempDir string

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
		Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
		md.APIVersion = metadata.APIVersionV1Alpha1
		md.Kind = deviceMetadataKind
		data, err := json.Marshal(md)
		Expect(err).ToNot(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(dir, driverName+metadataFileSuffix), data, 0o600)).To(Succeed())
	}

	createMetadataFile := func(claimName, requestName string, md *metadata.DeviceMetadata) {
		dir := filepath.Join(tempDir, resourceClaimsSubdir, claimName, requestName)
		writeMetadataJSON(dir, "gpu.example.com", md)
	}

	createTemplateMetadataFile := func(podClaimName, requestName string, md *metadata.DeviceMetadata) {
		dir := filepath.Join(tempDir, resourceClaimTemplatesSubdir, podClaimName, requestName)
		writeMetadataJSON(dir, "gpu.example.com", md)
	}

	metadataWithAttributes := func(
		name string,
		requestName string,
		attributes map[resourcev1.QualifiedName]resourcev1.DeviceAttribute,
	) *metadata.DeviceMetadata {
		return &metadata.DeviceMetadata{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Requests: []metadata.DeviceMetadataRequest{{
				Name: requestName,
				Devices: []metadata.Device{{
					Attributes: attributes,
				}},
			}},
		}
	}

	metadataWithoutAttributes := func(name, requestName string) *metadata.DeviceMetadata {
		return metadataWithAttributes(name, requestName, map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{})
	}

	metadataWithStringAttribute := func(
		name string,
		requestName string,
		attribute resourcev1.QualifiedName,
		value string,
	) *metadata.DeviceMetadata {
		return metadataWithAttributes(name, requestName, map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
			attribute: {StringValue: ptr.To(value)},
		})
	}

	templateMetadataWithStringAttribute := func(
		name string,
		podClaimName string,
		requestName string,
		attribute resourcev1.QualifiedName,
		value string,
	) *metadata.DeviceMetadata {
		md := metadataWithStringAttribute(name, requestName, attribute, value)
		md.PodClaimName = ptr.To(podClaimName)
		return md
	}

	Context("lazy resolution", func() {
		It("should resolve metadata for pre-existing claims (ResourceClaimName)", func() {
			pciAddr := "0000:02:00.0"
			createMetadataFile("my-gpu-claim", "gpu-request", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "my-gpu-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "gpu-request",
					Devices: []metadata.Device{{
						Driver: gpuDriver,
						Pool:   gpuPool,
						Name:   gpuDeviceName,
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
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
						Driver: gpuDriver,
						Pool:   gpuPool,
						Name:   "vgpu-0",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &mdevUUID},
						},
					}},
				}},
			})

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:                      "vmi-template-ref",
				ResourceClaimTemplateName: ptr.To("my-template"),
			}}

			uuid, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "vmi-template-ref", "vgpu-request")
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})

		It("should return error when no metadata files exist", func() {
			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              "missing-claim",
				ResourceClaimName: ptr.To("nonexistent"),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "missing-claim", request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read metadata"))
		})
	})

	Context("GetPCIAddressForClaim", func() {
		It("should return the PCI address when present", func() {
			pciAddr := pciAddr0300
			createMetadataFile("pci-claim", request1Name, &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "pci-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: request1Name,
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To("pci-claim"),
			}}

			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return error when claim ref not found", func() {
			_, err := GetPCIAddressForClaim(tempDir, nil, "nonexistent", request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})

		It("should return error when request not found in metadata file", func() {
			createMetadataFile(claim1Name, "other-req", metadataWithoutAttributes(claim1Name, "other-req"))

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To(claim1Name),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, "missing-req")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read metadata for claim"))
		})

		It("should return error when pciBusID attribute not present", func() {
			createMetadataFile(claim1Name, request1Name, metadataWithoutAttributes(claim1Name, request1Name))

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To(claim1Name),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pciBusID not found"))
		})

		It("should return error when request has multiple devices (count > 1)", func() {
			pciAddr1 := pciAddr0300
			pciAddr2 := pciAddr0400
			createMetadataFile(claim1Name, request1Name, &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: claim1Name},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: request1Name,
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

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To(claim1Name),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("count > 1 is not supported"))
		})
	})

	Context("GetMDevUUIDForClaim", func() {
		It("should return the mdev UUID when present", func() {
			uuid := "abcd1234-5678-90ab-cdef-1234567890ab"
			createMetadataFile("mdev-claim", vgpuRequestName, &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "mdev-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: vgpuRequestName,
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &uuid},
						},
					}},
				}},
			})

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              "my-vgpu",
				ResourceClaimName: ptr.To("mdev-claim"),
			}}

			result, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "my-vgpu", vgpuRequestName)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(uuid))
		})

		It("should return error when claim ref not found", func() {
			_, err := GetMDevUUIDForClaim(tempDir, nil, "nonexistent", request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("metadata not found"))
		})

		It("should return error when mdevUUID attribute not present", func() {
			pciAddr := "0000:01:00.0"
			createMetadataFile("pci-only", request1Name, &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "pci-only"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: request1Name,
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To("pci-only"),
			}}

			_, err := GetMDevUUIDForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mdevUUID not found"))
		})
	})

	Context("multiple claims and requests", func() {
		It("should handle multiple claims with different device types", func() {
			pciAddr := pciAddr0400
			mdevUUID := "11111111-2222-3333-4444-555555555555"

			createMetadataFile("gpu-claim", gpuRequestName, &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "gpu-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: gpuRequestName,
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			createMetadataFile("vgpu-claim", vgpuRequestName, &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: "vgpu-claim"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: vgpuRequestName,
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &mdevUUID},
						},
					}},
				}},
			})

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{
				{Name: "claim-gpu", ResourceClaimName: ptr.To("gpu-claim")},
				{Name: "claim-vgpu", ResourceClaimName: ptr.To("vgpu-claim")},
			}

			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, "claim-gpu", gpuRequestName)
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))

			uuid, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "claim-vgpu", vgpuRequestName)
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})
	})

	Context("ResourceClaimTemplateName claims", func() {
		It("should return PCI address for template-generated claim", func() {
			pciAddr := "0000:05:00.0"
			createTemplateMetadataFile(
				"template-gpu-claim",
				"pci-req",
				templateMetadataWithStringAttribute(
					"generated-pci-claim-xyz",
					"template-gpu-claim",
					"pci-req",
					metadata.PCIBusIDAttribute,
					pciAddr,
				),
			)

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:                      "template-gpu-claim",
				ResourceClaimTemplateName: ptr.To("gpu-template"),
			}}

			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, "template-gpu-claim", "pci-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return mdev UUID for template-generated claim", func() {
			mdevUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			createTemplateMetadataFile(
				"template-vgpu-claim",
				vgpuRequestName,
				templateMetadataWithStringAttribute(
					"generated-vgpu-claim-abc",
					"template-vgpu-claim",
					vgpuRequestName,
					metadata.MDevUUIDAttribute,
					mdevUUID,
				),
			)

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:                      "template-vgpu-claim",
				ResourceClaimTemplateName: ptr.To("vgpu-template"),
			}}

			uuid, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "template-vgpu-claim", vgpuRequestName)
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

			createTemplateMetadataFile("my-template-claim", vgpuRequestName, &metadata.DeviceMetadata{
				ObjectMeta:   metav1.ObjectMeta{Name: "generated-claim-def456"},
				PodClaimName: ptr.To("my-template-claim"),
				Requests: []metadata.DeviceMetadataRequest{{
					Name: vgpuRequestName,
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &mdevUUID},
						},
					}},
				}},
			})

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{
				{Name: "existing-ref", ResourceClaimName: ptr.To("preexisting-claim")},
				{Name: "my-template-claim", ResourceClaimTemplateName: ptr.To("some-template")},
			}

			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, "existing-ref", "pci-req")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))

			uuid, err := GetMDevUUIDForClaim(tempDir, resourceClaims, "my-template-claim", vgpuRequestName)
			Expect(err).ToNot(HaveOccurred())
			Expect(uuid).To(Equal(mdevUUID))
		})

		It("should return error when template claim metadata not found", func() {
			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:                      "missing-template-claim",
				ResourceClaimTemplateName: ptr.To("nonexistent-template"),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, "missing-template-claim", request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read metadata"))
		})
	})

	Context("request name mismatch in metadata content", func() {
		It("should return error with available requests when request not found in metadata JSON", func() {
			createMetadataFile(claim1Name, request1Name, &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: claim1Name},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "actual-req",
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{},
					}},
				}},
			})

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To(claim1Name),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found in metadata for claim"))
			Expect(err.Error()).To(ContainSubstring(claim1Name))
			Expect(err.Error()).To(ContainSubstring("available requests: [actual-req]"))
		})
	})

	Context("multiple driver files per request", func() {
		It("should reject when multiple metadata files exist for the same request", func() {
			pciAddr1 := "0000:08:00.0"
			pciAddr2 := "0000:09:00.0"

			dir := filepath.Join(tempDir, resourceClaimsSubdir, "multi-driver-claim", gpuRequestName)
			writeMetadataJSON(dir, "gpu-a.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: multiDriverClaim},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: gpuRequestName,
					Devices: []metadata.Device{{
						Driver: "gpu-a.example.com",
						Pool:   worker0Pool,
						Name:   gpuDeviceName,
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr1},
						},
					}},
				}},
			})
			writeMetadataJSON(dir, "gpu-b.example.com", &metadata.DeviceMetadata{
				ObjectMeta: metav1.ObjectMeta{Name: multiDriverClaim},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: gpuRequestName,
					Devices: []metadata.Device{{
						Driver: "gpu-b.example.com",
						Pool:   worker0Pool,
						Name:   "gpu-1",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr2},
						},
					}},
				}},
			})

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To("multi-driver-claim"),
			}}

			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, gpuRequestName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("only supports exactly one driver per request"))
		})
	})

	Context("JSON stream version negotiation", func() {
		writeRawStreamFile := func(claimName, requestName, driverName string, objects ...string) {
			dir := filepath.Join(tempDir, resourceClaimsSubdir, claimName, requestName)
			Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
			var content []byte
			for _, obj := range objects {
				content = append(content, []byte(obj+"\n")...)
			}
			Expect(os.WriteFile(filepath.Join(dir, driverName+metadataFileSuffix), content, 0o600)).To(Succeed())
		}

		It("should skip unknown apiVersion and decode v1alpha1 from stream", func() {
			pciAddr := "0000:07:00.0"
			v2Obj := fmt.Sprintf(
				`{"apiVersion":"metadata.resource.k8s.io/v2beta1","kind":%q,"metadata":{"name":%q},"newField":"ignored"}`,
				deviceMetadataKind, claim1Name)
			v1Obj, err := json.Marshal(&metadata.DeviceMetadata{
				TypeMeta:   metav1.TypeMeta{APIVersion: metadata.APIVersionV1Alpha1, Kind: deviceMetadataKind},
				ObjectMeta: metav1.ObjectMeta{Name: claim1Name},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: request1Name,
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})
			Expect(err).ToNot(HaveOccurred())

			writeRawStreamFile(claim1Name, request1Name, "gpu.example.com", v2Obj, string(v1Obj))

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To(claim1Name),
			}}
			addr, err := GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal(pciAddr))
		})

		It("should return error when stream contains only unsupported versions", func() {
			writeRawStreamFile("claim2", request1Name, "gpu.example.com",
				fmt.Sprintf(`{"apiVersion":"metadata.resource.k8s.io/v99","kind":%q}`, deviceMetadataKind),
			)

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To("claim2"),
			}}
			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"no compatible metadata version found in stream (unknown versions: metadata.resource.k8s.io/v99)",
			))
		})

		It("should return error on empty stream", func() {
			writeRawStreamFile("claim3", request1Name, "gpu.example.com")

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To("claim3"),
			}}
			_, err := GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no metadata objects"))
		})

		It("should fail if an entry's apiVersion cannot be peeked even when a later entry is valid", func() {
			pciAddr := "0000:0a:00.0"
			v1Obj, err := json.Marshal(&metadata.DeviceMetadata{
				TypeMeta:   metav1.TypeMeta{APIVersion: metadata.APIVersionV1Alpha1, Kind: deviceMetadataKind},
				ObjectMeta: metav1.ObjectMeta{Name: "claim4"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: request1Name,
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})
			Expect(err).ToNot(HaveOccurred())
			writeRawStreamFile("claim4", request1Name, "gpu.example.com",
				`"not-an-object"`,
				string(v1Obj),
			)

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To("claim4"),
			}}
			_, err = GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("decode metadata object"))
		})

		It("should fail if a supported-version entry fails to unmarshal even when a later entry is valid", func() {
			pciAddr := "0000:0b:00.0"
			badV1 := fmt.Sprintf(
				`{"apiVersion":%q,"kind":%q,"metadata":{"name":"claim5"},"requests":"this-should-be-an-array"}`,
				metadata.APIVersionV1Alpha1,
				deviceMetadataKind,
			)
			goodV1, err := json.Marshal(&metadata.DeviceMetadata{
				TypeMeta:   metav1.TypeMeta{APIVersion: metadata.APIVersionV1Alpha1, Kind: deviceMetadataKind},
				ObjectMeta: metav1.ObjectMeta{Name: "claim5"},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: request1Name,
					Devices: []metadata.Device{{
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})
			Expect(err).ToNot(HaveOccurred())
			writeRawStreamFile("claim5", request1Name, "gpu.example.com", badV1, string(goodV1))

			resourceClaims := []v1.VirtualMachineInstanceResourceClaim{{
				Name:              testClaimName,
				ResourceClaimName: ptr.To("claim5"),
			}}
			_, err = GetPCIAddressForClaim(tempDir, resourceClaims, testClaimName, request1Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("decode %s", metadata.APIVersionV1Alpha1)))
		})
	})
})
