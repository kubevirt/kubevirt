package testutils

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	configMapName = "kubevirt-config"
	namespace     = "kubevirt"
)

func MakeFakeClusterConfig(configMaps []v1.ConfigMap, stopChan chan struct{}) *virtconfig.ClusterConfig {
	cmListWatch := &cache.ListWatch{
		ListFunc: func(options v12.ListOptions) (runtime.Object, error) {
			return &v1.ConfigMapList{Items: configMaps}, nil
		},
		WatchFunc: func(options v12.ListOptions) (watch.Interface, error) {
			fakeWatch := watch.NewFake()
			for _, cfgMap := range configMaps {
				fakeWatch.Add(&cfgMap)
			}
			return fakeWatch, nil
		},
	}
	cmInformer := cache.NewSharedIndexInformer(cmListWatch, &v1.ConfigMap{}, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	go cmInformer.Run(stopChan)
	cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)
	return virtconfig.NewClusterConfig(cmInformer.GetStore(), namespace)
}

func NewFakeClusterConfig(cfgMap *v1.ConfigMap) (*virtconfig.ClusterConfig, cache.Store) {
	store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
	copy := copy(cfgMap)
	store.Add(copy)
	return virtconfig.NewClusterConfig(store, namespace), store
}

func UpdateFakeClusterConfig(store cache.Store, cfgMap *v1.ConfigMap) {
	copy := copy(cfgMap)
	store.Update(copy)
}

func copy(cfgMap *v1.ConfigMap) *v1.ConfigMap {
	copy := cfgMap.DeepCopy()
	copy.ObjectMeta = v12.ObjectMeta{
		Namespace: namespace,
		Name:      configMapName,
		// Change the resource version, or the config will not be updated
		ResourceVersion: rand.String(10),
	}
	return copy
}
