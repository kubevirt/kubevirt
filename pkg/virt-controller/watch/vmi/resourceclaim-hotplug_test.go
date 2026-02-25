package vmi

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("ResourceClaim hotplug matching", func() {
	newClaim := func(name string) *v1.ResourceClaim {
		return &v1.ResourceClaim{PodResourceClaim: k8sv1.PodResourceClaim{Name: name}}
	}

	It("does not match when pod has fewer claims than desired", func() {
		pod := &k8sv1.Pod{Spec: k8sv1.PodSpec{ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim-a"}}}}
		readyClaims := []*v1.ResourceClaim{newClaim("claim-a"), newClaim("claim-b")}

		Expect(podResourceClaimsMatchesReadyResourceClaims(pod, readyClaims)).To(BeFalse())
	})

	It("matches when pod and desired claims are equal", func() {
		pod := &k8sv1.Pod{Spec: k8sv1.PodSpec{ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim-a"}, {Name: "claim-b"}}}}
		readyClaims := []*v1.ResourceClaim{newClaim("claim-a"), newClaim("claim-b")}

		Expect(podResourceClaimsMatchesReadyResourceClaims(pod, readyClaims)).To(BeTrue())
	})
})
