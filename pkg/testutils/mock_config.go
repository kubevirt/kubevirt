package testutils

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
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
	return virtconfig.NewClusterConfig(cmInformer.GetStore(), "kubevirt")
}
