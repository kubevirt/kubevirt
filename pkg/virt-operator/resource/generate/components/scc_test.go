package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	secv1 "github.com/openshift/api/security/v1"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("SCC", func() {
	Context("virt-controller", func() {
		var scc *secv1.SecurityContextConstraints

		BeforeEach(func() {
			scc = NewKubeVirtControllerSCC("test")
		})

		It("should have allowPrivilegedContainer to false", func() {
			Expect(scc.AllowPrivilegedContainer).To(BeFalse())
		})

		It("should allow seccomp profiles used by Kubevirt", func() {
			Expect(scc.SeccompProfiles).To(ConsistOf(
				"runtime/default",
				"unconfined",
				"localhost/kubevirt/kubevirt.json",
			))
		})

		It("should allow capabilities used by Kubevirt", func() {
			Expect(scc.AllowedCapabilities).To(ConsistOf(
				v1.Capability("SYS_NICE"),
				v1.Capability("NET_BIND_SERVICE"),
			))
		})

		It("should allow HostDir volume plugin for host-disk", func() {
			Expect(scc.AllowHostDirVolumePlugin).To(BeTrue())
		})

		It("should allow any user", func() {
			Expect(scc.RunAsUser).To(BeEquivalentTo(
				secv1.RunAsUserStrategyOptions{
					Type: secv1.RunAsUserStrategyRunAsAny,
				}))
		})

		It("should allow any SELinux", func() {
			Expect(scc.SELinuxContext).To(BeEquivalentTo(
				secv1.SELinuxContextStrategyOptions{
					Type: secv1.SELinuxStrategyRunAsAny,
				}))
		})
	})
})
