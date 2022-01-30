package testutils

import (
	"runtime"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"

	KVv1 "kubevirt.io/api/core/v1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	namespace = "kubevirt"
)

func NewFakeClusterConfigUsingKV(kv *KVv1.KubeVirt) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.SharedIndexInformer) {
	return NewFakeClusterConfigUsingKVWithCPUArch(kv, runtime.GOARCH)
}

func NewFakeClusterConfigUsingKVWithCPUArch(kv *KVv1.KubeVirt, CPUArch string) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.SharedIndexInformer) {
	kv.ResourceVersion = rand.String(10)
	kv.Status.Phase = "Deployed"
	crdInformer, _ := NewFakeInformerFor(&extv1.CustomResourceDefinition{})
	kubeVirtInformer, _ := NewFakeInformerFor(&KVv1.KubeVirt{})

	kubeVirtInformer.GetStore().Add(kv)

	AddDataVolumeAPI(crdInformer)

	return virtconfig.NewClusterConfigWithCPUArch(crdInformer, kubeVirtInformer, namespace, CPUArch), crdInformer, kubeVirtInformer
}

func NewFakeClusterConfigUsingKVConfig(config *KVv1.KubeVirtConfiguration) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.SharedIndexInformer) {
	kv := &KVv1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: "kubevirt",
		},
		Spec: KVv1.KubeVirtSpec{
			Configuration: *config,
		},
		Status: KVv1.KubeVirtStatus{
			Phase: "Deployed",
		},
	}
	return NewFakeClusterConfigUsingKV(kv)
}

func NewFakeContainerDiskSource() *KVv1.ContainerDiskSource {
	return &KVv1.ContainerDiskSource{
		Image:           "fake-image",
		ImagePullSecret: "fake-pull-secret",
		Path:            "fake-path",
	}
}

func RemoveDataVolumeAPI(crdInformer cache.SharedIndexInformer) {
	crdInformer.GetStore().Replace(nil, "")
}

func AddDataVolumeAPI(crdInformer cache.SharedIndexInformer) {
	crdInformer.GetStore().Add(&extv1.CustomResourceDefinition{
		Spec: extv1.CustomResourceDefinitionSpec{
			Names: extv1.CustomResourceDefinitionNames{
				Kind: "DataVolume",
			},
		},
	})
}

func UpdateFakeKubeVirtClusterConfig(kubeVirtInformer cache.SharedIndexInformer, kv *KVv1.KubeVirt) {
	clone := kv.DeepCopy()
	clone.ResourceVersion = rand.String(10)
	clone.Name = "kubevirt"
	clone.Namespace = "kubevirt"
	clone.Status.Phase = "Deployed"

	kubeVirtInformer.GetStore().Update(clone)
}
