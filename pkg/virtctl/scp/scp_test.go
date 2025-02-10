package scp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/scp"
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
			"myfile.yaml", "cirros@mynamespace/remote:myfile.yaml",
			&scp.LocalArgument{Path: "myfile.yaml"},
			&scp.RemoteArgument{
				Namespace: "mynamespace", Name: "remote", Username: "cirros", Path: "myfile.yaml",
			},
			true,
		),
		Entry("copy from remote location",
			"cirros@mynamespace/remote:myfile.yaml", "myfile.yaml",
			&scp.LocalArgument{Path: "myfile.yaml"},
			&scp.RemoteArgument{
				Namespace: "mynamespace", Name: "remote", Username: "cirros", Path: "myfile.yaml",
			},
			false,
		),
	)

	DescribeTable("ParseTarget should fail", func(arg0, arg1, expectedError string) {
		_, _, _, err := scp.ParseTarget(arg0, arg1)
		Expect(err).To(MatchError(expectedError))
	},
		Entry("when two local locations are specified",
			"myfile.yaml", "otherfile.yaml", "none of the two provided locations seems to be a remote location: \"myfile.yaml\" to \"otherfile.yaml\"",
		),
		Entry("when two remote locations are specified",
			"remotenode:myfile.yaml", "othernode:otherfile.yaml", "copying from a remote location to another remote location is not supported: \"remotenode:myfile.yaml\" to \"othernode:otherfile.yaml\"",
		),
	)
})
