package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-kit/log"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"
	kubevirtlog "kubevirt.io/client-go/log"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
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

	qemuLogs = `2020-07-02 09:04:39.037+0000: starting up libvirt version: 6.0.0, package: 16.fc31 (Unknown, 2020-04-07-15:55:55, ), qemu version: 4.2.0qemu-kvm-4.2.0-15.fc31, kernel: 3.10.0-1062.9.1.el7.x86_64, hostname: vmi-alpine-efi
LC_ALL=C \
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
HOME=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e \
XDG_DATA_HOME=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e/.local/share \
XDG_CACHE_HOME=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e/.cache \
XDG_CONFIG_HOME=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e/.config \
QEMU_AUDIO_DRV=none \
/usr/libexec/qemu-kvm \
-name guest=default_vmi-alpine-efi,debug-threads=on \
-S \
-object secret,id=masterKey0,format=raw,file=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e/master-key.aes \
-blockdev '{"driver":"file","filename":"/usr/share/OVMF/OVMF_CODE.fd","node-name":"libvirt-pflash0-storage","auto-read-only":true,"discard":"unmap"}' \
-blockdev '{"node-name":"libvirt-pflash0-format","read-only":true,"driver":"raw","file":"libvirt-pflash0-storage"}' \
-blockdev '{"driver":"file","filename":"/tmp/default_vmi-alpine-efi","node-name":"libvirt-pflash1-storage","auto-read-only":true,"discard":"unmap"}' \
-blockdev '{"node-name":"libvirt-pflash1-format","read-only":false,"driver":"raw","file":"libvirt-pflash1-storage"}' \
-machine pc-q35-rhel8.2.0,accel=kvm,usb=off,dump-guest-core=off,pflash0=libvirt-pflash0-format,pflash1=libvirt-pflash1-format \
-cpu Skylake-Client,ss=on,hypervisor=on,tsc-adjust=on,clflushopt=on,umip=on,arch-capabilities=on,pdpe1gb=on,skip-l1dfl-vmentry=on \
-m 1024 \
-overcommit mem-lock=off \
-smp 1,sockets=1,dies=1,cores=1,threads=1 \
-object iothread,id=iothread1 \
-uuid 5c6fa8f7-c3f6-4b3f-b596-d16ea3912302 \
-smbios type=1,manufacturer=KubeVirt,product=None,uuid=5c6fa8f7-c3f6-4b3f-b596-d16ea3912302,family=KubeVirt \
-no-user-config \
-nodefaults \
-chardev socket,id=charmonitor,fd=20,server,nowait \
-mon chardev=charmonitor,id=monitor,mode=control \
-rtc base=utc \
-no-shutdown \
-boot strict=on \
-device pcie-root-port,port=0x10,chassis=1,id=pci.1,bus=pcie.0,multifunction=on,addr=0x2 \
-device pcie-root-port,port=0x11,chassis=2,id=pci.2,bus=pcie.0,addr=0x2.0x1 \
-device pcie-root-port,port=0x12,chassis=3,id=pci.3,bus=pcie.0,addr=0x2.0x2 \
-device pcie-root-port,port=0x13,chassis=4,id=pci.4,bus=pcie.0,addr=0x2.0x3 \
-device virtio-serial-pci,id=virtio-serial0,bus=pci.2,addr=0x0 \
-blockdev '{"driver":"file","filename":"/var/run/kubevirt/container-disks/disk_0.img","node-name":"libvirt-2-storage","cache":{"direct":true,"no-flush":false},"auto-read-only":true,"discard":"unmap"}' \
-blockdev '{"node-name":"libvirt-2-format","read-only":true,"cache":{"direct":true,"no-flush":false},"driver":"raw","file":"libvirt-2-storage"}' \
-blockdev '{"driver":"file","filename":"/var/run/kubevirt-ephemeral-disks/disk-data/containerdisk/disk.qcow2","node-name":"libvirt-1-storage","cache":{"direct":true,"no-flush":false},"auto-read-only":true,"discard":"unmap"}' \
-blockdev '{"node-name":"libvirt-1-format","read-only":false,"cache":{"direct":true,"no-flush":false},"driver":"qcow2","file":"libvirt-1-storage","backing":"libvirt-2-format"}' \
-device virtio-blk-pci,scsi=off,bus=pci.3,addr=0x0,drive=libvirt-1-format,id=ua-containerdisk,bootindex=1,write-cache=on \
-netdev tap,fd=22,id=hostua-default,vhost=on,vhostfd=23 \
-device virtio-net-pci,host_mtu=1450,netdev=hostua-default,id=ua-default,mac=22:f8:ef:32:60:95,bus=pci.1,addr=0x0 \
-chardev socket,id=charserial0,fd=24,server,nowait \
-device isa-serial,chardev=charserial0,id=serial0 \
-chardev socket,id=charchannel0,fd=25,server,nowait \
-device virtserialport,bus=virtio-serial0.0,nr=1,chardev=charchannel0,id=channel0,name=org.qemu.guest_agent.0 \
-vnc vnc=unix:/var/run/kubevirt-private/6d220540-cae6-4aa3-a850-af62ff66e407/virt-vnc \
-device VGA,id=video0,vgamem_mb=16,bus=pcie.0,addr=0x1 \
-sandbox on,obsolete=deny,elevateprivileges=deny,spawn=deny,resourcecontrol=deny \
-msg timestamp=on`

	qemuFormattedLogs = `{"component":"virt-launcher","level":"info","msg":"2020-07-02 09:04:39.037+0000: starting up libvirt version: 6.0.0, package: 16.fc31 (Unknown, 2020-04-07-15:55:55, ), qemu version: 4.2.0qemu-kvm-4.2.0-15.fc31, kernel: 3.10.0-1062.9.1.el7.x86_64, hostname: vmi-alpine-efi","subcomponent":"qemu","timestamp":"2020-07-02T09:04:39.303235Z"}
{"component":"virt-launcher","level":"info","msg":"LC_ALL=C \\PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \\HOME=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e \\XDG_DATA_HOME=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e/.local/share \\XDG_CACHE_HOME=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e/.cache \\XDG_CONFIG_HOME=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e/.config \\QEMU_AUDIO_DRV=none \\/usr/libexec/qemu-kvm \\-name guest=default_vmi-alpine-efi,debug-threads=on \\-S \\-object secret,id=masterKey0,format=raw,file=/var/lib/libvirt/qemu/domain-1-default_vmi-alpine-e/master-key.aes \\-blockdev '{\"driver\":\"file\",\"filename\":\"/usr/share/OVMF/OVMF_CODE.fd\",\"node-name\":\"libvirt-pflash0-storage\",\"auto-read-only\":true,\"discard\":\"unmap\"}' \\-blockdev '{\"node-name\":\"libvirt-pflash0-format\",\"read-only\":true,\"driver\":\"raw\",\"file\":\"libvirt-pflash0-storage\"}' \\-blockdev '{\"driver\":\"file\",\"filename\":\"/tmp/default_vmi-alpine-efi\",\"node-name\":\"libvirt-pflash1-storage\",\"auto-read-only\":true,\"discard\":\"unmap\"}' \\-blockdev '{\"node-name\":\"libvirt-pflash1-format\",\"read-only\":false,\"driver\":\"raw\",\"file\":\"libvirt-pflash1-storage\"}' \\-machine pc-q35-rhel8.2.0,accel=kvm,usb=off,dump-guest-core=off,pflash0=libvirt-pflash0-format,pflash1=libvirt-pflash1-format \\-cpu Skylake-Client,ss=on,hypervisor=on,tsc-adjust=on,clflushopt=on,umip=on,arch-capabilities=on,pdpe1gb=on,skip-l1dfl-vmentry=on \\-m 1024 \\-overcommit mem-lock=off \\-smp 1,sockets=1,dies=1,cores=1,threads=1 \\-object iothread,id=iothread1 \\-uuid 5c6fa8f7-c3f6-4b3f-b596-d16ea3912302 \\-smbios type=1,manufacturer=KubeVirt,product=None,uuid=5c6fa8f7-c3f6-4b3f-b596-d16ea3912302,family=KubeVirt \\-no-user-config \\-nodefaults \\-chardev socket,id=charmonitor,fd=20,server,nowait \\-mon chardev=charmonitor,id=monitor,mode=control \\-rtc base=utc \\-no-shutdown \\-boot strict=on \\-device pcie-root-port,port=0x10,chassis=1,id=pci.1,bus=pcie.0,multifunction=on,addr=0x2 \\-device pcie-root-port,port=0x11,chassis=2,id=pci.2,bus=pcie.0,addr=0x2.0x1 \\-device pcie-root-port,port=0x12,chassis=3,id=pci.3,bus=pcie.0,addr=0x2.0x2 \\-device pcie-root-port,port=0x13,chassis=4,id=pci.4,bus=pcie.0,addr=0x2.0x3 \\-device virtio-serial-pci,id=virtio-serial0,bus=pci.2,addr=0x0 \\-blockdev '{\"driver\":\"file\",\"filename\":\"/var/run/kubevirt/container-disks/disk_0.img\",\"node-name\":\"libvirt-2-storage\",\"cache\":{\"direct\":true,\"no-flush\":false},\"auto-read-only\":true,\"discard\":\"unmap\"}' \\-blockdev '{\"node-name\":\"libvirt-2-format\",\"read-only\":true,\"cache\":{\"direct\":true,\"no-flush\":false},\"driver\":\"raw\",\"file\":\"libvirt-2-storage\"}' \\-blockdev '{\"driver\":\"file\",\"filename\":\"/var/run/kubevirt-ephemeral-disks/disk-data/containerdisk/disk.qcow2\",\"node-name\":\"libvirt-1-storage\",\"cache\":{\"direct\":true,\"no-flush\":false},\"auto-read-only\":true,\"discard\":\"unmap\"}' \\-blockdev '{\"node-name\":\"libvirt-1-format\",\"read-only\":false,\"cache\":{\"direct\":true,\"no-flush\":false},\"driver\":\"qcow2\",\"file\":\"libvirt-1-storage\",\"backing\":\"libvirt-2-format\"}' \\-device virtio-blk-pci,scsi=off,bus=pci.3,addr=0x0,drive=libvirt-1-format,id=ua-containerdisk,bootindex=1,write-cache=on \\-netdev tap,fd=22,id=hostua-default,vhost=on,vhostfd=23 \\-device virtio-net-pci,host_mtu=1450,netdev=hostua-default,id=ua-default,mac=22:f8:ef:32:60:95,bus=pci.1,addr=0x0 \\-chardev socket,id=charserial0,fd=24,server,nowait \\-device isa-serial,chardev=charserial0,id=serial0 \\-chardev socket,id=charchannel0,fd=25,server,nowait \\-device virtserialport,bus=virtio-serial0.0,nr=1,chardev=charchannel0,id=channel0,name=org.qemu.guest_agent.0 \\-vnc vnc=unix:/var/run/kubevirt-private/6d220540-cae6-4aa3-a850-af62ff66e407/virt-vnc \\-device VGA,id=video0,vgamem_mb=16,bus=pcie.0,addr=0x1 \\-sandbox on,obsolete=deny,elevateprivileges=deny,spawn=deny,resourcecontrol=deny \\-msg timestamp=on","subcomponent":"qemu","timestamp":"2020-07-02T09:04:39.303348Z"}`
)

var _ = Describe("LibvirtHelper", func() {

	It("should parse libvirt logs", func() {
		buffer := bytes.NewBuffer(nil)

		kubevirtlog.InitializeLogging("test")
		logger := log.NewJSONLogger(buffer)
		klog := kubevirtlog.MakeLogger(logger)

		scanner := bufio.NewScanner(strings.NewReader(logs))
		for scanner.Scan() {
			kubevirtlog.LogLibvirtLogLine(klog, scanner.Text())
		}

		scanner = bufio.NewScanner(buffer)

		loggedLines := []map[string]string{}

		for scanner.Scan() {
			entry := map[string]string{}
			Expect(json.Unmarshal(scanner.Bytes(), &entry)).To(Succeed())
			//delete(entry, "timestamp")
			loggedLines = append(loggedLines, entry)
		}
		Expect(scanner.Err()).ToNot(HaveOccurred())

		expectedLines := []map[string]string{}
		scanner = bufio.NewScanner(strings.NewReader(formattedLogs))
		for scanner.Scan() {
			entry := map[string]string{}
			Expect(json.Unmarshal(scanner.Bytes(), &entry)).To(Succeed())
			//delete(entry, "timestamp")
			expectedLines = append(expectedLines, entry)
		}
		Expect(scanner.Err()).ToNot(HaveOccurred())

		Expect(loggedLines).To(Equal(expectedLines))
	})

	It("should parse qemu logs", func() {
		buffer := bytes.NewBuffer(nil)

		kubevirtlog.InitializeLogging("virt-launcher")
		logger := log.NewJSONLogger(buffer)
		klog := kubevirtlog.MakeLogger(logger)

		scanner := bufio.NewScanner(strings.NewReader(qemuLogs))
		for scanner.Scan() {
			kubevirtlog.LogQemuLogLine(klog, scanner.Text())
		}

		scanner = bufio.NewScanner(buffer)

		loggedLines := []map[string]string{}

		for scanner.Scan() {
			entry := map[string]string{}
			Expect(json.Unmarshal(scanner.Bytes(), &entry)).To(Succeed())
			delete(entry, "timestamp")
			loggedLines = append(loggedLines, entry)
		}
		Expect(scanner.Err()).ToNot(HaveOccurred())

		expectedLines := []map[string]string{}
		scanner = bufio.NewScanner(strings.NewReader(qemuFormattedLogs))
		for scanner.Scan() {
			entry := map[string]string{}
			Expect(json.Unmarshal(scanner.Bytes(), &entry)).To(Succeed())
			delete(entry, "timestamp")
			expectedLines = append(expectedLines, entry)
		}
		Expect(scanner.Err()).ToNot(HaveOccurred())

		Expect(loggedLines).To(Equal(expectedLines))
	})

	It("should update the wantedSpec to reflect changes made by hooks", func() {
		vmiNamespace := "test-namespace"
		vmiName := "test-vmi"
		ctrl := gomock.NewController(GinkgoT())
		mockConn := cli.NewMockConnection(ctrl)
		mockDomain := cli.NewMockVirDomain(ctrl)

		vmi := api2.NewMinimalVMIWithNS(vmiNamespace, vmiName)
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		domain := &api.Domain{}
		c := &converter.ConverterContext{
			Architecture:     arch.NewConverter(runtime.GOARCH),
			VirtualMachine:   vmi,
			AllowEmulation:   true,
			SMBios:           &cmdv1.SMBios{},
			HotplugVolumes:   make(map[string]v1.VolumeStatus),
			PermanentVolumes: make(map[string]v1.VolumeStatus),
		}
		Expect(converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)).To(Succeed())
		api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

		wantedSpec := &domain.Spec
		wantedSpec.Devices.Disks = []api.Disk{
			{
				Device: "disk",
				Type:   "file",
				Source: api.DiskSource{
					File: "/var/run/kubevirt-private/vmi-disks/permvolume1/disk.img",
				},
				Target: api.DiskTarget{
					Bus:    v1.DiskBusVirtio,
					Device: "vda",
				},
				Driver: &api.DiskDriver{
					Cache:       "none",
					Name:        "qemu",
					Type:        "raw",
					ErrorPolicy: "stop",
				},
				Alias: api.NewUserDefinedAlias("permvolume1"),
			},
		}

		mutatedSpec := wantedSpec.DeepCopy()
		mutatedSpec.Devices.Disks[0].Source.File = "/var/run/kubevirt-private/vmi-disks/permvolume1/new-disk.img"
		mutatedSpecXml, err := xml.Marshal(mutatedSpec)
		Expect(err).NotTo(HaveOccurred())

		// mock hook manager
		mockHookManager := hooks.NewMockManager(ctrl)
		getHookManager = func() hooks.Manager {
			return mockHookManager
		}
		defer func() {
			getHookManager = hooks.GetManager
		}()
		mockHookManager.EXPECT().OnDefineDomain(wantedSpec, vmi).Return(string(mutatedSpecXml), nil)
		mockConn.EXPECT().DomainDefineXML(string(mutatedSpecXml)).Return(mockDomain, nil)

		_, err = SetDomainSpecStrWithHooks(mockConn, vmi, wantedSpec)
		Expect(err).NotTo(HaveOccurred())

		Expect(wantedSpec.Devices.Disks).To(Equal(mutatedSpec.Devices.Disks))
	})

	Context("getLibvirtLogFilters()", func() {

		DescribeTable("should return customLogFilters if defined and not empty with", func(libvirtLogVerbosityEnvVar *string, libvirtDebugLogsEnvVarDefined bool) {
			customLogFilters := pointer.P("3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access")

			logFilters, enableDebugLogs := getLibvirtLogFilters(customLogFilters, libvirtLogVerbosityEnvVar, libvirtDebugLogsEnvVarDefined)
			Expect(enableDebugLogs).To(BeTrue())
			Expect(logFilters).To(Equal(*customLogFilters))
		},
			Entry("libvirtLogVerbosityEnvVar not defined, libvirtDebugLogsEnvVarDefined false", nil, false),
			Entry("libvirtLogVerbosityEnvVar defined, libvirtDebugLogsEnvVarDefined false", pointer.P("2"), false),
			Entry("libvirtLogVerbosityEnvVar not defined, libvirtDebugLogsEnvVarDefined true", nil, true),
			Entry("libvirtLogVerbosityEnvVar defined, libvirtDebugLogsEnvVarDefined true", pointer.P("1"), true),
		)

		Context("with customLogFilters not defined", func() {

			const verbosityThreshold = services.EXT_LOG_VERBOSITY_THRESHOLD

			DescribeTable("logs should be enabled if debugLogs env var is defined when", func(libvirtLogVerbosityEnvVar *string) {
				_, enableDebugLogs := getLibvirtLogFilters(nil, libvirtLogVerbosityEnvVar, true)
				Expect(enableDebugLogs).To(BeTrue())
			},
				Entry("libvirtLogVerbosityEnvVar defined to 8", pointer.P("8")),
				Entry("libvirtLogVerbosityEnvVar defined to 3", pointer.P("3")),
				Entry("libvirtLogVerbosityEnvVar is not defined", nil),
			)

			DescribeTable("with debugLogs not defined logs should", func(libvirtLogVerbosity *int, expectedEnableDebugLogs bool) {

				var libvirtLogVerbosityEnvVar *string
				if libvirtLogVerbosity != nil {
					libvirtLogVerbosityEnvVar = pointer.P(fmt.Sprintf("%d", *libvirtLogVerbosity))
				}

				_, enableDebugLogs := getLibvirtLogFilters(nil, libvirtLogVerbosityEnvVar, false)
				Expect(enableDebugLogs).To(Equal(expectedEnableDebugLogs))
			},
				Entry("be disabled when libvirt log verbosity is below threshold", pointer.P(verbosityThreshold-1), false),
				Entry("be disabled when libvirt log verbosity is equal to threshold", pointer.P(verbosityThreshold), true),
				Entry("be enabled when libvirt log verbosity is above threshold", pointer.P(verbosityThreshold+1), true),
				Entry("be disabled when libvirt log verbosity is not defined", nil, false),
			)

		})

	})

	Context("configureQemuConf()", func() {
		DescribeTable("should set shared_filesystem on qemu.conf according to env", func(envInput string, expected string) {
			confPath := filepath.Join(GinkgoT().TempDir(), "qemu.conf")
			file, err := os.Create(confPath)
			Expect(err).ToNot(HaveOccurred())
			// Random content to ensure we append properly
			file.WriteString("dummy = 1\n")
			Expect(file.Close()).To(Succeed())
			Expect(os.Setenv(services.ENV_VAR_SHARED_FILESYSTEM_PATHS, envInput)).To(Succeed())
			DeferCleanup(func() {
				Expect(os.Unsetenv(services.ENV_VAR_SHARED_FILESYSTEM_PATHS)).To(Succeed())
				Expect(os.RemoveAll(confPath)).To(Succeed())
			})

			Expect(configureQemuConf(confPath)).To(Succeed())

			file, err = os.OpenFile(confPath, os.O_RDONLY, 0644)
			Expect(err).ToNot(HaveOccurred())
			fileInfo, err := file.Stat()
			Expect(err).ToNot(HaveOccurred())
			buf := make([]byte, fileInfo.Size())
			_, err = file.Read(buf)
			Expect(err).ToNot(HaveOccurred())

			lines := strings.Split(string(buf), "\n")
			Expect(lines).To(ContainElement(expected))
		},
			Entry("single shared filesystem", "/foo/bar", "shared_filesystems = [ \"/foo/bar\" ]"),
			Entry("multiple shared filesystems", "/foo/bar1:/foo/bar2", "shared_filesystems = [ \"/foo/bar1\", \"/foo/bar2\" ]"),
		)

	})
})
