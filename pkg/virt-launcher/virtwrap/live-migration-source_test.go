package virtwrap

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("live-migration-source", func() {
	var (
		qemuErrorLog = `
2023-11-07 12:49:18.610+0000: starting up libvirt version: 9.0.0, package: 10.3.el9_2 (Red Hat, Inc. <http://bugzilla.redhat.com/bugzilla>, 2023-08-24-06:08:50, ), qemu version: 7.2.0qemu-kvm-7.2.0-14.el9_2.5, kernel: 5.14.0-284[12/1890]
_2.x86_64, hostname: foo-37362155-f4mvp                                                                                                                                                                                                      
LC_ALL=C \                                                                                                                                                                                                                                   
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \                                                                                                                                                                          
HOME=/ \                                                                                                                                                                                                                                     
XDG_CACHE_HOME=/var/run/kubevirt-private/libvirt/qemu/lib/domain-1-clusters-foo_foo-373/.cache \                                                                                                                                             
/usr/libexec/qemu-kvm \                                                                                                                                                                                                                      
-name guest=clusters-foo_foo-37362155-f4mvp,debug-threads=on \                                                                                                                                                                               
-S \                                                                                                                                                                                                                                         
-object '{"qom-type":"secret","id":"masterKey0","format":"raw","file":"/var/run/kubevirt-private/libvirt/qemu/lib/domain-1-clusters-foo_foo-373/master-key.aes"}' \                                                                          
-machine pc-q35-rhel9.2.0,usb=off,dump-guest-core=off,memory-backend=pc.ram \                                                                                                                                                                
-accel kvm \                                                                                                                                                                                                                                 
-cpu Cascadelake-Server,ss=on,vmx=on,hypervisor=on,tsc-adjust=on,umip=on,pku=on,md-clear=on,stibp=on,arch-capabilities=on,ibpb=on,ibrs=on,amd-stibp=on,amd-ssbd=on,rdctl-no=on,ibrs-all=on,skip-l1dfl-vmentry=on,mds-no=on,pschange-mc-no=on$tsx-ctrl=on,hle=off,rtm=off,mpx=off \                                                                                                           
-m 6144 \                                                          
-object '{"qom-type":"memory-backend-ram","id":"pc.ram","size":6442450944}' \                                                                                                                                                                
-overcommit mem-lock=off \                                                                                            
-smp 4,sockets=1,dies=1,cores=4,threads=1 \                                                                                                                                                                                                  
-object '{"qom-type":"iothread","id":"iothread1"}' \                                                                  
-uuid 1fd80ebc-5e53-5afe-aa6a-4666ac29dd8a \                                                                                                                                                                                                 
-smbios 'type=1,manufacturer=Red Hat,product=OpenShift Virtualization,version=4.14.0,uuid=1fd80ebc-5e53-5afe-aa6a-4666ac29dd8a,sku=4.14.0,family=Red Hat' \                                                                                 
-no-user-config \                                                                                                                                                                                                                            
-nodefaults \                                                                                                         
-chardev socket,id=charmonitor,fd=18,server=on,wait=off \                                                                                                               
-mon chardev=charmonitor,id=monitor,mode=control \                                                                                                                                              
-rtc base=utc \                                                                                                                                                                                                                              
-no-shutdown \                                                                                                        
-boot strict=on \                                                                                                                                                                                                                            
-device '{"driver":"pcie-root-port","port":16,"chassis":1,"id":"pci.1","bus":"pcie.0","multifunction":true,"addr":"0x2"}' \                                                                                                                 
-device '{"driver":"pcie-root-port","port":17,"chassis":2,"id":"pci.2","bus":"pcie.0","addr":"0x2.0x1"}' \                                                              
-device '{"driver":"pcie-root-port","port":18,"chassis":3,"id":"pci.3","bus":"pcie.0","addr":"0x2.0x2"}' \                                                                                      
-device '{"driver":"pcie-root-port","port":19,"chassis":4,"id":"pci.4","bus":"pcie.0","addr":"0x2.0x3"}' \                                                                                                                                   
-device '{"driver":"pcie-root-port","port":20,"chassis":5,"id":"pci.5","bus":"pcie.0","addr":"0x2.0x4"}' \                                        
-device '{"driver":"pcie-root-port","port":21,"chassis":6,"id":"pci.6","bus":"pcie.0","addr":"0x2.0x5"}' \                                                                                
-device '{"driver":"pcie-root-port","port":22,"chassis":7,"id":"pci.7","bus":"pcie.0","addr":"0x2.0x6"}' \
-device '{"driver":"pcie-root-port","port":23,"chassis":8,"id":"pci.8","bus":"pcie.0","addr":"0x2.0x7"}' \
-device '{"driver":"pcie-root-port","port":24,"chassis":9,"id":"pci.9","bus":"pcie.0","multifunction":true,"addr":"0x3"}' \
-device '{"driver":"pcie-root-port","port":25,"chassis":10,"id":"pci.10","bus":"pcie.0","addr":"0x3.0x1"}' \
-device '{"driver":"virtio-scsi-pci-non-transitional","id":"scsi0","bus":"pci.5","addr":"0x0"}' \
-device '{"driver":"virtio-serial-pci-non-transitional","id":"virtio-serial0","bus":"pci.6","addr":"0x0"}' \
-blockdev '{"driver":"host_device","filename":"/dev/rhcos","aio":"native","node-name":"libvirt-2-storage","cache":{"direct":true,"no-flush":false},"auto-read-only":true,"discard":"unmap"}' \
-blockdev '{"node-name":"libvirt-2-format","read-only":false,"discard":"unmap","cache":{"direct":true,"no-flush":false},"driver":"raw","file":"libvirt-2-storage"}' \
-device '{"driver":"virtio-blk-pci-non-transitional","bus":"pci.7","addr":"0x0","drive":"libvirt-2-format","id":"ua-rhcos","bootindex":1,"write-cache":"on","werror":"stop","rerror":"stop"}' \
-blockdev '{"driver":"file","filename":"/var/run/kubevirt-ephemeral-disks/cloud-init-data/clusters-foo/foo-37362155-f4mvp/configdrive.iso","node-name":"libvirt-1-storage","cache":{"direct":true,"no-flush":false},"auto-read-only":true,"d$
scard":"unmap"}' \
-blockdev '{"node-name":"libvirt-1-format","read-only":false,"discard":"unmap","cache":{"direct":true,"no-flush":false},"driver":"raw","file":"libvirt-1-storage"}' \
-device '{"driver":"virtio-blk-pci-non-transitional","bus":"pci.8","addr":"0x0","drive":"libvirt-1-format","id":"ua-cloudinitvolume","write-cache":"on","werror":"stop","rerror":"stop"}' \
-netdev '{"type":"tap","fd":"19","vhost":true,"vhostfd":"21","id":"hostua-default"}' \
-device '{"driver":"virtio-net-pci-non-transitional","host_mtu":1400,"netdev":"hostua-default","id":"ua-default","mac":"0a:58:0a:86:00:5b","bus":"pci.1","addr":"0x0","romfile":""}' \
-chardev socket,id=charserial0,fd=16,server=on,wait=off \
-device '{"driver":"isa-serial","chardev":"charserial0","id":"serial0","index":0}' \
-chardev socket,id=charchannel0,fd=17,server=on,wait=off \
-device '{"driver":"virtserialport","bus":"virtio-serial0.0","nr":1,"chardev":"charchannel0","id":"channel0","name":"org.qemu.guest_agent.0"}' \
-audiodev '{"id":"audio1","driver":"none"}' \
-vnc vnc=unix:/var/run/kubevirt-private/7a734204-e3a3-4a09-a9cc-f510cc1967a1/virt-vnc,audiodev=audio1 \
-device '{"driver":"VGA","id":"video0","vgamem_mb":16,"bus":"pcie.0","addr":"0x1"}' \
-device '{"driver":"virtio-balloon-pci-non-transitional","id":"balloon0","free-page-reporting":false,"bus":"pci.9","addr":"0x0"}' \
-sandbox on,obsolete=deny,elevateprivileges=deny,spawn=deny,resourcecontrol=deny \
-msg timestamp=on
2023-11-07 12:49:59.831+0000: Domain id=1 is tainted: custom-ga-command
2023-11-07 13:07:39.275+0000: initiating migration
2023-11-07T13:07:48.561834Z qemu-kvm: qemu_savevm_state_complete_precopy_non_iterable: bdrv_inactivate_all() failed (-1)                                                                                                                    
2023-11-07T13:07:48.751290Z qemu-kvm: Unable to read from socket: Bad file descriptor
2023-11-07T13:07:48.751339Z qemu-kvm: Unable to read from socket: Bad file descriptor
2023-11-07T13:07:48.751346Z qemu-kvm: Unable to read from socket: Bad file descriptor                      
`
		expectedQemuFilteredLogs = `2023-11-07 13:07:39.275+0000: initiating migration
2023-11-07T13:07:48.561834Z qemu-kvm: qemu_savevm_state_complete_precopy_non_iterable: bdrv_inactivate_all() failed (-1)                                                                                                                    
2023-11-07T13:07:48.751290Z qemu-kvm: Unable to read from socket: Bad file descriptor
2023-11-07T13:07:48.751339Z qemu-kvm: Unable to read from socket: Bad file descriptor
2023-11-07T13:07:48.751346Z qemu-kvm: Unable to read from socket: Bad file descriptor                      
`
	)
	Context("when filtering qemu logs after live migration initiation", func() {
		FIt("should show the logs after 'initiating migration' string", func() {
			obtainedQemuFilteredLogs, err := dumpQemuLogsAfterMigration(strings.NewReader(qemuErrorLog))
			Expect(err).ToNot(HaveOccurred())
			Expect(obtainedQemuFilteredLogs).To(Equal(expectedQemuFilteredLogs))
		})
	})
})
