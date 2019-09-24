package virtconfig_test

import (
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"

	"kubevirt.io/client-go/log"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	testutils "kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("ConfigMap", func() {

	log.Log.SetIOWriter(GinkgoWriter)

	var stopChan chan struct{}

	BeforeEach(func() {
		stopChan = make(chan struct{})
	})

	AfterEach(func() {
		close(stopChan)
	})

	table.DescribeTable(" when useEmulation", func(value string, result bool) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.LessPVCSpaceTolerationKey: value},
		})
		Expect(clusterConfig.GetLessPVCSpaceToleration()).To(Equal(result))
	},
		table.Entry("is set, GetLessPVCSpaceToleration should return correct value", "5", 5),
		table.Entry("is unset, GetLessPVCSpaceToleration should return the default", "", virtconfig.DefaultLessPVCSpaceToleration),
		table.Entry("is invalid, GetLessPVCSpaceToleration should return the default", "-1", virtconfig.DefaultLessPVCSpaceToleration),
	)

	table.DescribeTable(" when defaultNetworkInterface", func(value string, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.NodeSelectorsKey: value},
		})
		Expect(clusterConfig.GetNodeSelectors()).To(Equal(result))
	},
		table.Entry("is set, GetNodeSelectors should return correct value", nodeSelectorsStr, nodeSelectors),
		table.Entry("is unset, GetNodeSelectors should return the default", "", nil),
		table.Entry("is invalid, GetNodeSelectors should return the default", "-1", nil),
	)

	table.DescribeTable(" when machineType", func(value string, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MachineTypeKey: value},
		})
		Expect(clusterConfig.GetMachineType()).To(Equal(result))
	},
		table.Entry("when set, GetMachineType should return the value", "pc-q35-3.0", "pc-q35-3.0"),
		table.Entry("when unset, GetMachineType should return the default", "", virtconfig.DefaultMachineType),
	)

	table.DescribeTable(" when cpuModel", func(value string, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.CpuModelKey: value},
		})
		Expect(clusterConfig.GetCPUModel()).To(Equal(result))
	},
		table.Entry("when set, GetCPUModel should return the value", "Haswell", "Haswell"),
		table.Entry("when unset, GetCPUModel should return empty string", "", ""),
	)

	table.DescribeTable(" when cpuRequest", func(value string, result string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.CpuRequestKey: value},
		})
		cpuRequest := clusterConfig.GetCPURequest()
		Expect(cpuRequest.String()).To(Equal(result))
	},
		table.Entry("when set, GetCPURequest should return the value", "400m", "400m"),
		table.Entry("when unset, GetCPURequest should return the default", "", virtconfig.DefaultCPURequest),
	)

	table.DescribeTable(" when memoryOvercommit", func(value string, result int) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MemoryOvercommitKey: value},
		})
		Expect(clusterConfig.GetMemoryOvercommit()).To(Equal(result))
	},
		table.Entry("when set, GetMemoryOvercommit should return the value", "150", 150),
		table.Entry("when unset, GetMemoryOvercommit should return the default", "", virtconfig.DefaultMemoryOvercommit),
	)

	table.DescribeTable(" when emulatedMachines", func(value string, result []string) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.EmulatedMachinesKey: value},
		})
		emulatedMachines := clusterConfig.GetEmulatedMachines()
		Expect(emulatedMachines).To(ConsistOf(result))
	},
		table.Entry("when set, GetEmulatedMachines should return the value", "q35, i440*", []string{"q35", "i440*"}),
		table.Entry("when unset, GetEmulatedMachines should return the defaults", "", strings.Split(virtconfig.DefaultEmulatedMachines, ",")),
	)

	It("Should return migration config values if specified as json", func() {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10, "parallelMigrationsPerCluster": 20, "bandwidthPerMigration": "110Mi", "progressTimeout" : 5, "completionTimeoutPerGiB": 5, "unsafeMigrationOverride": true, "allowAutoConverge": true}`},
		})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 20))
		Expect(result.BandwidthPerMigration.String()).To(Equal("110Mi"))
		Expect(*result.ProgressTimeout).To(BeNumerically("==", 5))
		Expect(*result.CompletionTimeoutPerGiB).To(BeNumerically("==", 5))
		Expect(result.UnsafeMigrationOverride).To(BeTrue())
		Expect(result.AllowAutoConverge).To(BeTrue())
	})

	It("Should return migration config values if specified as yaml", func() {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `"parallelOutboundMigrationsPerNode" : 10
"parallelMigrationsPerCluster": 20
"bandwidthPerMigration": "110Mi"`},
		})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 20))
		Expect(result.BandwidthPerMigration.String()).To(Equal("110Mi"))
	})

	It("Should return defaults if parts of the config are not set", func() {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 5))
		Expect(result.BandwidthPerMigration.String()).To(Equal("64Mi"))
	})

	It("Should update the config if a newer version is available", func() {
		clusterConfig, store, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))

		testutils.UpdateFakeClusterConfig(store, &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 9}`},
		})
		Eventually(func() uint32 {
			return *clusterConfig.GetMigrationConfig().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 9))
	})

	It("Should stick with the last good config", func() {
		clusterConfig, store, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))

		testutils.UpdateFakeClusterConfig(store, &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "invalid"}`},
		})
		Consistently(func() uint32 {
			return *clusterConfig.GetMigrationConfig().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 10))
	})

	It("Should pick up the latest config once it is fixed and parsable again", func() {
		clusterConfig, store, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))

		invalidCfg := &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "invalid"}`},
		}
		testutils.UpdateFakeClusterConfig(store, invalidCfg)
		Consistently(func() uint32 {
			return *clusterConfig.GetMigrationConfig().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 10))

		validCfg := &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 9}`},
		}
		testutils.UpdateFakeClusterConfig(store, validCfg)
		Consistently(func() uint32 {
			return *clusterConfig.GetMigrationConfig().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 9))
	})

	It("should return the default config if no config map exists", func() {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 2))
	})

	It("should contain a default machine type that is supported by default", func() {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})
		Expect(clusterConfig.GetMachineType()).To(testutils.SatisfyAnyRegexp(clusterConfig.GetEmulatedMachines()))
	})

	table.DescribeTable("SMBIOS values from kubevirt-config", func(value string, result cmdv1.SMBios) {
		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.SmbiosConfigKey: value},
		})
		smbios := clusterConfig.GetSMBIOS()

		smbiosJson, err := json.Marshal(smbios)
		Expect(err).ToNot(HaveOccurred())

		resultJson, err := json.Marshal(result)
		Expect(err).ToNot(HaveOccurred())

		Expect(string(smbiosJson)).To(BeEquivalentTo(string(resultJson)))
	},
		table.Entry("when values set, should equal to result", `{"Family":"test","Product":"test", "Manufacturer":"None"}`, cmdv1.SMBios{Family: "test", Product: "test", Manufacturer: "None"}),
		table.Entry("When an invalid smbios value is set, should return default values", `{"invalid":"invalid"}`, cmdv1.SMBios{Family: "KubeVirt", Product: "None", Manufacturer: "KubeVirt"}),
	)
})
