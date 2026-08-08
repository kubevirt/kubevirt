package scp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/scp"
	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
)

var _ = Describe("SCP", func() {
	const transferFileName = "myfile.yaml"
	const testCirrosUsername = "cirros"
	const myNamespace = "mynamespace"
	const remoteVMIName = "remote"
	const remoteKind = "vmi"

	DescribeTable("ParseTarget", func(arg0, arg1 string, expLocal *scp.LocalArgument, expRemote *scp.RemoteArgument, expToRemote bool) {
		local, remote, toRemote, err := scp.ParseTarget(arg0, arg1)
		Expect(err).ToNot(HaveOccurred())
		Expect(local).To(Equal(expLocal))
		Expect(remote).To(Equal(expRemote))
		Expect(toRemote).To(Equal(expToRemote))
	},
		Entry("copy to remote location",
			transferFileName, testCirrosUsername+"@vmi/remote/mynamespace:"+transferFileName,
			&scp.LocalArgument{Path: transferFileName},
			&scp.RemoteArgument{
				Kind: remoteKind, Namespace: myNamespace, Name: remoteVMIName, Username: testCirrosUsername, Path: transferFileName,
			},
			true,
		),
		Entry("copy from remote location",
			testCirrosUsername+"@vmi/remote/mynamespace:"+transferFileName, transferFileName,
			&scp.LocalArgument{Path: transferFileName},
			&scp.RemoteArgument{
				Kind: remoteKind, Namespace: myNamespace, Name: remoteVMIName, Username: testCirrosUsername, Path: transferFileName,
			},
			false,
		),
	)

	DescribeTable("ParseTarget should fail", func(arg0, arg1, expectedError string) {
		_, _, _, err := scp.ParseTarget(arg0, arg1)
		Expect(err).To(MatchError(expectedError))
	},
		Entry("when two local locations are specified",
			transferFileName, "otherfile.yaml",
			"none of the two provided locations seems to be a remote location: \""+transferFileName+"\" to \"otherfile.yaml\"",
		),
		Entry("when two remote locations are specified",
			"remotenode:"+transferFileName, "othernode:otherfile.yaml",
			"copying from a remote location to another remote location is not supported: \"remotenode:"+
				transferFileName+"\" to \"othernode:otherfile.yaml\"",
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
