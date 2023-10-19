package testutils

import (
	"runtime"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"

	k8score "k8s.io/api/core/v1"

	KVv1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	kvObjectNamespace = "kubevirt"
	kvObjectName      = "kubevirt"
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
	cfg, _ := virtconfig.NewClusterConfigWithCPUArch(crdInformer, kubeVirtInformer, kvObjectNamespace, CPUArch)
	return cfg, crdInformer, kubeVirtInformer
}

func NewFakeClusterConfigUsingKVConfig(config *KVv1.KubeVirtConfiguration) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.SharedIndexInformer) {
	kv := &KVv1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvObjectName,
			Namespace: kvObjectNamespace,
		},
		Spec: KVv1.KubeVirtSpec{
			Configuration: *config,
		},
		Status: KVv1.KubeVirtStatus{
			DefaultArchitecture: runtime.GOARCH,
			Phase:               "Deployed",
		},
	}
	return NewFakeClusterConfigUsingKV(kv)
}

func NewFakeContainerDiskSource() *KVv1.ContainerDiskSource {
	return &KVv1.ContainerDiskSource{
		Image:           "fake-image",
		ImagePullSecret: "fake-pull-secret",
		Path:            "/fake-path",
	}
}

func NewFakePersistentVolumeSource() *KVv1.PersistentVolumeClaimVolumeSource {
	return &KVv1.PersistentVolumeClaimVolumeSource{
		PersistentVolumeClaimVolumeSource: k8score.PersistentVolumeClaimVolumeSource{
			ClaimName: "fake-pvc",
		},
	}
}

func NewFakeMemoryDumpSource(name string) *KVv1.MemoryDumpVolumeSource {
	return &KVv1.MemoryDumpVolumeSource{
		PersistentVolumeClaimVolumeSource: KVv1.PersistentVolumeClaimVolumeSource{
			PersistentVolumeClaimVolumeSource: k8score.PersistentVolumeClaimVolumeSource{
				ClaimName: name,
			},
			Hotpluggable: true,
		},
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

func GetFakeKubeVirtClusterConfig(kubeVirtInformer cache.SharedIndexInformer) *KVv1.KubeVirt {
	obj, _, _ := kubeVirtInformer.GetStore().GetByKey(kvObjectNamespace + "/" + kvObjectName)
	return obj.(*KVv1.KubeVirt)

}

func UpdateFakeKubeVirtClusterConfig(kubeVirtInformer cache.SharedIndexInformer, kv *KVv1.KubeVirt) {
	clone := kv.DeepCopy()
	clone.ResourceVersion = rand.String(10)
	clone.Name = kvObjectName
	clone.Namespace = kvObjectNamespace
	clone.Status.Phase = "Deployed"

	kubeVirtInformer.GetStore().Update(clone)
}

func AddServiceMonitorAPI(crdInformer cache.SharedIndexInformer) {
	crdInformer.GetStore().Add(&extv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "service-monitors.monitoring.coreos.com",
		},
		Spec: extv1.CustomResourceDefinitionSpec{
			Names: extv1.CustomResourceDefinitionNames{
				Kind: "ServiceMonitor",
			},
		},
	})
}

func RemoveServiceMonitorAPI(crdInformer cache.SharedIndexInformer) {
	crdInformer.GetStore().Replace(nil, "")
}

func AddPrometheusRuleAPI(crdInformer cache.SharedIndexInformer) {
	crdInformer.GetStore().Add(&extv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prometheusrules.monitoring.coreos.com",
		},
		Spec: extv1.CustomResourceDefinitionSpec{
			Names: extv1.CustomResourceDefinitionNames{
				Kind: "PrometheusRule",
			},
		},
	})
}

func RemovePrometheusRuleAPI(crdInformer cache.SharedIndexInformer) {
	crdInformer.GetStore().Replace(nil, "")
}
