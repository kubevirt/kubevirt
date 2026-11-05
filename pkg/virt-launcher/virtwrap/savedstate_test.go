/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package virtwrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hibernation"
	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Saved state", func() {
	const uuid = "5b0c5a2e-1f53-4a8a-9d41-4d1b3a8f0e11"

	domainXML := `<domain type="kvm">
  <name>default_testvm</name>
  <uuid>` + uuid + `</uuid>
  <memory unit="KiB">2097152</memory>
  <os><type arch="x86_64" machine="pc-q35-rhel9.6.0">hvm</type></os>
  <cpu mode="host-model"><topology sockets="1" cores="1" threads="1"></topology></cpu>
  <devices><disk type="file" device="disk"><source file="/var/run/kubevirt-private/vmi-disks/root/disk.img"></source><target dev="vda" bus="virtio"></target></disk></devices>
</domain>`

	savedXML := `<domain type="kvm">
  <name>default_testvm</name>
  <uuid>` + uuid + `</uuid>
  <memory unit="KiB">2097152</memory>
  <os><type arch="x86_64" machine="pc-q35-rhel9.4.0">hvm</type></os>
  <cpu mode="custom" match="exact"><model fallback="forbid">EPYC-Milan</model><feature policy="require" name="x2apic"></feature><topology sockets="1" cores="1" threads="1"></topology></cpu>
  <devices><disk type="file" device="disk"><source file="/old/path/disk.img"></source><target dev="vda" bus="virtio"></target></disk></devices>
</domain>`

	It("should restore with the definition of the pod and the CPU and machine of the saved state", func() {
		restoreXML, err := restoreDomainXML(domainXML, savedXML)
		Expect(err).ToNot(HaveOccurred())

		restored := &libvirtxml.Domain{}
		Expect(restored.Unmarshal(restoreXML)).To(Succeed())
		Expect(restored.CPU.Mode).To(Equal("custom"))
		Expect(restored.CPU.Model.Value).To(Equal("EPYC-Milan"))
		Expect(restored.CPU.Features).To(HaveLen(1))
		Expect(restored.OS.Type.Machine).To(Equal("pc-q35-rhel9.4.0"))
		Expect(restored.Devices.Disks[0].Source.File.File).To(Equal("/var/run/kubevirt-private/vmi-disks/root/disk.img"))
	})

	It("should restore with the vCPU enable order of the saved state", func() {
		withVCPUs := func(xml, vcpus string) string {
			return strings.Replace(xml, "<os>", vcpus+"<os>", 1)
		}
		pod := withVCPUs(domainXML, `<vcpus><vcpu id="0" enabled="yes" hotpluggable="no"></vcpu><vcpu id="1" enabled="no" hotpluggable="yes"></vcpu></vcpus>`)
		booted := withVCPUs(savedXML, `<vcpus><vcpu id="0" enabled="yes" hotpluggable="no" order="1"></vcpu><vcpu id="1" enabled="no" hotpluggable="yes"></vcpu></vcpus>`)

		restoreXML, err := restoreDomainXML(pod, booted)
		Expect(err).ToNot(HaveOccurred())
		restored := &libvirtxml.Domain{}
		Expect(restored.Unmarshal(restoreXML)).To(Succeed())
		Expect(restored.VCPUs.VCPU).To(HaveLen(2))
		Expect(restored.VCPUs.VCPU[0].Order).To(HaveValue(BeEquivalentTo(1)))
	})

	It("should restore with the MAC addresses of the saved state", func() {
		withNIC := func(xml, mac string) string {
			return strings.Replace(xml, "</devices>", `<interface type="ethernet"><mac address="`+mac+`"></mac><alias name="ua-default"></alias></interface></devices>`, 1)
		}
		restoreXML, err := restoreDomainXML(withNIC(domainXML, "0a:4c:42:9e:f3:b6"), withNIC(savedXML, "02:b1:4a:84:b1:41"))
		Expect(err).ToNot(HaveOccurred())
		restored := &libvirtxml.Domain{}
		Expect(restored.Unmarshal(restoreXML)).To(Succeed())
		Expect(restored.Devices.Interfaces).To(HaveLen(1))
		Expect(restored.Devices.Interfaces[0].MAC.Address).To(Equal("02:b1:4a:84:b1:41"))
	})

	It("should refuse a saved state of another domain", func() {
		other := `<domain type="kvm"><name>default_testvm</name><uuid>00000000-0000-0000-0000-000000000000</uuid></domain>`
		_, err := restoreDomainXML(domainXML, other)
		Expect(err).To(MatchError(ContainSubstring("belongs to domain")))
	})

	DescribeTable("should pick the number of parallel channels", func(annotations map[string]string, expected int) {
		vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}}
		Expect(savedStateChannels(vmi)).To(Equal(expected))
	},
		Entry("by default", nil, defaultSavedStateChannels),
		Entry("from the annotation", map[string]string{v1.SavedStateParallelChannelsAnnotation: "8"}, 8),
		Entry("ignoring a non number", map[string]string{v1.SavedStateParallelChannelsAnnotation: "many"}, defaultSavedStateChannels),
		Entry("ignoring zero", map[string]string{v1.SavedStateParallelChannelsAnnotation: "0"}, defaultSavedStateChannels),
		Entry("ignoring too many", map[string]string{v1.SavedStateParallelChannelsAnnotation: "1000"}, defaultSavedStateChannels),
	)

	It("should resume from the hot plugged saved state source", func() {
		vmi := &v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{
			SavedStateSource: &v1.SavedStateSource{ClaimName: "state", FileName: "domain-0123456789ab.save"},
		}}
		Expect(savedStateSourcePath(vmi)).To(Equal(filepath.Join(hotplugdisk.GetVolumeMountDir(hibernation.SourceVolumeName), "domain-0123456789ab.save")))
	})

	It("should neither retry a failed restore nor boot instead", func() {
		vmi := &v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{
			SavedStateSource: &v1.SavedStateSource{ClaimName: "state", FileName: "domain-0123456789ab.save"},
		}}
		manager := &LibvirtDomainManager{metadataCache: metadata.NewCache()}
		manager.metadataCache.SavedState.Set(api.SavedStateMetadata{
			Operation: api.SavedStateOperationRestore, FileName: "domain-0123456789ab.save", Completed: true, Failed: true, FailureReason: "corrupt",
		})
		// A retry would use the domain, which is nil here.
		Expect(manager.startFromSavedState(vmi, nil)).To(MatchError(ContainSubstring("corrupt")))
	})

	It("should drop the page cache of a file being written until stopped", func() {
		path := filepath.Join(GinkgoT().TempDir(), "domain.save")
		stop := dropPageCacheWhileWriting(path, time.Millisecond)
		Expect(os.WriteFile(path, make([]byte, 1<<20), 0o600)).To(Succeed())
		time.Sleep(10 * time.Millisecond)
		stop()
		content, err := os.ReadFile(path)
		Expect(err).ToNot(HaveOccurred())
		Expect(content).To(HaveLen(1<<20), "dropping the cache never changes the file")
	})

	It("should report a failed save rather than retry it", func() {
		vmi := &v1.VirtualMachineInstance{}
		manager := &LibvirtDomainManager{metadataCache: metadata.NewCache()}
		failed := api.SavedStateMetadata{
			Operation: api.SavedStateOperationSave, FileName: "domain-0123456789ab.save", Completed: true, Failed: true, FailureReason: "aborted",
		}
		manager.metadataCache.SavedState.Set(failed)
		// A retry would look the domain up, which needs a connection.
		Expect(manager.saveVMI(vmi, "/var/run/kubevirt/hotplug-disks/saved-state-target/domain-0123456789ab.save")).To(Succeed())
		current, _ := manager.metadataCache.SavedState.Load()
		Expect(current).To(Equal(failed))
		Expect(manager.isSavingState()).To(BeFalse())
	})

	It("should recognize running out of space", func() {
		Expect(isNoSpace(fmt.Errorf("operation failed: %w", unix.ENOSPC))).To(BeTrue())
		Expect(isNoSpace(fmt.Errorf("virError(Code=38, Message='unable to write: No space left on device')"))).To(BeTrue())
		Expect(isNoSpace(fmt.Errorf("operation aborted"))).To(BeFalse())
		Expect(freeSpace(GinkgoT().TempDir())).ToNot(Equal("unknown"))
	})
})
