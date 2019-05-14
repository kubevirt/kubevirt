package virtconfig_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/log"
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
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{"debug.useEmulation": value},
		})
		Expect(clusterConfig.IsUseEmulation()).To(Equal(result))
	},
		table.Entry("is true, it should return true", "true", true),
		table.Entry("is false, it should return false", "false", false),
		table.Entry("when unset, it should return false", "", false),
		table.Entry("when invalid, it should return the default", "invalid", false),
	)

	table.DescribeTable(" when imagePullPolicy", func(value string, result kubev1.PullPolicy) {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.ImagePullPolicyKey: value},
		})
		Expect(clusterConfig.GetImagePullPolicy()).To(Equal(result))
	},
		table.Entry("is PullAlways, it should return PullAlways", "Always", kubev1.PullAlways),
		table.Entry("is Never, it should return Never", "Never", kubev1.PullNever),
		table.Entry("is IsNotPresent, it should return IsNotPresent", "IfNotPresent", kubev1.PullIfNotPresent),
		table.Entry("when unset, it should return PullIfNotPresent", "", kubev1.PullIfNotPresent),
		table.Entry("when invalid, it should return the default", "invalid", kubev1.PullIfNotPresent),
	)

	table.DescribeTable(" when lessPVCSpaceToleration", func(value string, result int) {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.LessPVCSpaceTolerationKey: value},
		})
		Expect(clusterConfig.GetLessPVCSpaceToleration()).To(Equal(result))
	},
		table.Entry("is set, it should return correct value", "5", 5),
		table.Entry("is unset, it should return the default", "", virtconfig.DefaultLessPVCSpaceToleration),
		table.Entry("is invalid, it should return the default", "-1", virtconfig.DefaultLessPVCSpaceToleration),
	)

	nodeSelectorsStr := "kubernetes.io/hostname=node02\nnode-role.kubernetes.io/compute=true\n"
	nodeSelectors := map[string]string{
		"kubernetes.io/hostname":          "node02",
		"node-role.kubernetes.io/compute": "true",
	}
	table.DescribeTable(" when nodeSelectors", func(value string, result map[string]string) {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.NodeSelectorsKey: value},
		})
		Expect(clusterConfig.GetNodeSelectors()).To(Equal(result))
	},
		table.Entry("is set, it should return correct value", nodeSelectorsStr, nodeSelectors),
		table.Entry("is unset, it should return the default", "", nil),
		table.Entry("is invalid, it should return the default", "-1", nil),
	)

	table.DescribeTable(" when machineType", func(value string, result string) {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MachineTypeKey: value},
		})
		Expect(clusterConfig.GetMachineType()).To(Equal(result))
	},
		table.Entry("when set, it should return the value", "pc-q35-3.0", "pc-q35-3.0"),
		table.Entry("when unset, it should return the default", "", virtconfig.DefaultMachineType),
	)

	table.DescribeTable(" when cpuModel", func(value string, result string) {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.CpuModelKey: value},
		})
		Expect(clusterConfig.GetCPUModel()).To(Equal(result))
	},
		table.Entry("when set, it should return the value", "Haswell", "Haswell"),
		table.Entry("when unset, it should return empty string", "", ""),
	)

	table.DescribeTable(" when cpuRequest", func(value string, result string) {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.CpuRequestKey: value},
		})
		cpuRequest := clusterConfig.GetCPURequest()
		Expect(cpuRequest.String()).To(Equal(result))
	},
		table.Entry("when set, it should return the value", "400m", "400m"),
		table.Entry("when unset, it should return the default", "", virtconfig.DefaultCPURequest),
	)

	table.DescribeTable(" when memoryRequest", func(value string, result string) {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MemoryRequestKey: value},
		})
		memoryRequest := clusterConfig.GetMemoryRequest()
		Expect(memoryRequest.String()).To(Equal(result))
	},
		table.Entry("when set, it should return the value", "512Mi", "512Mi"),
		table.Entry("when unset, it should return the default", "", virtconfig.DefaultMemoryRequest),
	)

	table.DescribeTable(" when emulatedMachines", func(value string, result []string) {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.EmulatedMachinesKey: value},
		})
		emulatedMachines := clusterConfig.GetEmulatedMachines()
		Expect(emulatedMachines).To(ConsistOf(result))
	},
		table.Entry("when set, it should return the value", "q35, i440*", []string{"q35", "i440*"}),
		table.Entry("when unset, it should return the defaults", "", strings.Split(virtconfig.DefaultEmulatedMachines, ",")),
	)

	It("Should return migration config values if specified as json", func() {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10, "parallelMigrationsPerCluster": 20, "bandwidthPerMigration": "110Mi", "progressTimeout" : 5, "completionTimeoutPerGiB": 5, "unsafeMigrationOverride": true}`},
		})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 20))
		Expect(result.BandwidthPerMigration.String()).To(Equal("110Mi"))
		Expect(*result.ProgressTimeout).To(BeNumerically("==", 5))
		Expect(*result.CompletionTimeoutPerGiB).To(BeNumerically("==", 5))
		Expect(result.UnsafeMigrationOverride).To(Equal(true))
	})

	It("Should return migration config values if specified as yaml", func() {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
			Data: map[string]string{virtconfig.MigrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 5))
		Expect(result.BandwidthPerMigration.String()).To(Equal("64Mi"))
	})

	It("Should update the config if a newer version is available", func() {
		clusterConfig, store := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, store := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, store := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{
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
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 2))
	})

	It("should contain a default machine type that is supported by default", func() {
		clusterConfig, _ := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})
		Expect(clusterConfig.GetMachineType()).To(testutils.SatisfyAnyRegexp(clusterConfig.GetEmulatedMachines()))
	})
})
