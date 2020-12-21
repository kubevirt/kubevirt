package ndp

import (
	"net"

	"github.com/mdlayher/ndp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Neighbour Discovery Protocol", func() {
	Context("prepareRAOptions", func() {
		const (
			ipv6CIDRString      = "2001:1234::/64"
			routerMACAddrString = "02:03:04:05:06:07"
		)

		var ipv6Addr net.IP
		var ipv6CIDR *net.IPNet
		var routerMACAddr net.HardwareAddr
		var routerAdvertisementOptions []ndp.Option

		BeforeEach(func() {
			ipv6Addr, ipv6CIDR, _ = net.ParseCIDR(ipv6CIDRString)
			routerMACAddr, _ = net.ParseMAC(routerMACAddrString)

			prefixLength, _ := ipv6CIDR.Mask.Size()
			routerAdvertisementOptions = prepareRAOptions(ipv6Addr, uint8(prefixLength), routerMACAddr)
		})

		It("should have 2 elements", func() {
			Expect(routerAdvertisementOptions).To(HaveLen(2), "Two router advertisement options are expected")
		})

		It("should feature the `prefixInformation` attribute", func() {
			expectedOption := &ndp.PrefixInformation{
				PrefixLength:                   64,
				OnLink:                         true,
				AutonomousAddressConfiguration: false,
				ValidLifetime:                  ndp.Infinity,
				PreferredLifetime:              ndp.Infinity,
				Prefix:                         ipv6Addr,
			}

			Expect(routerAdvertisementOptions).To(ContainElement(expectedOption))
		})

		It("should feature the `SourceLinkAddress` attribute", func() {
			expectedOption := &ndp.LinkLayerAddress{
				Direction: ndp.Source,
				Addr:      routerMACAddr,
			}

			Expect(routerAdvertisementOptions).To(ContainElement(expectedOption))
		})
	})
})
