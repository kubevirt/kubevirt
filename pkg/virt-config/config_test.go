package virtconfig

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/log"
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
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "1234",
				Namespace:       "kubevirt",
				Name:            "kubevirt-config",
			},
			Data: map[string]string{"debug.useEmulation": value},
		}
		clusterConfig, _ := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		Expect(clusterConfig.IsUseEmulation()).To(Equal(result))
	},
		table.Entry("is true, it should return true", "true", true),
		table.Entry("is false, it should return false", "false", false),
		table.Entry("when unset, it should return false", "", false),
		table.Entry("when invalid, it should return the default", "invalid", false),
	)

	table.DescribeTable(" when imagePullPolicy", func(value string, result kubev1.PullPolicy) {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "1234",
				Namespace:       "kubevirt",
				Name:            "kubevirt-config",
			},
			Data: map[string]string{imagePullPolicyKey: value},
		}
		clusterConfig, _ := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		Expect(clusterConfig.GetImagePullPolicy()).To(Equal(result))
	},
		table.Entry("is PullAlways, it should return PullAlways", "Always", kubev1.PullAlways),
		table.Entry("is Never, it should return Never", "Never", kubev1.PullNever),
		table.Entry("is IsNotPresent, it should return IsNotPresent", "IfNotPresent", kubev1.PullIfNotPresent),
		table.Entry("when unset, it should return PullIfNotPresent", "", kubev1.PullIfNotPresent),
		table.Entry("when invalid, it should return the default", "invalid", kubev1.PullIfNotPresent),
	)

	table.DescribeTable(" when machineType", func(value string, result string) {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "1234",
				Namespace:       "kubevirt",
				Name:            "kubevirt-config",
			},
			Data: map[string]string{machineTypeKey: value},
		}
		clusterConfig, _ := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		Expect(clusterConfig.GetMachineType()).To(Equal(result))
	},
		table.Entry("when set, it should return the value", "pc-q35-3.0", "pc-q35-3.0"),
		table.Entry("when unset, it should return the default", "", DefaultMachineType),
	)

	It("Should return migration config values if specified as json", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "1234",
				Namespace:       "kubevirt",
				Name:            "kubevirt-config",
			},
			Data: map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10, "parallelMigrationsPerCluster": 20, "bandwidthPerMigration": "110Mi", "progressTimeout" : 5, "completionTimeoutPerGiB": 5}`},
		}
		clusterConfig, _ := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 20))
		Expect(result.BandwidthPerMigration.String()).To(Equal("110Mi"))
		Expect(*result.ProgressTimeout).To(BeNumerically("==", 5))
		Expect(*result.CompletionTimeoutPerGiB).To(BeNumerically("==", 5))
	})

	It("Should return migration config values if specified as yaml", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "1234",
				Namespace:       "kubevirt",
				Name:            "kubevirt-config",
			},
			Data: map[string]string{migrationsConfigKey: `"parallelOutboundMigrationsPerNode" : 10
"parallelMigrationsPerCluster": 20
"bandwidthPerMigration": "110Mi"`},
		}
		clusterConfig, _ := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 20))
		Expect(result.BandwidthPerMigration.String()).To(Equal("110Mi"))
	})

	It("Should return defaults if parts of the config are not set", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "1234",
				Namespace:       "kubevirt",
				Name:            "kubevirt-config",
			},
			Data: map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		}
		clusterConfig, _ := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 5))
		Expect(result.BandwidthPerMigration.String()).To(Equal("64Mi"))
	})

	It("Should update the config if a newer version is available", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "1234",
				Namespace:       "kubevirt",
				Name:            "kubevirt-config",
			},
			Data: map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		}
		clusterConfig, store := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))

		newCfg := cfgMap.DeepCopy()
		newCfg.ResourceVersion = "12345"
		newCfg.Data = map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 9}`}
		store.Add(newCfg)
		Eventually(func() uint32 {
			return *clusterConfig.GetMigrationConfig().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 9))
	})

	It("Should stick with the last good config", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "1234",
				Namespace:       "kubevirt",
				Name:            "kubevirt-config",
			},
			Data: map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		}
		clusterConfig, store := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))

		newCfg := cfgMap.DeepCopy()
		newCfg.ResourceVersion = "12345"
		newCfg.Data = map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "invalid"}`}
		store.Add(newCfg)
		Consistently(func() uint32 {
			return *clusterConfig.GetMigrationConfig().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 10))
	})

	It("Should pick up the latest config once it is fixed and parsable again", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "1234",
				Namespace:       "kubevirt",
				Name:            "kubevirt-config",
			},
			Data: map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		}
		clusterConfig, store := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))

		invalidCfg := cfgMap.DeepCopy()
		invalidCfg.ResourceVersion = "12345"
		invalidCfg.Data = map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : "invalid"}`}
		store.Add(invalidCfg)
		Consistently(func() uint32 {
			return *clusterConfig.GetMigrationConfig().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 10))

		validCfg := cfgMap.DeepCopy()
		validCfg.ResourceVersion = "123456"
		validCfg.Data = map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 9}`}
		store.Add(validCfg)
		Consistently(func() uint32 {
			return *clusterConfig.GetMigrationConfig().ParallelOutboundMigrationsPerNode
		}).Should(BeNumerically("==", 9))
	})

	It("should return the default config if no config map exists", func() {
		clusterConfig, _ := MakeClusterConfig([]kubev1.ConfigMap{}, stopChan)
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 2))
	})
})

func MakeClusterConfig(configMaps []kubev1.ConfigMap, stopChan chan struct{}) (*ClusterConfig, cache.Store) {
	cmListWatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return &kubev1.ConfigMapList{Items: configMaps}, nil
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			fakeWatch := watch.NewFake()
			for _, cfgMap := range configMaps {
				fakeWatch.Add(&cfgMap)
			}
			return fakeWatch, nil
		},
	}
	cmInformer := cache.NewSharedIndexInformer(cmListWatch, &kubev1.ConfigMap{}, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	go cmInformer.Run(stopChan)
	cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)
	return NewClusterConfig(cmInformer.GetStore(), "kubevirt"), cmInformer.GetStore()
}
