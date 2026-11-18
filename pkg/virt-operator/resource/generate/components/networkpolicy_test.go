package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkv1 "k8s.io/api/networking/v1"
)

var _ = Describe("NetworkPolicies", func() {
	Describe("newSyncControllerPeerNP", func() {
		It("allows ingress on the default synchronization port from any source", func() {
			np := newSyncControllerPeerNP("kubevirt")
			Expect(np.Name).To(Equal(allowSyncControllerPeer))
			Expect(np.Spec.PolicyTypes).To(ConsistOf(networkv1.PolicyTypeIngress, networkv1.PolicyTypeEgress))
			Expect(np.Spec.Ingress).To(HaveLen(1))
			Expect(np.Spec.Ingress[0].From).To(BeEmpty())
			Expect(np.Spec.Ingress[0].Ports).To(HaveLen(1))
			Expect(np.Spec.Ingress[0].Ports[0].Port.IntVal).To(Equal(synchronizationPeerPort))
		})

		It("allows egress to any destination and port for remote-cluster endpoints", func() {
			np := newSyncControllerPeerNP("kubevirt")
			Expect(np.Spec.Egress).To(HaveLen(1))
			// No To: remote Ingress/Route peers have no local Pod identity.
			Expect(np.Spec.Egress[0].To).To(BeEmpty())
			// No Ports: remote synchronization listen port is configurable.
			Expect(np.Spec.Egress[0].Ports).To(BeEmpty())
		})
	})
})
