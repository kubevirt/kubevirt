package testutils

import (
	v1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	configMapName = "kubevirt-config"
	namespace     = "kubevirt"
)

func NewFakeClusterConfig(cfgMap *v1.ConfigMap) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.SharedIndexInformer) {
	configMapInformer, _ := NewFakeInformerFor(&v1.ConfigMap{})
	crdInformer, _ := NewFakeInformerFor(&extv1beta1.CustomResourceDefinition{})

	copy := copy(cfgMap)
	configMapInformer.GetStore().Add(copy)

	AddDataVolumeAPI(crdInformer)

	return virtconfig.NewClusterConfig(configMapInformer, crdInformer, namespace), configMapInformer, crdInformer
}

func RemoveDataVolumeAPI(crdInformer cache.SharedIndexInformer) {
	crdInformer.GetStore().Replace(nil, "")
}

func AddDataVolumeAPI(crdInformer cache.SharedIndexInformer) {
	crdInformer.GetStore().Add(&extv1beta1.CustomResourceDefinition{
		Spec: extv1beta1.CustomResourceDefinitionSpec{
			Names: extv1beta1.CustomResourceDefinitionNames{
				Kind: "DataVolume",
			},
		},
	})
}

func UpdateFakeClusterConfig(configMapInformer cache.SharedIndexInformer, cfgMap *v1.ConfigMap) {
	copy := copy(cfgMap)
	configMapInformer.GetStore().Update(copy)
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
