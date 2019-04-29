package testutils

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	configMapName = "kubevirt-config"
	namespace     = "kubevirt"
)

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
