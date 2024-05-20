package downwardmetrics

import (
	"kubevirt.io/kubevirt/pkg/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reporter", func() {
	It("should return a hash of the node's host name", func() {
		// A dummy domain that doesn't exist
		nodeName := "node.example.invalid"
		hashedNodeName := util.HashString(nodeName)

		reporter := NewReporter(nodeName)
		Expect(reporter.staticHostInfo.HostName).ToNot(Equal(nodeName))
		Expect(reporter.staticHostInfo.HostName).To(Equal(hashedNodeName))
	})

	It("should return a 'HostName' string not longer than the maximum allowed", func() {
		// The metrics `HostName` value is used as an uuid of the node (to detect changes),
		// and is not used to do dns queries. So, we don't care about the maximum length of
		// 63 chars of each label (i.e., a fqdn is a dot-separated list of labels), let's
		// just check the total hostname length to be lees than 253 characters
		// (the effective maximum length of a DNS name), in case the client has reserved
		// that space as maximum.

		// A dummy domain that doesn't exist
		nodeName := "node.example.invalid"
		reporter := NewReporter(nodeName)

		Expect(len(reporter.staticHostInfo.HostName)).To(BeNumerically("<=", 253))
	})
})
