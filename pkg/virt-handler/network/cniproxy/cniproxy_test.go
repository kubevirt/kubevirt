package cniproxy

import (
	"fmt"
	"testing"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pod CNI", func() {
	Context("Mocked utils", func() {
		var mockedPidFunc = func(string) (int, error) {
			return 12345, nil
		}

		var failPidFunc = func(string) (int, error) {
			return 0, fmt.Errorf("unit test simulated failure")
		}

		It("Should return namespaces", func() {
			res, err := _getLibvirtNS(mockedPidFunc)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Pid).To(Equal("/proc/12345/ns/pid"))
			Expect(res.Cgroup).To(Equal("/proc/12345/ns/cgroup"))
		})

		It("Should raise errors", func() {
			res, err := _getLibvirtNS(failPidFunc)
			Expect(err).To(HaveOccurred())
			Expect(res).To(BeNil())
		})

		It("Should build a valid runtime config", func() {
			res, err := _buildRuntimeConfig(mockedPidFunc, "fake-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.NetNS).To(Equal("/proc/12345/ns/net"))
			Expect(res.IfName).To(Equal("fake-1"))
			Expect(res.ContainerID).To(Equal("1"))
		})

		It("Shouldn't panic on invalid name", func() {
			res, err := _buildRuntimeConfig(mockedPidFunc, "bad_interface")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid interface name: bad_interface"))
			Expect(res).To(BeNil())
		})
	})

	Context("Mocked CNI", func() {
		var mockConfFileFunc = func(dir string, extensions []string) ([]string, error) {
			return []string{"/tmp/test"}, nil
		}

		var failConfFileFunc = func(dir string, extensions []string) ([]string, error) {
			return nil, fmt.Errorf("fake conf file error")
		}

		var mockConfFromFileFunc = func(filename string) (*libcni.NetworkConfig, error) {
			return &libcni.NetworkConfig{}, nil
		}

		var failConfFromFileFunc = func(filename string) (*libcni.NetworkConfig, error) {
			return nil, fmt.Errorf("fake config load error")
		}

		It("Should succeed without errors", func() {
			res, err := _getCNINetworkConfig(mockConfFileFunc, mockConfFromFileFunc)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Network).To(BeNil())
			Expect(res.Bytes).To(BeNil())
		})

		It("Should fail looking up files", func() {
			res, err := _getCNINetworkConfig(failConfFileFunc, mockConfFromFileFunc)
			Expect(res).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fake conf file error"))
		})

		It("Should fail loading config", func() {
			res, err := _getCNINetworkConfig(mockConfFileFunc, failConfFromFileFunc)
			Expect(res).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("No valid networks found in /etc/cni/net.d"))
		})
	})

	Context("Mocked Proxy", func() {
		var mockedGetConf = func() (*libcni.NetworkConfig, error) {
			return &libcni.NetworkConfig{
				Network: &types.NetConf{Type: "test"}}, nil
		}
		runtime := libcni.RuntimeConf{}

		It("Should create a valid proxy", func() {
			res, err := _getProxy(mockedGetConf, &runtime)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.cniConfig.Path).To(Equal([]string{CNIPluginsDir}))
		})

		It("Should fail creation on invalid plugin", func() {
			proxy, err := _getProxy(mockedGetConf, &runtime)
			Expect(err).ToNot(HaveOccurred())
			res, err := proxy.AddToNetwork()
			Expect(err).To(HaveOccurred())
			Expect(res).To(BeNil())
			Expect(err.Error()).To(Equal("failed to find plugin \"test\" in path [/opt/cni/bin]"))
		})

		It("Should fail deletion on invalid plugin", func() {
			proxy, err := _getProxy(mockedGetConf, &runtime)
			Expect(err).ToNot(HaveOccurred())
			err = proxy.DeleteFromNetwork()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to find plugin \"test\" in path [/opt/cni/bin]"))
		})
	})
})

func TestCNI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pod CNI")
}
