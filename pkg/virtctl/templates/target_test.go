package templates_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var _ = Describe("Target", func() {

	DescribeTable("ParseTarget", func(arg, targetNamespace, targetName, targetKind string, success bool) {
		kind, namespace, name, err := templates.ParseTarget(arg)
		Expect(namespace).To(Equal(targetNamespace))
		Expect(name).To(Equal(targetName))
		Expect(kind).To(Equal(targetKind))
		if success {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}
	},
		Entry("only name", "testvmi", "", "testvmi", "vmi", true),
		Entry("name and namespace", "testvmi.default", "default", "testvmi", "vmi", true),
		Entry("kind vmi with name", "vmi/testvmi", "", "testvmi", "vmi", true),
		Entry("kind vmi with name and namespace", "vmi/testvmi.default", "default", "testvmi", "vmi", true),
		Entry("kind vm with name", "vm/testvm", "", "testvm", "vm", true),
		Entry("kind vm with name and namespace", "vm/testvm.default", "default", "testvm", "vm", true),
		Entry("kind invalid with name and namespace", "invalid/testvm.default", "", "", "", false),
		Entry("name with separator but missing namespace", "testvm.", "", "", "", false),
		Entry("namespace with separator but missing name", ".default", "", "", "", false),
		Entry("only valid kind", "vmi/", "", "", "", false),
		Entry("only separators", "/.", "", "", "", false),
		Entry("only dot", ".", "", "", "", false),
		Entry("only slash", "/", "", "", "", false),
	)
	DescribeTable("ParseSSHTarget", func(arg, targetNamespace, targetName, targetKind, targetUsername string, success bool) {
		kind, namespace, name, username, err := templates.ParseSSHTarget(arg)
		Expect(namespace).To(Equal(targetNamespace))
		Expect(name).To(Equal(targetName))
		Expect(kind).To(Equal(targetKind))
		Expect(username).To(Equal(targetUsername))
		if success {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}
	},
		Entry("username and name", "user@testvmi", "", "testvmi", "vmi", "user", true),
		Entry("username and name and namespace", "user@testvmi.default", "default", "testvmi", "vmi", "user", true),
		Entry("kind vmi with name and username", "user@vmi/testvmi", "", "testvmi", "vmi", "user", true),
		Entry("kind vmi with name and namespace and username", "user@vmi/testvmi.default", "default", "testvmi", "vmi", "user", true),
		Entry("only username", "user@", "", "", "", "", false),
		Entry("only at and target", "@testvmi", "", "", "", "", false),
		Entry("only separators", "@/.", "", "", "", "", false),
		Entry("only at", "@", "", "", "", "", false),
	)
	DescribeTable("ParseSCPTargets", func(arg0, arg1 string, expLocal templates.LocalSCPArgument, expRemote templates.RemoteSCPArgument, expToRemote bool) {
		local, remote, toRemote, err := templates.ParseSCPArguments(arg0, arg1)
		Expect(err).ToNot(HaveOccurred())
		Expect(local).To(Equal(expLocal))
		Expect(remote).To(Equal(expRemote))
		if expToRemote {
			Expect(toRemote).To(BeTrue())
		} else {
			Expect(toRemote).To(BeFalse())
		}
	},
		Entry("copy to remote location",
			"myfile.yaml", "cirros@remote.mynamespace:myfile.yaml",
			templates.LocalSCPArgument{Path: "myfile.yaml"},
			templates.RemoteSCPArgument{
				Kind: "vmi", Namespace: "mynamespace", Name: "remote", Username: "cirros", Path: "myfile.yaml",
			},
			true,
		),
		Entry("copy from remote location",
			"cirros@remote.mynamespace:myfile.yaml", "myfile.yaml",
			templates.LocalSCPArgument{Path: "myfile.yaml"},
			templates.RemoteSCPArgument{
				Kind: "vmi", Namespace: "mynamespace", Name: "remote", Username: "cirros", Path: "myfile.yaml",
			},
			false,
		),
	)

	DescribeTable("ParseSCPTargets should fail", func(arg0, arg1 string) {
		_, _, _, err := templates.ParseSCPArguments(arg0, arg1)
		Expect(err).To(HaveOccurred())
	},
		Entry("when two local locations are specified",
			"myfile.yaml", "otherfile.yaml",
		),
		Entry("when two remote locations are specified",
			"remotenode:myfile.yaml", "othernode:otherfile.yaml",
		),
	)
})
