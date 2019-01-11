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
	"math/rand"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	cdiv1informers "kubevirt.io/containerized-data-importer/pkg/client/informers/externalversions"
	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
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
}

type kubeInformerFactory struct {
	restClient    *rest.RESTClient
	clientSet     kubecli.KubevirtClient
	lock          sync.Mutex
	defaultResync time.Duration

	informers         map[string]cache.SharedIndexInformer
	startedInformers  map[string]bool
	kubevirtNamespace string
}

func NewKubeInformerFactory(restClient *rest.RESTClient, clientSet kubecli.KubevirtClient, kubevirtNamespace string) KubeInformerFactory {
	return &kubeInformerFactory{
		restClient: restClient,
		clientSet:  clientSet,
		// Resulting resync period will be between 12 and 24 hours, like the default for k8s
		defaultResync:     resyncPeriod(12 * time.Hour),
		informers:         make(map[string]cache.SharedIndexInformer),
		startedInformers:  make(map[string]bool),
		kubevirtNamespace: kubevirtNamespace,
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

func (f *kubeInformerFactory) VMI() cache.SharedIndexInformer {
	return f.getInformer("vmiInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.VirtualMachineInstance{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
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

func (f *kubeInformerFactory) DataVolume() cache.SharedIndexInformer {
	return f.getInformer("dataVolumeInformer", func() cache.SharedIndexInformer {
		cdiClient := f.clientSet.CdiClient()
		cdiInformerFactory := cdiv1informers.NewSharedInformerFactory(cdiClient, f.defaultResync)
		return cdiInformerFactory.Cdi().V1alpha1().DataVolumes().Informer()
	})
}

func (f *kubeInformerFactory) DummyDataVolume() cache.SharedIndexInformer {
	return f.getInformer("fakeDataVolumeInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		return informer
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
		return cache.NewSharedIndexInformer(lw, &k8sv1.PersistentVolumeClaim{}, f.defaultResync, cache.Indexers{})
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
