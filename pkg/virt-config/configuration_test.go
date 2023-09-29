package virtconfig_test

import (
	"encoding/json"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("test configuration", func() {
	validMemBalloonStatsPeriod := uint32(3)
	DescribeTable("when memBalloonStatsPeriod", func(value *uint32, result uint32) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			MemBalloonStatsPeriod: value,
		})

		Expect(clusterConfig.GetMemBalloonStatsPeriod()).To(Equal(result))
	},
		Entry("is positive, GetMemBalloonStatsPeriod should return period", &validMemBalloonStatsPeriod, uint32(3)),
		Entry("when unset, GetMemBalloonStatsPeriod should return 10", nil, uint32(10)),
	)

	DescribeTable(" when useEmulation", func(value bool, result bool) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				UseEmulation: value,
			},
		})
		Expect(clusterConfig.AllowEmulation()).To(Equal(result))
	},
		Entry("is true, AllowEmulation should return true", true, true),
		Entry("is false, AllowEmulation should return false", false, false),
	)

	trueValue := true
	falseValue := false
	DescribeTable(" when permitSlirpInterface", func(value *bool, result bool) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			NetworkConfiguration: &v1.NetworkConfiguration{
				PermitSlirpInterface: value,
			},
		})

		Expect(clusterConfig.IsSlirpInterfaceEnabled()).To(Equal(result))
	},
		Entry("is true, IsSlirpInterfaceEnabled should return true", &trueValue, true),
		Entry("is false, IsSlirpInterfaceEnabled should return false", &falseValue, false),
		Entry("when unset, IsSlirpInterfaceEnabled should return false", nil, false),
	)

	DescribeTable(" when permitBridgeInterfaceOnPodNetwork", func(value *bool, result bool) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			NetworkConfiguration: &v1.NetworkConfiguration{
				PermitBridgeInterfaceOnPodNetwork: value,
			},
		})

		Expect(clusterConfig.IsBridgeInterfaceOnPodNetworkEnabled()).To(Equal(result))
	},
		Entry("is true, IsBridgeInterfaceOnPodNetworkEnabled should return true", &trueValue, true),
		Entry("is false, IsBridgeInterfaceOnPodNetworkEnabled should return false", &falseValue, false),
		Entry("when unset, IsBridgeInterfaceOnPodNetworkEnabled should return true", nil, true),
	)

	DescribeTable(" when defaultNetworkInterface", func(value string, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			NetworkConfiguration: &v1.NetworkConfiguration{
				NetworkInterface: value,
			},
		})
		Expect(clusterConfig.GetDefaultNetworkInterface()).To(Equal(result))
	},
		Entry("is bridge, GetDefaultNetworkInterface should return bridge", "bridge", "bridge"),
		Entry("is slirp, GetDefaultNetworkInterface should return slirp", "slirp", "slirp"),
		Entry("is masquerade, GetDefaultNetworkInterface should return masquerade", "masquerade", "masquerade"),
		Entry("when unset, GetDefaultNetworkInterface should return the default", "", "bridge"),
		Entry("when invalid, GetDefaultNetworkInterface should return the default", "invalid", "bridge"),
	)

	DescribeTable(" when imagePullPolicy", func(value string, result kubev1.PullPolicy) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			ImagePullPolicy: kubev1.PullPolicy(value),
		})
		Expect(clusterConfig.GetImagePullPolicy()).To(Equal(result))
	},
		Entry("is PullAlways, GetImagePullPolicy should return PullAlways", "Always", kubev1.PullAlways),
		Entry("is Never, GetImagePullPolicy should return Never", "Never", kubev1.PullNever),
		Entry("is IsNotPresent, GetImagePullPolicy should return IsNotPresent", "IfNotPresent", kubev1.PullIfNotPresent),
		Entry("when unset, GetImagePullPolicy should return PullIfNotPresent", "", kubev1.PullIfNotPresent),
		Entry("when invalid, GetImagePullPolicy should return the default", "invalid", kubev1.PullIfNotPresent),
	)

	DescribeTable(" when lessPVCSpaceToleration", func(value int, result int) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				LessPVCSpaceToleration: value,
			},
		})
		Expect(clusterConfig.GetLessPVCSpaceToleration()).To(Equal(result))
	},
		Entry("is set, GetLessPVCSpaceToleration should return correct value", 5, 5),
		Entry("is unset, GetLessPVCSpaceToleration should return the default", 0, virtconfig.DefaultLessPVCSpaceToleration),
		Entry("is invalid, GetLessPVCSpaceToleration should return the default", -1, virtconfig.DefaultLessPVCSpaceToleration),
	)

	nodeSelectors := map[string]string{
		"kubernetes.io/hostname":          "node02",
		"node-role.kubernetes.io/compute": "true",
	}
	DescribeTable(" when nodeSelectors", func(value, result map[string]string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				NodeSelectors: value,
			},
		})
		Expect(clusterConfig.GetNodeSelectors()).To(Equal(result))
	},
		Entry("is set, GetNodeSelectors should return correct value", nodeSelectors, nodeSelectors),
		Entry("is unset, GetNodeSelectors should return the default", nil, nil),
		Entry("is empty, GetNodeSelectors should return the default", map[string]string{}, nil),
	)

	DescribeTable(" when machineType", func(cpuArch string, machineTypeAMD64 string, machineTypeARM64 string, machineTypePPC64le string, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVWithCPUArch(&v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ArchitectureConfiguration: &v1.ArchConfiguration{
						Amd64:   &v1.ArchSpecificConfiguration{MachineType: machineTypeAMD64},
						Arm64:   &v1.ArchSpecificConfiguration{MachineType: machineTypeARM64},
						Ppc64le: &v1.ArchSpecificConfiguration{MachineType: machineTypePPC64le},
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: "Deployed",
			},
		}, cpuArch)
		Expect(clusterConfig.GetMachineType(cpuArch)).To(Equal(result))
	},
		Entry("when amd64 set, GetMachineType should return the value", "amd64", "pc-q35-3.0", "", "", "pc-q35-3.0"),
		Entry("when arm64 set, GetMachineType should return the value", "arm64", "", "virt", "", "virt"),
		Entry("when ppc64le set, GetMachineType should return the value", "ppc64le", "", "", "pseries", "pseries"),
		Entry("when amd64 unset, GetMachineType should return the default with amd64", "amd64", "", "", "", virtconfig.DefaultAMD64MachineType),
		Entry("when arm64 unset, GetMachineType should return the default with arm64", "arm64", "", "", "", virtconfig.DefaultAARCH64MachineType),
		Entry("when ppc64le unset, GetMachineType should return the default with ppc64le", "ppc64le", "", "", "", virtconfig.DefaultPPC64LEMachineType),
	)

	Context("when deprecated machineType is set", func() {
		It("it should have higher priority than the architectureConfiguration", func() {
			const machineType = "quantum-qc35"
			const cpuArch = "amd64"

			clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVWithCPUArch(&v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "kubevirt",
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						MachineType: machineType,
						ArchitectureConfiguration: &v1.ArchConfiguration{
							Amd64: &v1.ArchSpecificConfiguration{MachineType: virtconfig.DefaultAMD64MachineType},
						},
					},
				},
				Status: v1.KubeVirtStatus{
					Phase: "Deployed",
				},
			}, cpuArch)

			Expect(clusterConfig.GetMachineType(cpuArch)).To(Equal(machineType))
		})
	})

	DescribeTable(" when cpuModel", func(value string, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			CPUModel: value,
		})
		Expect(clusterConfig.GetCPUModel()).To(Equal(result))
	},
		Entry("when set, GetCPUModel should return the value", "Haswell", "Haswell"),
		Entry("when unset, GetCPUModel should return empty string", "", ""),
	)

	validCpuRequest := resource.MustParse("400m")
	DescribeTable(" when cpuRequest", func(value *resource.Quantity, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			CPURequest: value,
		})
		cpuRequest := clusterConfig.GetCPURequest()
		Expect(cpuRequest.String()).To(Equal(result))
	},
		Entry("when set, GetCPURequest should return the value", &validCpuRequest, "400m"),
		Entry("when unset, GetCPURequest should return the default", nil, virtconfig.DefaultCPURequest),
	)

	DescribeTable(" when memoryOvercommit", func(value int, result int) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				MemoryOvercommit: value,
			},
		})
		Expect(clusterConfig.GetMemoryOvercommit()).To(Equal(result))
	},
		Entry("when set, GetMemoryOvercommit should return the value", 150, 150),
		Entry("when unset, GetMemoryOvercommit should return the default", 0, virtconfig.DefaultMemoryOvercommit),
		Entry("when negative, GetMemoryOvercommit should return the default", -150, virtconfig.DefaultMemoryOvercommit),
	)

	DescribeTable(" when CPUAllocationRatio", func(value int, result int) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				CPUAllocationRatio: value,
			},
		})
		Expect(clusterConfig.GetCPUAllocationRatio()).To(Equal(result))
	},
		Entry("when set, GetCPUAllocationRatio should return the value", 150, 150),
		Entry("when unset, GetCPUAllocationRatio should return the default", 0, virtconfig.DefaultCPUAllocationRatio),
		Entry("when negative, GetCPUAllocationRatio should return the default", -150, virtconfig.DefaultCPUAllocationRatio),
	)

	DescribeTable(" when emulatedMachines", func(cpuArch string, emuMachinesAMD64 []string, emuMachinesARM64 []string, emuMachinesAPC64le64 []string, result []string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVWithCPUArch(&v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ArchitectureConfiguration: &v1.ArchConfiguration{
						Amd64:   &v1.ArchSpecificConfiguration{EmulatedMachines: emuMachinesAMD64},
						Arm64:   &v1.ArchSpecificConfiguration{EmulatedMachines: emuMachinesARM64},
						Ppc64le: &v1.ArchSpecificConfiguration{EmulatedMachines: emuMachinesAPC64le64},
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: "Deployed",
			},
		}, cpuArch)
		emulatedMachines := clusterConfig.GetEmulatedMachines(cpuArch)
		Expect(emulatedMachines).To(ConsistOf(result))
	},
		Entry("when amd64 set, GetEmulatedMachines should return the value", "amd64", []string{"q35", "i440*"}, nil, nil, []string{"q35", "i440*"}),
		Entry("when arm64 set, GetEmulatedMachines should return the value", "arm64", nil, []string{"virt*"}, nil, []string{"virt*"}),
		Entry("when ppc64le set, GetEmulatedMachines should return the value", "ppc64le", nil, nil, []string{"pseries*"}, []string{"pseries*"}),
		Entry("when unset, GetEmulatedMachines should return the defaults with amd64", "amd64", nil, nil, nil, strings.Split(virtconfig.DefaultAMD64EmulatedMachines, ",")),
		Entry("when empty, GetEmulatedMachines should return the defaults with amd64", "amd64", []string{}, []string{}, []string{}, strings.Split(virtconfig.DefaultAMD64EmulatedMachines, ",")),
		Entry("when unset, GetEmulatedMachines should return the defaults with arm64", "arm64", nil, nil, nil, strings.Split(virtconfig.DefaultAARCH64EmulatedMachines, ",")),
		Entry("when empty, GetEmulatedMachines should return the defaults with arm64", "arm64", []string{}, []string{}, []string{}, strings.Split(virtconfig.DefaultAARCH64EmulatedMachines, ",")),
		Entry("when unset, GetEmulatedMachines should return the defaults with ppc64le", "ppc64le", nil, nil, nil, strings.Split(virtconfig.DefaultPPC64LEEmulatedMachines, ",")),
		Entry("when empty, GetEmulatedMachines should return the defaults with ppc64le", "ppc64le", []string{}, []string{}, []string{}, strings.Split(virtconfig.DefaultPPC64LEEmulatedMachines, ",")),
	)

	DescribeTable("when virtualMachineOptions", func(virtualMachineOptions *v1.VirtualMachineOptions, expected bool) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKV(&v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					VirtualMachineOptions: virtualMachineOptions,
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: "Deployed",
			},
		})
		Expect(clusterConfig.IsFreePageReportingDisabled()).To(BeEquivalentTo(expected))
	},
		Entry("is nil, IsFreePageReportingDisabled should return false",
			nil, false,
		),
		Entry("is an empty struct, IsFreePageReportingDisabled should return false",
			&v1.VirtualMachineOptions{}, false,
		),
		Entry("contains disableFreePageReporting, IsFreePageReportingDisabled should return true",
			&v1.VirtualMachineOptions{DisableFreePageReporting: &v1.DisableFreePageReporting{}}, true,
		),
	)

	DescribeTable(" when maxHotplugRatio", func(value int, expected int) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			LiveUpdateConfiguration: &v1.LiveUpdateConfiguration{
				MaxHotplugRatio: uint32(value),
			},
		})
		Expect(clusterConfig.GetMaxHotplugRatio()).To(Equal(uint32(expected)))
	},
		Entry("is set, GetMaxHotplugRatio should return the set value", 100, 100),
		Entry("is unset, GetMaxHotplugRatio should return the default", 0, virtconfig.DefaultMaxHotplugRatio),
	)

	// deprecated
	DescribeTable(" when supportedGuestAgentVersions", func(value []string, result []string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			SupportedGuestAgentVersions: value,
		})
		supportedGuestAgentVersions := clusterConfig.GetSupportedAgentVersions()
		Expect(supportedGuestAgentVersions).To(ConsistOf(result))
	},
		Entry("when set, GetSupportedAgentVersions should return the value", []string{"5.*", "6.*"}, []string{"5.*", "6.*"}),
		Entry("when unset, GetSupportedAgentVersions should return the defaults", nil, strings.Split(virtconfig.SupportedGuestAgentVersions, ",")),
		Entry("when empty, GetSupportedAgentVersions should return the defaults", []string{}, strings.Split(virtconfig.SupportedGuestAgentVersions, ",")),
	)

	It("Should return migration config values", func() {

		parallelOutboundMigrationsPerNode := uint32(10)
		parallelMigrationsPerCluster := uint32(20)
		bandwidthPerMigration := resource.MustParse("110Mi")
		progressTimeout := int64(5)
		completionTimeoutPerGiB := int64(5)
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			MigrationConfiguration: &v1.MigrationConfiguration{
				ParallelOutboundMigrationsPerNode: &parallelOutboundMigrationsPerNode,
				ParallelMigrationsPerCluster:      &parallelMigrationsPerCluster,
				BandwidthPerMigration:             &bandwidthPerMigration,
				ProgressTimeout:                   &progressTimeout,
				CompletionTimeoutPerGiB:           &completionTimeoutPerGiB,
				UnsafeMigrationOverride:           &trueValue,
				AllowAutoConverge:                 &trueValue,
			},
		})

		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 20))
		bandwidth := *result.BandwidthPerMigration
		Expect(bandwidth.String()).To(Equal("110Mi"))
		Expect(*result.ProgressTimeout).To(BeNumerically("==", 5))
		Expect(*result.CompletionTimeoutPerGiB).To(BeNumerically("==", 5))
		Expect(*result.UnsafeMigrationOverride).To(BeTrue())
		Expect(*result.AllowAutoConverge).To(BeTrue())
	})

	It("Should return defaults if parts of the config are not set", func() {
		parallelOutboundMigrationsPerNode := uint32(10)
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			MigrationConfiguration: &v1.MigrationConfiguration{
				ParallelOutboundMigrationsPerNode: &parallelOutboundMigrationsPerNode,
			},
		})

		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 5))
		Expect(result.BandwidthPerMigration.String()).To(Equal("0"))
	})

	It("Should update the config if a newer version is available", func() {
		oldValue := uint32(10)
		clusterConfig, _, kvInformer := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			MigrationConfiguration: &v1.MigrationConfiguration{
				ParallelOutboundMigrationsPerNode: &oldValue,
			},
		})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeEquivalentTo(10))

		newValue := uint32(9)
		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MigrationConfiguration: &v1.MigrationConfiguration{
						ParallelOutboundMigrationsPerNode: &newValue,
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: "Deployed",
			},
		}
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)
		Eventually(func() uint32 {
			return *clusterConfig.GetMigrationConfiguration().ParallelOutboundMigrationsPerNode
		}).Should(BeEquivalentTo(9))
	})

	It("Should stick with the last good config", func() {

		clusterConfig, _, kvInformer := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			ImagePullPolicy: kubev1.PullAlways,
		})
		Expect(clusterConfig.GetImagePullPolicy()).To(Equal(kubev1.PullAlways))

		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ImagePullPolicy: kubev1.PullPolicy("invalid"),
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: "Deployed",
			},
		}
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)

		Consistently(func() kubev1.PullPolicy {
			return clusterConfig.GetImagePullPolicy()
		}).Should(Equal(kubev1.PullAlways))
	})

	It("should return the default config if no config map exists", func() {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeEquivalentTo(2))
	})

	DescribeTable("SMBIOS values", func(value *v1.SMBiosConfiguration, result *cmdv1.SMBios) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			SMBIOSConfig: value,
		})
		smbios := clusterConfig.GetSMBIOS()

		smbiosJSON, err := json.Marshal(smbios)
		Expect(err).ToNot(HaveOccurred())

		resultJSON, err := json.Marshal(result)
		Expect(err).ToNot(HaveOccurred())

		Expect(string(smbiosJSON)).To(BeEquivalentTo(string(resultJSON)))
	},
		Entry("when values set, should equal to result", &v1.SMBiosConfiguration{Family: "test", Product: "test", Manufacturer: "None"}, &cmdv1.SMBios{Family: "test", Product: "test", Manufacturer: "None"}),
		Entry("When unset, should return default values", nil, &cmdv1.SMBios{Family: "KubeVirt", Product: "None", Manufacturer: "KubeVirt"}),
	)

	DescribeTable(" when SELinuxLauncherType", func(value string, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			SELinuxLauncherType: value,
		})
		selinuxLauncherType := clusterConfig.GetSELinuxLauncherType()
		Expect(selinuxLauncherType).To(Equal(result))
	},
		Entry("when set, GetSELinuxLauncherType should return the value", "spc_t", "spc_t"),
		Entry("when unset, GetSELinuxLauncherType should return the default", virtconfig.DefaultSELinuxLauncherType, virtconfig.DefaultSELinuxLauncherType),
	)

	DescribeTable(" when OVMFPath", func(cpuArch string, ovmfPathKeyAMD64 string, ovmfPathKeyARM64 string, ovmfPathKeyPPC64le64 string, result string) {

		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ArchitectureConfiguration: &v1.ArchConfiguration{
						Amd64:   &v1.ArchSpecificConfiguration{OVMFPath: ovmfPathKeyAMD64},
						Arm64:   &v1.ArchSpecificConfiguration{OVMFPath: ovmfPathKeyARM64},
						Ppc64le: &v1.ArchSpecificConfiguration{OVMFPath: ovmfPathKeyPPC64le64},
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: "Deployed",
			},
		}

		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVWithCPUArch(kv, cpuArch)
		ovmfPath := clusterConfig.GetOVMFPath(cpuArch)
		Expect(ovmfPath).To(Equal(result))
	},
		Entry("when amd64 set, GetOVMFPath should return the value", "amd64", "/usr/share/ovmf/x64", "", "", "/usr/share/ovmf/x64"),
		Entry("when arm64 set, GetOVMFPath should return the value", "arm64", "", "/usr/share/AAVMF", "", "/usr/share/AAVMF"),
		Entry("when ppc64le set, GetOVMFPath should return the value", "ppc64le", "", "", "/usr/share/ovmf/x64", "/usr/share/ovmf/x64"),
		Entry("when unset, GetOVMFPath should return the default with amd64", "amd64", "", "", "", virtconfig.DefaultARCHOVMFPath),
		Entry("when unset, GetOVMFPath should return the default with arm64", "arm64", "", "", "", virtconfig.DefaultAARCH64OVMFPath),
		Entry("when unset, GetOVMFPath should return the default with ppc64le", "ppc64le", "", "", "", virtconfig.DefaultARCHOVMFPath),
	)

	It("verifies that SetConfigModifiedCallback works as expected ", func() {
		lock := &sync.Mutex{}
		var callbackSet1, callbackSet2 bool
		callback1 := func() {
			lock.Lock()
			defer lock.Unlock()
			callbackSet1 = true
		}
		callback2 := func() {
			lock.Lock()
			defer lock.Unlock()
			callbackSet2 = true
		}
		KV := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: rand.String(10),
				Name:            "kubevirt",
				Namespace:       "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						LogVerbosity: &v1.LogVerbosity{
							VirtLauncher: 3,
						},
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
			},
		}
		clusterConfig, _, kubeVirtInformer := testutils.NewFakeClusterConfigUsingKV(KV)
		callbackSet1 = false
		callbackSet2 = false
		clusterConfig.SetConfigModifiedCallback(callback1)
		clusterConfig.SetConfigModifiedCallback(callback2)

		Expect(clusterConfig.GetVirtLauncherVerbosity()).To(Equal(uint(3)))
		KV.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtLauncher = 6
		testutils.UpdateFakeKubeVirtClusterConfig(kubeVirtInformer, KV)
		Expect(clusterConfig.GetVirtLauncherVerbosity()).To(Equal(uint(6)))
		Eventually(func() bool {
			lock.Lock()
			defer lock.Unlock()
			return callbackSet1 && callbackSet2
		}).Should(BeTrue())
	})
	DescribeTable("mdev configuration", func(nodeLabels map[string]string, expectedResult []string) {
		node := &kubev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testNode",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: v1.GroupVersion.String(),
			},
		}

		node.Status.Phase = kubev1.NodeRunning
		node.ObjectMeta.Labels = nodeLabels
		KV := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: rand.String(10),
				Name:            "kubevirt",
				Namespace:       "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MediatedDevicesConfiguration: &v1.MediatedDevicesConfiguration{
						MediatedDeviceTypes: []string{

							"nvidia-222",
							"nvidia-228",
							"i915-GVTg_V5_4",
						},
						NodeMediatedDeviceTypes: []v1.NodeMediatedDeviceTypesConfig{
							{
								NodeSelector: map[string]string{
									"testLabel1": "true",
								},
								MediatedDeviceTypes: []string{
									"nvidia-223",
								},
							},
							{
								NodeSelector: map[string]string{
									"testLabel2": "true",
								},
								MediatedDeviceTypes: []string{
									"nvidia-229",
								},
							},
							{
								NodeSelector: map[string]string{
									"testLabel3": "true",
									"testLabel4": "true",
								},
								MediatedDeviceTypes: []string{
									"nvidia-230",
								},
							},
						},
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
			},
		}
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKV(KV)
		nodeMdevConf := clusterConfig.GetDesiredMDEVTypes(node)
		Expect(nodeMdevConf).Should(ConsistOf(expectedResult))
	},
		Entry("expect default config when no selecors matched",
			map[string]string{},
			[]string{"nvidia-222", "nvidia-228", "i915-GVTg_V5_4"}),
		Entry("specific label match",
			map[string]string{"testLabel1": "true"},
			[]string{"nvidia-223"}),
		Entry("should match node by multiple selectors",
			map[string]string{"testLabel3": "true", "testLabel4": "true"},
			[]string{"nvidia-230"}),
		Entry("expect a merged result when several selectors match the same node",
			map[string]string{"testLabel1": "true", "testLabel2": "true"},
			[]string{"nvidia-223", "nvidia-229"}),
	)

	DescribeTable("when kubevirt CR holds config", func(value v1.KubeVirtConfiguration, getPart func(*v1.KubeVirtConfiguration) interface{}, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKV(&v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: rand.String(10),
				Name:            "kubevirt",
				Namespace:       "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: value,
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
			},
		})

		kubevirtConfig := clusterConfig.GetConfig()
		partJson, err := json.Marshal(getPart(kubevirtConfig))
		Expect(err).ToNot(HaveOccurred())

		Expect(string(partJson)).To(BeEquivalentTo(result))
	},
		Entry("when machineType set, should equal to result",
			v1.KubeVirtConfiguration{
				MachineType: "test",
			},
			func(c *v1.KubeVirtConfiguration) interface{} {
				return c.MachineType
			},
			`"test"`),
		Entry("when developerConfiguration set, should equal to result",
			v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					FeatureGates:           []string{"test1", "test2"},
					LessPVCSpaceToleration: 5,
					MemoryOvercommit:       150,
					NodeSelectors:          map[string]string{"test": "test"},
					UseEmulation:           true,
					CPUAllocationRatio:     25,
					DiskVerification: &v1.DiskVerification{
						MemoryLimit: resource.NewScaledQuantity(1, resource.Giga),
					},
				},
			},
			func(c *v1.KubeVirtConfiguration) interface{} {
				return c.DeveloperConfiguration
			},
			`{"featureGates":["test1","test2"],"pvcTolerateLessSpaceUpToPercent":5,"minimumReservePVCBytes":131072,"memoryOvercommit":150,"nodeSelectors":{"test":"test"},"useEmulation":true,"cpuAllocationRatio":25,"diskVerification":{"memoryLimit":"1G"},"logVerbosity":{"virtAPI":2,"virtController":2,"virtHandler":2,"virtLauncher":2,"virtOperator":2}}`),
		Entry("when wrong networkConfiguration set, should use the default",
			v1.KubeVirtConfiguration{
				NetworkConfiguration: &v1.NetworkConfiguration{
					NetworkInterface:                  "test",
					PermitSlirpInterface:              pointer.BoolPtr(true),
					PermitBridgeInterfaceOnPodNetwork: pointer.BoolPtr(false),
				},
			},
			func(c *v1.KubeVirtConfiguration) interface{} {
				return c.NetworkConfiguration
			},
			`{"defaultNetworkInterface":"bridge","permitSlirpInterface":false,"permitBridgeInterfaceOnPodNetwork":true}`),
		Entry("when networkConfiguration set, should equal to result",
			v1.KubeVirtConfiguration{
				NetworkConfiguration: &v1.NetworkConfiguration{
					NetworkInterface:                  string(v1.SlirpInterface),
					PermitSlirpInterface:              pointer.BoolPtr(true),
					PermitBridgeInterfaceOnPodNetwork: pointer.BoolPtr(false),
				},
			},
			func(c *v1.KubeVirtConfiguration) interface{} {
				return c.NetworkConfiguration
			},
			`{"defaultNetworkInterface":"slirp","permitSlirpInterface":true,"permitBridgeInterfaceOnPodNetwork":false}`),
		Entry("when networkConfiguration set with empty NetworkInterface, should use the default",
			v1.KubeVirtConfiguration{
				NetworkConfiguration: &v1.NetworkConfiguration{
					PermitSlirpInterface:              pointer.BoolPtr(true),
					PermitBridgeInterfaceOnPodNetwork: pointer.BoolPtr(false),
				},
			},
			func(c *v1.KubeVirtConfiguration) interface{} {
				return c.NetworkConfiguration
			},
			`{"defaultNetworkInterface":"bridge","permitSlirpInterface":true,"permitBridgeInterfaceOnPodNetwork":false}`),
	)

	DescribeTable("when ClusterProfiler feature-gate", func(openFeatureGates []string, isEnabled bool) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				FeatureGates: openFeatureGates,
			},
		})

		Expect(clusterConfig.ClusterProfilerEnabled()).To(Equal(isEnabled))
	},
		Entry("ClusterProfiler feature gate not set should result in cluster profiler being disabled",
			nil, false),
		Entry("ClusterProfiler feature gate empty should result in cluster profiler being disabled",
			[]string{}, false),
		Entry("ClusterProfiler feature gate enabled should result in cluster profiler being enabled",
			[]string{virtconfig.ClusterProfiler}, true),
	)

	Context("deprecated feature gates should always be considered as enabled", func() {
		var clusterConfig *virtconfig.ClusterConfig

		BeforeEach(func() {
			clusterConfig, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					FeatureGates: nil,
				},
			})
		})

		It("live migration feature gate", func() {
			Expect(clusterConfig.LiveMigrationEnabled()).To(BeTrue())
		})

		It("SR-IOV live migration feature gate", func() {
			Expect(clusterConfig.SRIOVLiveMigrationEnabled()).To(BeTrue())
		})
	})
})
