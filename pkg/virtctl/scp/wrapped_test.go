package scp

import (
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/ssh"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Wrapped SCP", func() {

	var fakeLocal *LocalArgument
	var fakeRemote *RemoteArgument
	var fakeToRemote bool
	var scp SCP

	BeforeEach(func() {
		fakeLocal = &LocalArgument{
			Path: "/local/fakepath",
		}
		fakeRemote = &RemoteArgument{
			Kind:      "fake-kind",
			Namespace: "fake-ns",
			Name:      "fake-name",
			Path:      "/remote/fakepath",
		}
		fakeToRemote = false
		scp = SCP{}
	})

	Context("buildSCPTarget", func() {

		It("with SCP username", func() {
			scp.options = &ssh.SSHOptions{SSHUsername: "testuser"}
			scpTarget := scp.buildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("testuser@fake-kind.fake-name.fake-ns:/remote/fakepath"))
		})

		It("without SCP username", func() {
			scpTarget := scp.buildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("fake-kind.fake-name.fake-ns:/remote/fakepath"))
		})

		It("with recursive", func() {
			scp.recursive = true
			scpTarget := scp.buildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("-r"))
		})

		It("with preserve", func() {
			scp.preserve = true
			scpTarget := scp.buildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("-p"))
		})

		It("with recursive and preserve", func() {
			scp.recursive = true
			scp.preserve = true
			scpTarget := scp.buildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("-r"))
			Expect(scpTarget[1]).To(Equal("-p"))
		})

		It("toRemote = false", func() {
			scpTarget := scp.buildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("fake-kind.fake-name.fake-ns:/remote/fakepath"))
			Expect(scpTarget[1]).To(Equal("/local/fakepath"))
		})

		It("toRemote = true", func() {
			fakeToRemote = true
			scpTarget := scp.buildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("/local/fakepath"))
			Expect(scpTarget[1]).To(Equal("fake-kind.fake-name.fake-ns:/remote/fakepath"))
		})
	})
})
