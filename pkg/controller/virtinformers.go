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
 * Copyright The KubeVirt Authors.
 *
 */

package controller

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"kubevirt.io/api/snapshot"

	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	clonebase "kubevirt.io/api/clone"
	clone "kubevirt.io/api/clone/v1beta1"

	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	routev1 "github.com/openshift/api/route/v1"
	secv1 "github.com/openshift/api/security/v1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	k8sv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"kubevirt.io/api/core"
	kubev1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/api/migrations"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	/*
		TODO: replace the assignment to expression that accepts only kubev1.ManagedByLabelOperatorValue after few releases (after 0.47)
		The new assignment is to avoid error on update
		(operator can't recognize components with the old managed-by label's value)
	*/
	OperatorLabel    = kubev1.ManagedByLabel + " in (" + kubev1.ManagedByLabelOperatorValue + "," + kubev1.ManagedByLabelOperatorOldValue + " )"
	NotOperatorLabel = kubev1.ManagedByLabel + " notin (" + kubev1.ManagedByLabelOperatorValue + "," + kubev1.ManagedByLabelOperatorOldValue + " )"
)

var unexpectedObjectError = errors.New("unexpected object")

type newSharedInformer func() cache.SharedIndexInformer

type KubeInformerFactory interface {
	// Starts any informers that have not been started yet
	// This function is thread safe and idempotent
	Start(stopCh <-chan struct{})

	// Waits for all informers to sync
	WaitForCacheSync(stopCh <-chan struct{})

	// Watches for vmi objects
	VMI() cache.SharedIndexInformer

	// Watches for vmi objects assigned to a specific host
	VMISourceHost(hostName string) cache.SharedIndexInformer

	// Watches for vmi objects assigned to a specific host
	// as a migration target
	VMITargetHost(hostName string) cache.SharedIndexInformer

	// Watches for VirtualMachineInstanceReplicaSet objects
	VMIReplicaSet() cache.SharedIndexInformer

	// Watches for VirtualMachinePool objects
	VMPool() cache.SharedIndexInformer

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

	// Watches VirtualMachineExport objects
	VirtualMachineExport() cache.SharedIndexInformer

	// Watches VirtualMachineSnapshot objects
	VirtualMachineSnapshot() cache.SharedIndexInformer

	// Watches VirtualMachineSnapshot objects
	VirtualMachineSnapshotContent() cache.SharedIndexInformer

	// Watches VirtualMachineRestore objects
	VirtualMachineRestore() cache.SharedIndexInformer

	// Watches MigrationPolicy objects
	MigrationPolicy() cache.SharedIndexInformer

	// Watches VirtualMachineClone objects
	VirtualMachineClone() cache.SharedIndexInformer

	// Watches VirtualMachineInstancetype objects
	VirtualMachineInstancetype() cache.SharedIndexInformer

	// Watches VirtualMachineClusterInstancetype objects
	VirtualMachineClusterInstancetype() cache.SharedIndexInformer

	// Watches VirtualMachinePreference objects
	VirtualMachinePreference() cache.SharedIndexInformer

	// Watches VirtualMachineClusterPreference objects
	VirtualMachineClusterPreference() cache.SharedIndexInformer

	// Watches for k8s extensions api configmap
	ApiAuthConfigMap() cache.SharedIndexInformer

	// Watches for the kubevirt CA config map
	KubeVirtCAConfigMap() cache.SharedIndexInformer

	// Watches for the kubevirt export CA config map
	KubeVirtExportCAConfigMap() cache.SharedIndexInformer

	// Watches for changes in kubevirt leases
	Leases() cache.SharedIndexInformer

	// Watches for the export route config map
	ExportRouteConfigMap() cache.SharedIndexInformer

	// Watches for the kubevirt export service
	ExportService() cache.SharedIndexInformer

	// ConfigMaps which are managed by the operator
	OperatorConfigMap() cache.SharedIndexInformer

	// Watches for PersistentVolumeClaim objects
	PersistentVolumeClaim() cache.SharedIndexInformer

	// Watches for ControllerRevision objects
	ControllerRevision() cache.SharedIndexInformer

	// Watches for CDI DataVolume objects
	DataVolume() cache.SharedIndexInformer

	// Fake CDI DataVolume informer used when feature gate is disabled
	DummyDataVolume() cache.SharedIndexInformer

	// Watches for CDI DataSource objects
	DataSource() cache.SharedIndexInformer

	// Fake CDI DataSource informer used when feature gate is disabled
	DummyDataSource() cache.SharedIndexInformer

	// Watches for CDI StorageProfile objects
	StorageProfile() cache.SharedIndexInformer

	// Fake CDI StorageProfile informer used when feature gate is disabled
	DummyStorageProfile() cache.SharedIndexInformer

	// Watches for CDI objects
	CDI() cache.SharedIndexInformer

	// Fake CDI informer used when feature gate is disabled
	DummyCDI() cache.SharedIndexInformer

	// Watches for CDIConfig objects
	CDIConfig() cache.SharedIndexInformer

	// Fake CDIConfig informer used when feature gate is disabled
	DummyCDIConfig() cache.SharedIndexInformer

	// CRD
	CRD() cache.SharedIndexInformer

	// Watches for KubeVirt objects
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

	// Routes
	OperatorRoute() cache.SharedIndexInformer

	// Fake Routes informer used when not on openshift
	DummyOperatorRoute() cache.SharedIndexInformer

	// Ingress
	Ingress() cache.SharedIndexInformer

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

	// Unmanaged secrets for things like Ingress TLS
	UnmanagedSecrets() cache.SharedIndexInformer

	// Fake ServiceMonitor informer used when Prometheus is not installed
	DummyOperatorServiceMonitor() cache.SharedIndexInformer

	// ValidatingAdmissionPolicyBinding created/managed by virt operator
	OperatorValidatingAdmissionPolicyBinding() cache.SharedIndexInformer

	// Fake OperatorValidatingAdmissionPolicyBinding informer used when ValidatingAdmissionPolicyBinding is not installed
	DummyOperatorValidatingAdmissionPolicyBinding() cache.SharedIndexInformer

	// ValidatingAdmissionPolicies created/managed by virt operator
	OperatorValidatingAdmissionPolicy() cache.SharedIndexInformer

	// Fake OperatorValidatingAdmissionPolicy informer used when ValidatingAdmissionPolicy is not installed
	DummyOperatorValidatingAdmissionPolicy() cache.SharedIndexInformer

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

	ResourceQuota() cache.SharedIndexInformer

	ResourceClaim() cache.SharedIndexInformer

	ResourceSlice() cache.SharedIndexInformer

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

func (f *kubeInformerFactory) WaitForCacheSync(stopCh <-chan struct{}) {
	syncs := []cache.InformerSynced{}

	f.lock.Lock()
	for name, informer := range f.informers {
		log.Log.Infof("Waiting for cache sync of informer %s", name)
		syncs = append(syncs, informer.HasSynced)
	}
	f.lock.Unlock()

	cache.WaitForCacheSync(stopCh, syncs...)
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
			&k8sv1.Namespace{},
			f.defaultResync,
			cache.Indexers{
				"namespace_name": func(obj interface{}) ([]string, error) {
					return []string{obj.(*k8sv1.Namespace).GetName()}, nil
				},
			},
		)
	})
}

func GetVMIInformerIndexers() cache.Indexers {
	return cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
		"node": func(obj interface{}) (strings []string, e error) {
			return []string{obj.(*kubev1.VirtualMachineInstance).Status.NodeName}, nil
		},
		"dv": func(obj interface{}) ([]string, error) {
			vmi, ok := obj.(*kubev1.VirtualMachineInstance)
			if !ok {
				return nil, unexpectedObjectError
			}
			var dvs []string
			for _, vol := range vmi.Spec.Volumes {
				if vol.DataVolume != nil {
					dvs = append(dvs, fmt.Sprintf("%s/%s", vmi.Namespace, vol.DataVolume.Name))
				}
			}
			return dvs, nil
		},
		"pvc": func(obj interface{}) ([]string, error) {
			vmi, ok := obj.(*kubev1.VirtualMachineInstance)
			if !ok {
				return nil, unexpectedObjectError
			}
			var pvcs []string
			for _, vol := range vmi.Spec.Volumes {
				if vol.PersistentVolumeClaim != nil {
					pvcs = append(pvcs, fmt.Sprintf("%s/%s", vmi.Namespace, vol.PersistentVolumeClaim.ClaimName))
				}
			}
			return pvcs, nil
		},
	}
}

func (f *kubeInformerFactory) VMI() cache.SharedIndexInformer {
	return f.getInformer("vmiInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.VirtualMachineInstance{}, f.defaultResync, GetVMIInformerIndexers())
	})
}

func (f *kubeInformerFactory) VMISourceHost(hostName string) cache.SharedIndexInformer {
	labelSelector, err := labels.Parse(fmt.Sprintf(kubev1.NodeNameLabel+" in (%s)", hostName))
	if err != nil {
		panic(err)
	}

	return f.getInformer("vmiInformer-sources", func() cache.SharedIndexInformer {
		lw := NewListWatchFromClient(f.restClient, "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &kubev1.VirtualMachineInstance{}, f.defaultResync, cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
			"node": func(obj interface{}) (strings []string, e error) {
				return []string{obj.(*kubev1.VirtualMachineInstance).Status.NodeName}, nil
			},
		})
	})
}

func (f *kubeInformerFactory) VMITargetHost(hostName string) cache.SharedIndexInformer {
	labelSelector, err := labels.Parse(fmt.Sprintf(kubev1.MigrationTargetNodeNameLabel+" in (%s)", hostName))
	if err != nil {
		panic(err)
	}

	return f.getInformer("vmiInformer-targets", func() cache.SharedIndexInformer {
		lw := NewListWatchFromClient(f.restClient, "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
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

func (f *kubeInformerFactory) VMPool() cache.SharedIndexInformer {
	return f.getInformer("vmpool", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().PoolV1alpha1().RESTClient(), "virtualmachinepools", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &poolv1.VirtualMachinePool{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
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

func GetVirtualMachineInformerIndexers() cache.Indexers {
	return cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
		"dv": func(obj interface{}) ([]string, error) {
			vm, ok := obj.(*kubev1.VirtualMachine)
			if !ok {
				return nil, unexpectedObjectError
			}
			var dvs []string
			for _, vol := range vm.Spec.Template.Spec.Volumes {
				if vol.DataVolume != nil {
					dvs = append(dvs, fmt.Sprintf("%s/%s", vm.Namespace, vol.DataVolume.Name))
				}
			}
			return dvs, nil
		},
		"pvc": func(obj interface{}) ([]string, error) {
			vm, ok := obj.(*kubev1.VirtualMachine)
			if !ok {
				return nil, unexpectedObjectError
			}
			var pvcs []string
			for _, vol := range vm.Spec.Template.Spec.Volumes {
				if vol.PersistentVolumeClaim != nil {
					pvcs = append(pvcs, fmt.Sprintf("%s/%s", vm.Namespace, vol.PersistentVolumeClaim.ClaimName))
				}
			}
			return pvcs, nil
		},
	}
}

func (f *kubeInformerFactory) VirtualMachine() cache.SharedIndexInformer {
	return f.getInformer("vmInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "virtualmachines", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.VirtualMachine{}, f.defaultResync, GetVirtualMachineInformerIndexers())
	})
}

func GetVirtualMachineExportInformerIndexers() cache.Indexers {
	return cache.Indexers{
		"pvc": func(obj interface{}) ([]string, error) {
			export, ok := obj.(*exportv1.VirtualMachineExport)
			if !ok {
				return nil, unexpectedObjectError
			}

			if (export.Spec.Source.APIGroup == nil ||
				*export.Spec.Source.APIGroup == "" || *export.Spec.Source.APIGroup == "v1") &&
				export.Spec.Source.Kind == "PersistentVolumeClaim" {
				return []string{fmt.Sprintf("%s/%s", export.Namespace, export.Spec.Source.Name)}, nil
			}

			return nil, nil
		},
		"vmsnapshot": func(obj interface{}) ([]string, error) {
			export, ok := obj.(*exportv1.VirtualMachineExport)
			if !ok {
				return nil, unexpectedObjectError
			}

			if export.Spec.Source.APIGroup != nil &&
				*export.Spec.Source.APIGroup == snapshotv1.SchemeGroupVersion.Group &&
				export.Spec.Source.Kind == "VirtualMachineSnapshot" {
				return []string{fmt.Sprintf("%s/%s", export.Namespace, export.Spec.Source.Name)}, nil
			}

			return nil, nil
		},
		"vm": func(obj interface{}) ([]string, error) {
			export, ok := obj.(*exportv1.VirtualMachineExport)
			if !ok {
				return nil, unexpectedObjectError
			}

			if export.Spec.Source.APIGroup != nil &&
				*export.Spec.Source.APIGroup == core.GroupName &&
				export.Spec.Source.Kind == "VirtualMachine" {
				return []string{fmt.Sprintf("%s/%s", export.Namespace, export.Spec.Source.Name)}, nil
			}

			return nil, nil
		},
	}
}

func (f *kubeInformerFactory) VirtualMachineExport() cache.SharedIndexInformer {
	return f.getInformer("vmExportInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().ExportV1beta1().RESTClient(), "virtualmachineexports", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &exportv1.VirtualMachineExport{}, f.defaultResync, GetVirtualMachineExportInformerIndexers())
	})
}

func GetVirtualMachineSnapshotInformerIndexers() cache.Indexers {
	return cache.Indexers{
		"vm": func(obj interface{}) ([]string, error) {
			vms, ok := obj.(*snapshotv1.VirtualMachineSnapshot)
			if !ok {
				return nil, unexpectedObjectError
			}

			if vms.Spec.Source.APIGroup != nil &&
				*vms.Spec.Source.APIGroup == core.GroupName &&
				vms.Spec.Source.Kind == "VirtualMachine" {
				return []string{fmt.Sprintf("%s/%s", vms.Namespace, vms.Spec.Source.Name)}, nil
			}

			return nil, nil
		},
	}
}

func (f *kubeInformerFactory) VirtualMachineSnapshot() cache.SharedIndexInformer {
	return f.getInformer("vmSnapshotInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().SnapshotV1beta1().RESTClient(), "virtualmachinesnapshots", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &snapshotv1.VirtualMachineSnapshot{}, f.defaultResync, GetVirtualMachineSnapshotInformerIndexers())
	})
}

func GetVirtualMachineSnapshotContentInformerIndexers() cache.Indexers {
	return cache.Indexers{
		"volumeSnapshot": func(obj interface{}) ([]string, error) {
			vmsc, ok := obj.(*snapshotv1.VirtualMachineSnapshotContent)
			if !ok {
				return nil, unexpectedObjectError
			}
			var volumeSnapshots []string
			for _, v := range vmsc.Spec.VolumeBackups {
				if v.VolumeSnapshotName != nil {
					k := fmt.Sprintf("%s/%s", vmsc.Namespace, *v.VolumeSnapshotName)
					volumeSnapshots = append(volumeSnapshots, k)
				}
			}
			return volumeSnapshots, nil
		},
	}
}

func (f *kubeInformerFactory) VirtualMachineSnapshotContent() cache.SharedIndexInformer {
	return f.getInformer("vmSnapshotContentInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().SnapshotV1beta1().RESTClient(), "virtualmachinesnapshotcontents", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &snapshotv1.VirtualMachineSnapshotContent{}, f.defaultResync, GetVirtualMachineSnapshotContentInformerIndexers())
	})
}

func GetVirtualMachineRestoreInformerIndexers() cache.Indexers {
	return cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
		"vm": func(obj interface{}) ([]string, error) {
			vmr, ok := obj.(*snapshotv1.VirtualMachineRestore)
			if !ok {
				return nil, unexpectedObjectError
			}

			if vmr.Spec.Target.APIGroup != nil &&
				*vmr.Spec.Target.APIGroup == core.GroupName &&
				vmr.Spec.Target.Kind == "VirtualMachine" {
				return []string{fmt.Sprintf("%s/%s", vmr.Namespace, vmr.Spec.Target.Name)}, nil
			}

			return nil, nil
		},
	}
}

func (f *kubeInformerFactory) VirtualMachineRestore() cache.SharedIndexInformer {
	return f.getInformer("vmRestoreInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().SnapshotV1beta1().RESTClient(), "virtualmachinerestores", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &snapshotv1.VirtualMachineRestore{}, f.defaultResync, GetVirtualMachineRestoreInformerIndexers())
	})
}

func (f *kubeInformerFactory) MigrationPolicy() cache.SharedIndexInformer {
	return f.getInformer("migrationPolicyInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().MigrationsV1alpha1().RESTClient(), migrations.ResourceMigrationPolicies, k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &migrationsv1.MigrationPolicy{}, f.defaultResync, cache.Indexers{})
	})
}

func GetVirtualMachineCloneInformerIndexers() cache.Indexers {
	getkey := func(vmClone *clone.VirtualMachineClone, resourceName string) string {
		return fmt.Sprintf("%s/%s", vmClone.Namespace, resourceName)
	}

	return cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
		// Gets: vm key. Returns: clones that their source or target is the specified vm
		"vmSource": func(obj interface{}) ([]string, error) {
			vmClone, ok := obj.(*clone.VirtualMachineClone)
			if !ok {
				return nil, unexpectedObjectError
			}

			source := vmClone.Spec.Source
			if source != nil && source.APIGroup != nil && *source.APIGroup == core.GroupName && source.Kind == "VirtualMachine" {
				return []string{getkey(vmClone, source.Name)}, nil
			}

			return nil, nil
		},
		"vmTarget": func(obj interface{}) ([]string, error) {
			vmClone, ok := obj.(*clone.VirtualMachineClone)
			if !ok {
				return nil, unexpectedObjectError
			}

			target := vmClone.Spec.Target
			if target != nil && target.APIGroup != nil && *target.APIGroup == core.GroupName && target.Kind == "VirtualMachine" {
				return []string{getkey(vmClone, target.Name)}, nil
			}

			return nil, nil
		},
		// Gets: snapshot key. Returns: clones that their source is the specified snapshot
		"snapshotSource": func(obj interface{}) ([]string, error) {
			vmClone, ok := obj.(*clone.VirtualMachineClone)
			if !ok {
				return nil, unexpectedObjectError
			}

			source := vmClone.Spec.Source
			if source != nil && *source.APIGroup == snapshot.GroupName && source.Kind == "VirtualMachineSnapshot" {
				return []string{getkey(vmClone, source.Name)}, nil
			}

			return nil, nil
		},
		// Gets: snapshot key. Returns: clones in phase SnapshotInProgress that wait for the specified snapshot
		string(clone.SnapshotInProgress): func(obj interface{}) ([]string, error) {
			vmClone, ok := obj.(*clone.VirtualMachineClone)
			if !ok {
				return nil, unexpectedObjectError
			}

			if vmClone.Status.Phase == clone.SnapshotInProgress && vmClone.Status.SnapshotName != nil {
				return []string{getkey(vmClone, *vmClone.Status.SnapshotName)}, nil
			}

			return nil, nil
		},
		// Gets: restore key. Returns: clones in phase RestoreInProgress that wait for the specified restore
		string(clone.RestoreInProgress): func(obj interface{}) ([]string, error) {
			vmClone, ok := obj.(*clone.VirtualMachineClone)
			if !ok {
				return nil, unexpectedObjectError
			}

			if vmClone.Status.Phase == clone.RestoreInProgress && vmClone.Status.RestoreName != nil {
				return []string{getkey(vmClone, *vmClone.Status.RestoreName)}, nil
			}

			return nil, nil
		},
		// Gets: restore key. Returns: clones in phase Succeeded
		string(clone.Succeeded): func(obj interface{}) ([]string, error) {
			vmClone, ok := obj.(*clone.VirtualMachineClone)
			if !ok {
				return nil, unexpectedObjectError
			}

			if vmClone.Status.Phase == clone.Succeeded && vmClone.Status.RestoreName != nil {
				return []string{getkey(vmClone, *vmClone.Status.RestoreName)}, nil
			}

			return nil, nil
		},
	}
}

func (f *kubeInformerFactory) VirtualMachineClone() cache.SharedIndexInformer {
	return f.getInformer("virtualMachineCloneInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().CloneV1beta1().RESTClient(), clonebase.ResourceVMClonePlural, k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &clone.VirtualMachineClone{}, f.defaultResync, GetVirtualMachineCloneInformerIndexers())
	})
}

func (f *kubeInformerFactory) VirtualMachineInstancetype() cache.SharedIndexInformer {
	return f.getInformer("vmInstancetypeInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().InstancetypeV1beta1().RESTClient(), instancetypeapi.PluralResourceName, k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &instancetypev1beta1.VirtualMachineInstancetype{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) VirtualMachineClusterInstancetype() cache.SharedIndexInformer {
	return f.getInformer("vmClusterInstancetypeInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().InstancetypeV1beta1().RESTClient(), instancetypeapi.ClusterPluralResourceName, k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &instancetypev1beta1.VirtualMachineClusterInstancetype{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) VirtualMachinePreference() cache.SharedIndexInformer {
	return f.getInformer("vmPreferenceInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().InstancetypeV1beta1().RESTClient(), instancetypeapi.PluralPreferenceResourceName, k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &instancetypev1beta1.VirtualMachinePreference{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) VirtualMachineClusterPreference() cache.SharedIndexInformer {
	return f.getInformer("vmClusterPreferenceInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.GeneratedKubeVirtClient().InstancetypeV1beta1().RESTClient(), instancetypeapi.ClusterPluralPreferenceResourceName, k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &instancetypev1beta1.VirtualMachineClusterPreference{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) DataVolume() cache.SharedIndexInformer {
	return f.getInformer("dataVolumeInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.CdiClient().CdiV1beta1().RESTClient(), "datavolumes", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &cdiv1.DataVolume{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) DummyDataVolume() cache.SharedIndexInformer {
	return f.getInformer("fakeDataVolumeInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		return informer
	})
}

func (f *kubeInformerFactory) DataSource() cache.SharedIndexInformer {
	return f.getInformer("dataSourceInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.CdiClient().CdiV1beta1().RESTClient(), "datasources", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &cdiv1.DataSource{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) DummyDataSource() cache.SharedIndexInformer {
	return f.getInformer("fakeDataSourceInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
		return informer
	})
}

func (f *kubeInformerFactory) StorageProfile() cache.SharedIndexInformer {
	return f.getInformer("storageProfileInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.CdiClient().CdiV1beta1().RESTClient(), "storageprofiles", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &cdiv1.StorageProfile{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) DummyStorageProfile() cache.SharedIndexInformer {
	return f.getInformer("fakeStorageProfileInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
		return informer
	})
}

func (f *kubeInformerFactory) CDI() cache.SharedIndexInformer {
	return f.getInformer("cdiInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CdiClient().CdiV1beta1().RESTClient()
		lw := cache.NewListWatchFromClient(restClient, "cdis", k8sv1.NamespaceAll, fields.Everything())

		return cache.NewSharedIndexInformer(lw, &cdiv1.CDI{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) DummyCDI() cache.SharedIndexInformer {
	return f.getInformer("fakeCdiInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&cdiv1.CDI{})
		return informer
	})
}

func (f *kubeInformerFactory) CDIConfig() cache.SharedIndexInformer {
	return f.getInformer("cdiConfigInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CdiClient().CdiV1beta1().RESTClient()
		lw := cache.NewListWatchFromClient(restClient, "cdiconfigs", k8sv1.NamespaceAll, fields.Everything())

		return cache.NewSharedIndexInformer(lw, &cdiv1.CDIConfig{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) DummyCDIConfig() cache.SharedIndexInformer {
	return f.getInformer("fakeCdiConfigInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&cdiv1.CDIConfig{})
		return informer
	})
}

func (f *kubeInformerFactory) Leases() cache.SharedIndexInformer {
	return f.getInformer("leasesInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CoordinationV1().RESTClient()
		lw := cache.NewListWatchFromClient(restClient, "leases", f.kubevirtNamespace, fields.Everything())

		return cache.NewSharedIndexInformer(lw, &coordinationv1.Lease{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) DummyLeases() cache.SharedIndexInformer {
	return f.getInformer("fakeLeasesInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&coordinationv1.Lease{})
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

func (f *kubeInformerFactory) KubeVirtExportCAConfigMap() cache.SharedIndexInformer {
	return f.getInformer("extensionsKubeVirtExportCAConfigMapInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CoreV1().RESTClient()
		fieldSelector := fields.OneTermEqualSelector("metadata.name", "kubevirt-export-ca")
		lw := cache.NewListWatchFromClient(restClient, "configmaps", f.kubevirtNamespace, fieldSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.ConfigMap{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) ExportRouteConfigMap() cache.SharedIndexInformer {
	return f.getInformer("extensionsExportRouteConfigMapInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CoreV1().RESTClient()
		fieldSelector := fields.OneTermEqualSelector("metadata.name", "kube-root-ca.crt")
		lw := cache.NewListWatchFromClient(restClient, "configmaps", f.kubevirtNamespace, fieldSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.ConfigMap{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) ExportService() cache.SharedIndexInformer {
	return f.getInformer("exportService", func() cache.SharedIndexInformer {
		// Watch all service with the kubevirt app label
		labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", kubev1.AppLabel, exportv1.App))
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "services", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.Service{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) PersistentVolumeClaim() cache.SharedIndexInformer {
	return f.getInformer("persistentVolumeClaimInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.CoreV1().RESTClient()
		lw := cache.NewListWatchFromClient(restClient, "persistentvolumeclaims", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &k8sv1.PersistentVolumeClaim{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func GetControllerRevisionInformerIndexers() cache.Indexers {
	return cache.Indexers{
		"vm": func(obj interface{}) ([]string, error) {
			cr, ok := obj.(*appsv1.ControllerRevision)
			if !ok {
				return nil, unexpectedObjectError
			}

			for _, ref := range cr.OwnerReferences {
				if ref.Kind == "VirtualMachine" {
					return []string{string(ref.UID)}, nil
				}
			}

			return nil, nil
		},
		"vmpool": func(obj interface{}) ([]string, error) {
			cr, ok := obj.(*appsv1.ControllerRevision)
			if !ok {
				return nil, unexpectedObjectError
			}

			for _, ref := range cr.OwnerReferences {
				if ref.Kind == "VirtualMachinePool" {
					return []string{string(ref.UID)}, nil
				}
			}

			return nil, nil
		},
	}
}

func (f *kubeInformerFactory) ControllerRevision() cache.SharedIndexInformer {
	return f.getInformer("controllerRevisionInformer", func() cache.SharedIndexInformer {
		restClient := f.clientSet.AppsV1().RESTClient()
		lw := cache.NewListWatchFromClient(restClient, "controllerrevisions", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &appsv1.ControllerRevision{}, f.defaultResync, GetControllerRevisionInformerIndexers())
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
	// #nosec no need for better randomness
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
		labelSelector, err := labels.Parse(fmt.Sprintf("!%s, %s", kubev1.InstallStrategyLabel, OperatorLabel))
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

		lw := NewListWatchFromClient(ext.ApiextensionsV1().RESTClient(), "customresourcedefinitions", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &extv1.CustomResourceDefinition{}, f.defaultResync, cache.Indexers{})
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

func (f *kubeInformerFactory) Ingress() cache.SharedIndexInformer {
	return f.getInformer("Ingress", func() cache.SharedIndexInformer {
		restClient := f.clientSet.NetworkingV1().RESTClient()
		lw := cache.NewListWatchFromClient(restClient, "ingresses", f.kubevirtNamespace, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &networkingv1.Ingress{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorRoute() cache.SharedIndexInformer {
	return f.getInformer("OperatorRoute", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}
		restClient := f.clientSet.RouteClient().RESTClient()
		lw := NewListWatchFromClient(restClient, "routes", f.kubevirtNamespace, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &routev1.Route{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) DummyOperatorRoute() cache.SharedIndexInformer {
	return f.getInformer("FakeOperatorRoute", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&routev1.Route{})
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

		lw := NewListWatchFromClient(f.clientSet.AdmissionregistrationV1().RESTClient(), "validatingwebhookconfigurations", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &admissionregistrationv1.ValidatingWebhookConfiguration{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorMutatingWebhook() cache.SharedIndexInformer {
	return f.getInformer("operatorMutatingWebhookInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.AdmissionregistrationV1().RESTClient(), "mutatingwebhookconfigurations", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &admissionregistrationv1.MutatingWebhookConfiguration{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
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
		return cache.NewSharedIndexInformer(lw, &k8sv1.Secret{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) UnmanagedSecrets() cache.SharedIndexInformer {
	return f.getInformer("secretsInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(NotOperatorLabel)
		if err != nil {
			panic(err)
		}

		restClient := f.clientSet.CoreV1().RESTClient()
		lw := NewListWatchFromClient(restClient, "secrets", f.kubevirtNamespace, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.Secret{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorAPIService() cache.SharedIndexInformer {
	return f.getInformer("operatorAPIServiceInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.aggregatorClient.ApiregistrationV1().RESTClient(), "apiservices", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &apiregv1.APIService{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) OperatorPodDisruptionBudget() cache.SharedIndexInformer {
	return f.getInformer("operatorPodDisruptionBudgetInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.PolicyV1().RESTClient(), "poddisruptionbudgets", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &policyv1.PodDisruptionBudget{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
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

func (f *kubeInformerFactory) OperatorValidatingAdmissionPolicyBinding() cache.SharedIndexInformer {
	return f.getInformer("operatorValidatingAdmissionPolicyBindingInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.AdmissionregistrationV1().RESTClient(), "validatingadmissionpolicybindings", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) DummyOperatorValidatingAdmissionPolicyBinding() cache.SharedIndexInformer {
	return f.getInformer("FakeOperatorValidatingAdmissionPolicyBindingInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&admissionregistrationv1.ValidatingAdmissionPolicyBinding{})
		return informer
	})
}

func (f *kubeInformerFactory) OperatorValidatingAdmissionPolicy() cache.SharedIndexInformer {
	return f.getInformer("operatorValidatingAdmissionPolicyInformer", func() cache.SharedIndexInformer {
		labelSelector, err := labels.Parse(OperatorLabel)
		if err != nil {
			panic(err)
		}

		lw := NewListWatchFromClient(f.clientSet.AdmissionregistrationV1().RESTClient(), "validatingadmissionpolicies", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &admissionregistrationv1.ValidatingAdmissionPolicy{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) DummyOperatorValidatingAdmissionPolicy() cache.SharedIndexInformer {
	return f.getInformer("FakeOperatorValidatingAdmissionPolicyInformer", func() cache.SharedIndexInformer {
		informer, _ := testutils.NewFakeInformerFor(&admissionregistrationv1.ValidatingAdmissionPolicy{})
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

		restClient := ext.ApiextensionsV1().RESTClient()

		lw := cache.NewListWatchFromClient(restClient, "customresourcedefinitions", k8sv1.NamespaceAll, fields.Everything())

		return cache.NewSharedIndexInformer(lw, &extv1.CustomResourceDefinition{}, f.defaultResync, cache.Indexers{})
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

func (f *kubeInformerFactory) ResourceQuota() cache.SharedIndexInformer {
	return f.getInformer("resourceQuotaInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "resourcequotas", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &k8sv1.ResourceQuota{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) ResourceClaim() cache.SharedIndexInformer {
	return f.getInformer("resourceClaimInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.ResourceV1beta1().RESTClient(), "resourceclaims", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &resourcev1beta1.ResourceClaim{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

func (f *kubeInformerFactory) ResourceSlice() cache.SharedIndexInformer {
	return f.getInformer("resourceSliceInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.clientSet.ResourceV1beta1().RESTClient(), "resourceslices", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &resourcev1beta1.ResourceSlice{}, f.defaultResync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
}

// VolumeSnapshotInformer returns an informer for VolumeSnapshots
func VolumeSnapshotInformer(clientSet kubecli.KubevirtClient, resyncPeriod time.Duration) cache.SharedIndexInformer {
	restClient := clientSet.KubernetesSnapshotClient().SnapshotV1().RESTClient()
	lw := cache.NewListWatchFromClient(restClient, "volumesnapshots", k8sv1.NamespaceAll, fields.Everything())
	return cache.NewSharedIndexInformer(lw, &vsv1.VolumeSnapshot{}, resyncPeriod, cache.Indexers{})
}

// VolumeSnapshotClassInformer returns an informer for VolumeSnapshotClasses
func VolumeSnapshotClassInformer(clientSet kubecli.KubevirtClient, resyncPeriod time.Duration) cache.SharedIndexInformer {
	restClient := clientSet.KubernetesSnapshotClient().SnapshotV1().RESTClient()
	lw := cache.NewListWatchFromClient(restClient, "volumesnapshotclasses", k8sv1.NamespaceAll, fields.Everything())
	return cache.NewSharedIndexInformer(lw, &vsv1.VolumeSnapshotClass{}, resyncPeriod, cache.Indexers{})
}
