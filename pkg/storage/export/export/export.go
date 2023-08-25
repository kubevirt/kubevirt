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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package export

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/openshift/library-go/pkg/build/naming"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	validation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"

	virtv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/storage/snapshot"
	"kubevirt.io/kubevirt/pkg/storage/types"
	kutil "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	unexpectedResourceFmt  = "unexpected resource %+v"
	failedKeyFromObjectFmt = "failed to get key from object: %v, %v"
	enqueuedForSyncFmt     = "enqueued %q for sync"

	pvcNotFoundReason  = "PVCNotFound"
	pvcBoundReason     = "PVCBound"
	pvcPendingReason   = "PVCPending"
	unknownReason      = "Unknown"
	initializingReason = "Initializing"
	inUseReason        = "InUse"
	podPendingReason   = "PodPending"
	podReadyReason     = "PodReady"
	podCompletedReason = "PodCompleted"

	exportServiceLabel = "kubevirt.io.virt-export-service"

	exportPrefix = "virt-export"

	blockVolumeMountPath = "/dev/export-volumes"
	fileSystemMountPath  = "/export-volumes"
	urlBasePath          = "/volumes"

	// annContentType is an annotation on a PVC indicating the content type. This is populated by CDI.
	annContentType = "cdi.kubevirt.io/storage.contentType"
	// annCertParams stores "current" cert rotation params in pod in order to detect changes
	annCertParams = "kubevirt.io/export.certParameters"

	caDefaultPath = "/etc/virt-controller/exportca"
	caCertFile    = caDefaultPath + "/tls.crt"
	caKeyFile     = caDefaultPath + "/tls.key"
	// name of certificate secret volume in pod
	certificates = "certificates"

	exporterPodFailedOrCompletedEvent     = "ExporterPodFailedOrCompleted"
	exporterPodCreatedEvent               = "ExporterPodCreated"
	ExportPaused                          = "ExportPaused"
	secretCreatedEvent                    = "SecretCreated"
	serviceCreatedEvent                   = "ServiceCreated"
	certParamsChangedEvent                = "CertificateParametersChanged"
	exporterManifestConfigMapCreatedEvent = "DataManifestCreated"

	kvm = 107

	// secretTokenLength is the lenght of the randomly generated token
	secretTokenLength = 20
	// secretTokenKey is the entry used to store the token in the virtualMachineExport secret
	secretTokenKey = "token"

	requeueTime = time.Second * 3

	vmManifest             = "virtualmachine-manifest"
	exportNameKey          = "export-name"
	manifestData           = "manifest-data"
	manifestsPath          = "/manifests/all"
	secretManifestPath     = "/manifests/secret"
	externalHostKey        = "external_host"
	internalHostKey        = "internal_host"
	externalCaConfigMapKey = "external_ca_cm"
	internalCaConfigMapKey = "internal_ca_cm"
)

// variable so can be overridden in tests
var currentTime = func() *metav1.Time {
	t := metav1.Now()
	return &t
}

var exportGVK = schema.GroupVersionKind{
	Group:   exportv1.SchemeGroupVersion.Group,
	Version: exportv1.SchemeGroupVersion.Version,
	Kind:    "VirtualMachineExport",
}

var datavolumeGVK = schema.GroupVersionKind{
	Group:   cdiv1.SchemeGroupVersion.Group,
	Version: cdiv1.SchemeGroupVersion.Version,
	Kind:    "DataVolume",
}

func rawURI(pvc *corev1.PersistentVolumeClaim) string {
	return path.Join(fmt.Sprintf("%s/%s/disk.img", urlBasePath, pvc.Name))
}

func rawGzipURI(pvc *corev1.PersistentVolumeClaim) string {
	return path.Join(fmt.Sprintf("%s/%s/disk.img.gz", urlBasePath, pvc.Name))
}

func archiveURI(pvc *corev1.PersistentVolumeClaim) string {
	return path.Join(fmt.Sprintf("%s/%s/disk.tar.gz", urlBasePath, pvc.Name))
}

func dirURI(pvc *corev1.PersistentVolumeClaim) string {
	return path.Join(fmt.Sprintf("%s/%s/dir", urlBasePath, pvc.Name)) + "/"
}

type sourceVolumes struct {
	volumes          []*corev1.PersistentVolumeClaim
	inUse            bool
	isPopulated      bool
	availableMessage string
}

func (sv *sourceVolumes) isSourceAvailable() bool {
	return !sv.inUse && sv.isPopulated
}

// VMExportController is resonsible for exporting VMs
type VMExportController struct {
	Client kubecli.KubevirtClient

	TemplateService services.TemplateService

	VMExportInformer            cache.SharedIndexInformer
	PVCInformer                 cache.SharedIndexInformer
	VMSnapshotInformer          cache.SharedIndexInformer
	VMSnapshotContentInformer   cache.SharedIndexInformer
	PodInformer                 cache.SharedIndexInformer
	DataVolumeInformer          cache.SharedIndexInformer
	ConfigMapInformer           cache.SharedIndexInformer
	ServiceInformer             cache.SharedIndexInformer
	VMInformer                  cache.SharedIndexInformer
	VMIInformer                 cache.SharedIndexInformer
	RouteConfigMapInformer      cache.SharedInformer
	RouteCache                  cache.Store
	IngressCache                cache.Store
	SecretInformer              cache.SharedIndexInformer
	CRDInformer                 cache.SharedIndexInformer
	KubeVirtInformer            cache.SharedIndexInformer
	VolumeSnapshotProvider      snapshot.VolumeSnapshotProvider
	InstancetypeInformer        cache.SharedIndexInformer
	ClusterInstancetypeInformer cache.SharedIndexInformer
	PreferenceInformer          cache.SharedIndexInformer
	ClusterPreferenceInformer   cache.SharedIndexInformer
	ControllerRevisionInformer  cache.SharedIndexInformer

	Recorder record.EventRecorder

	KubevirtNamespace string

	vmExportQueue workqueue.RateLimitingInterface

	caCertManager *bootstrap.FileCertificateManager

	clusterConfig *virtconfig.ClusterConfig

	instancetypeMethods instancetype.Methods
}

type CertParams struct {
	Duration    time.Duration
	RenewBefore time.Duration
}

type getExportVolumeName func(pvc *corev1.PersistentVolumeClaim, vmExport *exportv1.VirtualMachineExport) string

// Default getExportVolumeName function
func getVolumeName(pvc *corev1.PersistentVolumeClaim, vmExport *exportv1.VirtualMachineExport) string {
	return pvc.Name
}

func serializeCertParams(cp *CertParams) (string, error) {
	bs, err := json.Marshal(cp)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func deserializeCertParams(value string) (*CertParams, error) {
	cp := &CertParams{}
	if err := json.Unmarshal([]byte(value), cp); err != nil {
		return nil, err
	}
	return cp, nil
}

var initCert = func(ctrl *VMExportController) {
	ctrl.caCertManager = bootstrap.NewFileCertificateManager(caCertFile, caKeyFile)
	go ctrl.caCertManager.Start()
}

// Init initializes the export controller
func (ctrl *VMExportController) Init() error {
	var err error
	ctrl.clusterConfig, err = virtconfig.NewClusterConfig(ctrl.CRDInformer, ctrl.KubeVirtInformer, ctrl.KubevirtNamespace)
	if err != nil {
		return err
	}
	ctrl.vmExportQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-export-vmexport")

	_, err = ctrl.VMExportInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMExport,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMExport(newObj) },
		},
	)
	if err != nil {
		return err
	}

	_, err = ctrl.PodInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePod,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePod(newObj) },
			DeleteFunc: ctrl.handlePod,
		},
	)
	if err != nil {
		return err
	}
	_, err = ctrl.ServiceInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleService,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleService(newObj) },
			DeleteFunc: ctrl.handleService,
		},
	)
	if err != nil {
		return err
	}
	_, err = ctrl.PVCInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePVC,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePVC(newObj) },
			DeleteFunc: ctrl.handlePVC,
		},
	)
	if err != nil {
		return err
	}
	_, err = ctrl.VMSnapshotInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMSnapshot,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMSnapshot(newObj) },
			DeleteFunc: ctrl.handleVMSnapshot,
		},
	)
	if err != nil {
		return err
	}
	_, err = ctrl.VMIInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMI,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMI(newObj) },
			DeleteFunc: ctrl.handleVMI,
		},
	)
	if err != nil {
		return err
	}
	_, err = ctrl.VMInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVM,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVM(newObj) },
			DeleteFunc: ctrl.handleVM,
		},
	)
	if err != nil {
		return err
	}
	_, err = ctrl.KubeVirtInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: ctrl.handleKubeVirt,
		},
	)
	if err != nil {
		return err
	}
	ctrl.instancetypeMethods = &instancetype.InstancetypeMethods{
		InstancetypeStore:        ctrl.InstancetypeInformer.GetStore(),
		ClusterInstancetypeStore: ctrl.ClusterInstancetypeInformer.GetStore(),
		PreferenceStore:          ctrl.PreferenceInformer.GetStore(),
		ClusterPreferenceStore:   ctrl.ClusterPreferenceInformer.GetStore(),
		ControllerRevisionStore:  ctrl.ControllerRevisionInformer.GetStore(),
		Clientset:                ctrl.Client,
	}

	initCert(ctrl)
	return nil
}

// Run the controller
func (ctrl *VMExportController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.vmExportQueue.ShutDown()

	log.Log.Info("Starting export controller.")
	defer log.Log.Info("Shutting down export controller.")

	if !cache.WaitForCacheSync(
		stopCh,
		ctrl.VMExportInformer.HasSynced,
		ctrl.PVCInformer.HasSynced,
		ctrl.PodInformer.HasSynced,
		ctrl.DataVolumeInformer.HasSynced,
		ctrl.ConfigMapInformer.HasSynced,
		ctrl.ServiceInformer.HasSynced,
		ctrl.RouteConfigMapInformer.HasSynced,
		ctrl.SecretInformer.HasSynced,
		ctrl.VMSnapshotInformer.HasSynced,
		ctrl.VMSnapshotContentInformer.HasSynced,
		ctrl.VMInformer.HasSynced,
		ctrl.VMIInformer.HasSynced,
		ctrl.CRDInformer.HasSynced,
		ctrl.KubeVirtInformer.HasSynced,
		ctrl.InstancetypeInformer.HasSynced,
		ctrl.ClusterInstancetypeInformer.HasSynced,
		ctrl.PreferenceInformer.HasSynced,
		ctrl.ClusterPreferenceInformer.HasSynced,
		ctrl.ControllerRevisionInformer.HasSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.vmExportWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (ctrl *VMExportController) vmExportWorker() {
	for ctrl.processVMExportWorkItem() {
	}
}

func (ctrl *VMExportController) processVMExportWorkItem() bool {
	return watchutil.ProcessWorkItem(ctrl.vmExportQueue, func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("vmExport worker processing key [%s]", key)

		storeObj, exists, err := ctrl.VMExportInformer.GetStore().GetByKey(key)
		if !exists || err != nil {
			return 0, err
		}

		vmExport, ok := storeObj.(*exportv1.VirtualMachineExport)
		if !ok {
			return 0, fmt.Errorf(unexpectedResourceFmt, storeObj)
		}

		return ctrl.updateVMExport(vmExport.DeepCopy())
	})
}

func (ctrl *VMExportController) handlePod(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if pod, ok := obj.(*corev1.Pod); ok {
		key := ctrl.getOwnerVMexportKey(pod)
		_, exists, err := ctrl.VMExportInformer.GetStore().GetByKey(key)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		if exists {
			log.Log.V(3).Infof("Adding VMExport due to pod %s", key)
			ctrl.vmExportQueue.Add(key)
		}
	}
}

func (ctrl *VMExportController) handleService(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if service, ok := obj.(*corev1.Service); ok {
		serviceKey := ctrl.getOwnerVMexportKey(service)
		_, exists, err := ctrl.VMExportInformer.GetStore().GetByKey(serviceKey)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		if exists {
			log.Log.V(3).Infof("Adding VMExport due to service %s", serviceKey)
			ctrl.vmExportQueue.Add(serviceKey)
		}
	}
}

func (ctrl *VMExportController) handleKubeVirt(oldObj, newObj interface{}) {
	okv, ok := oldObj.(*virtv1.KubeVirt)
	if !ok {
		return
	}

	nkv, ok := newObj.(*virtv1.KubeVirt)
	if !ok {
		return
	}

	if equality.Semantic.DeepEqual(okv.Spec.CertificateRotationStrategy, nkv.Spec.CertificateRotationStrategy) {
		return
	}

	// queue everything
	keys := ctrl.VMExportInformer.GetStore().ListKeys()
	for _, key := range keys {
		ctrl.vmExportQueue.Add(key)
	}
}

func (ctrl *VMExportController) getPVCsFromName(namespace, name string) *corev1.PersistentVolumeClaim {
	pvc, exists, err := ctrl.getPvc(namespace, name)
	if err != nil {
		log.Log.V(3).Infof("Error getting pvc by name %v", err)
		return nil
	}
	if exists {
		return pvc
	}
	return nil
}

func (ctrl *VMExportController) updateVMExport(vmExport *exportv1.VirtualMachineExport) (time.Duration, error) {
	log.Log.V(3).Infof("Updating VirtualMachineExport %s/%s", vmExport.Namespace, vmExport.Name)

	if vmExport.DeletionTimestamp != nil {
		return 0, nil
	}

	service, err := ctrl.getOrCreateExportService(vmExport)
	if err != nil {
		return 0, err
	}

	if vmExport.Status == nil {
		populateInitialVMExportStatus(vmExport)
	}

	if err := ctrl.handleVMExportToken(vmExport); err != nil {
		return 0, err
	}

	if ctrl.isSourcePvc(&vmExport.Spec) {
		return ctrl.handleSource(vmExport, service, ctrl.getPVCFromSourcePVC, ctrl.updateVMExportPvcStatus)
	}
	if ctrl.isSourceVMSnapshot(&vmExport.Spec) {
		return ctrl.handleSource(vmExport, service, ctrl.getPVCFromSourceVMSnapshot, ctrl.updateVMExporVMSnapshotStatus)
	}
	if ctrl.isSourceVM(&vmExport.Spec) {
		return ctrl.handleSource(vmExport, service, ctrl.getPVCFromSourceVM, ctrl.updateVMExportVMStatus)
	}
	return 0, nil
}

type pvcFromSourceFunc func(*exportv1.VirtualMachineExport) (*sourceVolumes, error)
type updateVMExportStatusFunc func(*exportv1.VirtualMachineExport, *corev1.Pod, *corev1.Service, *sourceVolumes) (time.Duration, error)

func (ctrl *VMExportController) handleSource(vmExport *exportv1.VirtualMachineExport, service *corev1.Service, getPVCFromSource pvcFromSourceFunc, updateStatus updateVMExportStatusFunc) (time.Duration, error) {
	sourceVolumes, err := getPVCFromSource(vmExport)
	if err != nil {
		return 0, err
	}
	log.Log.V(4).Infof("Source volumes %v", sourceVolumes)

	pod, err := ctrl.manageExporterPod(vmExport, service, sourceVolumes)
	if err != nil {
		return 0, err
	}

	return updateStatus(vmExport, pod, service, sourceVolumes)
}

func (ctrl *VMExportController) manageExporterPod(vmExport *exportv1.VirtualMachineExport, service *corev1.Service, sourceVolumes *sourceVolumes) (*corev1.Pod, error) {
	pod, podExists, err := ctrl.getExporterPod(vmExport)
	if err != nil {
		return nil, err
	}
	if !podExists {
		if sourceVolumes.isSourceAvailable() {
			if len(sourceVolumes.volumes) > 0 {
				pod, err = ctrl.createExporterPod(vmExport, service, sourceVolumes.volumes)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if pod != nil {
		if pod.Status.Phase == corev1.PodPending {
			if err := ctrl.createCertSecret(vmExport, pod); err != nil {
				return nil, err
			}
		}

		if sourceVolumes.isSourceAvailable() {
			if err := ctrl.checkPod(vmExport, pod); err != nil {
				return nil, err
			}
		} else {
			// source is not available, stop the exporter pod if started
			if err := ctrl.deleteExporterPod(vmExport, pod, ExportPaused, sourceVolumes.availableMessage); err != nil {
				return nil, err
			}
			pod = nil
		}
	}
	return pod, nil
}

func (ctrl *VMExportController) deleteExporterPod(vmExport *exportv1.VirtualMachineExport, pod *corev1.Pod, deleteReason, message string) error {
	ctrl.Recorder.Eventf(vmExport, corev1.EventTypeWarning, deleteReason, message)
	if err := ctrl.Client.CoreV1().Pods(vmExport.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{}); !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (ctrl *VMExportController) checkPod(vmExport *exportv1.VirtualMachineExport, pod *corev1.Pod) error {
	if pod.DeletionTimestamp != nil {
		return nil
	}

	if ttlExpiration := getExpirationTime(vmExport); !time.Now().Before(ttlExpiration) {
		if err := ctrl.Client.VirtualMachineExport(vmExport.Namespace).Delete(context.Background(), vmExport.Name, metav1.DeleteOptions{}); err != nil {
			return err
		}
		return nil
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		// The server died or completed, delete the pod.
		return ctrl.deleteExporterPod(vmExport, pod, exporterPodFailedOrCompletedEvent, fmt.Sprintf("Exporter pod %s/%s is in phase %s", pod.Namespace, pod.Name, pod.Status.Phase))
	}

	certParams, err := ctrl.getCertParams()
	if err != nil {
		return err
	}
	scp, err := serializeCertParams(certParams)
	if err != nil {
		return err
	}

	if pod.Annotations[annCertParams] != scp {
		// must recreate pod/secret because params changed
		return ctrl.deleteExporterPod(vmExport, pod, certParamsChangedEvent, "Exporter TLS certificate parameters updated")
	}
	return nil
}

func (ctrl *VMExportController) isPVCPopulated(pvc *corev1.PersistentVolumeClaim) (bool, error) {
	return cdiv1.IsPopulated(pvc, func(name, namespace string) (*cdiv1.DataVolume, error) {
		obj, exists, err := ctrl.DataVolumeInformer.GetStore().GetByKey(controller.NamespacedKey(namespace, name))
		if err != nil {
			return nil, err
		}
		if exists {
			dv, ok := obj.(*cdiv1.DataVolume)
			if ok {
				return dv, nil
			}
		}
		return nil, fmt.Errorf("datavolume %s/%s not found", namespace, name)
	})
}

func (ctrl *VMExportController) createCertSecret(vmExport *exportv1.VirtualMachineExport, ownerPod *corev1.Pod) error {
	secret, err := ctrl.createCertSecretManifest(vmExport, ownerPod)
	if err != nil {
		return err
	}
	_, err = ctrl.Client.CoreV1().Secrets(vmExport.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	} else {
		log.Log.V(3).Infof("Created new exporter pod secret")
		ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, secretCreatedEvent, "Created exporter pod secret")
	}
	return nil
}

func (ctrl *VMExportController) createCertSecretManifest(vmExport *exportv1.VirtualMachineExport, ownerPod *corev1.Pod) (*corev1.Secret, error) {
	v, exists := ownerPod.Annotations[annCertParams]
	if !exists {
		return nil, fmt.Errorf("pod missing cert parameter annotation")
	}

	certParams, err := deserializeCertParams(v)
	if err != nil {
		return nil, err
	}

	caCert := ctrl.caCertManager.Current()
	caKeyPair := &triple.KeyPair{
		Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
		Cert: caCert.Leaf,
	}
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		fmt.Sprintf(components.LocalPodDNStemplateString, ctrl.getExportServiceName(vmExport), vmExport.Namespace),
		ctrl.getExportServiceName(vmExport),
		vmExport.Namespace,
		components.CaClusterLocal,
		nil,
		nil,
		certParams.Duration,
	)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrl.getExportSecretName(ownerPod),
			Namespace: vmExport.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ownerPod, schema.GroupVersionKind{
					Group:   corev1.SchemeGroupVersion.Group,
					Version: corev1.SchemeGroupVersion.Version,
					Kind:    "Pod",
				}),
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": cert.EncodeCertPEM(keyPair.Cert),
			"tls.key": cert.EncodePrivateKeyPEM(keyPair.Key),
		},
	}, nil
}

// handleVMExportToken checks if a secret has been specified for the current export object and, if not, creates one specific to it
func (ctrl *VMExportController) handleVMExportToken(vmExport *exportv1.VirtualMachineExport) error {
	// If a tokenSecretRef has been specified, we assume that the corresponding
	// secret has already been created and managed appropiately by the user
	if vmExport.Spec.TokenSecretRef != nil {
		vmExport.Status.TokenSecretRef = vmExport.Spec.TokenSecretRef
		return nil
	}

	// If not, the secret name is constructed so it can be specific to the current vmExport object
	if vmExport.Status.TokenSecretRef == nil {
		generatedSecretName := getDefaultTokenSecretName(vmExport)
		vmExport.Status.TokenSecretRef = &generatedSecretName
	}

	token, err := kutil.GenerateSecureRandomString(secretTokenLength)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *vmExport.Status.TokenSecretRef,
			Namespace: vmExport.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmExport, schema.GroupVersionKind{
					Group:   exportv1.SchemeGroupVersion.Group,
					Version: exportv1.SchemeGroupVersion.Version,
					Kind:    "VirtualMachineExport",
				}),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretTokenKey: []byte(token),
		},
	}

	secret, err = ctrl.Client.CoreV1().Secrets(vmExport.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, secretCreatedEvent, "Created default secret %s/%s", secret.Namespace, secret.Name)
	return nil
}

func (ctrl *VMExportController) getExportSecretName(ownerPod *corev1.Pod) string {
	var certSecretName string
	for _, volume := range ownerPod.Spec.Volumes {
		if volume.Name == certificates {
			certSecretName = volume.Secret.SecretName
		}
	}
	return certSecretName
}

// getDefaultTokenSecretName returns a secret name specifically created for the current export object
func getDefaultTokenSecretName(vme *exportv1.VirtualMachineExport) string {
	return naming.GetName("export-token", vme.Name, validation.DNS1035LabelMaxLength)
}

func (ctrl *VMExportController) getExportServiceName(vmExport *exportv1.VirtualMachineExport) string {
	return naming.GetName(exportPrefix, vmExport.Name, validation.DNS1035LabelMaxLength)
}

func (ctrl *VMExportController) getExportPodName(vmExport *exportv1.VirtualMachineExport) string {
	return naming.GetName(exportPrefix, vmExport.Name, validation.DNS1035LabelMaxLength)
}

func (ctrl *VMExportController) getOrCreateExportService(vmExport *exportv1.VirtualMachineExport) (*corev1.Service, error) {
	key := controller.NamespacedKey(vmExport.Namespace, ctrl.getExportServiceName(vmExport))
	if service, exists, err := ctrl.ServiceInformer.GetStore().GetByKey(key); err != nil {
		return nil, err
	} else if !exists {
		service := ctrl.createServiceManifest(vmExport)
		log.Log.V(3).Infof("Creating new exporter service %s/%s", service.Namespace, service.Name)
		service, err := ctrl.Client.CoreV1().Services(vmExport.Namespace).Create(context.Background(), service, metav1.CreateOptions{})
		if err == nil {
			ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, serviceCreatedEvent, "Created service %s/%s", service.Namespace, service.Name)
		}
		return service, err
	} else {
		return service.(*corev1.Service), nil
	}
}

func (ctrl *VMExportController) createServiceManifest(vmExport *exportv1.VirtualMachineExport) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrl.getExportServiceName(vmExport),
			Namespace: vmExport.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmExport, schema.GroupVersionKind{
					Group:   exportGVK.Group,
					Version: exportGVK.Version,
					Kind:    exportGVK.Kind,
				}),
			},
			Labels: map[string]string{
				virtv1.AppLabel: exportv1.App,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol: "TCP",
					Port:     443,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8443,
					},
				},
			},
			Selector: map[string]string{
				exportServiceLabel: vmExport.Name,
			},
		},
	}
	return service
}

func (ctrl *VMExportController) getExporterPod(vmExport *exportv1.VirtualMachineExport) (*corev1.Pod, bool, error) {
	key := controller.NamespacedKey(vmExport.Namespace, ctrl.getExportPodName(vmExport))
	if obj, exists, err := ctrl.PodInformer.GetStore().GetByKey(key); err != nil {
		log.Log.Errorf("error %v", err)
		return nil, false, err
	} else if !exists {
		return nil, exists, nil
	} else {
		pod := obj.(*corev1.Pod)
		return pod, exists, nil
	}
}

func (ctrl *VMExportController) createExporterPod(vmExport *exportv1.VirtualMachineExport, service *corev1.Service, pvcs []*corev1.PersistentVolumeClaim) (*corev1.Pod, error) {
	log.Log.V(3).Infof("Checking if pod exists: %s/%s", vmExport.Namespace, ctrl.getExportPodName(vmExport))
	key := controller.NamespacedKey(vmExport.Namespace, ctrl.getExportPodName(vmExport))
	if obj, exists, err := ctrl.PodInformer.GetStore().GetByKey(key); err != nil {
		log.Log.Errorf("error %v", err)
		return nil, err
	} else if !exists {
		manifest, err := ctrl.createExporterPodManifest(vmExport, service, pvcs)
		if err != nil {
			return nil, err
		}

		log.Log.V(3).Infof("Creating new exporter pod %s/%s", manifest.Namespace, manifest.Name)
		pod, err := ctrl.Client.CoreV1().Pods(vmExport.Namespace).Create(context.Background(), manifest, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, exporterPodCreatedEvent, "Created exporter pod %s/%s", manifest.Namespace, manifest.Name)
		return pod, nil
	} else {
		pod := obj.(*corev1.Pod)
		return pod, nil
	}
}

func (ctrl *VMExportController) createExporterPodManifest(vmExport *exportv1.VirtualMachineExport, service *corev1.Service, pvcs []*corev1.PersistentVolumeClaim) (*corev1.Pod, error) {
	certParams, err := ctrl.getCertParams()
	if err != nil {
		return nil, err
	}

	scp, err := serializeCertParams(certParams)
	if err != nil {
		return nil, err
	}

	deadline := certParams.Duration - certParams.RenewBefore
	podManifest := ctrl.TemplateService.RenderExporterManifest(vmExport, exportPrefix)
	podManifest.Labels = map[string]string{exportServiceLabel: vmExport.Name}
	podManifest.Annotations = map[string]string{annCertParams: scp}
	podManifest.Spec.SecurityContext = &corev1.PodSecurityContext{
		RunAsNonRoot:   pointer.Bool(true),
		FSGroup:        pointer.Int64Ptr(kvm),
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}
	for i, pvc := range pvcs {
		var mountPoint string
		if types.IsPVCBlock(pvc.Spec.VolumeMode) {
			mountPoint = fmt.Sprintf("%s/%s", blockVolumeMountPath, pvc.Name)
			podManifest.Spec.Containers[0].VolumeDevices = append(podManifest.Spec.Containers[0].VolumeDevices, corev1.VolumeDevice{
				Name:       pvc.Name,
				DevicePath: mountPoint,
			})
		} else {
			mountPoint = fmt.Sprintf("%s/%s", fileSystemMountPath, pvc.Name)
			podManifest.Spec.Containers[0].VolumeMounts = append(podManifest.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      pvc.Name,
				ReadOnly:  true,
				MountPath: mountPoint,
			})
		}
		podManifest.Spec.Volumes = append(podManifest.Spec.Volumes, corev1.Volume{
			Name: pvc.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		})
		ctrl.addVolumeEnvironmentVariables(&podManifest.Spec.Containers[0], pvc, i, mountPoint)
	}

	// Add token and certs ENV variables
	podManifest.Spec.Containers[0].Env = append(podManifest.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "CERT_FILE",
		Value: "/cert/tls.crt",
	}, corev1.EnvVar{
		Name:  "KEY_FILE",
		Value: "/cert/tls.key",
	}, corev1.EnvVar{
		Name:  "TOKEN_FILE",
		Value: "/token/token",
	}, corev1.EnvVar{
		Name:  "DEADLINE",
		Value: getDeadlineValue(deadline, vmExport).Format(time.RFC3339),
	}, corev1.EnvVar{
		Name:  "EXPORT_VM_DEF_URI",
		Value: manifestsPath,
	}, corev1.EnvVar{
		Name:  "EXPORT_SECRET_DEF_URI",
		Value: secretManifestPath,
	})

	tokenSecretRef := ""
	if vmExport.Status != nil && vmExport.Status.TokenSecretRef != nil {
		tokenSecretRef = *vmExport.Status.TokenSecretRef
	}

	secretName := fmt.Sprintf("secret-%s", rand.String(10))
	podManifest.Spec.Volumes = append(podManifest.Spec.Volumes, corev1.Volume{
		Name: certificates,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}, corev1.Volume{
		Name: tokenSecretRef,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: tokenSecretRef,
			},
		},
	})

	podManifest.Spec.Containers[0].VolumeMounts = append(podManifest.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      certificates,
		MountPath: "/cert",
	}, corev1.VolumeMount{
		Name:      tokenSecretRef,
		MountPath: "/token",
	})

	if vm, err := ctrl.getVmFromExport(vmExport); err != nil {
		return nil, err
	} else {
		if vm != nil {
			if err := ctrl.createDataManifestAndAddToPod(vmExport, vm, podManifest, service); err != nil {
				return nil, err
			}
		}
	}
	return podManifest, nil
}

func (ctrl *VMExportController) createDataManifestAndAddToPod(vmExport *exportv1.VirtualMachineExport, vm *virtv1.VirtualMachine, podManifest *corev1.Pod, service *corev1.Service) error {
	vmManifestConfigMap, err := ctrl.createDataManifestConfigMap(vmExport, vm, service)
	if err != nil {
		return err
	}
	cm, err := ctrl.Client.CoreV1().ConfigMaps(vmExport.Namespace).Create(context.Background(), vmManifestConfigMap, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	} else {
		ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, exporterManifestConfigMapCreatedEvent, "Created exporter data manifest %s/%s", cm.Namespace, cm.Name)
	}

	podManifest.Spec.Containers[0].VolumeMounts = append(podManifest.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      manifestData,
		MountPath: "/manifest_data",
	})
	podManifest.Spec.Volumes = append(podManifest.Spec.Volumes, corev1.Volume{
		Name: manifestData,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: vmManifestConfigMap.Name,
				},
			},
		},
	})
	return nil
}

func (ctrl *VMExportController) createDataManifestConfigMap(vmExport *exportv1.VirtualMachineExport, vm *virtv1.VirtualMachine, service *corev1.Service) (*corev1.ConfigMap, error) {
	data := make(map[string]string)

	data[internalHostKey] = fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace)
	cert, err := ctrl.internalExportCa()
	if err != nil {
		return nil, err
	}
	internalCaCm := ctrl.createExportCaConfigMap(cert, vmExport.Name)
	caCmBytes, err := json.Marshal(internalCaCm)
	if err != nil {
		return nil, err
	}
	data[internalCaConfigMapKey] = string(caCmBytes)

	externalUrlPath := fmt.Sprintf(externalUrlLinkFormat, vmExport.Namespace, vmExport.Name)
	externalLinkHost, cert := ctrl.getExternalLinkHostAndCert()
	if externalLinkHost != "" {
		data[externalHostKey] = path.Join(externalLinkHost, externalUrlPath)
		externalCaCm := ctrl.createExportCaConfigMap(cert, vmExport.Name)
		caCmBytes, err := json.Marshal(externalCaCm)
		if err != nil {
			return nil, err
		}
		data[externalCaConfigMapKey] = string(caCmBytes)
	}
	vmBytes, err := ctrl.generateVMDefinitionFromVm(vm)
	if err != nil {
		return nil, err
	}
	data[vmManifest] = string(vmBytes)

	datavolumes := ctrl.generateDataVolumesFromVm(vm)
	for _, datavolume := range datavolumes {
		if datavolume != nil {
			dvBytes, err := json.Marshal(datavolume)
			if err != nil {
				return nil, err
			}
			data[fmt.Sprintf("dv-%s", datavolume.Name)] = string(dvBytes)
		}
	}
	data[exportNameKey] = vmExport.Name
	res := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vm.Namespace,
			Name:      ctrl.getVmManifestConfigMapName(vmExport),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmExport, exportGVK),
			},
		},
		Data: data,
	}
	return res, nil
}

func (ctrl *VMExportController) createExportCaConfigMap(ca, vmExportName string) *corev1.ConfigMap {
	data := make(map[string]string)
	data["ca.pem"] = ca
	res := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("export-ca-cm-%s", vmExportName),
		},
		Data: data,
	}
	return res
}

func (ctrl *VMExportController) getVmManifestConfigMapName(vmExport *exportv1.VirtualMachineExport) string {
	return fmt.Sprintf("exporter-vm-manifest-%s", vmExport.Name)
}

func (ctrl *VMExportController) getVmFromExport(vmExport *exportv1.VirtualMachineExport) (*virtv1.VirtualMachine, error) {
	vmName := ""
	if ctrl.isSourceVMSnapshot(&vmExport.Spec) {
		vmName = ctrl.getVmNameFromVmSnapshot(vmExport)
	} else if ctrl.isSourceVM(&vmExport.Spec) {
		vmName = vmExport.Spec.Source.Name
	}
	if vmName == "" {
		return nil, nil
	}
	vm, exists, err := ctrl.getVm(vmExport.Namespace, vmName)
	if err != nil {
		return nil, err
	}
	if exists {
		return ctrl.expandVirtualMachine(vm)
	}
	return nil, nil
}

func (ctrl *VMExportController) addVolumeEnvironmentVariables(exportContainer *corev1.Container, pvc *corev1.PersistentVolumeClaim, index int, mountPoint string) {
	exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
		Name:  fmt.Sprintf("VOLUME%d_EXPORT_PATH", index),
		Value: mountPoint,
	})
	if types.IsPVCBlock(pvc.Spec.VolumeMode) {
		exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
			Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_URI", index),
			Value: rawURI(pvc),
		}, corev1.EnvVar{
			Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_GZIP_URI", index),
			Value: rawGzipURI(pvc),
		})
	} else {
		if ctrl.isKubevirtContentType(pvc) {
			exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_URI", index),
				Value: rawURI(pvc),
			}, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_GZIP_URI", index),
				Value: rawGzipURI(pvc),
			})
		} else {
			exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_ARCHIVE_URI", index),
				Value: archiveURI(pvc),
			}, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_DIR_URI", index),
				Value: dirURI(pvc),
			})
		}
	}
}

func (ctrl *VMExportController) isKubevirtContentType(pvc *corev1.PersistentVolumeClaim) bool {
	// Block volumes are assumed always KubevirtContentType
	if types.IsPVCBlock(pvc.Spec.VolumeMode) {
		return true
	}
	contentType, ok := pvc.Annotations[annContentType]
	isKubevirt := ok && (contentType == string(cdiv1.DataVolumeKubeVirt) || contentType == "")
	if isKubevirt {
		return true
	}
	ownerRef := metav1.GetControllerOf(pvc)
	if ownerRef == nil {
		return false
	}
	if ownerRef.Kind == datavolumeGVK.Kind && ownerRef.APIVersion == datavolumeGVK.GroupVersion().String() {
		obj, exists, err := ctrl.DataVolumeInformer.GetStore().GetByKey(controller.NamespacedKey(pvc.GetNamespace(), ownerRef.Name))
		if err != nil {
			log.Log.V(3).Infof("Error getting DataVolume %v", err)
		}
		if exists {
			dv, ok := obj.(*cdiv1.DataVolume)
			isKubevirt = ok && (dv.Spec.ContentType == cdiv1.DataVolumeKubeVirt || dv.Spec.ContentType == "")
		}
	}
	return isKubevirt
}

func (ctrl *VMExportController) updateCommonVMExportStatusFields(vmExport, vmExportCopy *exportv1.VirtualMachineExport, exporterPod *corev1.Pod, service *corev1.Service, sourceVolumes *sourceVolumes, getVolumeName getExportVolumeName) error {
	var err error

	vmExportCopy.Status.ServiceName = service.Name
	vmExportCopy.Status.Links = &exportv1.VirtualMachineExportLinks{}
	if exporterPod == nil {
		vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, inUseReason, sourceVolumes.availableMessage))
		vmExportCopy.Status.Phase = exportv1.Pending
	} else {
		if exporterPod.Status.Phase == corev1.PodRunning {
			vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionTrue, podReadyReason, ""))
			vmExportCopy.Status.Phase = exportv1.Ready
			vmExportCopy.Status.Links.Internal, err = ctrl.getInteralLinks(sourceVolumes.volumes, exporterPod, service, getVolumeName, vmExport)
			if err != nil {
				return err
			}
			vmExportCopy.Status.Links.External, err = ctrl.getExternalLinks(sourceVolumes.volumes, exporterPod, getVolumeName, vmExport)
			if err != nil {
				return err
			}
		} else if exporterPod.Status.Phase == corev1.PodSucceeded {
			vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, podCompletedReason, ""))
			vmExportCopy.Status.Phase = exportv1.Terminated
		} else if exporterPod.Status.Phase == corev1.PodPending {
			vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, podPendingReason, ""))
			vmExportCopy.Status.Phase = exportv1.Pending
		} else {
			vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, unknownReason, ""))
			vmExportCopy.Status.Phase = exportv1.Pending
		}
	}

	return nil
}

func (ctrl *VMExportController) updateVMExportStatus(vmExport, vmExportCopy *exportv1.VirtualMachineExport) error {
	if !equality.Semantic.DeepEqual(vmExport.Status, vmExportCopy.Status) {
		if _, err := ctrl.Client.VirtualMachineExport(vmExportCopy.Namespace).Update(context.Background(), vmExportCopy, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (ctrl *VMExportController) getCertParams() (*CertParams, error) {
	kv := ctrl.clusterConfig.GetConfigFromKubeVirtCR()
	if kv == nil {
		return nil, fmt.Errorf("no KubeVirt CR")
	}
	duration := apply.GetCertDuration(kv.Spec.CertificateRotationStrategy.SelfSigned)
	renewBefore := apply.GetCertRenewBefore(kv.Spec.CertificateRotationStrategy.SelfSigned)
	return &CertParams{
		Duration:    duration.Duration,
		RenewBefore: renewBefore.Duration,
	}, nil
}

func populateInitialVMExportStatus(vmExport *exportv1.VirtualMachineExport) {
	expireAt := metav1.NewTime(getExpirationTime(vmExport))
	vmExport.Status = &exportv1.VirtualMachineExportStatus{
		Phase: exportv1.Pending,
		Conditions: []exportv1.Condition{
			newReadyCondition(corev1.ConditionFalse, initializingReason, ""),
			newPvcCondition(corev1.ConditionFalse, unknownReason, ""),
		},
		TTLExpirationTime: &expireAt,
	}
}

func getDeadlineValue(deadline time.Duration, vmExport *exportv1.VirtualMachineExport) time.Time {
	// Pod needs to shutdown to either cert rotate or because export TTL expired altogether
	rotate := currentTime().Add(deadline)
	ttlExpiration := getExpirationTime(vmExport)

	if ttlExpiration.After(rotate) {
		return rotate
	}
	return ttlExpiration
}

func getExpirationTime(vmExport *exportv1.VirtualMachineExport) time.Time {
	ttl := exportv1.DefaultDurationTTL
	if vmExport.Spec.TTLDuration != nil {
		ttl = vmExport.Spec.TTLDuration.Duration
	}

	return vmExport.GetCreationTimestamp().Time.Add(ttl)
}

func newReadyCondition(status corev1.ConditionStatus, reason, message string) exportv1.Condition {
	return exportv1.Condition{
		Type:               exportv1.ConditionReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: *currentTime(),
	}
}

func newPvcCondition(status corev1.ConditionStatus, reason, message string) exportv1.Condition {
	return exportv1.Condition{
		Type:               exportv1.ConditionPVC,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: *currentTime(),
	}
}

func newVolumesCreatedCondition(status corev1.ConditionStatus, reason, message string) exportv1.Condition {
	return exportv1.Condition{
		Type:               exportv1.ConditionVolumesCreated,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: *currentTime(),
	}
}

func updateCondition(conditions []exportv1.Condition, c exportv1.Condition) []exportv1.Condition {
	found := false
	for i := range conditions {
		if conditions[i].Type == c.Type {
			if conditions[i].Status != c.Status || conditions[i].Reason != c.Reason || conditions[i].Message != c.Message {
				conditions[i] = c
			}
			found = true
			break
		}
	}

	if !found {
		conditions = append(conditions, c)
	}

	return conditions
}

func (ctrl *VMExportController) pvcConditionFromPVC(pvcs []*corev1.PersistentVolumeClaim) exportv1.Condition {
	cond := exportv1.Condition{
		Type:               exportv1.ConditionPVC,
		LastTransitionTime: *currentTime(),
	}
	phase := corev1.ClaimBound
	// Figure out most severe status.
	// Bound least, pending more, lost is most severe status
	for _, pvc := range pvcs {
		if pvc.Status.Phase == corev1.ClaimPending && phase != corev1.ClaimLost {
			phase = corev1.ClaimPending
		}
		if pvc.Status.Phase == corev1.ClaimLost {
			phase = corev1.ClaimLost
		}
	}
	switch phase {
	case corev1.ClaimBound:
		cond.Status = corev1.ConditionTrue
		cond.Reason = pvcBoundReason
	case corev1.ClaimPending:
		cond.Status = corev1.ConditionFalse
		cond.Reason = pvcPendingReason
	default:
		cond.Status = corev1.ConditionFalse
		cond.Reason = unknownReason
	}
	return cond
}

func (ctrl *VMExportController) expandVirtualMachine(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachine, error) {
	instancetypeSpec, err := ctrl.instancetypeMethods.FindInstancetypeSpec(vm)
	if err != nil {
		return nil, err
	}
	preferenceSpec, err := ctrl.instancetypeMethods.FindPreferenceSpec(vm)
	if err != nil {
		return nil, err
	}

	if instancetypeSpec == nil && preferenceSpec == nil {
		return vm, nil
	}

	conflicts := ctrl.instancetypeMethods.ApplyToVmi(field.NewPath("spec", "template", "spec"), instancetypeSpec, preferenceSpec, &vm.Spec.Template.Spec, &vm.Spec.Template.ObjectMeta)
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("cannot expand instancetype to VM, due to %d conflicts", len(conflicts))
	}

	// Remove InstancetypeMatcher and PreferenceMatcher, so the returned VM object can be used and not cause a conflict
	vm.Spec.Instancetype = nil
	vm.Spec.Preference = nil

	return vm, nil
}

func (ctrl *VMExportController) updateHttpSourceDataVolumeTemplate(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		volumeName := ""
		if volume.DataVolume != nil {
			volumeName = volume.DataVolume.Name
		}
		if volume.PersistentVolumeClaim != nil {
			volumeName = volume.PersistentVolumeClaim.ClaimName
		}
		if volumeName != "" {
			vm.Spec.DataVolumeTemplates = ctrl.replaceUrlDVTemplate(volumeName, vm.Namespace, vm.Spec.DataVolumeTemplates)
		}
	}
	return vm
}

func (ctrl *VMExportController) replaceUrlDVTemplate(volumeName, namespace string, templates []virtv1.DataVolumeTemplateSpec) []virtv1.DataVolumeTemplateSpec {
	res := make([]virtv1.DataVolumeTemplateSpec, 0)
	for _, template := range templates {
		if template.ObjectMeta.Name == volumeName {
			//Replace template
			replacement := template.DeepCopy()
			replacement.Spec.Source = &cdiv1.DataVolumeSource{
				HTTP: &cdiv1.DataVolumeSourceHTTP{
					URL: "",
				},
			}
			res = append(res, *replacement)
		} else {
			res = append(res, template)
		}
	}
	return res
}

func (ctrl *VMExportController) generateVMDefinitionFromVm(vm *virtv1.VirtualMachine) ([]byte, error) {
	expandedVm, err := ctrl.expandVirtualMachine(vm)
	if err != nil {
		return nil, err
	}
	// Clear status
	expandedVm.Status = virtv1.VirtualMachineStatus{}
	expandedVm.ManagedFields = nil
	cleanedObjectMeta := metav1.ObjectMeta{}
	cleanedObjectMeta.Name = expandedVm.ObjectMeta.Name
	cleanedObjectMeta.Namespace = expandedVm.ObjectMeta.Namespace
	cleanedObjectMeta.Labels = expandedVm.ObjectMeta.Labels
	cleanedObjectMeta.Annotations = expandedVm.Annotations
	expandedVm.ObjectMeta = cleanedObjectMeta

	// Update dvTemplates if exists
	expandedVm = ctrl.updateHttpSourceDataVolumeTemplate(vm)
	vmBytes, err := json.Marshal(expandedVm)
	if err != nil {
		return nil, err
	}
	return vmBytes, nil
}

func (ctrl *VMExportController) generateDataVolumesFromVm(vm *virtv1.VirtualMachine) []*cdiv1.DataVolume {
	res := make([]*cdiv1.DataVolume, 0)
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		volumeName := ""
		if volume.DataVolume != nil {
			volumeName = volume.DataVolume.Name
		}
		if volume.PersistentVolumeClaim != nil {
			volumeName = volume.PersistentVolumeClaim.ClaimName
		}
		if volumeName != "" {
			found := false
			for _, template := range vm.Spec.DataVolumeTemplates {
				if template.ObjectMeta.Name == volumeName {
					found = true
				}
			}
			if !found {
				res = append(res, ctrl.createExportHttpDvFromPVC(vm.Namespace, volumeName))
			}
		}
	}
	return res
}

func (ctrl *VMExportController) createExportHttpDvFromPVC(namespace, name string) *cdiv1.DataVolume {
	pvc := ctrl.getPVCsFromName(namespace, name)
	if pvc != nil {
		pvc.Spec.VolumeName = ""
		pvc.Spec.StorageClassName = nil
		// Don't copy datasources, will be populated by CDI with the datavolume
		pvc.Spec.DataSource = nil
		pvc.Spec.DataSourceRef = nil
		return &cdiv1.DataVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: cdiv1.DataVolumeSpec{
				Source: &cdiv1.DataVolumeSource{
					HTTP: &cdiv1.DataVolumeSourceHTTP{
						URL: "",
					},
				},
				PVC: &pvc.Spec,
			},
		}
	}
	return nil
}
