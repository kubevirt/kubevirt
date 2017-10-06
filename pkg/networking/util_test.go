package networking

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
)

var _ = Describe("Util", func() {
	It("should return lo interface for 127.0.0.1", func() {
		iface, err := GetInterfaceFromIP("127.0.0.1")
		Expect(err).ToNot(HaveOccurred())
		Expect(iface.Name).To(Equal("lo"))
	})

	It("should return the internal IP from the node status", func() {
		node := &v1.Node{
			Status: v1.NodeStatus{
				Addresses: []v1.NodeAddress{
					{
						Type:    v1.NodeExternalIP,
						Address: "4.3.2.1",
					},
					{
						Type:    v1.NodeInternalIP,
						Address: "1.2.3.4",
					},
				},
			},
		}
		Expect(GetNodeInternalIP(node)).To(Equal("1.2.3.4"))
	})

	It("should return an empyt string if no internal node IP is present", func() {
		node := &v1.Node{
			Status: v1.NodeStatus{
				Addresses: []v1.NodeAddress{
					{
						Type:    v1.NodeExternalIP,
						Address: "4.3.2.1",
					},
				},
			},
		}
		Expect(GetNodeInternalIP(node)).To(BeEmpty())
	})

})
