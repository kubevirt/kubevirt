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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	// This xml will be the contents of the Autounattend.xml and Unattend.xml files, which are the answer files that Windows uses
	// when booting from a sealed image at diffrent stages, in particulare to answer the questions at the OOBE stage.
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
	keyFromFile, err := ioutil.ReadFile(productKeyFilePath)
	Expect(err).ToNot(HaveOccurred())
	productKey := strings.TrimSpace(string(keyFromFile))
	return fmt.Sprintf(answerFileTemplate, productKey)
}

var _ = Describe("[Serial][Sysprep][sig-compute]Syspreped VirtualMachineInstance", func() {
	const windowsSysprepedVMIUser = "Admin"
	const windowsSysprepedVMIPassword = "Gauranga"
	const OSWindowsSysprep = "windows-sysprep"
	const windowsSealedDisk = "windows-disk"

	Context("[ref_id:5105]should create the Admin user as specified in the Autounattend.xml", func() {
		var err error
		var virtClient kubecli.KubevirtClient
		var windowsVMI *v1.VirtualMachineInstance
		var winrmcliPod *k8sv1.Pod
		var cli []string
		var output string
		var vmiIp string

		BeforeEach(func() {
			virtClient, err = kubecli.GetKubevirtClient()
			Expect(err).ToNot(HaveOccurred())

			checks.SkipIfMissingRequiredImage(virtClient, tests.DiskWindowsSysprep)
			libstorage.CreatePVC(OSWindowsSysprep, "35Gi", libstorage.Config.StorageClassWindows, true)
			answerFileWithKey := insertProductKeyToAnswerFileTemplate(answerFileTemplate)
			tests.CreateConfigMap("sysprepautounattend", map[string]string{"Autounattend.xml": answerFileWithKey, "Unattend.xml": answerFileWithKey})
			masqueradeInterface := libvmi.InterfaceDeviceWithMasqueradeBinding()
			masqueradeInterface.Model = "e1000"
			windowsVMI = libvmi.New(
				withCoreNumber(uint32(2)),
				withACPIFeature(),
				withAPICFeature(),
				withRelaxedHypervFeature(),
				withRelaxedVAPICHypervFeature(),
				withRelaxedSpinlocksHypervFeature(uint32(8191)),
				withClockOffsetUTC(),
				withPITTimer(v1.PITTickPolicyDelay),
				withRTCTimer(v1.RTCTickPolicyCatchup),
				withHypervTimer(),
				withFirmwareUUID(windowsFirmware),
				libvmi.WithResourceMemory("2048Mi"),
				withEphemeralVolumeSource(windowsSealedDisk, tests.DiskWindowsSysprep),
				withDiskDevice(windowsSealedDisk, v1.DiskBusSATA),
				withSysprepVolumesource("sysprep", "sysprepautounattend"),
				withCDRomDiskDevice("sysprep", v1.DiskBusSATA),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(masqueradeInterface),
			)

			By("Creating winrm-cli pod for the future use")
			winrmcliPod = winRMCliPod()
			winrmcliPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), winrmcliPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Starting the windows VirtualMachineInstance")
			windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 720)

			windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(windowsVMI.Name, &metav1.GetOptions{})
			vmiIp = windowsVMI.Status.Interfaces[0].IP
			cli = []string{
				winrmCliCmd,
				"-hostname",
				vmiIp,
				"-username",
				windowsSysprepedVMIUser,
				"-password",
				windowsSysprepedVMIPassword,
			}
		})

		It("[test_id:5843]Should run echo command on machine using the credentials specified in the Autounattend.xml file", func() {
			command := append(cli, "echo works")
			Eventually(func() error {
				fmt.Printf("Running \"%s\" command via winrm-cli\n", command)
				output, err = tests.ExecuteCommandOnPod(
					virtClient,
					winrmcliPod,
					winrmcliPod.Spec.Containers[0].Name,
					command,
				)
				fmt.Printf("Result \"%v\" command via winrm-cli\n", err)
				return err
			}, time.Minute*10, time.Second*60).ShouldNot(HaveOccurred())
			By("Checking that the Windows VirtualMachineInstance has expected UUID")
			Expect(output).Should(ContainSubstring("works"))
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
					Image:   fmt.Sprintf("%s/%s:%s", flags.KubeVirtUtilityRepoPrefix, winrmCli, flags.KubeVirtUtilityVersionTag),
					Command: []string{"sleep"},
					Args:    []string{"3600"},
					SecurityContext: &k8sv1.SecurityContext{
						AllowPrivilegeEscalation: pointer.Bool(false),
						Capabilities:             &k8sv1.Capabilities{Drop: []k8sv1.Capability{"ALL"}},
					},
				},
			},
			SecurityContext: &k8sv1.PodSecurityContext{
				RunAsNonRoot:   pointer.Bool(true),
				RunAsUser:      &user,
				SeccompProfile: &k8sv1.SeccompProfile{Type: k8sv1.SeccompProfileTypeRuntimeDefault},
			},
		},
	}
}

func withCoreNumber(number uint32) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}
		vmi.Spec.Domain.CPU.Cores = number
	}
}

func withACPIFeature() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &v1.Features{}
		}
		vmi.Spec.Domain.Features.ACPI = v1.FeatureState{}
	}
}

func withAPICFeature() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &v1.Features{}
		}
		vmi.Spec.Domain.Features.APIC = &v1.FeatureAPIC{}
	}
}

func withRelaxedHypervFeature() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &v1.Features{}
		}
		if vmi.Spec.Domain.Features.Hyperv == nil {
			vmi.Spec.Domain.Features.Hyperv = &v1.FeatureHyperv{}
		}
		vmi.Spec.Domain.Features.Hyperv.Relaxed = &v1.FeatureState{}
	}
}

func withRelaxedVAPICHypervFeature() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &v1.Features{}
		}
		if vmi.Spec.Domain.Features.Hyperv == nil {
			vmi.Spec.Domain.Features.Hyperv = &v1.FeatureHyperv{}
		}
		vmi.Spec.Domain.Features.Hyperv.VAPIC = &v1.FeatureState{}
	}
}

func withRelaxedSpinlocksHypervFeature(spinlocks uint32) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &v1.Features{}
		}
		if vmi.Spec.Domain.Features.Hyperv == nil {
			vmi.Spec.Domain.Features.Hyperv = &v1.FeatureHyperv{}
		}
		vmi.Spec.Domain.Features.Hyperv.Spinlocks = &v1.FeatureSpinlocks{Retries: &spinlocks}
	}
}

func withClockOffsetUTC() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Clock == nil {
			vmi.Spec.Domain.Clock = &v1.Clock{}
		}
		vmi.Spec.Domain.Clock.UTC = &v1.ClockOffsetUTC{}

	}
}

func withPITTimer(tickPolicy v1.PITTickPolicy) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Clock == nil {
			vmi.Spec.Domain.Clock = &v1.Clock{}
		}
		if vmi.Spec.Domain.Clock.Timer == nil {
			vmi.Spec.Domain.Clock.Timer = &v1.Timer{}
		}
		vmi.Spec.Domain.Clock.Timer.PIT = &v1.PITTimer{TickPolicy: tickPolicy}
	}
}

func withRTCTimer(tickPolicy v1.RTCTickPolicy) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Clock == nil {
			vmi.Spec.Domain.Clock = &v1.Clock{}
		}
		if vmi.Spec.Domain.Clock.Timer == nil {
			vmi.Spec.Domain.Clock.Timer = &v1.Timer{}
		}
		vmi.Spec.Domain.Clock.Timer.RTC = &v1.RTCTimer{TickPolicy: tickPolicy}
	}
}

func withHypervTimer() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Clock == nil {
			vmi.Spec.Domain.Clock = &v1.Clock{}
		}
		if vmi.Spec.Domain.Clock.Timer == nil {
			vmi.Spec.Domain.Clock.Timer = &v1.Timer{}
		}
		vmi.Spec.Domain.Clock.Timer.Hyperv = &v1.HypervTimer{}
	}
}

func withFirmwareUUID(firmware types.UID) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		vmi.Spec.Domain.Firmware.UUID = firmware
	}
}

func withDiskDevice(diskName string, bus v1.DiskBus) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		disk := v1.Disk{
			Name:       diskName,
			DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: bus}},
		}
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)
	}
}

func withCDRomDiskDevice(diskName string, bus v1.DiskBus) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		disk := v1.Disk{
			Name:       diskName,
			DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{Bus: bus}},
		}
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)
	}
}

func withEphemeralVolumeSource(name, claimName string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		volume := v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				Ephemeral: &v1.EphemeralVolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: claimName,
					}},
			},
		}
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, volume)
	}
}

func withSysprepVolumesource(name, sysprepSourceName string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		volume := v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				Sysprep: &v1.SysprepSource{
					ConfigMap: &k8sv1.LocalObjectReference{
						Name: sysprepSourceName,
					}},
			},
		}
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, volume)
	}
}
