package util_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"

	"github.com/go-kit/kit/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kubevirtlog "kubevirt.io/kubevirt/pkg/log"
)

const (
	logs = `
2018-10-04 09:20:33.702+0000: 38: info : libvirt version: 4.2.0, package: 1.fc28 (Unknown, 2018-04-04-03:04:18, a0570af3fea64d0ba2df52242c71403f)
2018-10-04 09:20:33.702+0000: 38: info : hostname: vmi-nocloud
2018-10-04 09:20:33.702+0000: 38: error : virDBusGetSystemBus:109 : internal error: Unable to get DBus system bus connection: Failed to connect to socket /run/dbus/system_bus_socket: No such file or directory
2018-10-04 09:20:33.924+0000: 38: error : virDBusGetSystemBus:109 : internal error: Unable to get DBus system bus connection: Failed to connect to socket /run/dbus/system_bus_socket: No such file or directory

2018-10-04 09:20:33.924+0000: 38: warning : networkStateInitialize:763 : DBus not available, disabling firewalld support in bridge_network_driver: internal error: Unable to get DBus system bus connection: Failed to connect to socket /run/dbus/system_bus_socket: No such file or directory
2018-10-04 09:20:33.942+0000: 43: error : virPCIDeviceConfigOpen:312 : Failed to open config space file '/sys/bus/pci/devices/0000:00:00.0/config': Read-only file system
2018-10-04 09:20:33.942+0000: 43: error : virPCIDeviceConfigOpen:312 : Failed to open config space file '/sys/bus/pci/devices/0000:00:01.0/config': Read-only file system
2018-10-04 09:20:33.942+0000: 43: error : virPCIDeviceConfigOpen:312 : Failed to open config space file '/sys/bus/pci/devices/0000:00:01.1/config': Read-only file system
2018-10-04 09:20:33.944+0000: 43: error : virPCIDeviceConfigOpen:312 : Failed to open config space file '/sys/bus/pci/devices/0000:00:01.3/config': Read-only file system
2018-10-04 09:20:33.948+0000: 43: error : virPCIDeviceConfigOpen:312 : Failed to open config space file '/sys/bus/pci/devices/0000:00:02.0/config': Read-only file system
2018-10-04 09:error:33.948+0000: 43: error : virPCIDeviceConfigOpen:312 : Failed to open config space file '/sys/bus/pci/devices/0000:00:02.0/config': Read-only file system
2018-10-04 09:20:33.950+0000: 43: error : virPCIDeviceConfigOpen:312 : Failed to open config space file '/sys/bus/pci/devices/0000:00:03.0/config': Read-only file system
2018-10-04 09:20:33.950+0000: 43: error : virPCIDeviceConfigOpen:312 : Failed to open config space file '/sys/bus/pci/devices/0000:00:04.0/config': Read-only file system
2018-10-04 09:20:33.950+0000: 43: error : virPCIDeviceConfigOpen:312 : Failed to open config space file '/sys/bus/pci/devices/0000:00:05.0/config': Read-only file system
2018-10-04 09:20:34.465+0000: 38: error : virCommandWait:2600 : internal error: Child process (/usr/sbin/dmidecode -q -t 0,1,2,3,4,17) unexpected exit status 1: /dev/mem: No such file or directory
2018-10-04 09:20:34.474+0000: 38: error : virNodeSuspendSupportsTarget:336 : internal error: Cannot probe for supported suspend types
2018-10-04 09:20:34.474+0000: 38: warning : virQEMUCapsInit:1229 : Failed to get host power management capabilities
2018-10-04 09:20:44.174+0000: 26: error : virDBusGetSystemBus:109 : internal error: Unable to get DBus system bus connection: Failed to connect to socket /run/dbus/system_bus_socket: No such file or directory
2018-10-04 09:20:44.177+0000: 26: warning : qemuInterfaceOpenVhostNet:687 : Unable to open vhost-net. Opened so far 0, requested 1
2018-10-04 09:20:44.284+0000: 26: error : virCgroupDetect:714 : At least one cgroup controller is required: No such device or address
2018-10-04 13:39:13.905+0000: 26: error : virCgroupDetect:715 : At least one cgroup controller is required: No such device or address
`

	formattedLogs = `{"component":"test","level":"info","msg":"libvirt version: 4.2.0, package: 1.fc28 (Unknown, 2018-04-04-03:04:18, a0570af3fea64d0ba2df52242c71403f)","subcomponent":"libvirt","thread":"38","timestamp":"2018-10-04T09:20:33.702000Z"}
{"component":"test","level":"info","msg":"hostname: vmi-nocloud","subcomponent":"libvirt","thread":"38","timestamp":"2018-10-04T09:20:33.702000Z"}
{"component":"test","level":"error","msg":"internal error: Unable to get DBus system bus connection: Failed to connect to socket /run/dbus/system_bus_socket: No such file or directory","pos":"virDBusGetSystemBus:109","subcomponent":"libvirt","thread":"38","timestamp":"2018-10-04T09:20:33.702000Z"}
{"component":"test","level":"error","msg":"internal error: Unable to get DBus system bus connection: Failed to connect to socket /run/dbus/system_bus_socket: No such file or directory","pos":"virDBusGetSystemBus:109","subcomponent":"libvirt","thread":"38","timestamp":"2018-10-04T09:20:33.924000Z"}
{"component":"test","level":"warning","msg":"DBus not available, disabling firewalld support in bridge_network_driver: internal error: Unable to get DBus system bus connection: Failed to connect to socket /run/dbus/system_bus_socket: No such file or directory","pos":"networkStateInitialize:763","subcomponent":"libvirt","thread":"38","timestamp":"2018-10-04T09:20:33.924000Z"}
{"component":"test","level":"error","msg":"Failed to open config space file '/sys/bus/pci/devices/0000:00:00.0/config': Read-only file system","pos":"virPCIDeviceConfigOpen:312","subcomponent":"libvirt","thread":"43","timestamp":"2018-10-04T09:20:33.942000Z"}
{"component":"test","level":"error","msg":"Failed to open config space file '/sys/bus/pci/devices/0000:00:01.0/config': Read-only file system","pos":"virPCIDeviceConfigOpen:312","subcomponent":"libvirt","thread":"43","timestamp":"2018-10-04T09:20:33.942000Z"}
{"component":"test","level":"error","msg":"Failed to open config space file '/sys/bus/pci/devices/0000:00:01.1/config': Read-only file system","pos":"virPCIDeviceConfigOpen:312","subcomponent":"libvirt","thread":"43","timestamp":"2018-10-04T09:20:33.942000Z"}
{"component":"test","level":"error","msg":"Failed to open config space file '/sys/bus/pci/devices/0000:00:01.3/config': Read-only file system","pos":"virPCIDeviceConfigOpen:312","subcomponent":"libvirt","thread":"43","timestamp":"2018-10-04T09:20:33.944000Z"}
{"component":"test","level":"error","msg":"Failed to open config space file '/sys/bus/pci/devices/0000:00:02.0/config': Read-only file system","pos":"virPCIDeviceConfigOpen:312","subcomponent":"libvirt","thread":"43","timestamp":"2018-10-04T09:20:33.948000Z"}
{"component":"test","level":"error","msg":"Failed to open config space file '/sys/bus/pci/devices/0000:00:03.0/config': Read-only file system","pos":"virPCIDeviceConfigOpen:312","subcomponent":"libvirt","thread":"43","timestamp":"2018-10-04T09:20:33.950000Z"}
{"component":"test","level":"error","msg":"Failed to open config space file '/sys/bus/pci/devices/0000:00:04.0/config': Read-only file system","pos":"virPCIDeviceConfigOpen:312","subcomponent":"libvirt","thread":"43","timestamp":"2018-10-04T09:20:33.950000Z"}
{"component":"test","level":"error","msg":"Failed to open config space file '/sys/bus/pci/devices/0000:00:05.0/config': Read-only file system","pos":"virPCIDeviceConfigOpen:312","subcomponent":"libvirt","thread":"43","timestamp":"2018-10-04T09:20:33.950000Z"}
{"component":"test","level":"error","msg":"internal error: Child process (/usr/sbin/dmidecode -q -t 0,1,2,3,4,17) unexpected exit status 1: /dev/mem: No such file or directory","pos":"virCommandWait:2600","subcomponent":"libvirt","thread":"38","timestamp":"2018-10-04T09:20:34.465000Z"}
{"component":"test","level":"error","msg":"internal error: Cannot probe for supported suspend types","pos":"virNodeSuspendSupportsTarget:336","subcomponent":"libvirt","thread":"38","timestamp":"2018-10-04T09:20:34.474000Z"}
{"component":"test","level":"warning","msg":"Failed to get host power management capabilities","pos":"virQEMUCapsInit:1229","subcomponent":"libvirt","thread":"38","timestamp":"2018-10-04T09:20:34.474000Z"}
{"component":"test","level":"error","msg":"internal error: Unable to get DBus system bus connection: Failed to connect to socket /run/dbus/system_bus_socket: No such file or directory","pos":"virDBusGetSystemBus:109","subcomponent":"libvirt","thread":"26","timestamp":"2018-10-04T09:20:44.174000Z"}
{"component":"test","level":"warning","msg":"Unable to open vhost-net. Opened so far 0, requested 1","pos":"qemuInterfaceOpenVhostNet:687","subcomponent":"libvirt","thread":"26","timestamp":"2018-10-04T09:20:44.177000Z"}
{"component":"test","level":"error","msg":"At least one cgroup controller is required: No such device or address","pos":"virCgroupDetect:714","subcomponent":"libvirt","thread":"26","timestamp":"2018-10-04T09:20:44.284000Z"}
{"component":"test","level":"error","msg":"At least one cgroup controller is required: No such device or address","pos":"virCgroupDetect:715","subcomponent":"libvirt","thread":"26","timestamp":"2018-10-04T13:39:13.905000Z"}`
)

var _ = Describe("LibvirtHelper", func() {

	It("should parse libvirt logs", func() {
		buffer := bytes.NewBuffer(nil)

		kubevirtlog.InitializeLogging("test")
		logger := log.NewContext(log.NewJSONLogger(buffer))
		klog := kubevirtlog.MakeLogger(logger)

		scanner := bufio.NewScanner(strings.NewReader(logs))
		for scanner.Scan() {
			kubevirtlog.LogLibvirtLogLine(klog, scanner.Text())
		}

		scanner = bufio.NewScanner(buffer)

		loggedLines := []map[string]string{}

		for scanner.Scan() {
			entry := map[string]string{}
			err := json.Unmarshal(scanner.Bytes(), &entry)
			Expect(err).To(Not(HaveOccurred()))
			//delete(entry, "timestamp")
			loggedLines = append(loggedLines, entry)
		}
		Expect(scanner.Err()).To(Not(HaveOccurred()))

		expectedLines := []map[string]string{}
		scanner = bufio.NewScanner(strings.NewReader(formattedLogs))
		for scanner.Scan() {
			entry := map[string]string{}
			err := json.Unmarshal(scanner.Bytes(), &entry)
			Expect(err).To(Not(HaveOccurred()))
			//delete(entry, "timestamp")
			expectedLines = append(expectedLines, entry)
		}
		Expect(scanner.Err()).To(Not(HaveOccurred()))

		Expect(loggedLines).To(Equal(expectedLines))
	})
})
