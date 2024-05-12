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

package ipamclaims

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-controller/ipamclaims/libipam"
	"kubevirt.io/kubevirt/pkg/virt-controller/network"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	ipamclaims "github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1"
	ipamclaimsclient "github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1/apis/clientset/versioned"

	networkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned"
)

type IPAMClaimsManager struct {
	networkClient    networkclient.Interface
	ipamClaimsClient ipamclaimsclient.Interface
}

func NewIPAMClaimsManager(networkClient networkclient.Interface, ipamClaimsClient ipamclaimsclient.Interface) *IPAMClaimsManager {
	return &IPAMClaimsManager{
		networkClient:    networkClient,
		ipamClaimsClient: ipamClaimsClient,
	}
}

func (m *IPAMClaimsManager) CreateNewPodIPAMClaims(vmi *virtv1.VirtualMachineInstance, ownerRef *v1.OwnerReference) error {
	if ownerRef == nil {
		ownerRef = v1.NewControllerRef(vmi, virtv1.VirtualMachineInstanceGroupVersionKind)
	}
	nonAbsentNetworks := filterNonAbsentNetworks(vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks)
	networkToIPAMClaimParams, err := m.GetNetworkToIPAMClaimParams(
		vmi.Namespace,
		vmi.Name,
		vmispec.FilterMultusNonDefaultNetworks(nonAbsentNetworks))
	if err != nil {
		return fmt.Errorf("failed composing networkToIPAMClaimName: %w", err)
	}

	claims := composeIPAMClaims(vmi.Namespace, ownerRef, networkToIPAMClaimParams)
	if err := m.createNewPodIPAMClaims(vmi.Namespace, claims); err != nil {
		return fmt.Errorf("failed IPAMClaims creation for VMI %s: %w", vmi.Name, err)
	}

	return nil
}

func composeIPAMClaims(namespace string, ownerRef *v1.OwnerReference, networkToIPAMClaimParams map[string]libipam.IPAMClaimParams) []*ipamclaims.IPAMClaim {
	claims := []*ipamclaims.IPAMClaim{}
	for netName, ipamClaimParams := range networkToIPAMClaimParams {
		claims = append(claims, composeIPAMClaim(
			namespace,
			*ownerRef,
			ipamClaimParams,
			namescheme.GenerateHashedInterfaceName(netName),
		))
	}

	return claims
}

func (m *IPAMClaimsManager) createNewPodIPAMClaims(namespace string, claims []*ipamclaims.IPAMClaim) error {
	for _, claim := range claims {
		_, err := m.ipamClaimsClient.K8sV1alpha1().IPAMClaims(namespace).Create(
			context.Background(),
			claim,
			v1.CreateOptions{},
		)

		if err != nil {
			if !k8serrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create IPAMClaim: %w", err)
			}

			err = m.ensureValidIPAMClaimForVMI(namespace, claim.Name, claim.OwnerReferences[0].UID)
			if err != nil {
				return fmt.Errorf("failed validating IPAMClaim %s/%s: %w", namespace, claim.Name, err)
			}
		}
	}

	return nil
}

func (m *IPAMClaimsManager) ensureValidIPAMClaimForVMI(namespace string, claimName string, expectedOwnerUID types.UID) error {
	currentClaim, err := m.ipamClaimsClient.K8sV1alpha1().IPAMClaims(namespace).Get(
		context.Background(),
		claimName,
		v1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed getting IPAMClaim: %w", err)
	}

	if len(currentClaim.OwnerReferences) != 1 || currentClaim.OwnerReferences[0].UID != expectedOwnerUID {
		return fmt.Errorf("failed validating IPAMClaim, wrong IPAMClaim with the same name still exists")
	}

	return nil
}

func composeIPAMClaim(namespace string, ownerRef v1.OwnerReference, ipamClaimParams libipam.IPAMClaimParams, interfaceName string) *ipamclaims.IPAMClaim {
	return &ipamclaims.IPAMClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      ipamClaimParams.ClaimName,
			Namespace: namespace,
			OwnerReferences: []v1.OwnerReference{
				ownerRef,
			},
		},
		Spec: ipamclaims.IPAMClaimSpec{
			Network:   ipamClaimParams.NetworkName,
			Interface: interfaceName,
		},
	}
}

func (m *IPAMClaimsManager) GetNetworkToIPAMClaimParams(namespace string, vmiName string, multusNonDefaultNetworks []virtv1.Network) (map[string]libipam.IPAMClaimParams, error) {
	nads, err := network.GetNetworkAttachmentDefinitionByName(m.networkClient.K8sCniCncfIoV1(), namespace, multusNonDefaultNetworks)
	if err != nil {
		return nil, fmt.Errorf("failed retrieving network attachment definitions: %w", err)
	}

	networkToIPAMClaimParams, err := ExtractNetworkToIPAMClaimParams(nads, vmiName)
	if err != nil {
		return nil, fmt.Errorf("failed extracting ipam claim params: %w", err)
	}

	return networkToIPAMClaimParams, nil
}

func ExtractNetworkToIPAMClaimParams(nadMap map[string]*networkv1.NetworkAttachmentDefinition, vmiName string) (map[string]libipam.IPAMClaimParams, error) {
	networkToIPAMClaimParams := map[string]libipam.IPAMClaimParams{}
	for networkName, nad := range nadMap {
		netConf, err := libipam.GetPersistentIPsConf(nad)
		if err != nil {
			return nil, fmt.Errorf("failed retrieving netConf: %w", err)
		}
		if netConf.AllowPersistentIPs {
			networkToIPAMClaimParams[networkName] = libipam.IPAMClaimParams{
				ClaimName:   fmt.Sprintf("%s.%s", vmiName, networkName),
				NetworkName: netConf.Name,
			}
		}
	}
	return networkToIPAMClaimParams, nil
}

func filterNonAbsentNetworks(interfaces []virtv1.Interface, networks []virtv1.Network) []virtv1.Network {
	nonAbsentIfaces := vmispec.FilterInterfacesSpec(interfaces, func(iface virtv1.Interface) bool {
		return iface.State != virtv1.InterfaceStateAbsent
	})

	return vmispec.FilterNetworksByInterfaces(networks, nonAbsentIfaces)
}

func WithIPAMClaimRef(networkToIPAMClaimParams map[string]libipam.IPAMClaimParams, networkToPodIfaceMap map[string]string) network.Option {
	return func(mnap network.MultusNetworkAnnotationPool, namespace string, networks []virtv1.Network) network.MultusNetworkAnnotationPool {
		for _, net := range vmispec.FilterMultusNonDefaultNetworks(networks) {
			namespace, networkName := vmispec.GetNamespaceAndNetworkName(namespace, net.Multus.NetworkName)
			i, element := mnap.FindMultusAnnotation(namespace, networkName, networkToPodIfaceMap[net.Name])
			if element == nil {
				continue
			}
			element.IPAMClaimReference = networkToIPAMClaimParams[net.Name].ClaimName
			mnap.Set(i, *element)
		}
		return mnap
	}
}
