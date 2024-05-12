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

package ipamclaims_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-controller/ipamclaims"
	"kubevirt.io/kubevirt/pkg/virt-controller/ipamclaims/libipam"
	"kubevirt.io/kubevirt/pkg/virt-controller/network"

	fakenetworkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/fake"

	ipamv1alpha1 "github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1"
	fakeipamclaimclient "github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1/apis/clientset/versioned/fake"
)

const (
	nadSuffix              = "-net"
	nsSuffix               = "-ns"
	redNetworkLogicalName  = "red"
	redNetworkNadName      = redNetworkLogicalName + nadSuffix
	namespace              = redNetworkLogicalName + nsSuffix
	blueNetworkLogicalName = "blue"
	blueNetworkNadName     = blueNetworkLogicalName + nadSuffix
	defaultNamespace       = "default"
)

const (
	vmiName        = "testvmi"
	vmUID          = "vmUID"
	vmiUID         = "vmiUID"
	nadNetworkName = "nad_network_name"
)

var _ = Describe("CreateNewPodIPAMClaims", func() {
	var networkClient *fakenetworkclient.Clientset
	var ipamClaimsClient *fakeipamclaimclient.Clientset
	var ipamClaimsManager *ipamclaims.IPAMClaimsManager
	var vmi *virtv1.VirtualMachineInstance

	BeforeEach(func() {
		networkClient = fakenetworkclient.NewSimpleClientset()
		ipamClaimsClient = fakeipamclaimclient.NewSimpleClientset()
		ipamClaimsManager = ipamclaims.NewIPAMClaimsManager(networkClient, ipamClaimsClient)
	})

	BeforeEach(func() {
		vmi = libvmi.New(
			libvmi.WithNamespace(namespace),
			libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(redNetworkLogicalName, redNetworkNadName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(blueNetworkLogicalName, defaultNamespace+"/"+blueNetworkNadName)),
			libvmi.WithNetwork(libvmi.MultusNetwork("absent", "absent-net")),
			libvmi.WithInterface(virtv1.Interface{Name: defaultNamespace}),
			libvmi.WithInterface(virtv1.Interface{Name: redNetworkLogicalName}),
			libvmi.WithInterface(virtv1.Interface{Name: blueNetworkLogicalName}),
			libvmi.WithInterface(virtv1.Interface{Name: "absent", State: virtv1.InterfaceStateAbsent}),
		)
		vmi.UID = vmiUID
	})

	Context("With allowPersistentIPs enabled in the NADs", func() {
		BeforeEach(func() {
			persistentIPs := map[string]struct{}{redNetworkNadName: {}, blueNetworkNadName: {}}
			Expect(createNADs(networkClient, vmi.Namespace, vmi.Spec.Networks, persistentIPs)).To(Succeed())
		})

		It("should create the expected IPAMClaims", func() {
			ownerRef := &v1.OwnerReference{
				APIVersion:         "kubevirt.io/v1",
				Kind:               "VirtualMachine",
				Name:               vmi.Name,
				UID:                vmUID,
				Controller:         pointer.P(true),
				BlockOwnerDeletion: pointer.P(true),
			}
			Expect(ipamClaimsManager.CreateNewPodIPAMClaims(vmi, ownerRef)).To(Succeed())

			ipamClaimsList, err := ipamClaimsClient.K8sV1alpha1().IPAMClaims(vmi.Namespace).List(
				context.Background(),
				v1.ListOptions{},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipamClaimsList.Items).To(HaveLen(2))
			assertIPAMClaim(ipamClaimsList.Items[0], vmi.Namespace, vmi.Name, blueNetworkLogicalName, "pod16477688c0e", "VirtualMachine")
			assertIPAMClaim(ipamClaimsList.Items[1], vmi.Namespace, vmi.Name, redNetworkLogicalName, "podb1f51a511f1", "VirtualMachine")
		})

		Context("When IPAMClaims already exist", func() {
			var ownerRef *v1.OwnerReference

			BeforeEach(func() {
				ownerRef = &v1.OwnerReference{
					APIVersion:         "kubevirt.io/v1",
					Kind:               "VirtualMachine",
					Name:               vmi.Name,
					UID:                vmUID,
					Controller:         pointer.P(true),
					BlockOwnerDeletion: pointer.P(true),
				}
				Expect(ipamClaimsManager.CreateNewPodIPAMClaims(vmi, ownerRef)).To(Succeed())

				ipamClaimsList, err := ipamClaimsClient.K8sV1alpha1().IPAMClaims(vmi.Namespace).List(
					context.Background(),
					v1.ListOptions{},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(ipamClaimsList.Items).To(HaveLen(2))
			})

			It("with the expected owner UID, should not fail re-creation/validation attempt", func() {
				Expect(ipamClaimsManager.CreateNewPodIPAMClaims(vmi, ownerRef)).To(Succeed())

				ipamClaimsList, err := ipamClaimsClient.K8sV1alpha1().IPAMClaims(vmi.Namespace).List(
					context.Background(),
					v1.ListOptions{},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(ipamClaimsList.Items).To(HaveLen(2))
			})

			It("with a different owner UID, should fail fail re-creation/validation attempt", func() {
				ownerRef.UID = "differentUID"
				err := ipamClaimsManager.CreateNewPodIPAMClaims(vmi, ownerRef)
				Expect(err).To(MatchError(ContainSubstring("wrong IPAMClaim with the same name still exists")))
			})
		})
	})

	Context("With allowPersistentIPs disabled in the NADs", func() {
		BeforeEach(func() {
			Expect(createNADs(networkClient, vmi.Namespace, vmi.Spec.Networks, map[string]struct{}{})).To(Succeed())
		})

		It("should not create IPAMClaims", func() {
			ownerRef := &v1.OwnerReference{
				APIVersion:         "kubevirt.io/v1",
				Kind:               "VirtualMachine",
				Name:               vmi.Name,
				UID:                vmUID,
				Controller:         pointer.P(true),
				BlockOwnerDeletion: pointer.P(true),
			}
			Expect(ipamClaimsManager.CreateNewPodIPAMClaims(vmi, ownerRef)).To(Succeed())

			ipamClaimsList, err := ipamClaimsClient.K8sV1alpha1().IPAMClaims(vmi.Namespace).List(
				context.Background(),
				v1.ListOptions{},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipamClaimsList.Items).To(BeEmpty())
		})
	})

	Context("With mixed allowPersistentIPs settings, standalone VMI", func() {
		BeforeEach(func() {
			Expect(createNADs(networkClient, vmi.Namespace, vmi.Spec.Networks, map[string]struct{}{redNetworkNadName: {}})).To(Succeed())
		})

		It("should create IPAMClaims just for networks with persistent IP", func() {
			Expect(ipamClaimsManager.CreateNewPodIPAMClaims(vmi, nil)).To(Succeed())

			ipamClaimsList, err := ipamClaimsClient.K8sV1alpha1().IPAMClaims(vmi.Namespace).List(
				context.Background(),
				v1.ListOptions{},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipamClaimsList.Items).To(HaveLen(1))
			assertIPAMClaim(ipamClaimsList.Items[0], vmi.Namespace, vmi.Name, redNetworkLogicalName, "podb1f51a511f1", "VirtualMachineInstance")
		})
	})
})

var _ = Describe("GetNetworkToIPAMClaimParams", func() {
	var networkClient *fakenetworkclient.Clientset
	var ipamClaimsManager *ipamclaims.IPAMClaimsManager
	var networks []virtv1.Network

	BeforeEach(func() {
		networkClient = fakenetworkclient.NewSimpleClientset()
		ipamClaimsManager = ipamclaims.NewIPAMClaimsManager(networkClient, fakeipamclaimclient.NewSimpleClientset())
	})

	BeforeEach(func() {
		networks = []virtv1.Network{
			*libvmi.MultusNetwork(redNetworkLogicalName, redNetworkNadName),
			*libvmi.MultusNetwork(blueNetworkLogicalName, blueNetworkNadName),
		}

		persistentIPs := map[string]struct{}{redNetworkNadName: {}}
		Expect(createNADs(networkClient, namespace, networks, persistentIPs)).To(Succeed())
	})

	It("should return the expected IPAMClaim parameters", func() {
		networkToIPAMClaimParams, err := ipamClaimsManager.GetNetworkToIPAMClaimParams(namespace, vmiName, networks)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkToIPAMClaimParams).To(Equal(map[string]libipam.IPAMClaimParams{
			redNetworkLogicalName: {
				ClaimName:   fmt.Sprintf("%s.%s", vmiName, redNetworkLogicalName),
				NetworkName: nadNetworkName,
			}}))
	})
})

var _ = Describe("ExtractNetworkToIPAMClaimParams", func() {
	It("should successfully extract expected network to IPAM claim params", func() {
		nadMap := map[string]*networkv1.NetworkAttachmentDefinition{
			blueNetworkLogicalName: {
				Spec: networkv1.NetworkAttachmentDefinitionSpec{
					Config: fmt.Sprintf(`{"name": "%s"}`, nadNetworkName),
				},
			},
			redNetworkLogicalName: {
				Spec: networkv1.NetworkAttachmentDefinitionSpec{
					Config: fmt.Sprintf(`{"allowPersistentIPs": true, "name": "%s"}`, nadNetworkName),
				},
			},
		}

		expected := map[string]libipam.IPAMClaimParams{
			redNetworkLogicalName: {
				ClaimName:   fmt.Sprintf("%s.%s", vmiName, redNetworkLogicalName),
				NetworkName: "nad_network_name",
			},
		}

		networkToIPAMClaimParams, err := ipamclaims.ExtractNetworkToIPAMClaimParams(nadMap, vmiName)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkToIPAMClaimParams).To(Equal(expected))
	})

	It("should fail when nad is misconfigured", func() {
		nadMap := map[string]*networkv1.NetworkAttachmentDefinition{
			redNetworkLogicalName: {
				Spec: networkv1.NetworkAttachmentDefinitionSpec{
					Config: `{"allowPersistentIPs": true}`,
				},
			},
		}
		_, err := ipamclaims.ExtractNetworkToIPAMClaimParams(nadMap, vmiName)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("failed retrieving netConf: failed to obtain network name: missing required field"))
	})
})

var _ = Describe("WithIPAMClaimRef", func() {
	It("should add ipam-claim-reference to multus annotation according networkToIPAMClaimParams", func() {
		vmi := libvmi.New(
			libvmi.WithNamespace("default"),
			libvmi.WithInterface(virtv1.Interface{Name: "blue"}),
			libvmi.WithInterface(virtv1.Interface{Name: "red"}),
			libvmi.WithNetwork(libvmi.MultusNetwork("blue", "test1")),
			libvmi.WithNetwork(libvmi.MultusNetwork("red", "other-namespace/test2")),
		)
		vmi.Name = "testvmi"

		networkToIPAMClaimParams := map[string]libipam.IPAMClaimParams{
			"red": {
				ClaimName:   "testvmi.red",
				NetworkName: "network_name",
			}}
		networkToPodIfaceMap := map[string]string{"red": "podb1f51a511f1"}
		Expect(network.GenerateMultusCNIAnnotation(
			vmi.Namespace,
			vmi.Spec.Domain.Devices.Interfaces,
			vmi.Spec.Networks,
			nil,
			ipamclaims.WithIPAMClaimRef(networkToIPAMClaimParams, networkToPodIfaceMap))).To(MatchJSON(
			`[
				{"name": "test1","namespace": "default","interface": "pod16477688c0e"},
				{"name": "test2","namespace": "other-namespace","interface": "podb1f51a511f1","ipam-claim-reference": "testvmi.red"}
			]`,
		))
	})
})

func assertIPAMClaim(claim ipamv1alpha1.IPAMClaim, namespace, vmiName, networkLogicalName, interfaceName, ownerKind string) {
	uid := types.UID(vmUID)
	if ownerKind == "VirtualMachineInstance" {
		uid = types.UID(vmiUID)
	}

	ExpectWithOffset(1, claim.OwnerReferences).To(ConsistOf(v1.OwnerReference{
		APIVersion:         "kubevirt.io/v1",
		Kind:               ownerKind,
		Name:               vmiName,
		UID:                uid,
		Controller:         pointer.P(true),
		BlockOwnerDeletion: pointer.P(true),
	}))
	ExpectWithOffset(1, claim.Name).To(Equal(fmt.Sprintf("%s.%s", vmiName, networkLogicalName)))
	ExpectWithOffset(1, claim.Namespace).To(Equal(namespace))
	ExpectWithOffset(1, claim.Spec).To(Equal(ipamv1alpha1.IPAMClaimSpec{
		Network:   "nad_network_name",
		Interface: interfaceName,
	}))
}

func createNADs(networkClient *fakenetworkclient.Clientset, namespace string, networks []virtv1.Network, persistentIPs map[string]struct{}) error {
	gvr := schema.GroupVersionResource{
		Group:    "k8s.cni.cncf.io",
		Version:  "v1",
		Resource: "network-attachment-definitions",
	}
	for _, net := range networks {
		if net.Multus == nil {
			continue
		}
		ns, networkName := vmispec.GetNamespaceAndNetworkName(namespace, net.Multus.NetworkName)
		nad := &networkv1.NetworkAttachmentDefinition{
			ObjectMeta: v1.ObjectMeta{
				Name:      networkName,
				Namespace: ns,
			},
		}

		if _, exists := persistentIPs[networkName]; exists {
			nad.Spec.Config = fmt.Sprintf(`{"allowPersistentIPs": true, "name": "%s"}`, nadNetworkName)
		} else {
			nad.Spec.Config = fmt.Sprintf(`{"name": "%s"}`, nadNetworkName)
		}

		err := networkClient.Tracker().Create(gvr, nad, ns)
		if err != nil {
			return err
		}
	}

	return nil
}
