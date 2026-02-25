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

package libnet

import (
	"context"
	"encoding/json"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/gomega"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

// VfConfigParameters represents the parameters for SR-IOV VF configuration
type VfConfigParameters struct {
	APIVersion       string `json:"apiVersion"`
	Kind             string `json:"kind"`
	NetAttachDefName string `json:"netAttachDefName"`
	Driver           string `json:"driver"`
	AddVhostMount    bool   `json:"addVhostMount"`
}

// NewSRIOVResourceClaimTemplate creates a ResourceClaimTemplate for SR-IOV network devices
// This creates a template with proper VfConfig parameters that reference a NetworkAttachmentDefinition
func NewSRIOVResourceClaimTemplate(name, namespace, netAttachDefName, driverName string) *resourcev1.ResourceClaimTemplate {
	vfConfig := VfConfigParameters{
		APIVersion:       "sriovnetwork.k8snetworkplumbingwg.io/v1alpha1",
		Kind:             "VfConfig",
		NetAttachDefName: netAttachDefName,
		Driver:           "vfio-pci",
		AddVhostMount:    true,
	}

	vfConfigJSON, err := json.Marshal(vfConfig)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return &resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: resourcev1.ResourceClaimTemplateSpec{
			Spec: resourcev1.ResourceClaimSpec{
				Devices: resourcev1.DeviceClaim{
					Requests: []resourcev1.DeviceRequest{
						{
							Name: "vf",
							Exactly: &resourcev1.ExactDeviceRequest{
								DeviceClassName: driverName,
							},
						},
					},
					Config: []resourcev1.DeviceClaimConfiguration{
						{
							Requests: []string{"vf"},
							DeviceConfiguration: resourcev1.DeviceConfiguration{
								Opaque: &resourcev1.OpaqueDeviceConfiguration{
									Driver: driverName,
									Parameters: runtime.RawExtension{
										Raw: vfConfigJSON,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// NewSriovNetAttachDefWithIPAM creates a SR-IOV NetworkAttachmentDefinition with IPAM configuration
// matching the format used by DRA SR-IOV drivers
func NewSriovNetAttachDefWithIPAM(name string, vlanID int, opts ...pluginConfOption) *nadv1.NetworkAttachmentDefinition {
	config := map[string]interface{}{
		"cniVersion": "0.4.0",
		"name":       name,
		"type":       "sriov",
		"vlan":       vlanID,
		"spoofchk":   "on",
		"trust":      "on",
		"vlanQoS":    0,
		"logLevel":   "info",
		"ipam": map[string]interface{}{
			"type": "host-local",
			"ranges": [][]map[string]interface{}{
				{
					{
						"subnet": "10.0.1.0/24",
					},
				},
			},
		},
	}

	// Apply plugin configuration options (e.g., WithLinkState)
	for _, opt := range opts {
		opt(config)
	}

	configJSON, err := json.Marshal(config)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return NewNetAttachDef(name, string(configJSON))
}

// CreateSRIOVNetworkWithDRA creates both a NetworkAttachmentDefinition and ResourceClaimTemplate
// This matches the real-world DRA SR-IOV setup with proper VfConfig
func CreateSRIOVNetworkWithDRA(ctx context.Context, namespace, networkName, driverName string, vlanID int, opts ...pluginConfOption) error {
	// Create NetworkAttachmentDefinition with SR-IOV CNI config
	netAttachDef := NewSriovNetAttachDefWithIPAM(networkName, vlanID, opts...)
	_, err := CreateNetAttachDef(ctx, namespace, netAttachDef)
	if err != nil {
		return err
	}

	// Create ResourceClaimTemplate that references the NAD
	template := NewSRIOVResourceClaimTemplate(
		"single-vf-"+networkName,
		namespace,
		networkName,
		driverName,
	)
	_, err = CreateResourceClaimTemplate(ctx, namespace, template)
	return err
}

// CreateResourceClaimTemplate creates a ResourceClaimTemplate in the specified namespace
func CreateResourceClaimTemplate(
	ctx context.Context, namespace string, template *resourcev1.ResourceClaimTemplate,
) (*resourcev1.ResourceClaimTemplate, error) {
	kvclient := kubevirt.Client()
	return kvclient.ResourceV1().ResourceClaimTemplates(namespace).Create(
		ctx, template, metav1.CreateOptions{},
	)
}

// DeleteResourceClaimTemplate deletes a ResourceClaimTemplate from the specified namespace
func DeleteResourceClaimTemplate(ctx context.Context, namespace, name string) error {
	kvclient := kubevirt.Client()
	return kvclient.ResourceV1().ResourceClaimTemplates(namespace).Delete(
		ctx, name, metav1.DeleteOptions{},
	)
}

// CreateResourceClaim creates a ResourceClaim in the specified namespace
func CreateResourceClaim(
	ctx context.Context, namespace string, resourceClaim *resourcev1.ResourceClaim,
) (*resourcev1.ResourceClaim, error) {
	kvclient := kubevirt.Client()
	return kvclient.ResourceV1().ResourceClaims(namespace).Create(
		ctx, resourceClaim, metav1.CreateOptions{},
	)
}

// DeleteResourceClaim deletes a ResourceClaim from the specified namespace
func DeleteResourceClaim(ctx context.Context, namespace, name string) error {
	kvclient := kubevirt.Client()
	return kvclient.ResourceV1().ResourceClaims(namespace).Delete(
		ctx, name, metav1.DeleteOptions{},
	)
}

// NewSRIOVResourceClaim creates a simple ResourceClaim for SR-IOV network devices (for tests)
func NewSRIOVResourceClaim(name, driverName, resourceName string, count int) *resourcev1.ResourceClaim {
	return &resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: []resourcev1.DeviceRequest{
					{
						Name: "vf",
						Exactly: &resourcev1.ExactDeviceRequest{
							DeviceClassName: driverName,
							Count:           int64(count),
						},
					},
				},
			},
		},
	}
}
