package scp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/scp"
	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
)

var _ = Describe("SCP", func() {
	DescribeTable("ParseTarget", func(arg0, arg1 string, expLocal *scp.LocalArgument, expRemote *scp.RemoteArgument, expToRemote bool) {
		local, remote, toRemote, err := scp.ParseTarget(arg0, arg1)
		Expect(err).ToNot(HaveOccurred())
		Expect(local).To(Equal(expLocal))
		Expect(remote).To(Equal(expRemote))
		Expect(toRemote).To(Equal(expToRemote))
	},
		Entry("copy to remote location",
			"myfile.yaml", "cirros@vmi/remote/mynamespace:myfile.yaml",
			&scp.LocalArgument{Path: "myfile.yaml"},
			&scp.RemoteArgument{
				Kind: "vmi", Namespace: "mynamespace", Name: "remote", Username: "cirros", Path: "myfile.yaml",
			},
			true,
		),
		Entry("copy from remote location",
			"cirros@vmi/remote/mynamespace:myfile.yaml", "myfile.yaml",
			&scp.LocalArgument{Path: "myfile.yaml"},
			&scp.RemoteArgument{
				Kind: "vmi", Namespace: "mynamespace", Name: "remote", Username: "cirros", Path: "myfile.yaml",
			},
			false,
		),
	)

	DescribeTable("ParseTarget should fail", func(arg0, arg1, expectedError string) {
		_, _, _, err := scp.ParseTarget(arg0, arg1)
		Expect(err).To(MatchError(expectedError))
	},
		Entry("when two local locations are specified",
			"myfile.yaml", "otherfile.yaml",
			"none of the two provided locations seems to be a remote location: \"myfile.yaml\" to \"otherfile.yaml\"",
		),
		Entry("when two remote locations are specified",
			"remotenode:myfile.yaml", "othernode:otherfile.yaml",
			"copying from a remote location to another remote location is not supported: \"remotenode:myfile.yaml\" to \"othernode:otherfile.yaml\"",
		),
	)

	Context("BuildSCPTarget", func() {
		const fakeToRemote = false

		var (
			fakeLocal  *scp.LocalArgument
			fakeRemote *scp.RemoteArgument
		)

		BeforeEach(func() {
			fakeLocal = &scp.LocalArgument{
				Path: "/local/fakepath",
			}
			fakeRemote = &scp.RemoteArgument{
				Kind:      "fake-kind",
				Namespace: "fake-ns",
				Name:      "fake-name",
				Path:      "/remote/fakepath",
			}
		})

		It("with SCP username", func() {
			c := scp.NewSCP(&ssh.SSHOptions{SSHUsername: "testuser"}, false, false)
			scpTarget := c.BuildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("testuser@fake-kind.fake-name.fake-ns:/remote/fakepath"))
		})

		It("without SCP username", func() {
			c := scp.NewSCP(&ssh.SSHOptions{}, false, false)
			scpTarget := c.BuildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("fake-kind.fake-name.fake-ns:/remote/fakepath"))
		})

		It("with recursive", func() {
			c := scp.NewSCP(&ssh.SSHOptions{}, true, false)
			scpTarget := c.BuildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("-r"))
		})

		It("with preserve", func() {
			c := scp.NewSCP(&ssh.SSHOptions{}, false, true)
			scpTarget := c.BuildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("-p"))
		})

		It("with Recursive and Preserve", func() {
			c := scp.NewSCP(&ssh.SSHOptions{}, true, true)
			scpTarget := c.BuildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("-r"))
			Expect(scpTarget[1]).To(Equal("-p"))
		})

		It("toRemote = false", func() {
			c := scp.NewSCP(&ssh.SSHOptions{}, false, false)
			scpTarget := c.BuildSCPTarget(fakeLocal, fakeRemote, fakeToRemote)
			Expect(scpTarget[0]).To(Equal("fake-kind.fake-name.fake-ns:/remote/fakepath"))
			Expect(scpTarget[1]).To(Equal("/local/fakepath"))
		})

		It("toRemote = true", func() {
			c := scp.NewSCP(&ssh.SSHOptions{}, false, false)
			scpTarget := c.BuildSCPTarget(fakeLocal, fakeRemote, true)
			Expect(scpTarget[0]).To(Equal("/local/fakepath"))
			Expect(scpTarget[1]).To(Equal("fake-kind.fake-name.fake-ns:/remote/fakepath"))
		})
	})
})
