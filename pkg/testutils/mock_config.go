package testutils

import (
	v1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"

	KVv1 "kubevirt.io/client-go/api/v1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	configMapName = "kubevirt-config"
	HostDevicesConfigMapName = "kubevirt-host-device-plugin-config"
	namespace     = "kubevirt"
)

func NewFakeClusterConfig(cfgMap *v1.ConfigMap) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.SharedIndexInformer, cache.SharedIndexInformer, cache.SharedIndexInformer) {
	configMapInformer, _ := NewFakeInformerFor(&v1.ConfigMap{})
	hostDevConfigMapInformer, _ := NewFakeInformerFor(&v1.ConfigMap{})
	crdInformer, _ := NewFakeInformerFor(&extv1beta1.CustomResourceDefinition{})
	kubeVirtInformer, _ := NewFakeInformerFor(&KVv1.KubeVirt{})

	if cfgMap != nil {
		copy := copyByName(cfgMap, configMapName)
		configMapInformer.GetStore().Add(copy)
	}

	AddDataVolumeAPI(crdInformer)

	return virtconfig.NewClusterConfig(configMapInformer, crdInformer, kubeVirtInformer, hostDevConfigMapInformer, namespace), configMapInformer, crdInformer, kubeVirtInformer, hostDevConfigMapInformer
}

func NewFakeClusterConfigUsingKV(kv *KVv1.KubeVirt) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.SharedIndexInformer, cache.SharedIndexInformer) {
	kv.ResourceVersion = rand.String(10)
	configMapInformer, _ := NewFakeInformerFor(&v1.ConfigMap{})
	hostDevConfigMapInformer, _ := NewFakeInformerFor(&v1.ConfigMap{})
	crdInformer, _ := NewFakeInformerFor(&extv1beta1.CustomResourceDefinition{})
	kubeVirtInformer, _ := NewFakeInformerFor(&KVv1.KubeVirt{})

	kubeVirtInformer.GetStore().Add(kv)

	AddDataVolumeAPI(crdInformer)

	return virtconfig.NewClusterConfig(configMapInformer, crdInformer, kubeVirtInformer, hostDevConfigMapInformer, namespace), configMapInformer, crdInformer, kubeVirtInformer
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
	UpdateFakeClusterConfigByName(configMapInformer, cfgMap, configMapName)
}

func UpdateFakeClusterConfigByName(configMapInformer cache.SharedIndexInformer, cfgMap *v1.ConfigMap, name string) {
	copy := copyByName(cfgMap, name)
	configMapInformer.GetStore().Update(copy)
}

func UpdateFakeKubeVirtClusterConfig(kubeVirtInformer cache.SharedIndexInformer, kv *KVv1.KubeVirt) {
	copy := kv.DeepCopy()
	copy.ResourceVersion = rand.String(10)
	copy.Name = "kubevirt"
	copy.Namespace = "kubevirt"

	kubeVirtInformer.GetStore().Update(copy)
}

func copyByName(cfgMap *v1.ConfigMap, name string) *v1.ConfigMap {
	copy := cfgMap.DeepCopy()
	copy.ObjectMeta = v12.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		// Change the resource version, or the config will not be updated
		ResourceVersion: rand.String(10),
	}
	return copy
}
