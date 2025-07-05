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

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	ipamclaims "github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1"
)

// CreateIPAMClaim creates an IPAM claim in the specified namespace
func CreateIPAMClaim(ctx context.Context, namespace string, claim *ipamclaims.IPAMClaim) (*ipamclaims.IPAMClaim, error) {
	kvclient := kubevirt.Client()
	return kvclient.IPAMClaimsClient().K8sV1alpha1().IPAMClaims(namespace).Create(
		ctx, claim, metav1.CreateOptions{},
	)
}

// CreateIPAMClaimForVM creates an IPAM claim specifically for a VM with the VM label
func CreateIPAMClaimForVM(ctx context.Context, namespace, vmName, claimName string, network, inface string) (*ipamclaims.IPAMClaim, error) {
	claim := &ipamclaims.IPAMClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: namespace,
			Labels: map[string]string{
				"kubevirt.io/vm": vmName,
			},
		},
		Spec: ipamclaims.IPAMClaimSpec{
			Network:   network,
			Interface: inface,
		},
	}
	return CreateIPAMClaim(ctx, namespace, claim)
}
