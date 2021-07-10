package virtconfig_test

import (
	"encoding/json"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/client-go/api/v1"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	testutils "kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("ConfigMap", func() {
	table.DescribeTable("when memBalloonStatsPeriod", func(value string, result uint32) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{"memBalloonStatsPeriod": value},
		})

		Expect(clusterConfig.GetMemBalloonStatsPeriod()).To(Equal(result))
	},
		table.Entry("is positive, GetMemBalloonStatsPeriod should return period", "3", uint32(3)),
		table.Entry("is negative, GetMemBalloonStatsPeriod should return 10", "-1", uint32(10)),
		table.Entry("when unset, GetMemBalloonStatsPeriod should return 10", "", uint32(10)),
		table.Entry("when invalid, GetMemBalloonStatsPeriod should return 10", "invalid", uint32(10)))

	table.DescribeTable(" when useEmulation", func(value string, result bool) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{"debug.useEmulation": value},
		})
		Expect(clusterConfig.IsUseEmulation()).To(Equal(result))
	},
		table.Entry("is true, IsUseEmulation should return true", "true", true),
		table.Entry("is false, IsUseEmulation should return false", "false", false),
		table.Entry("when unset, IsUseEmulation should return false", "", false),
		table.Entry("when invalid, IsUseEmulation should return the default", "invalid", false),
	)

	table.DescribeTable(" when permitSlirpInterface", func(value string, result bool) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{"permitSlirpInterface": value},
		})
		Expect(clusterConfig.IsSlirpInterfaceEnabled()).To(Equal(result))
	},
		table.Entry("is true, IsSlirpInterfaceEnabled should return true", "true", true),
		table.Entry("is false, IsSlirpInterfaceEnabled should return false", "false", false),
		table.Entry("when unset, IsSlirpInterfaceEnabled should return false", "", false),
		table.Entry("when invalid, IsSlirpInterfaceEnabled should return the default", "invalid", false),
	)

	table.DescribeTable(" when permitBridgeInterfaceOnPodNetwork", func(value string, result bool) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{"permitBridgeInterfaceOnPodNetwork": value},
		})
		Expect(clusterConfig.IsBridgeInterfaceOnPodNetworkEnabled()).To(Equal(result))
	},
		table.Entry("is true, IsBridgeInterfaceOnPodNetworkEnabled should return true", "true", true),
		table.Entry("is false, IsBridgeInterfaceOnPodNetworkEnabled should return false", "false", false),
		table.Entry("when unset, IsBridgeInterfaceOnPodNetworkEnabled should return true", "", true),
		table.Entry("when invalid, IsBridgeInterfaceOnPodNetworkEnabled should return the default", "invalid", true),
	)

	table.DescribeTable(" when imagePullPolicy", func(value string, result kubev1.PullPolicy) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.ImagePullPolicyKey: value},
		})
		Expect(clusterConfig.GetImagePullPolicy()).To(Equal(result))
	},
		table.Entry("is PullAlways, GetImagePullPolicy should return PullAlways", "Always", kubev1.PullAlways),
		table.Entry("is Never, GetImagePullPolicy should return Never", "Never", kubev1.PullNever),
		table.Entry("is IsNotPresent, GetImagePullPolicy should return IsNotPresent", "IfNotPresent", kubev1.PullIfNotPresent),
		table.Entry("when unset, GetImagePullPolicy should return PullIfNotPresent", "", kubev1.PullIfNotPresent),
		table.Entry("when invalid, GetImagePullPolicy should return the default", "invalid", kubev1.PullIfNotPresent),
	)

	table.DescribeTable(" when lessPVCSpaceToleration", func(value string, result int) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.LessPVCSpaceTolerationKey: value},
		})
		Expect(clusterConfig.GetLessPVCSpaceToleration()).To(Equal(result))
	},
		table.Entry("is set, GetLessPVCSpaceToleration should return correct value", "5", 5),
		table.Entry("is unset, GetLessPVCSpaceToleration should return the default", "", virtconfig.DefaultLessPVCSpaceToleration),
		table.Entry("is invalid, GetLessPVCSpaceToleration should return the default", "-1", virtconfig.DefaultLessPVCSpaceToleration),
	)

	table.DescribeTable(" when defaultNetworkInterface", func(value string, result string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.NetworkInterfaceKey: value},
		})
		Expect(clusterConfig.GetDefaultNetworkInterface()).To(Equal(result))
	},
		table.Entry("is bridge, GetDefaultNetworkInterface should return bridge", "bridge", "bridge"),
		table.Entry("is slirp, GetDefaultNetworkInterface should return slirp", "slirp", "slirp"),
		table.Entry("is masquerade, GetDefaultNetworkInterface should return masquerade", "masquerade", "masquerade"),
		table.Entry("when unset, GetDefaultNetworkInterface should return the default", "", "bridge"),
		table.Entry("when invalid, GetDefaultNetworkInterface should return the default", "invalid", "bridge"),
	)

	nodeSelectorsStr := "kubernetes.io/hostname=node02\nnode-role.kubernetes.io/compute=true\n"
	nodeSelectors := map[string]string{
		"kubernetes.io/hostname":          "node02",
		"node-role.kubernetes.io/compute": "true",
	}
	table.DescribeTable(" when nodeSelectors", func(value string, result map[string]string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.NodeSelectorsKey: value},
		})
		Expect(clusterConfig.GetNodeSelectors()).To(Equal(result))
	},
		table.Entry("is set, GetNodeSelectors should return correct value", nodeSelectorsStr, nodeSelectors),
		table.Entry("is unset, GetNodeSelectors should return the default", "", nil),
		table.Entry("is invalid, GetNodeSelectors should return the default", "-1", nil),
	)

	table.DescribeTable(" when machineType", func(cpuArch string, machineType string, result string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfigWithCPUArch(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MachineTypeKey: machineType},
		}, cpuArch)
		Expect(clusterConfig.GetMachineType()).To(Equal(result))
	},
		table.Entry("when set, GetMachineType should return the value", "", "pc-q35-3.0", "pc-q35-3.0"),
		table.Entry("when unset, GetMachineType should return the default with amd64", "amd64", "", virtconfig.DefaultAMD64MachineType),
		table.Entry("when unset, GetMachineType should return the default with arm64", "arm64", "", virtconfig.DefaultAARCH64MachineType),
		table.Entry("when unset, GetMachineType should return the default with ppc64le", "ppc64le", "", virtconfig.DefaultPPC64LEMachineType),
	)

	table.DescribeTable(" when cpuModel", func(value string, result string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.CPUModelKey: value},
		})
		Expect(clusterConfig.GetCPUModel()).To(Equal(result))
	},
		table.Entry("when set, GetCPUModel should return the value", "Haswell", "Haswell"),
		table.Entry("when unset, GetCPUModel should return empty string", "", ""),
	)

	table.DescribeTable(" when cpuRequest", func(value string, result string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.CPURequestKey: value},
		})
		cpuRequest := clusterConfig.GetCPURequest()
		Expect(cpuRequest.String()).To(Equal(result))
	},
		table.Entry("when set, GetCPURequest should return the value", "400m", "400m"),
		table.Entry("when unset, GetCPURequest should return the default", "", virtconfig.DefaultCPURequest),
	)

	table.DescribeTable(" when memoryOvercommit", func(value string, result int) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MemoryOvercommitKey: value},
		})
		Expect(clusterConfig.GetMemoryOvercommit()).To(Equal(result))
	},
		table.Entry("when set, GetMemoryOvercommit should return the value", "150", 150),
		table.Entry("when unset, GetMemoryOvercommit should return the default", "", virtconfig.DefaultMemoryOvercommit),
	)

	table.DescribeTable(" when emulatedMachines", func(cpuArch string, emuMachinesKey string, result []string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfigWithCPUArch(&kubev1.ConfigMap{
			Data: map[string]string{
				virtconfig.EmulatedMachinesKey: emuMachinesKey,
			},
		}, cpuArch)
		emulatedMachines := clusterConfig.GetEmulatedMachines()
		Expect(emulatedMachines).To(ConsistOf(result))
	},
		table.Entry("when set, GetEmulatedMachines should return the value", "", "q35, i440*", []string{"q35", "i440*"}),
		table.Entry("when unset, GetEmulatedMachines should return the defaults with amd64", "amd64", "", strings.Split(virtconfig.DefaultAMD64EmulatedMachines, ",")),
		table.Entry("when unset, GetEmulatedMachines should return the defaults with arm64", "arm64", "", strings.Split(virtconfig.DefaultAARCH64EmulatedMachines, ",")),
		table.Entry("when unset, GetEmulatedMachines should return the defaults with ppc64le", "ppc64le", "", strings.Split(virtconfig.DefaultPPC64LEEmulatedMachines, ",")),
	)

	table.DescribeTable(" when supportedGuestAgentVersions", func(value string, result []string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.SupportedGuestAgentVersionsKey: value},
		})
		supportedGuestAgentVersions := clusterConfig.GetSupportedAgentVersions()
		Expect(supportedGuestAgentVersions).To(ConsistOf(result))
	},
		table.Entry("when set, GetSupportedAgentVersions should return the value", "5.*,6.*", []string{"5.*", "6.*"}),
		table.Entry("when unset, GetSupportedAgentVersions should return the defaults", "", strings.Split(virtconfig.SupportedGuestAgentVersions, ",")),
	)

	It("Should return migration config values if specified as json", func() {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "10", "parallelMigrationsPerCluster": "20", "bandwidthPerMigration": "110Mi", "progressTimeout" : "5", "completionTimeoutPerGiB": "5", "unsafeMigrationOverride": "true", "allowAutoConverge": "true"}`},
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

	It("Should return migration config values if specified as yaml", func() {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `"parallelOutboundMigrationsPerNode" : "10"
"parallelMigrationsPerCluster": "20"
"bandwidthPerMigration": "110Mi"`},
		})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 20))
		Expect(result.BandwidthPerMigration.String()).To(Equal("110Mi"))
	})

	It("Should return defaults if parts of the config are not set", func() {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "10"}`},
		})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 5))
		Expect(result.BandwidthPerMigration.String()).To(Equal("0"))
	})

	It("Should update the config if a newer version is available", func() {
		clusterConfig, store, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "10"}`},
		})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))

		testutils.UpdateFakeClusterConfig(store, &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "9"}`},
		})
		Eventually(func() uint32 {
			return *clusterConfig.GetMigrationConfiguration().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 9))
	})

	It("Should stick with the last good config", func() {
		clusterConfig, store, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "10"}`},
		})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))

		testutils.UpdateFakeClusterConfig(store, &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "invalid"}`},
		})
		Consistently(func() uint32 {
			return *clusterConfig.GetMigrationConfiguration().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 10))
	})

	It("Should pick up the latest config once it is fixed and parsable again", func() {
		clusterConfig, store, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "10"}`},
		})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))

		invalidCfg := &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "invalid"}`},
		}
		testutils.UpdateFakeClusterConfig(store, invalidCfg)
		Consistently(func() uint32 {
			return *clusterConfig.GetMigrationConfiguration().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 10))

		validCfg := &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "9"}`},
		}
		testutils.UpdateFakeClusterConfig(store, validCfg)
		Consistently(func() uint32 {
			return *clusterConfig.GetMigrationConfiguration().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 9))
	})

	It("should return the default config if no config map exists", func() {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 2))
	})

	It("should contain a default machine type that is supported by default", func() {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})
		Expect(clusterConfig.GetMachineType()).To(testutils.SatisfyAnyRegexp(clusterConfig.GetEmulatedMachines()))
	})

	table.DescribeTable("SMBIOS values from kubevirt-config", func(value string, result *cmdv1.SMBios) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.SmbiosConfigKey: value},
		})
		smbios := clusterConfig.GetSMBIOS()

		smbiosJson, err := json.Marshal(smbios)
		Expect(err).ToNot(HaveOccurred())

		resultJson, err := json.Marshal(result)
		Expect(err).ToNot(HaveOccurred())

		Expect(string(smbiosJson)).To(BeEquivalentTo(string(resultJson)))
	},
		table.Entry("when values set, should equal to result", `{"Family":"test","Product":"test", "Manufacturer":"None"}`, &cmdv1.SMBios{Family: "test", Product: "test", Manufacturer: "None"}),
		table.Entry("When an invalid smbios value is set, should return default values", `{"invalid":"invalid"}`, &cmdv1.SMBios{Family: "KubeVirt", Product: "None", Manufacturer: "KubeVirt"}),
	)

	table.DescribeTable(" when SELinuxLauncherType", func(value string, result string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.SELinuxLauncherTypeKey: value},
		})
		selinuxLauncherType := clusterConfig.GetSELinuxLauncherType()
		Expect(selinuxLauncherType).To(Equal(result))
	},
		table.Entry("when set, GetSELinuxLauncherType should return the value", "spc_t", "spc_t"),
		table.Entry("when unset, GetSELinuxLauncherType should return the default", virtconfig.DefaultSELinuxLauncherType, virtconfig.DefaultSELinuxLauncherType),
	)

	table.DescribeTable(" when OVMFPath", func(cpuArch string, ovmfPathKey string, result string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfigWithCPUArch(&kubev1.ConfigMap{
			Data: map[string]string{
				virtconfig.OVMFPathKey: ovmfPathKey,
			},
		}, cpuArch)
		ovmfPath := clusterConfig.GetOVMFPath()
		Expect(ovmfPath).To(Equal(result))
	},
		table.Entry("when set, GetOVMFPath should return the value", "", "/usr/share/ovmf/x64", "/usr/share/ovmf/x64"),
		table.Entry("when unset, GetOVMFPath should return the default with amd64", "amd64", "", virtconfig.DefaultARCHOVMFPath),
		table.Entry("when unset, GetOVMFPath should return the default with arm64", "arm64", "", virtconfig.DefaultAARCH64OVMFPath),
		table.Entry("when unset, GetOVMFPath should return the default with ppc64le", "ppc64le", "", virtconfig.DefaultARCHOVMFPath),
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
		clusterConfig, _, _, kubeVirtInformer := testutils.NewFakeClusterConfigUsingKV(KV)
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

	It("Should still get GetPermittedHostDevices after invalid update", func() {
		expectedDevices := `{"pciHostDevices":[{"pciVendorSelector":"10DE:1EB8","resourceName":"nvidia.com/TU104GL_Tesla_T4"}],"mediatedDevices":[{"mdevNameSelector":"GRID T4-1Q","resourceName":"nvidia.com/GRID_T4-1Q"}]}`
		invalidPermittedHostDevicesConfig := "something wrong"
		clusterConfig, store, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})
		testutils.UpdateFakeClusterConfig(store, &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.PermittedHostDevicesKey: expectedDevices},
		})
		clusterConfig.GetPermittedHostDevices()
		testutils.UpdateFakeClusterConfig(store, &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.PermittedHostDevicesKey: invalidPermittedHostDevicesConfig},
		})
		hostdevs := clusterConfig.GetPermittedHostDevices()

		hostdevsJson, err := json.Marshal(hostdevs)
		Expect(err).ToNot(HaveOccurred())

		Expect(string(hostdevsJson)).To(BeEquivalentTo(expectedDevices))
	})

	table.DescribeTable("when kubevirt CR holds config", func(value v1.KubeVirtConfiguration, getPart func(*v1.KubeVirtConfiguration) interface{}, result string) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfigUsingKV(&v1.KubeVirt{
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
		table.Entry("when machineType set, should equal to result",
			v1.KubeVirtConfiguration{
				MachineType: "test",
			},
			func(c *v1.KubeVirtConfiguration) interface{} {
				return c.MachineType
			},
			`"test"`),
		table.Entry("when developerConfiguration set, should equal to result",
			v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					FeatureGates:           []string{"test1", "test2"},
					LessPVCSpaceToleration: 5,
					MemoryOvercommit:       150,
					NodeSelectors:          map[string]string{"test": "test"},
					UseEmulation:           true,
					CPUAllocationRatio:     25,
				},
			},
			func(c *v1.KubeVirtConfiguration) interface{} {
				return c.DeveloperConfiguration
			},
			`{"featureGates":["test1","test2"],"pvcTolerateLessSpaceUpToPercent":5,"minimumReservePVCBytes":131072,"memoryOvercommit":150,"nodeSelectors":{"test":"test"},"useEmulation":true,"cpuAllocationRatio":25,"logVerbosity":{"virtAPI":2,"virtController":2,"virtHandler":2,"virtLauncher":2,"virtOperator":2}}`),
		table.Entry("when networkConfiguration set, should equal to result",
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
			`{"defaultNetworkInterface":"test","permitSlirpInterface":true,"permitBridgeInterfaceOnPodNetwork":false}`),
	)

	It("should use configmap value over kubevirt configuration", func() {
		clusterConfig, cminformer, _, _ := testutils.NewFakeClusterConfigUsingKV(&v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: rand.String(10),
				Name:            "kubevirt",
				Namespace:       "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						UseEmulation: true,
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
			},
		})

		emulation := clusterConfig.IsUseEmulation()
		Expect(emulation).To(BeTrue())

		cminformer.GetStore().Add(&kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       "kubevirt",
				Name:            virtconfig.ConfigMapName,
				ResourceVersion: rand.String(10),
			},
			Data: map[string]string{virtconfig.UseEmulationKey: "false"},
		})

		emulation = clusterConfig.IsUseEmulation()
		Expect(emulation).To(BeFalse())
	})

	table.DescribeTable("when feature-gate", func(openFeatureGates string, isLiveMigrationEnabled, isSRIOVLiveMigrationEnabled bool) {
		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.FeatureGatesKey: openFeatureGates},
		})

		Expect(clusterConfig.LiveMigrationEnabled()).To(Equal(isLiveMigrationEnabled))
		Expect(clusterConfig.SRIOVLiveMigrationEnabled()).To(Equal(isSRIOVLiveMigrationEnabled))
	},
		table.Entry("LiveMigration and SRIOVLiveMigration are closed, both should be closed",
			"", false, false),
		table.Entry("LiveMigration and SRIOVLiveMigration are open, both should be open",
			virtconfig.LiveMigrationGate+","+virtconfig.SRIOVLiveMigrationGate, true, true),
		table.Entry("SRIOVLiveMigration is open, LiveMigration should be open",
			virtconfig.SRIOVLiveMigrationGate, true, true),
		table.Entry("LiveMigration is open, SRIOVLiveMigration should be close",
			virtconfig.LiveMigrationGate, true, false),
	)
})
