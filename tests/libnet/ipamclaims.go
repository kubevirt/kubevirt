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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

var ipamGVR = schema.GroupVersionResource{
	Group:    "k8s.cni.cncf.io",
	Version:  "v1alpha1",
	Resource: "ipamclaims",
}

// CreateIPAMClaim creates an IPAM claim in the specified namespace using a DynamicClient.
func CreateIPAMClaim(ctx context.Context, namespace string, claim map[string]interface{}) (*unstructured.Unstructured, error) {
	kvclient := kubevirt.Client()
	unstr := &unstructured.Unstructured{Object: claim}
	return kvclient.DynamicClient().Resource(ipamGVR).Namespace(namespace).Create(
		ctx, unstr, metav1.CreateOptions{},
	)
}

// CreateIPAMClaimForVM creates an IPAM claim specifically for a VM with the VM label
func CreateIPAMClaimForVM(ctx context.Context, namespace, vmName, claimName, network, inface string) (*unstructured.Unstructured, error) {
	claim := map[string]interface{}{
		"apiVersion": "k8s.cni.cncf.io/v1alpha1",
		"kind":       "IPAMClaim",
		"metadata": map[string]interface{}{
			"name":      claimName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"kubevirt.io/vm": vmName,
			},
		},
		"spec": map[string]interface{}{
			"network":   network,
			"interface": inface,
		},
	}
	return CreateIPAMClaim(ctx, namespace, claim)
}
