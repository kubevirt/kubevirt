/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package controller

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	vsv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	secv1 "github.com/openshift/api/security/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	v1beta12 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	kubev1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	OperatorLabel = kubev1.ManagedByLabel + "=" + kubev1.ManagedByLabelOperatorValue
)

type newSharedInformer func() cache.SharedIndexInformer

type KubeInformerFactory interface {
	// Starts any informers that have not been started yet
	// This function is thread safe and idempotent
	Start(stopCh <-chan struct{})

	// Watches for vmi objects
	VMI() cache.SharedIndexInformer

	// Watches for VirtualMachineInstanceReplicaSet objects
	VMIReplicaSet() cache.SharedIndexInformer

	// Watches for VirtualMachineInstancePreset objects
	VirtualMachinePreset() cache.SharedIndexInformer

	// Watches for pods related only to kubevirt
	KubeVirtPod() cache.SharedIndexInformer

	// Watches for nodes
	KubeVirtNode() cache.SharedIndexInformer

	// VirtualMachine handles the VMIs that are stopped or not running
	VirtualMachine() cache.SharedIndexInformer

	// Watches VirtualMachineInstanceMigration objects
	VirtualMachineInstanceMigration() cache.SharedIndexInformer

	// Watches VirtualMachineSnapshot objects
	VirtualMachineSnapshot() cache.SharedIndexInformer

	// Watches VirtualMachineSnapshot objects
	VirtualMachineSnapshotContent() cache.SharedIndexInformer

	// Watches VirtualMachineRestore objects
	VirtualMachineRestore() cache.SharedIndexInformer

	// Watches for k8s extensions api configmap
	ApiAuthConfigMap() cache.SharedIndexInformer

	// Watches for the kubevirt CA config map
	KubeVirtCAConfigMap() cache.SharedIndexInformer

	// ConfigMaps which are managed by the operator
	OperatorConfigMap() cache.SharedIndexInformer

	// Watches for ConfigMap objects
	ConfigMap() cache.SharedIndexInformer

	// Watches for PersistentVolumeClaim objects
	PersistentVolumeClaim() cache.SharedIndexInformer

	// Watches for LimitRange objects
	LimitRanges() cache.SharedIndexInformer

	// Watches for CDI DataVolume objects
	DataVolume() cache.SharedIndexInformer

	// Fake CDI DataVolume informer used when feature gate is disabled
	DummyDataVolume() cache.SharedIndexInformer

	// CRD
	CRD() cache.SharedIndexInformer

	// Wachtes for KubeVirt objects
	KubeVirt() cache.SharedIndexInformer

	// Service Accounts
	OperatorServiceAccount() cache.SharedIndexInformer

	// ClusterRole
	OperatorClusterRole() cache.SharedIndexInformer

	// ClusterRoleBinding
	OperatorClusterRoleBinding() cache.SharedIndexInformer

	// Roles
	OperatorRole() cache.SharedIndexInformer

	// RoleBinding
	OperatorRoleBinding() cache.SharedIndexInformer

	// CRD
	OperatorCRD() cache.SharedIndexInformer

	// Service
	OperatorService() cache.SharedIndexInformer

	// DaemonSet
	OperatorDaemonSet() cache.SharedIndexInformer

	// Deployment
	OperatorDeployment() cache.SharedIndexInformer

	// SecurityContextConstraints
	OperatorSCC() cache.SharedIndexInformer

	// Fake SecurityContextConstraints informer used when not on openshift
	DummyOperatorSCC() cache.SharedIndexInformer

	// ConfigMaps for operator install strategies
	OperatorInstallStrategyConfigMaps() cache.SharedIndexInformer

	// Jobs for dumping operator install strategies
	OperatorInstallStrategyJob() cache.SharedIndexInformer

	// KubeVirt infrastructure pods
	OperatorPod() cache.SharedIndexInformer

	// Webhooks created/managed by virt operator
	OperatorValidationWebhook() cache.SharedIndexInformer

	// Webhooks created/managed by virt operator
	OperatorMutatingWebhook() cache.SharedIndexInformer

	// APIServices created/managed by virt operator
	OperatorAPIService() cache.SharedIndexInformer

	// PodDisruptionBudgets created/managed by virt operator
	OperatorPodDisruptionBudget() cache.SharedIndexInformer

	// ServiceMonitors created/managed by virt operator
	OperatorServiceMonitor() cache.SharedIndexInformer

	// Managed secrets which hold data like certificates
	Secrets() cache.SharedIndexInformer

	// Fake ServiceMonitor informer used when Prometheus is not installed
	DummyOperatorServiceMonitor() cache.SharedIndexInformer

	// The namespace where kubevirt is deployed in
	Namespace() cache.SharedIndexInformer

	// PrometheusRules created/managed by virt operator
	OperatorPrometheusRule() cache.SharedIndexInformer

	// Fake PrometheusRule informer used when Prometheus not installed
	DummyOperatorPrometheusRule() cache.SharedIndexInformer

	// PVC StorageClasses
	StorageClass() cache.SharedIndexInformer

	// Pod returns an informer for ALL Pods in the system
	Pod() cache.SharedIndexInformer

	K8SInformerFactory() informers.SharedInformerFactory
}

type kubeInformerFactory struct {
	restClient       *rest.RESTClient
	clientSet        kubecli.KubevirtClient
	aggregatorClient aggregatorclient.Interface
	lock             sync.Mutex
	defaultResync    time.Duration

	informers         map[string]cache.SharedIndexInformer
	startedInformers  map[string]bool
	kubevirtNamespace string
	k8sInformers      informers.SharedInformerFactory
}

func NewKubeInformerFactory(restClient *rest.RESTClient, clientSet kubecli.KubevirtClient, aggregatorClient aggregatorclient.Interface, kubevirtNamespace string) KubeInformerFactory {
	return &kubeInformerFactory{
		restClient:       restClient,
		clientSet:        clientSet,
		aggregatorClient: aggregatorClient,
		// Resulting resync period will be between 12 and 24 hours, like the default for k8s
		defaultResync:     resyncPeriod(12 * time.Hour),
		informers:         make(map[string]cache.SharedIndexInformer),
		startedInformers:  make(map[string]bool),
		kubevirtNamespace: kubevirtNamespace,
		k8sInformers:      informers.NewSharedInformerFactoryWithOptions(clientSet, 0),
	}
}

// Start can be called from multiple controllers in different go routines safely.
// Only informers that have not started are triggered by this function.
// Multiple calls to this function are idempotent.
func (f *kubeInformerFactory) Start(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for name, informer := range f.informers {
		if f.startedInformers[name] {
			// skip informers that have already started.
			log.Log.Infof("SKIPPING informer %s", name)
			continue
		}
		log.Log.Infof("STARTING informer %s", name)
		go informer.Run(stopCh)
		f.startedInformers[name] = true
	}
	f.k8sInformers.Start(stopCh)
}

// internal function used to retrieve an already created informer
// or create a new informer if one does not already exist.
// Thread safe
func (f *kubeInformerFactory) getInformer(key string, newFunc newSharedInformer) cache.SharedIndexInformer {
	f.lock.Lock()
	defer f.lock.Unlock()

	informer, exists := f.informers[key]
	if exists {
		return informer
	}
	informer = newFunc()
	f.informers[key] = informer

	return informer
}

func (f *kubeInformerFactory) Namespace() cache.SharedIndexInformer {
	return f.getInformer("namespaceInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "namespaces", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(
			lw,
			&corev1.Namespace{},
			f.defaultResync,
			cache.Indexers{
				"namespace_name": func(obj interface{}) ([]string, error) {
					return []string{obj.(*corev1.Namespace).GetName()}, nil
				},
			},
		)
	})
}

func (f *kubeInformerFactory) VMI() cache.SharedIndexInformer {
	return f.getInformer("vmiInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.VirtualMachineInstance{}, f.defaultResync, cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
			"node": func(obj interface{}) (strings []string, e error) {
				return []string{obj.(*kubev1.VirtualMachineInstance).Status.NodeName}, nil
			},
		})
	})
}

func (f *kubeInformerFactory) VMIReplicaSet() cache.SharedIndexInformer {
	return f.getInformer("vmirsInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "virtualmachineinstancereplicasets", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.VirtualMachineInstanceReplicaSet{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) VirtualMachinePreset() cache.SharedIndexInformer {
	return f.getInformer("vmiPresetInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "virtualmachineinstancepresets", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.VirtualMachineInstancePreset{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) VirtualMachineInstanceMigration() cache.SharedIndexInformer {
	return f.getInformer("vmimInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "virtualmachineinstancemigrations", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.VirtualMachineInstanceMigration{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) KubeVirtPod() cache.SharedIndexInformer {
	return f.getInformer("kubeVirtPodInformer", func() cache.SharedIndexInformer {
		// Watch all pods with the kubevirt app label
		labelSelector, err := labels.Parse(kubev1.AppLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "pods", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.Pod{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) KubeVirtNode() cache.SharedIndexInformer {
	return f.getInformer("kubeVirtNodeInformer", func() cache.SharedIndexInformer {
		lw := NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "nodes", k8sv1.NamespaceAll, fields.Everything(), labels.Everything())
		return cache.NewSharedIndexInformer(lw, &k8sv1.Node{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) VirtualMachine() cache.SharedIndexInformer {
	return f.getInformer("vmInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "virtualmachines", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.VirtualMachine{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) VirtualMachineSnapshot() cache.SharedIndexInformer {
	return f.getInformer("vmSnapshotInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().SnapshotV1alpha1().RESTClient(), "virtualmachinesnapshots", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &snapshotv1.VirtualMachineSnapshot{}, f.defaultResync, cache.Indexers{
			"vm": func(obj interface{}) ([]string, error) {
				vms, ok := obj.(*snapshotv1.VirtualMachineSnapshot)
				if !ok {
					return nil, fmt.Errorf("unexpected object")
				}

				if vms.Spec.Source.APIGroup != nil {
					gv, err := schema.ParseGroupVersion(*vms.Spec.Source.APIGroup)
					if err != nil {
						return nil, err
					}

					if gv.Group == kubev1.GroupName &&
						vms.Spec.Source.Kind == "VirtualMachine" {
						return []string{vms.Spec.Source.Name}, nil
					}
				}

				return nil, nil
			},
		})
	})
}

func (f *kubeInformerFactory) VirtualMachineSnapshotContent() cache.SharedIndexInformer {
	return f.getInformer("vmSnapshotContentInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().SnapshotV1alpha1().RESTClient(), "virtualmachinesnapshotcontents", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &snapshotv1.VirtualMachineSnapshotContent{}, f.defaultResync, cache.Indexers{
			"volumeSnapshot": func(obj interface{}) ([]string, error) {
				vmsc, ok := obj.(*snapshotv1.VirtualMachineSnapshotContent)
				if !ok {
					return nil, fmt.Errorf("unexpected object")
				}
				var volumeSnapshots []string
				for _, v := range vmsc.Spec.VolumeBackups {
					if v.VolumeSnapshotName != nil {
						volumeSnapshots = append(volumeSnapshots, *v.VolumeSnapshotName)
					}
				}
				return volumeSnapshots, nil
			},
		})
	})
}

func (f *kubeInformerFactory) VirtualMachineRestore() cache.SharedIndexInformer {
	return f.getInformer("vmRestoreInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().SnapshotV1alpha1().RESTClient(), "virtualmachinerestores", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &snapshotv1.VirtualMachineRestore{}, f.defaultResync, cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
			"vm": func(obj interface{}) ([]string, error) {
				vmr, ok := obj.(*snapshotv1.VirtualMachineRestore)
				if !ok {
					return nil, fmt.Errorf("unexpected object")
				}

				if vmr.Spec.Target.APIGroup != nil {
					gv, err := schema.ParseGroupVersion(*vmr.Spec.Target.APIGroup)
					if err != nil {
						return nil, err
					}

					if gv.Group == kubev1.GroupName &&
						vmr.Spec.Target.Kind == "VirtualMachine" {
						return []string{vmr.Spec.Target.Name}, nil
					}
				}

				return nil, nil
			},
		})
	})
}

func (f *kubeInformerFactory) DataVolume() cache.SharedIndexInformer {
	return f.getInformer("dataVolumeInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.CdiClient().CdiV1alpha1().RESTClient(), "datavolumes", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &cdiv1.DataVolume{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) DummyDataVolume() cache.SharedIndexInformer {
	return f.getInformer("fakeDataVolumeInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		return informer
	})
}

func (f *kubeInformerFactory) ApiAuthConfigMap() cache.SharedIndexInformer {
	return f.getInformer("extensionsConfigMapInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CoreV1().RESTClient()
		fieldSelector := fields.OneTermEqualSelector("metadata.name", "extension-apiserver-authentication")
		lw := cache.NewListWatchFromClient(restClient, "configmaps", metav1.NamespaceSystem, fieldSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.ConfigMap{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) KubeVirtCAConfigMap() cache.SharedIndexInformer {
	return f.getInformer("extensionsKubeVirtCAConfigMapInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CoreV1().RESTClient()
		fieldSelector := fields.OneTermEqualSelector("metadata.name", "kubevirt-ca")
		lw := cache.NewListWatchFromClient(restClient, "configmaps", f.kubevirtNamespace, fieldSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.ConfigMap{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) ConfigMap() cache.SharedIndexInformer {
	return f.getInformer("configMapInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CoreV1().RESTClient()
		fieldSelector := fields.OneTermEqualSelector("metadata.name", "kubevirt-config")
		lw := cache.NewListWatchFromClient(restClient, "configmaps", f.kubevirtNamespace, fieldSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.ConfigMap{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) PersistentVolumeClaim() cache.SharedIndexInformer {
	return f.getInformer("persistentVolumeClaimInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CoreV1().RESTClient()
		lw := cache.NewListWatchFromClient(restClient, "persistentvolumeclaims", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &k8sv1.PersistentVolumeClaim{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) LimitRanges() cache.SharedIndexInformer {
	return f.getInformer("limitrangeInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CoreV1().RESTClient()
		lw := cache.NewListWatchFromClient(restClient, "limitranges", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &k8sv1.LimitRange{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) KubeVirt() cache.SharedIndexInformer {
	return f.getInformer("kubeVirtInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "kubevirts", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.KubeVirt{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

// resyncPeriod computes the time interval a shared informer waits before resyncing with the api server
func resyncPeriod(minResyncPeriod time.Duration) time.Duration {
	factor := rand.Float64() + 1
	return time.Duration(float64(minResyncPeriod.Nanoseconds()) * factor)
}

func (f *kubeInformerFactory) OperatorServiceAccount() cache.SharedIndexInformer {
	return f.getInformer("OperatorServiceAccountInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "serviceaccounts", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.ServiceAccount{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorConfigMap() cache.SharedIndexInformer {
	// filter out install strategies
	return f.getInformer("OperatorConfigMapInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(fmt.Sprintf("!%s, %s=%s", kubev1.InstallStrategyLabel, kubev1.ManagedByLabel, kubev1.ManagedByLabelOperatorValue))
		if err != nil {
			panic(err)
		}
		restClient := f.clientSet.CoreV1().RESTClient()
		lw := NewListWatchFromClient(restClient, "configmaps", f.kubevirtNamespace, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.ConfigMap{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorClusterRole() cache.SharedIndexInformer {
	return f.getInformer("OperatorClusterRoleInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.RbacV1().RESTClient(), "clusterroles", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &rbacv1.ClusterRole{}, f.defaultResync, cache.Indexers{})
	})
}
func (f *kubeInformerFactory) OperatorClusterRoleBinding() cache.SharedIndexInformer {
	return f.getInformer("OperatorClusterRoleBindingInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.RbacV1().RESTClient(), "clusterrolebindings", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &rbacv1.ClusterRoleBinding{}, f.defaultResync, cache.Indexers{})
	})
}
func (f *kubeInformerFactory) OperatorRole() cache.SharedIndexInformer {
	return f.getInformer("OperatorRoleInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.RbacV1().RESTClient(), "roles", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &rbacv1.Role{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorRoleBinding() cache.SharedIndexInformer {
	return f.getInformer("OperatorRoleBindingInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.RbacV1().RESTClient(), "rolebindings", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &rbacv1.RoleBinding{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorCRD() cache.SharedIndexInformer {
	return f.getInformer("OperatorCRDInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		ext, err := extclient.NewForConfig(f.clientSet.Config())
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(ext.ApiextensionsV1beta1().RESTClient(), "customresourcedefinitions", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &extv1beta1.CustomResourceDefinition{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) OperatorService() cache.SharedIndexInformer {
	return f.getInformer("OperatorServiceInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "services", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.Service{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorDeployment() cache.SharedIndexInformer {
	return f.getInformer("OperatorDeploymentInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.AppsV1().RESTClient(), "deployments", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &appsv1.Deployment{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorDaemonSet() cache.SharedIndexInformer {
	return f.getInformer("OperatorDaemonSetInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.AppsV1().RESTClient(), "daemonsets", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &appsv1.DaemonSet{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorSCC() cache.SharedIndexInformer {
	return f.getInformer("OperatorSCC", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.SecClient().RESTClient(), "securitycontextconstraints", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &secv1.SecurityContextConstraints{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) DummyOperatorSCC() cache.SharedIndexInformer {
	return f.getInformer("FakeOperatorSCC", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&secv1.SecurityContextConstraints{})
		return informer
	})
}

func (f *kubeInformerFactory) OperatorInstallStrategyConfigMaps() cache.SharedIndexInformer {
	return f.getInformer("installStrategyConfigMapInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(kubev1.InstallStrategyLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "configmaps", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.ConfigMap{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorInstallStrategyJob() cache.SharedIndexInformer {
	return f.getInformer("installStrategyJobsInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(kubev1.InstallStrategyLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.BatchV1().RESTClient(), "jobs", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &batchv1.Job{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorPod() cache.SharedIndexInformer {
	return f.getInformer("operatorPodsInformer", func() cache.SharedIndexInformer {
		// Watch all kubevirt infrastructure pods with the operator label
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "pods", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.Pod{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorValidationWebhook() cache.SharedIndexInformer {
	return f.getInformer("operatorValidatingWebhookInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.AdmissionregistrationV1beta1().RESTClient(), "validatingwebhookconfigurations", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorMutatingWebhook() cache.SharedIndexInformer {
	return f.getInformer("operatorMutatingWebhookInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.AdmissionregistrationV1beta1().RESTClient(), "mutatingwebhookconfigurations", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &admissionregistrationv1beta1.MutatingWebhookConfiguration{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) Secrets() cache.SharedIndexInformer {
	return f.getInformer("secretsInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		restClient := f.clientSet.CoreV1().RESTClient()
		lw := NewListWatchFromClient(restClient, "secrets", f.kubevirtNamespace, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &corev1.Secret{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorAPIService() cache.SharedIndexInformer {
	return f.getInformer("operatorAPIServiceInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.aggregatorClient.ApiregistrationV1beta1().RESTClient(), "apiservices", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &v1beta12.APIService{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorPodDisruptionBudget() cache.SharedIndexInformer {
	return f.getInformer("operatorPodDisruptionBudgetInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.PolicyV1beta1().RESTClient(), "poddisruptionbudgets", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &v1beta1.PodDisruptionBudget{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorServiceMonitor() cache.SharedIndexInformer {
	return f.getInformer("operatorServiceMonitorInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.PrometheusClient().MonitoringV1().RESTClient(), "servicemonitors", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &promv1.ServiceMonitor{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) DummyOperatorServiceMonitor() cache.SharedIndexInformer {
	return f.getInformer("FakeOperatorServiceMonitor", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&promv1.ServiceMonitor{})
		return informer
	})
}

func (f *kubeInformerFactory) K8SInformerFactory() informers.SharedInformerFactory {
	return f.k8sInformers
}

func (f *kubeInformerFactory) CRD() cache.SharedIndexInformer {
	return f.getInformer("CRDInformer", func() cache.SharedIndexInformer {

		ext, err := extclient.NewForConfig(f.clientSet.Config())
		if err != nil {
			panic(err)
		}

		restClient := ext.ApiextensionsV1beta1().RESTClient()

		lw := cache.NewListWatchFromClient(restClient, "customresourcedefinitions", k8sv1.NamespaceAll, fields.Everything())

		return cache.NewSharedIndexInformer(lw, &extv1beta1.CustomResourceDefinition{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) OperatorPrometheusRule() cache.SharedIndexInformer {
	return f.getInformer("OperatorPrometheusRuleInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.PrometheusClient().MonitoringV1().RESTClient(), "prometheusrules", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &promv1.PrometheusRule{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) DummyOperatorPrometheusRule() cache.SharedIndexInformer {
	return f.getInformer("FakeOperatorPrometheusRuleInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&promv1.PrometheusRule{})
		return informer
	})
}

func (f *kubeInformerFactory) StorageClass() cache.SharedIndexInformer {
	return f.getInformer("storageClassInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.StorageV1().RESTClient()
		lw := cache.NewListWatchFromClient(restClient, "storageclasses", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &storagev1.StorageClass{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) Pod() cache.SharedIndexInformer {
	return f.getInformer("podInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "pods", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &k8sv1.Pod{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

// VolumeSnapshotInformer returns an informer for VolumeSnapshots
func VolumeSnapshotInformer(clientSet kubecli.KubevirtClient, resyncPeriod time.Duration) cache.SharedIndexInformer {
	restClient := clientSet.KubernetesSnapshotClient().SnapshotV1beta1().RESTClient()
	lw := cache.NewListWatchFromClient(restClient, "volumesnapshots", k8sv1.NamespaceAll, fields.Everything())
	return cache.NewSharedIndexInformer(lw, &vsv1beta1.VolumeSnapshot{}, resyncPeriod, cache.Indexers{})
}

// VolumeSnapshotClassInformer returns an informer for VolumeSnapshotClasses
func VolumeSnapshotClassInformer(clientSet kubecli.KubevirtClient, resyncPeriod time.Duration) cache.SharedIndexInformer {
	restClient := clientSet.KubernetesSnapshotClient().SnapshotV1beta1().RESTClient()
	lw := cache.NewListWatchFromClient(restClient, "volumesnapshotclasses", k8sv1.NamespaceAll, fields.Everything())
	return cache.NewSharedIndexInformer(lw, &vsv1beta1.VolumeSnapshotClass{}, resyncPeriod, cache.Indexers{})
}
