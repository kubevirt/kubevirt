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
 */

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
			"myfile.yaml", "otherfile.yaml", "none of the two provided locations seems to be a remote location: \"myfile.yaml\" to \"otherfile.yaml\"",
		),
		Entry("when two remote locations are specified",
			"remotenode:myfile.yaml", "othernode:otherfile.yaml", "copying from a remote location to another remote location is not supported: \"remotenode:myfile.yaml\" to \"othernode:otherfile.yaml\"",
		),
	)
})
