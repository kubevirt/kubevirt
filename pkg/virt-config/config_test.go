package virtconfig_test

import (
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	testutils "kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("ConfigMap", func() {

	log.Log.SetIOWriter(GinkgoWriter)

	var stopChan chan struct{}
	defaultCPURequest := resource.MustParse(virtconfig.DefaultCPURequest)

	BeforeEach(func() {
		stopChan = make(chan struct{})
	})

	AfterEach(func() {
		close(stopChan)
	})

	table.DescribeTable("when memBalloonStatsPeriod", func(value string, result uint32) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{"memBalloonStatsPeriod": value},
		})

		Expect(clusterConfig.GetMemBalloonStatsPeriod()).To(Equal(result))
	},
		table.Entry("is positive, GetMemBalloonStatsPeriod should return period", "3", uint32(3)),
		table.Entry("is negative, GetMemBalloonStatsPeriod should return 10", "-1", uint32(10)),
		table.Entry("when unset, GetMemBalloonStatsPeriod should return 10", "", uint32(10)),
		table.Entry("when invalid, GetMemBalloonStatsPeriod should return 10", "invalid", uint32(10)))

	table.DescribeTable(" when useEmulation", func(value string, result bool) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.LessPVCSpaceTolerationKey: value},
		})
		Expect(clusterConfig.GetLessPVCSpaceToleration()).To(Equal(result))
	},
		table.Entry("is set, GetLessPVCSpaceToleration should return correct value", "5", 5),
		table.Entry("is unset, GetLessPVCSpaceToleration should return the default", "", virtconfig.DefaultLessPVCSpaceToleration),
		table.Entry("is invalid, GetLessPVCSpaceToleration should return the default", "-1", virtconfig.DefaultLessPVCSpaceToleration),
	)

	table.DescribeTable(" when defaultNetworkInterface", func(value string, result string) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.NodeSelectorsKey: value},
		})
		Expect(clusterConfig.GetNodeSelectors()).To(Equal(result))
	},
		table.Entry("is set, GetNodeSelectors should return correct value", nodeSelectorsStr, nodeSelectors),
		table.Entry("is unset, GetNodeSelectors should return the default", "", nil),
		table.Entry("is invalid, GetNodeSelectors should return the default", "-1", nil),
	)

	table.DescribeTable(" when machineType", func(value string, result string) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MachineTypeKey: value},
		})
		Expect(clusterConfig.GetMachineType()).To(Equal(result))
	},
		table.Entry("when set, GetMachineType should return the value", "pc-q35-3.0", "pc-q35-3.0"),
		table.Entry("when unset, GetMachineType should return the default", "", virtconfig.DefaultMachineType),
	)

	table.DescribeTable(" when cpuModel", func(value string, result string) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.CPUModelKey: value},
		})
		Expect(clusterConfig.GetCPUModel()).To(Equal(result))
	},
		table.Entry("when set, GetCPUModel should return the value", "Haswell", "Haswell"),
		table.Entry("when unset, GetCPUModel should return empty string", "", ""),
	)

	table.DescribeTable(" when cpuRequest", func(value string, result string) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.CPURequestKey: value},
		})
		cpuRequest := clusterConfig.GetCPURequest()
		Expect(cpuRequest.String()).To(Equal(result))
	},
		table.Entry("when set, GetCPURequest should return the value", "400m", "400m"),
		table.Entry("when unset, GetCPURequest should return the default", "", virtconfig.DefaultCPURequest),
	)

	table.DescribeTable(" when memoryOvercommit", func(value string, result int) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MemoryOvercommitKey: value},
		})
		Expect(clusterConfig.GetMemoryOvercommit()).To(Equal(result))
	},
		table.Entry("when set, GetMemoryOvercommit should return the value", "150", 150),
		table.Entry("when unset, GetMemoryOvercommit should return the default", "", virtconfig.DefaultMemoryOvercommit),
	)

	table.DescribeTable(" when emulatedMachines", func(value string, result []string) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.EmulatedMachinesKey: value},
		})
		emulatedMachines := clusterConfig.GetEmulatedMachines()
		Expect(emulatedMachines).To(ConsistOf(result))
	},
		table.Entry("when set, GetEmulatedMachines should return the value", "q35, i440*", []string{"q35", "i440*"}),
		table.Entry("when unset, GetEmulatedMachines should return the defaults", "", strings.Split(virtconfig.DefaultEmulatedMachines, ",")),
	)

	table.DescribeTable(" when supportedGuestAgentVersions", func(value string, result []string) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.SupportedGuestAgentVersionsKey: value},
		})
		supportedGuestAgentVersions := clusterConfig.GetSupportedAgentVersions()
		Expect(supportedGuestAgentVersions).To(ConsistOf(result))
	},
		table.Entry("when set, GetSupportedAgentVersions should return the value", "5.*,6.*", []string{"5.*", "6.*"}),
		table.Entry("when unset, GetSupportedAgentVersions should return the defaults", "", strings.Split(virtconfig.SupportedGuestAgentVersions, ",")),
	)

	It("Should return migration config values if specified as json", func() {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "10"}`},
		})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 5))
		Expect(result.BandwidthPerMigration.String()).To(Equal("64Mi"))
	})

	It("Should update the config if a newer version is available", func() {
		clusterConfig, store, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, store, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, store, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})
		result := clusterConfig.GetMigrationConfiguration()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 2))
	})

	It("should contain a default machine type that is supported by default", func() {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})
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
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.SELinuxLauncherTypeKey: value},
		})
		selinuxLauncherType := clusterConfig.GetSELinuxLauncherType()
		Expect(selinuxLauncherType).To(Equal(result))
	},
		table.Entry("when set, GetSELinuxLauncherType should return the value", "spc_t", "spc_t"),
		table.Entry("when unset, GetSELinuxLauncherType should return the default", virtconfig.DefaultSELinuxLauncherType, virtconfig.DefaultSELinuxLauncherType),
	)

	table.DescribeTable(" when OVMFPath", func(value string, result string) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.OVMFPathKey: value},
		})
		ovmfPath := clusterConfig.GetOVMFPath()
		Expect(ovmfPath).To(Equal(result))
	},
		table.Entry("when set, GetOVMFPath should return the value", "/usr/share/ovmf/x64", "/usr/share/ovmf/x64"),
		table.Entry("when unset, GetOVMFPath should return the default", "", virtconfig.DefaultOVMFPath),
	)

	table.DescribeTable("when kubevirt CR holds config", func(value string, result v1.KubeVirtConfiguration) {
		clusterConfig, _, _, _, _ := testutils.NewFakeClusterConfigUsingKV(&v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: rand.String(10),
				Name:            "kubevirt",
				Namespace:       "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: result,
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
			},
		})

		kubevirtConfig := clusterConfig.GetConfig()
		kubevirtConfigJson, err := json.Marshal(kubevirtConfig)
		Expect(err).ToNot(HaveOccurred())

		// applying the new config to resultJson
		kubevirtConfig = clusterConfig.GetDefaultClusterConfig()
		resultJson, err := json.Marshal(result)
		err = json.Unmarshal(resultJson, kubevirtConfig)
		Expect(err).ToNot(HaveOccurred())
		resultJson, err = json.Marshal(kubevirtConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(kubevirtConfigJson)).To(BeEquivalentTo(string(resultJson)))
	},
		table.Entry("when values set, should equal to result",
			`{"machineType":"test","cpuModel":"test"}`,
			v1.KubeVirtConfiguration{CPURequest: &defaultCPURequest, MachineType: "test", CPUModel: "test"}),
		table.Entry("when networkConfigurations set in kubevirt.yaml, should equal to result",
			`{"network":{"defaultNetworkInterface":"test","permitSlirpInterface":"true","permitBridgeInterfaceOnPodNetwork":"false"}}`,
			v1.KubeVirtConfiguration{CPURequest: &defaultCPURequest, NetworkConfiguration: &v1.NetworkConfiguration{NetworkInterface: "test", PermitSlirpInterface: pointer.BoolPtr(true), PermitBridgeInterfaceOnPodNetwork: pointer.BoolPtr(false)}}),
		table.Entry("when developerConfigurations set in kubevirt.yaml, should equal to result",
			`{"dev":{"useEmulation":"true","featureGates":["test1","test2"],"nodeSelectors": {"test":"test"},"pvcTolerateLessSpaceUpToPercent":"5", "memoryOvercommit": "150"}}`,
			v1.KubeVirtConfiguration{CPURequest: &defaultCPURequest, DeveloperConfiguration: &v1.DeveloperConfiguration{UseEmulation: true, FeatureGates: []string{"test1", "test2"}, NodeSelectors: map[string]string{"test": "test"}, LessPVCSpaceToleration: 5, MemoryOvercommit: 150}}),
	)

	It("should use configmap value over kubevirt configuration", func() {
		clusterConfig, cminformer, _, _, _ := testutils.NewFakeClusterConfigUsingKV(&v1.KubeVirt{
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
})
