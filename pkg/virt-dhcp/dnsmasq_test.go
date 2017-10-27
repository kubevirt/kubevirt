package virtdhcp

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var testdnsmasq *DNSmasqInstance
var cmdArgs []string
var cmdArgsPrefix string

type MockMonitor struct {
	MonitorRuntime
}

func (mon MockMonitor) Start(cmd string, args []string) {
	cmdArgs = args
}

func FakeMonitor() Monitor {
	mon := MockMonitor{}
	mon.isRunning = false
	mon.stopChan = make(chan bool)
	mon.pid = 0
	return &mon
}

type MockOSHandler struct {
	OSHandler
}

func (o *MockOSHandler) writeDHCPHostsFile(file *os.File, dhcpHosts []string) error {
	errRet := fmt.Errorf("failed to match dhcphosts")
	if len(testdnsmasq.hosts) != len(dhcpHosts) {
		return errRet
	}
	for _, host := range dhcpHosts {
		data := strings.Split(host, ",")
		if val, ok := testdnsmasq.hosts[data[1]]; ok {
			lease, _ := strconv.Atoi(data[2])
			if val.Mac != data[0] || val.Lease != lease {
				return errRet
			}
		}
	}
	return nil
}

func (o *MockOSHandler) killProcessIfExist(pid int) error {
	return nil
}

var _ = Describe("DHCP", func() {

	var tempDir string

	BeforeEach(func() {
		tempDir, err := ioutil.TempDir("", "dhcptest")
		Expect(err).NotTo(HaveOccurred())
		DHCPLeaseFile = fmt.Sprintf("%s/dhcp.leases", tempDir)
		DHCPHostsFile = fmt.Sprintf("%s/dhcp.hosts", tempDir)
		cmdArgsPrefix = fmt.Sprintf("-k -d --strict-order --bind-dynamic --no-resolv --dhcp-no-override --dhcp-authoritative "+
			"--except-interface=lo --dhcp-hostsfile=%s --dhcp-leasefile=%s --pid-file=/tmp/dhcp.pid",
			DHCPHostsFile, DHCPLeaseFile)
		testdnsmasq = NewDNSmasq()
		OS = &MockOSHandler{}
		testdnsmasq.monitor = FakeMonitor()
	})

	Context("handle requests to add/remove a hosts", func() {
		It("should restart dnsmasq with new configuration", func() {
			err := testdnsmasq.AddHost("12:d4:4f:99:7d:3f", "10.32.0.15", 86400)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(testdnsmasq.hosts)).To(Equal(1))
			key, val := testdnsmasq.hosts["10.32.0.15"]
			Expect(val).To(Equal(true))
			Expect(key.Lease).To(Equal(86400))
			Expect(key.Mac).To(Equal("12:d4:4f:99:7d:3f"))
		})
		It("should start dnsmasq with 2 hosts", func() {
			err := testdnsmasq.AddHost("12:d4:4f:99:7d:3f", "10.32.0.15", 86400)
			Expect(err).NotTo(HaveOccurred())
			err = testdnsmasq.AddHost("af:21:ac:bd:c3:ba", "10.32.0.12", 86400)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(testdnsmasq.hosts)).To(Equal(2))
			key, val := testdnsmasq.hosts["10.32.0.15"]
			Expect(val).To(Equal(true))
			Expect(key.Lease).To(Equal(86400))
			Expect(key.Mac).To(Equal("12:d4:4f:99:7d:3f"))
			key, val = testdnsmasq.hosts["10.32.0.12"]
			Expect(val).To(Equal(true))
			Expect(key.Lease).To(Equal(86400))
			Expect(key.Mac).To(Equal("af:21:ac:bd:c3:ba"))
			testArgs := cmdArgsPrefix + " --dhcp-range=10.32.0.12,10.32.0.15"
			expectedArgs := strings.Join(cmdArgs, " ")
			Expect(expectedArgs).To(Equal(testArgs))
		})
		It("should start dnsmasq with 1 host after remove", func() {
			err := testdnsmasq.AddHost("12:d4:4f:99:7d:3f", "10.32.0.15", 86400)
			Expect(err).NotTo(HaveOccurred())
			err = testdnsmasq.AddHost("af:21:ac:bd:c3:ba", "10.32.0.12", 86400)
			Expect(err).NotTo(HaveOccurred())
			err = testdnsmasq.RemoveHost("10.32.0.15")
			Expect(err).NotTo(HaveOccurred())

			Expect(len(testdnsmasq.hosts)).To(Equal(1))
			key, val := testdnsmasq.hosts["10.32.0.15"]
			Expect(val).To(Equal(false))
			key, val = testdnsmasq.hosts["10.32.0.12"]
			Expect(val).To(Equal(true))
			Expect(key.Lease).To(Equal(86400))
			Expect(key.Mac).To(Equal("af:21:ac:bd:c3:ba"))
			testArgs := cmdArgsPrefix + " --dhcp-range=10.32.0.12,10.32.0.12"
			expectedArgs := strings.Join(cmdArgs, " ")
			Expect(expectedArgs).To(Equal(testArgs))
		})
		It("should not start dnsmasq with all hosts removed", func() {
			err := testdnsmasq.AddHost("12:d4:4f:99:7d:3f", "10.32.0.15", 86400)
			Expect(err).NotTo(HaveOccurred())
			err = testdnsmasq.AddHost("af:21:ac:bd:c3:ba", "10.32.0.12", 86400)
			Expect(err).NotTo(HaveOccurred())
			err = testdnsmasq.RemoveHost("10.32.0.15")
			Expect(err).NotTo(HaveOccurred())

			cmdArgs = []string{}
			err = testdnsmasq.RemoveHost("10.32.0.12")
			Expect(err).NotTo(HaveOccurred())

			Expect(len(testdnsmasq.hosts)).To(Equal(0))
			Expect(cmdArgs).To(Equal([]string{}))
		})
		It("should set known hosts", func() {
			hostsList := []AddArgs{{Mac: "12:d4:4f:99:7d:3f", IP: "10.32.0.15", Lease: 86400},
				{Mac: "af:21:ac:bd:c3:ba", IP: "10.32.0.12", Lease: 86400}}
			err := testdnsmasq.SetKnownHosts(&hostsList)

			Expect(err).NotTo(HaveOccurred())
			Expect(len(testdnsmasq.hosts)).To(Equal(2))
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})
	})
})
