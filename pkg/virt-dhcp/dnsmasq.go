package virtdhcp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/template"

	"kubevirt.io/kubevirt/pkg/log"
)

const (
	DNSmasqExec = "/usr/sbin/dnsmasq"
	DHCPPidFile = "/tmp/dhcp.pid"
)

var DHCPLeaseFile = "/var/run/kubevirt/dhcp.leases"
var DHCPHostsFile = "/var/run/kubevirt/dhcp.hosts"

type hostsDetails struct {
	Mac   string
	Lease int
}

type DNSmasqInstance struct {
	hosts    map[string]hostsDetails
	ipRange  [][]byte
	confpath string
	args     []string
	cmd      *exec.Cmd
	monitor  Monitor
}

func NewDNSmasq() *DNSmasqInstance {
	dnsMasq := new(DNSmasqInstance)
	dnsMasq.hosts = make(map[string]hostsDetails)
	dnsMasq.ipRange = make([][]byte, 2)
	dnsMasq.args = []string{"-k", "-d", "--strict-order", "--bind-dynamic", "--no-resolv", "--dhcp-no-override", "--dhcp-authoritative",
		"--except-interface=lo", "--dhcp-hostsfile=" + DHCPHostsFile, "--dhcp-leasefile=" + DHCPLeaseFile,
		"--pid-file=" + DHCPPidFile}
	dnsMasq.monitor = NewMonitor()
	OS = &OSHandler{}
	return dnsMasq
}

// implement `Interface` in sort package.
type sortByteArrays [][]byte

func (b sortByteArrays) Len() int {
	return len(b)
}

func (b sortByteArrays) Less(i, j int) bool {
	// bytes package already implements Comparable for []byte.
	switch bytes.Compare(b[i], b[j]) {
	case -1:
		return true
	case 0, 1:
		return false
	default:
		return false
	}
}

func (b sortByteArrays) Swap(i, j int) {
	b[j], b[i] = b[i], b[j]
}

type OSOps interface {
	IsFileExist(path string) (bool, error)
	getPidFromFile(file string) (int, error)
	checkProcessExist(pid int) (bool, *os.Process)
	killProcessIfExist(pid int) error
	RecreateHostsFile() (*os.File, error)
	readFromFile(file string) ([]byte, error)
}

type OSHandler struct{}

var OS OSOps

func (o *OSHandler) IsFileExist(path string) (bool, error) {
	_, err := os.Stat(path)
	exists := false

	if err == nil {
		exists = true
	} else if os.IsNotExist(err) {
		err = nil
	}
	return exists, err
}

func (o *OSHandler) readFromFile(file string) ([]byte, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to open file: %s", file)
		return nil, err
	}
	return content, nil
}

func (o *OSHandler) getPidFromFile(file string) (int, error) {
	content, err := OS.readFromFile(file)
	if err != nil {
		return -1, err
	}
	lines := strings.Split(string(content), "\n")
	pid, _ := strconv.Atoi(lines[0])
	return pid, nil
}

func (o *OSHandler) killProcessIfExist(pid int) error {
	procExist, proc := OS.checkProcessExist(pid)
	if procExist {
		killErr := proc.Kill()
		if killErr != nil {
			log.Log.Warningf("failed to kill process %d", proc.Pid)
		}
	}
	return nil
}

func (o *OSHandler) checkProcessExist(pid int) (bool, *os.Process) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		log.Log.Warningf("failed to find process %d", pid)
		return false, nil
	}
	procErr := proc.Signal(syscall.Signal(0))
	return procErr == nil, proc
}

func (o *OSHandler) RecreateHostsFile() (*os.File, error) {
	// Remove the config file
	ferr := os.RemoveAll(DHCPHostsFile)
	if ferr != nil {
		log.Log.Reason(ferr).Warningf("failed to remove dhcp hosts file: %s", DHCPHostsFile)
	}

	f, err := os.Create(DHCPHostsFile)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create dnsmasq hosts file in %s", DHCPHostsFile)
		return nil, err
	}
	return f, nil
}

func (dnsmasq *DNSmasqInstance) formatDhcpRange() string {
	return fmt.Sprintf("--dhcp-range=%s,%s", net.IP(dnsmasq.ipRange[0]).String(), net.IP(dnsmasq.ipRange[1]).String())
}

func (dnsmasq *DNSmasqInstance) Start() error {
	log.Log.Info("starting Dnsmasq")
	dhcpRange := dnsmasq.formatDhcpRange()
	startArgs := append(dnsmasq.args, dhcpRange)
	dnsmasq.monitor.Start(DNSmasqExec, startArgs)
	return nil
}

func (dnsmasq *DNSmasqInstance) stop() error {
	log.Log.Info("Stopping Dnsmasq")
	if dnsmasq.monitor.IsRunning() {
		dnsmasq.monitor.Stop()
	} else {
		pid, err1 := OS.getPidFromFile(DHCPPidFile)
		if err1 == nil {
			OS.killProcessIfExist(pid)
		}
	}
	return nil
}

func (dnsmasq *DNSmasqInstance) Restart() error {
	log.Log.Debug("restarting Dnsmasq")

	err := dnsmasq.stop()
	if err != nil {
		log.Log.Reason(err).Error("failed to stop Dnsmasq")
		return err
	}

	dnsmasq.updateRange()
	err = dnsmasq.handleDHCPHostsFile()
	if err != nil {
		return nil
	}
	if len(dnsmasq.hosts) != 0 {
		return dnsmasq.Start()
	}
	return nil
}

func (dnsmasq *DNSmasqInstance) AddHost(mac string, ipaddr string, lease int) error {
	dnsmasq.hosts[ipaddr] = hostsDetails{Mac: mac, Lease: lease}
	log.Log.Debugf("added host %s, ip %s, to hosts", mac, ipaddr)
	return dnsmasq.Restart()
}

func (dnsmasq *DNSmasqInstance) RemoveHost(ipaddr string) error {
	delete(dnsmasq.hosts, ipaddr)
	return dnsmasq.Restart()
}

func (dnsmasq *DNSmasqInstance) SetKnownHosts(hostsList *[]AddArgs) error {
	//Empty the known hosts store
	dnsmasq.hosts = make(map[string]hostsDetails)

	//Set the provided known hosts
	for _, host := range *hostsList {
		dnsmasq.hosts[host.IP] = hostsDetails{Mac: host.Mac, Lease: host.Lease}
	}
	return dnsmasq.Restart()
}

func (dnsmasq *DNSmasqInstance) writeDHCPHostsFile(file *os.File, dhcpHosts []string) error {
	if len(dhcpHosts) != 0 {
		t := template.Must(template.New("dhcpshosts").Parse(`{{range $k:=.}}{{printf "%s\n" $k }}{{end}}
`))

		err := t.Execute(file, dhcpHosts)
		if err != nil {
			log.Log.Reason(err).Error("failed to write dhcp hosts file")
			return err
		}
	}
	return nil
}

func (dnsmasq *DNSmasqInstance) handleDHCPHostsFile() error {
	f, err := OS.RecreateHostsFile()
	if err != nil {
		return err
	}
	defer f.Close()

	dhcpHosts := make([]string, 0, len(dnsmasq.hosts))
	for key, val := range dnsmasq.hosts {
		dhcpHosts = append(dhcpHosts, fmt.Sprintf("%s,%s,%d", val.Mac, key, val.Lease))
	}
	err = dnsmasq.writeDHCPHostsFile(f, dhcpHosts)
	if err != nil {
		return err
	}
	return nil
}

func (dnsmasq *DNSmasqInstance) updateRange() {
	sorted := sortByteArrays(getMapValues(dnsmasq.hosts))
	sort.Sort(sorted)
	if len(sorted) > 0 {
		dnsmasq.ipRange[0] = sorted[0]
		dnsmasq.ipRange[1] = sorted[len(sorted)-1]
		log.Log.Debugf("updated range %s - %s", net.IP(dnsmasq.ipRange[0]).String(), net.IP(dnsmasq.ipRange[1]).String())
	}
}

func (dnsmasq *DNSmasqInstance) loadHostsFile() error {
	content, err := OS.readFromFile(DHCPHostsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		data := strings.Split(line, ",")
		lease, _ := strconv.Atoi(data[2])
		dnsmasq.hosts[data[1]] = hostsDetails{Mac: data[0], Lease: lease}
	}
	dnsmasq.Restart()
	return nil
}

func getMapValues(dict map[string]hostsDetails) [][]byte {
	vals := make([][]byte, 0, len(dict))
	for key, _ := range dict {
		vals = append(vals, net.ParseIP(key))
	}
	return vals
}
