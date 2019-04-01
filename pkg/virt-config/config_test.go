package virtconfig

import (
	. "github.com/onsi/ginkgo"
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

	It("Should return false if configmap is not present", func() {
		clusterConfig := MakeClusterConfig([]kubev1.ConfigMap{}, stopChan)
		result, err := clusterConfig.IsUseEmulation()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Should return false if configmap doesn't have useEmulation set", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kubevirt",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{},
		}
		clusterConfig := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result, err := clusterConfig.IsUseEmulation()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Should return true if useEmulation = true", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kubevirt",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{"debug.useEmulation": "true"},
		}
		clusterConfig := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result, err := clusterConfig.IsUseEmulation()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})

	It("Should return IfNotPresent if configmap doesn't have imagePullPolicy set", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kubevirt",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{},
		}
		clusterConfig := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result, err := clusterConfig.GetImagePullPolicy()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(kubev1.PullIfNotPresent))
	})

	It("Should return migration config values", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kubevirt",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10, "parallelMigrationsPerCluster": 20, "bandwidthPerMigration": "110Mi"}`},
		}
		clusterConfig := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 20))
		Expect(result.BandwidthPerMigration.String()).To(Equal("110Mi"))
	})

	It("Should return defaults if parts of the config are not set", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kubevirt",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{migrationsConfigKey: `{"parallelOutboundMigrationsPerNode" : 10}`},
		}
		clusterConfig := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result := clusterConfig.GetMigrationConfig()
		Expect(*result.ParallelOutboundMigrationsPerNode).To(BeNumerically("==", 10))
		Expect(*result.ParallelMigrationsPerCluster).To(BeNumerically("==", 5))
		Expect(result.BandwidthPerMigration.String()).To(Equal("64Mi"))
	})

	It("Should return Always if imagePullPolicy = Always", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kubevirt",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{imagePullPolicyKey: "Always"},
		}
		clusterConfig := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)
		result, err := clusterConfig.GetImagePullPolicy()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(kubev1.PullAlways))
	})

	It("Should return an error if imagePullPolicy is not valid", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kubevirt",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{imagePullPolicyKey: "IHaveNoStrongFeelingsOneWayOrTheOther"},
		}
		clusterConfig := MakeClusterConfig([]kubev1.ConfigMap{cfgMap}, stopChan)

		_, err := clusterConfig.GetImagePullPolicy()
		Expect(err).To(HaveOccurred())
	})
})

func MakeClusterConfig(configMaps []kubev1.ConfigMap, stopChan chan struct{}) *ClusterConfig {
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
	return NewClusterConfig(cmInformer.GetStore())
}
