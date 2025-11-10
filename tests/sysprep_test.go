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

package tests_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libconfigmap"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	windowsSealedDisk  = "windows-disk"
	diskWindowsSysprep = "disk-windows-sysprep"
)

const (
	// This xml will be the contents of the Autounattend.xml and Unattend.xml files, which are the answer files that Windows uses
	// when booting from a sealed image at different stages, in particulare to answer the questions at the OOBE stage.
	answerFileTemplate = `
    <?xml version="1.0" encoding="utf-8"?>
    <unattend xmlns="urn:schemas-microsoft-com:unattend">
    <settings pass="windowsPE">
    <component name="Microsoft-Windows-International-Core-WinPE" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <SetupUILanguage>
    <UILanguage>en-US</UILanguage>
    </SetupUILanguage>
    <InputLocale>0c09:00000409</InputLocale>
    <SystemLocale>en-US</SystemLocale>
    <UILanguage>en-US</UILanguage>
    <UILanguageFallback>en-US</UILanguageFallback>
    <UserLocale>en-AU</UserLocale>
    </component>
    <component name="Microsoft-Windows-Setup" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <DiskConfiguration>
    <Disk wcm:action="add">
    <CreatePartitions>
    <CreatePartition wcm:action="add">
    <Order>1</Order>
    <Type>Primary</Type>
    <Size>100</Size>
    </CreatePartition>
    <CreatePartition wcm:action="add">
    <Extend>true</Extend>
    <Order>2</Order>
    <Type>Primary</Type>
    </CreatePartition>
    </CreatePartitions>
    <ModifyPartitions>
    <ModifyPartition wcm:action="add">
    <Active>true</Active>
    <Format>NTFS</Format>
    <Label>System Reserved</Label>
    <Order>1</Order>
    <PartitionID>1</PartitionID>
    <TypeID>0x27</TypeID>
    </ModifyPartition>
    <ModifyPartition wcm:action="add">
    <Active>true</Active>
    <Format>NTFS</Format>
    <Label>OS</Label>
    <Letter>C</Letter>
    <Order>2</Order>
    <PartitionID>2</PartitionID>
    </ModifyPartition>
    </ModifyPartitions>
    <DiskID>0</DiskID>
    <WillWipeDisk>true</WillWipeDisk>
    </Disk>
    </DiskConfiguration>
    <ImageInstall>
    <OSImage>
    <InstallTo>
    <DiskID>0</DiskID>
    <PartitionID>2</PartitionID>
    </InstallTo>
    <InstallToAvailablePartition>false</InstallToAvailablePartition>
    </OSImage>
    </ImageInstall>
    <UserData>
    <AcceptEula>true</AcceptEula>
    <FullName>admin</FullName>
    <Organization></Organization>
    </UserData>
    <EnableFirewall>true</EnableFirewall>
    </component>
    </settings>
    <settings pass="offlineServicing">
    <component name="Microsoft-Windows-LUA-Settings" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <EnableLUA>false</EnableLUA>
    </component>
    </settings>
    <settings pass="generalize">
    <component name="Microsoft-Windows-Security-SPP" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <SkipRearm>1</SkipRearm>
    </component>
    </settings>
    <settings pass="specialize">
    <component name="Microsoft-Windows-International-Core" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <InputLocale>0c09:00000409</InputLocale>
    <SystemLocale>en-AU</SystemLocale>
    <UILanguage>en-AU</UILanguage>
    <UILanguageFallback>en-AU</UILanguageFallback>
    <UserLocale>en-AU</UserLocale>
    </component>
    <component name="Microsoft-Windows-Security-SPP-UX" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <SkipAutoActivation>true</SkipAutoActivation>
    </component>
    <component name="Microsoft-Windows-SQMApi" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <CEIPEnabled>0</CEIPEnabled>
    </component>
    <component name="Microsoft-Windows-Shell-Setup" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <ComputerName>ANAME-PC</ComputerName>
    <ProductKey>%s</ProductKey>
    </component>
    </settings>
    <settings pass="oobeSystem">
    <component name="Microsoft-Windows-Shell-Setup" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <AutoLogon>
    <Password>
    <Value>Gauranga</Value>
    <PlainText>true</PlainText>
    </Password>
    <Enabled>false</Enabled>
    <Username>admin</Username>
    </AutoLogon>
    <OOBE>
    <HideEULAPage>true</HideEULAPage>
    <HideOEMRegistrationScreen>true</HideOEMRegistrationScreen>
    <HideOnlineAccountScreens>true</HideOnlineAccountScreens>
    <HideWirelessSetupInOOBE>true</HideWirelessSetupInOOBE>
    <NetworkLocation>Work</NetworkLocation>
    <ProtectYourPC>1</ProtectYourPC>
    <SkipUserOOBE>true</SkipUserOOBE>
    <SkipMachineOOBE>true</SkipMachineOOBE>
    </OOBE>
    <UserAccounts>
    <LocalAccounts>
    <LocalAccount wcm:action="add">
    <Password>
    <Value>Gauranga</Value>
    <PlainText>true</PlainText>
    </Password>
    <Description></Description>
    <DisplayName>admin</DisplayName>
    <Group>Administrators</Group>
    <Name>admin</Name>
    </LocalAccount>
    </LocalAccounts>
    </UserAccounts>
    <RegisteredOrganization></RegisteredOrganization>
    <RegisteredOwner>admin</RegisteredOwner>
    <DisableAutoDaylightTimeSet>false</DisableAutoDaylightTimeSet>
    <TimeZone>AUS Eastern Standard Time</TimeZone>
    <VisualEffects>
    <SystemDefaultBackgroundColor>2</SystemDefaultBackgroundColor>
    </VisualEffects>
    </component>
    <component name="Microsoft-Windows-ehome-reg-inf" processorArchitecture="x86" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="NonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <RestartEnabled>true</RestartEnabled>
    </component>
    <component name="Microsoft-Windows-ehome-reg-inf" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="NonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <RestartEnabled>true</RestartEnabled>
    </component>
    </settings>
    </unattend>
    `
)

func insertProductKeyToAnswerFileTemplate(answerFileTemplate string) string {
	productKeyFilePath := os.Getenv("KUBEVIRT_WINDOWS_PRODUCT_KEY_PATH")
	keyFromFile, err := os.ReadFile(productKeyFilePath)
	Expect(err).ToNot(HaveOccurred())
	productKey := strings.TrimSpace(string(keyFromFile))
	return fmt.Sprintf(answerFileTemplate, productKey)
}

func withFeatures(features v1.Features) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Features = &features
	}
}

func withFirmwareUID(uid types.UID) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		vmi.Spec.Domain.Firmware.UUID = uid
	}
}

const (
	windowsSysprepedVMIUser     = "Admin"
	windowsSysprepedVMIPassword = "Gauranga"
)

var _ = Describe("[Sysprep][sig-compute]Syspreped VirtualMachineInstance", Serial, decorators.Sysprep, decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient

	var windowsVMI *v1.VirtualMachineInstance

	BeforeEach(func() {
		const OSWindowsSysprep = "windows-sysprep"
		virtClient = kubevirt.Client()
		checks.RecycleImageOrFail(virtClient, diskWindowsSysprep)
		libstorage.CreatePVC(OSWindowsSysprep, testsuite.GetTestNamespace(nil), "35Gi", libstorage.Config.StorageClassWindows, true)

		// TODO: Figure out why we have both Autounattend and Unattend
		cm := libconfigmap.New("sysprepautounattend", map[string]string{"Autounattend.xml": insertProductKeyToAnswerFileTemplate(answerFileTemplate), "Unattend.xml": insertProductKeyToAnswerFileTemplate(answerFileTemplate)})
		_, err := virtClient.CoreV1().ConfigMaps(testsuite.GetTestNamespace(nil)).Create(context.Background(), cm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		windowsVMI = libvmi.New(libvmi.WithInterface(e1000DefaultInterface()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithTerminationGracePeriod(0),
			libvmi.WithCPUCount(2, 0, 0),
			libvmi.WithEphemeralPersistentVolumeClaim(windowsSealedDisk, diskWindowsSysprep),
			libvmi.WithCDRomAndVolume(v1.DiskBusSATA, v1.Volume{
				Name: "sysprep",
				VolumeSource: v1.VolumeSource{
					Sysprep: &v1.SysprepSource{
						ConfigMap: &k8sv1.LocalObjectReference{
							Name: cm.Name,
						},
					},
				},
			}),
			libvmi.WithMemoryRequest("2048Mi"),
			withFeatures(v1.Features{
				ACPI: v1.FeatureState{},
				APIC: &v1.FeatureAPIC{},
				Hyperv: &v1.FeatureHyperv{
					Relaxed:   &v1.FeatureState{},
					VAPIC:     &v1.FeatureState{},
					Spinlocks: &v1.FeatureSpinlocks{Retries: pointer.P(uint32(8191))},
				},
			}),
			libvmi.WithClock(v1.Clock{
				ClockOffset: v1.ClockOffset{UTC: &v1.ClockOffsetUTC{}},
				Timer: &v1.Timer{
					HPET:   &v1.HPETTimer{Enabled: pointer.P(false)},
					PIT:    &v1.PITTimer{TickPolicy: v1.PITTickPolicyDelay},
					RTC:    &v1.RTCTimer{TickPolicy: v1.RTCTickPolicyCatchup},
					Hyperv: &v1.HypervTimer{},
				},
			}),
			withFirmwareUID(types.UID(libvmifact.WindowsFirmware)),
		)
	})

	Context("[ref_id:5105]should create the Admin user as specified in the Autounattend.xml", func() {
		var winrmcliPod *k8sv1.Pod

		BeforeEach(func() {
			By("Creating winrm-cli pod for the future use")
			var err error
			winrmcliPod, err = virtClient.CoreV1().Pods(testsuite.NamespaceTestDefault).Create(context.Background(), winRMCliPod(), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Starting the windows VirtualMachineInstance")
			windowsVMI, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), windowsVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			windowsVMI = libwait.WaitForSuccessfulVMIStart(windowsVMI,
				libwait.WithTimeout(720),
			)
		})

		It("[test_id:5843]Should run echo command on machine using the credentials specified in the Autounattend.xml file", func() {
			command := []string{
				winrmCliCmd,
				"-hostname",
				windowsVMI.Status.Interfaces[0].IP,
				"-username",
				windowsSysprepedVMIUser,
				"-password",
				windowsSysprepedVMIPassword,
				"echo works"}
			Eventually(func() (string, error) {
				return exec.ExecuteCommandOnPod(
					winrmcliPod,
					winrmcliPod.Spec.Containers[0].Name,
					command,
				)
			}, time.Minute*10, time.Second*60).Should(ContainSubstring("works"))
		})
	})
})

func winRMCliPod() *k8sv1.Pod {
	user := int64(1001)
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{GenerateName: winrmCli},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				{
					Name:    winrmCli,
					Image:   libregistry.GetUtilityImageFromRegistry(winrmCli),
					Command: []string{"sleep"},
					Args:    []string{"3600"},
					SecurityContext: &k8sv1.SecurityContext{
						AllowPrivilegeEscalation: pointer.P(false),
						Capabilities:             &k8sv1.Capabilities{Drop: []k8sv1.Capability{"ALL"}},
					},
				},
			},
			SecurityContext: &k8sv1.PodSecurityContext{
				RunAsNonRoot:   pointer.P(true),
				RunAsUser:      &user,
				SeccompProfile: &k8sv1.SeccompProfile{Type: k8sv1.SeccompProfileTypeRuntimeDefault},
			},
		},
	}
}
