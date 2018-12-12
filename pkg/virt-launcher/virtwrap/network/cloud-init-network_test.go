// Inspired by pkg/virt-launcher/virtwrap/network/podinterface_test.go

package network

import (
	"bytes"
	"io/ioutil"
	"net"
	"os"
	"strconv"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gopkg.in/yaml.v2"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("cloud-init network", func() {
	var mockNetwork *MockNetworkHandler
	var ctrl *gomock.Controller

	var tmpDir string

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		tmpDir, _ := ioutil.TempDir("", "cloudinittest")
		setInterfaceCacheFile(tmpDir + "/cache-%s.json")

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = NewMockNetworkHandler(ctrl)
		Handler = mockNetwork
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Context("on successful GenNetworkFile()", func() {
		It("should create valid network-config contents", func() {
			count := 3
			domain := newSriovDomainWithInterface()
			vm := newSriovVMISriovInterface("testnamespace", "testVmName", count)
			api.SetObjectDefaults_Domain(domain)
			buildMockSriovNetwork(mockNetwork, count)
			cloudinit, err := GenNetworkFile(vm)
			parsedCloudInit := bytes.Split(cloudinit, []byte(cloudInitDelimiter))
			var config CloudInitNetConfig
			yaml.Unmarshal(parsedCloudInit[0], &config)

			Expect(err).To(BeNil())
			intNum := 1
			for intNum <= count {
				intArray := intNum - 1
				intString := strconv.Itoa(intNum)
				intName := "net" + intString
				intConfig := &config.Config[intArray]

				Expect(intConfig.NetworkType).To(Equal("physical"))
				Expect(intConfig.Name).To(Equal(intName))
				Expect(intConfig.MacAddress).To(Equal("de:ad:be:af:00:0" + intString))
				Expect(intConfig.Mtu).To(Equal(uint16(1400 + intNum)))
				Expect(len(intConfig.Subnets)).To(Equal(1))

				intSubnet := &intConfig.Subnets[0]
				Expect(intSubnet.SubnetType).To(Equal("static"))
				Expect(intSubnet.Address).To(Equal("10." + intString + ".0.2/24"))
				if intNum == 1 {
					Expect(intSubnet.Gateway).To(Equal("10." + intString + ".0.1"))
					Expect(intSubnet.Routes).To(BeNil())
				} else {
					Expect(intSubnet.Gateway).To(Equal(""))
					Expect(len(intSubnet.Routes)).To(Equal(intNum + 1))
					routeNum := 0
					for routeNum <= intNum {
						Expect(intSubnet.Routes[routeNum].Network).To(Equal("10." + intString + "." + strconv.Itoa(routeNum+1) + ".0"))
						Expect(intSubnet.Routes[routeNum].Netmask).To(Equal("255.255.255.0"))
						if routeNum == intNum {
							Expect(intSubnet.Routes[routeNum].Gateway).To(Equal(""))
						} else {
							Expect(intSubnet.Routes[routeNum].Gateway).To(Equal("10." + intString + ".0.1"))
						}
						routeNum++
					}
				}

				Expect(intConfig.Address).To(BeNil())
				Expect(intConfig.Search).To(BeNil())
				Expect(intConfig.Destination).To(Equal(""))
				Expect(intConfig.Gateway).To(Equal(""))
				Expect(intConfig.Metric).To(Equal(0))

				intNum++
			}

		})
	})
})

func buildMockSriovNetwork(mockNetwork *MockNetworkHandler, count int) {
	netInts := make(map[string]*netlink.Dummy, count)
	intNum := 1
	for intNum <= count {

		var addrList []netlink.Addr
		var routeList []netlink.Route
		var ipAddress netlink.Addr
		var macAddress net.HardwareAddr

		intName := "net" + strconv.Itoa(intNum)
		netInts[intName] = &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Index: count,
				MTU:   1400 + intNum,
				Name:  intName,
				Alias: intName,
			},
		}

		address := &net.IPNet{IP: net.IPv4(10, byte(intNum), 0, 2), Mask: net.CIDRMask(24, 32)}
		ipAddress = netlink.Addr{IPNet: address}
		addrList = []netlink.Addr{ipAddress}
		macAddrStr := "de:ad:be:af:00:0" + strconv.Itoa(intNum)
		macAddress, _ = net.ParseMAC(macAddrStr)

		routeList = append(routeList, netlink.Route{Src: net.IPv4(10, byte(intNum), 0, 2)})

		if intNum == 1 {
			gw := net.IPv4(10, byte(intNum), 0, 1)
			routeList = append(routeList, netlink.Route{Gw: gw})
		} else if intNum > 1 {
			routeCount := 1
			for routeCount <= intNum {
				staticRoute := netlink.Route{
					Dst: &net.IPNet{IP: net.IPv4(10, byte(intNum), byte(routeCount), 0), Mask: net.CIDRMask(24, 32)},
					Gw:  net.IPv4(10, byte(intNum), 0, 1),
				}
				routeList = append(routeList, staticRoute)
				routeCount++
			}
			gwRoute := netlink.Route{
				Dst: &net.IPNet{IP: net.IPv4(10, byte(intNum), byte(routeCount), 0), Mask: net.CIDRMask(24, 32)},
			}

			routeList = append(routeList, gwRoute)
		}

		netlink.AddrAdd(netInts[intName], &ipAddress)

		mockNetwork.EXPECT().LinkByName(intName).Return(netInts[intName], nil)
		mockNetwork.EXPECT().AddrList(netInts[intName], netlink.FAMILY_V4).Return(addrList, nil)
		mockNetwork.EXPECT().RouteList(netInts[intName], netlink.FAMILY_V4).Return(routeList, nil)
		mockNetwork.EXPECT().GetMacDetails(intName).Return(macAddress, nil)

		intNum++
	}
}

func newSriovVMI(namespace, name string, count int) *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineInstanceSpec{
			Domain:   v1.NewMinimalDomainSpec(),
			Networks: []v1.Network{*v1.DefaultPodNetwork()},
		},
	}
	intNum := 1
	for intNum <= count {
		intName := "sriov" + strconv.Itoa(intNum)
		sriovNetwork := v1.Network{
			Name: intName,
			NetworkSource: v1.NetworkSource{
				Multus: &v1.CniNetwork{NetworkName: intName},
			},
		}
		vmi.Spec.Networks = append(vmi.Spec.Networks, sriovNetwork)
		intNum++
	}
	return vmi
}

func newSriovVMISriovInterface(namespace string, name string, count int) *v1.VirtualMachineInstance {
	vmi := newSriovVMI(namespace, name, count)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
	intNum := 1
	for intNum <= count {
		sriovInterface := v1.Interface{
			Name: "sriov" + strconv.Itoa(intNum),
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				SRIOV: &v1.InterfaceSRIOV{},
			},
		}
		vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, sriovInterface)
		intNum++
	}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func newSriovDomainWithInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{
			Type: "virtio",
		},
		Type: "bridge",
		Source: api.InterfaceSource{
			Bridge: api.DefaultBridgeName,
		},
		Alias: &api.Alias{
			Name: "default",
		}},
	}
	return domain
}
