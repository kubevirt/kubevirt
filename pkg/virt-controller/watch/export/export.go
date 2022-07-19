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
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
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
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"

	"github.com/openshift/library-go/pkg/build/naming"
	validation "k8s.io/apimachinery/pkg/util/validation"

	routev1 "github.com/openshift/api/route/v1"
)

const (
	unexpectedResourceFmt  = "unexpected resource %+v"
	failedKeyFromObjectFmt = "failed to get key from object: %v, %v"
	enqueuedForSyncFmt     = "enqueued %q for sync"

	pvcNotFoundReason  = "pvcNotFound"
	pvcBoundReason     = "pvcBound"
	pvcPendingReason   = "pvcPending"
	pvcInUseReason     = "pvcInUse"
	unknownReason      = "unknown"
	initializingReason = "initializing"
	podPendingReason   = "podPending"
	podReadyReason     = "podReady"
	podCompletedReason = "podCompleted"

	exportServiceLabel = "kubevirt.io.virt-export-service"

	exportPrefix = "virt-export"

	blockVolumeMountPath = "/dev/export-volumes"
	fileSystemMountPath  = "/export-volumes"
	urlBasePath          = "/volumes"

	// annContentType is an annotation on a PVC indicating the content type. This is populated by CDI.
	annContentType = "cdi.kubevirt.io/storage.contentType"

	caDefaultPath        = "/etc/virt-controller/exportca"
	caCertFile           = caDefaultPath + "/tls.crt"
	caKeyFile            = caDefaultPath + "/tls.key"
	caBundle             = "ca-bundle"
	routeCAConfigMapName = "kube-root-ca.crt"
	routeCaKey           = "ca.crt"
	subjectAltNameId     = "2.5.29.17"
	// name of certificate secret volume in pod
	certificates = "certificates"

	exporterPodFailedOrCompletedEvent = "ExporterPodFailedOrCompleted"
	exporterPodCreatedEvent           = "ExporterPodCreated"
	secretCreatedEvent                = "SecretCreated"
	serviceCreatedEvent               = "ServiceCreated"

	certExpiry = time.Duration(30 * time.Hour) // 30 hours
	deadline   = time.Duration(24 * time.Hour) // 24 hours

	kvm = 107

	apiGroup              = "export.kubevirt.io"
	apiVersion            = "v1alpha1"
	exportResourceName    = "virtualmachineexports"
	gv                    = apiGroup + "/" + apiVersion
	externalUrlLinkFormat = "/api/" + gv + "/namespaces/%s/" + exportResourceName + "/%s"
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

// VMExportController is resonsible for exporting VMs
type VMExportController struct {
	Client kubecli.KubevirtClient

	TemplateService services.TemplateService

	VMExportInformer       cache.SharedIndexInformer
	PVCInformer            cache.SharedIndexInformer
	PodInformer            cache.SharedIndexInformer
	DataVolumeInformer     cache.SharedIndexInformer
	ConfigMapInformer      cache.SharedIndexInformer
	ServiceInformer        cache.SharedIndexInformer
	RouteConfigMapInformer cache.SharedInformer
	RouteCache             cache.Store
	IngressCache           cache.Store
	SecretInformer         cache.SharedIndexInformer

	Recorder record.EventRecorder

	KubevirtNamespace string
	ResyncPeriod      time.Duration

	vmExportQueue workqueue.RateLimitingInterface

	caCertManager *bootstrap.FileCertificateManager
}

// Init initializes the export controller
func (ctrl *VMExportController) Init() {
	ctrl.vmExportQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-export-vmexport")

	ctrl.VMExportInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMExport,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMExport(newObj) },
		},
		ctrl.ResyncPeriod,
	)
	ctrl.PodInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePod,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePod(newObj) },
			DeleteFunc: ctrl.handlePod,
		},
		ctrl.ResyncPeriod,
	)
	ctrl.ServiceInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleService,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleService(newObj) },
			DeleteFunc: ctrl.handleService,
		},
		ctrl.ResyncPeriod,
	)

	ctrl.caCertManager = bootstrap.NewFileCertificateManager(caCertFile, caKeyFile)
	go ctrl.caCertManager.Start()
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

func (ctrl *VMExportController) handleVMExport(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vmExport, ok := obj.(*exportv1.VirtualMachineExport); ok {
		objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(vmExport)
		if err != nil {
			log.Log.Errorf(failedKeyFromObjectFmt, err, vmExport)
			return
		}
		log.Log.V(3).Infof(enqueuedForSyncFmt, objName)
		ctrl.vmExportQueue.Add(objName)
	}
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
		} else if exists {
			ctrl.vmExportQueue.Add(key)
		}
	}
}

func (ctrl *VMExportController) handleService(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if service, ok := obj.(*corev1.Service); ok {
		key := ctrl.getOwnerVMexportKey(service)
		_, exists, err := ctrl.VMExportInformer.GetStore().GetByKey(key)
		if err != nil {
			utilruntime.HandleError(err)
			return
		} else if exists {
			ctrl.vmExportQueue.Add(key)
		}
	}
}

func (ctrl *VMExportController) getOwnerVMexportKey(obj metav1.Object) string {
	ownerRef := metav1.GetControllerOf(obj)
	var key string
	if ownerRef != nil {
		if ownerRef.Kind == exportGVK.Kind && ownerRef.APIVersion == exportGVK.GroupVersion().String() {
			key = controller.NamespacedKey(obj.GetNamespace(), ownerRef.Name)
		}
	}
	return key
}

func (ctrl *VMExportController) updateVMExport(vmExport *exportv1.VirtualMachineExport) (time.Duration, error) {
	log.Log.V(3).Infof("Updating VirtualMachineExport %s/%s", vmExport.Namespace, vmExport.Name)
	var retry time.Duration

	if vmExport.DeletionTimestamp != nil {
		return retry, nil
	}

	service, err := ctrl.getOrCreateExportService(vmExport)
	if err != nil {
		return 0, err
	}

	if ctrl.isSourcePvc(&vmExport.Spec) {
		return ctrl.handleIsSourcePvc(vmExport, service)
	}
	return retry, nil
}

func (ctrl *VMExportController) handleIsSourcePvc(vmExport *exportv1.VirtualMachineExport, service *corev1.Service) (time.Duration, error) {
	var retry time.Duration
	pvcs := make([]*corev1.PersistentVolumeClaim, 0)
	pvc, exists, err := ctrl.getPvc(vmExport.Namespace, vmExport.Spec.Source.Name)
	if err != nil {
		return 0, err
	} else if exists {
		pvcs = append(pvcs, pvc)
	}

	pod, exists, err := ctrl.getExporterPod(vmExport)
	inUse := false
	if err != nil {
		return 0, err
	} else if !exists {
		inUse, err = ctrl.isPVCInUse(vmExport, pvc)
		if err != nil {
			return retry, err
		}
		if !inUse && len(pvcs) > 0 {
			isPopulated, err := ctrl.isPVCPopulated(pvc)
			if err != nil {
				return retry, err
			} else if isPopulated {
				pod, err = ctrl.createExporterPod(vmExport, pvcs)
				if err != nil {
					return 0, err
				} else if pod == nil {
					return retry, nil
				}

				if err := ctrl.getOrCreateCertSecret(vmExport, pod); err != nil {
					return 0, err
				}
			}
		}
	} else {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			// The server died or completed, delete the pod.
			ctrl.Recorder.Eventf(vmExport, corev1.EventTypeWarning, exporterPodFailedOrCompletedEvent, "Exporter pod %s/%s succeeded or failed", pod.Namespace, pod.Name)
			if err := ctrl.Client.CoreV1().Pods(vmExport.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{}); err != nil {
				return 0, err
			} else {
				return retry, nil
			}
		}
	}

	return ctrl.updateVMExportPvcStatus(vmExport, pvcs, pod, service, inUse)

}

func (ctrl *VMExportController) isPVCPopulated(pvc *corev1.PersistentVolumeClaim) (bool, error) {
	return cdiv1.IsPopulated(pvc, func(name, namespace string) (*cdiv1.DataVolume, error) {
		obj, exists, err := ctrl.DataVolumeInformer.GetStore().GetByKey(controller.NamespacedKey(namespace, name))
		if err != nil {
			return nil, err
		} else if exists {
			dv, ok := obj.(*cdiv1.DataVolume)
			if ok {
				return dv, nil
			}
		}
		return nil, fmt.Errorf("datavolume not found")
	})
}

func (ctrl *VMExportController) getOrCreateCertSecret(vmExport *exportv1.VirtualMachineExport, ownerPod *corev1.Pod) error {
	_, err := ctrl.Client.CoreV1().Secrets(vmExport.Namespace).Create(context.Background(), ctrl.createCertSecretManifest(vmExport, ownerPod), metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	} else {
		log.Log.V(3).Infof("Created new exporter pod secret")
		ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, secretCreatedEvent, "Created exporter pod secret")
	}
	return nil
}

func (ctrl *VMExportController) createCertSecretManifest(vmExport *exportv1.VirtualMachineExport, ownerPod *corev1.Pod) *corev1.Secret {
	caCert := ctrl.caCertManager.Current()
	caKeyPair := &triple.KeyPair{
		Key:  caCert.PrivateKey.(*rsa.PrivateKey),
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
		certExpiry,
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
	}
}

func (ctrl *VMExportController) isPVCInUse(vmExport *exportv1.VirtualMachineExport, pvc *corev1.PersistentVolumeClaim) (bool, error) {
	if pvc == nil {
		return false, nil
	}
	pvcSet := sets.NewString(pvc.Name)
	if usedPods, err := watchutil.PodsUsingPVCs(ctrl.PodInformer, pvc.Namespace, pvcSet); err != nil {
		return false, err
	} else {
		for _, pod := range usedPods {
			if !metav1.IsControlledBy(&pod, vmExport) {
				return true, nil
			}
		}
		return false, nil
	}
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
		ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, serviceCreatedEvent, "Created service %s/%s", service.Namespace, service.Name)
		return ctrl.Client.CoreV1().Services(vmExport.Namespace).Create(context.Background(), service, metav1.CreateOptions{})
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
					Group:   exportv1.SchemeGroupVersion.Group,
					Version: exportv1.SchemeGroupVersion.Version,
					Kind:    "VirtualMachineExport",
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
		log.Log.V(3).Errorf("error %v", err)
		return nil, false, err
	} else if !exists {
		return nil, exists, nil
	} else {
		pod := obj.(*corev1.Pod)
		return pod, exists, nil
	}
}

func (ctrl *VMExportController) createExporterPod(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim) (*corev1.Pod, error) {
	log.Log.V(3).Infof("Checking if pod exist: %s/%s", vmExport.Namespace, ctrl.getExportPodName(vmExport))
	key := controller.NamespacedKey(vmExport.Namespace, ctrl.getExportPodName(vmExport))
	if obj, exists, err := ctrl.PodInformer.GetStore().GetByKey(key); err != nil {
		log.Log.V(3).Errorf("error %v", err)
		return nil, err
	} else if !exists {
		manifest := ctrl.createExporterPodManifest(vmExport, pvcs)

		log.Log.V(3).Infof("Creating new exporter pod %s/%s", manifest.Namespace, manifest.Name)
		ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, exporterPodCreatedEvent, "Creating exporter pod %s/%s", manifest.Namespace, manifest.Name)
		return ctrl.Client.CoreV1().Pods(vmExport.Namespace).Create(context.Background(), manifest, metav1.CreateOptions{})
	} else {
		pod := obj.(*corev1.Pod)
		return pod, nil
	}
}

func (ctrl *VMExportController) createExporterPodManifest(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim) *corev1.Pod {
	podManifest := ctrl.TemplateService.RenderExporterManifest(vmExport, exportPrefix)
	podManifest.ObjectMeta.Labels = map[string]string{exportServiceLabel: vmExport.Name}
	podManifest.Spec.SecurityContext = &corev1.PodSecurityContext{
		RunAsNonRoot: pointer.Bool(true),
		RunAsGroup:   pointer.Int64Ptr(kvm),
		FSGroup:      pointer.Int64Ptr(kvm),
	}
	for i, pvc := range pvcs {
		var mountPoint string
		if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == corev1.PersistentVolumeBlock {
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
		Value: currentTime().Add(deadline).Format(time.RFC3339),
	})

	secretName := fmt.Sprintf("secret-%s", rand.String(10))
	podManifest.Spec.Volumes = append(podManifest.Spec.Volumes, corev1.Volume{
		Name: certificates,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}, corev1.Volume{
		Name: vmExport.Spec.TokenSecretRef,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: vmExport.Spec.TokenSecretRef,
			},
		},
	})

	podManifest.Spec.Containers[0].VolumeMounts = append(podManifest.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      certificates,
		MountPath: "/cert",
	}, corev1.VolumeMount{
		Name:      vmExport.Spec.TokenSecretRef,
		MountPath: "/token",
	})
	return podManifest
}

func (ctrl *VMExportController) addVolumeEnvironmentVariables(exportContainer *corev1.Container, pvc *corev1.PersistentVolumeClaim, index int, mountPoint string) {
	exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
		Name:  fmt.Sprintf("VOLUME%d_EXPORT_PATH", index),
		Value: mountPoint,
	})
	if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == corev1.PersistentVolumeBlock {
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
	if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == corev1.PersistentVolumeBlock {
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
			log.Log.V(1).Infof("Error getting DataVolume %v", err)
		} else if exists {
			dv, ok := obj.(*cdiv1.DataVolume)
			isKubevirt = ok && (dv.Spec.ContentType == cdiv1.DataVolumeKubeVirt || dv.Spec.ContentType == "")
		}
	}
	return isKubevirt
}

func (ctrl *VMExportController) updateVMExportPvcStatus(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim, exporterPod *corev1.Pod, service *corev1.Service, pvcInUse bool) (time.Duration, error) {
	var retry time.Duration
	var err error
	vmExportCopy := vmExport.DeepCopy()
	if vmExportCopy.Status == nil {
		vmExportCopy.Status = &exportv1.VirtualMachineExportStatus{
			Phase: exportv1.Pending,
			Conditions: []exportv1.Condition{
				newReadyCondition(corev1.ConditionFalse, initializingReason),
				newPvcCondition(corev1.ConditionFalse, unknownReason),
			},
		}
	}

	if exporterPod == nil {
		if !pvcInUse {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, initializingReason), true)
		} else {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, pvcInUseReason), true)
		}
		vmExportCopy.Status.Phase = exportv1.Pending
	} else {
		if exporterPod.Status.Phase == corev1.PodRunning {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionTrue, podReadyReason), true)
			vmExportCopy.Status.Phase = exportv1.Ready
		} else if exporterPod.Status.Phase == corev1.PodSucceeded {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, podCompletedReason), true)
			vmExportCopy.Status.Phase = exportv1.Terminated
		} else if exporterPod.Status.Phase == corev1.PodPending {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, podPendingReason), true)
			vmExportCopy.Status.Phase = exportv1.Pending
		} else {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, unknownReason), true)
			vmExportCopy.Status.Phase = exportv1.Pending
		}
	}

	if len(pvcs) == 0 {
		log.Log.V(3).Info("PVC(s) not found, updating status to not found")
		updateCondition(vmExportCopy.Status.Conditions, newPvcCondition(corev1.ConditionFalse, pvcNotFoundReason), true)
	} else {
		updateCondition(vmExportCopy.Status.Conditions, ctrl.pvcConditionFromPVC(pvcs), true)
	}

	vmExportCopy.Status.ServiceName = service.Name
	vmExportCopy.Status.Links = &exportv1.VirtualMachineExportLinks{}
	vmExportCopy.Status.Links.Internal, err = ctrl.getInteralLinks(pvcs, exporterPod, service)
	if err != nil {
		return retry, err
	}
	vmExportCopy.Status.Links.External, err = ctrl.getExternalLinks(pvcs, exporterPod, vmExport)
	if err != nil {
		return retry, err
	}

	if !equality.Semantic.DeepEqual(vmExport, vmExportCopy) {
		if _, err := ctrl.Client.VirtualMachineExport(vmExportCopy.Namespace).Update(context.Background(), vmExportCopy, metav1.UpdateOptions{}); err != nil {
			return retry, err
		}
	}
	if vmExportCopy.Status.Phase == exportv1.Pending {
		log.Log.V(4).Info("Not ready requeueing")
		retry = time.Second
	}
	return retry, nil
}

func (ctrl *VMExportController) getInteralLinks(pvcs []*corev1.PersistentVolumeClaim, exporterPod *corev1.Pod, service *corev1.Service) (*exportv1.VirtualMachineExportLink, error) {
	internalCert, err := ctrl.internalExportCa()
	if err != nil {
		return nil, err
	}
	host := fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace)
	return ctrl.getLinks(pvcs, exporterPod, host, internalCert)
}

func (ctrl *VMExportController) getExternalLinks(pvcs []*corev1.PersistentVolumeClaim, exporterPod *corev1.Pod, export *exportv1.VirtualMachineExport) (*exportv1.VirtualMachineExportLink, error) {
	urlPath := fmt.Sprintf(externalUrlLinkFormat, export.Namespace, export.Name)
	externalLinkHost, cert := ctrl.getExternalLinkHostAndCert()
	if externalLinkHost != "" {
		hostAndBase := path.Join(externalLinkHost, urlPath)
		return ctrl.getLinks(pvcs, exporterPod, hostAndBase, cert)
	}
	return nil, nil
}

func (ctrl *VMExportController) getLinks(pvcs []*corev1.PersistentVolumeClaim, exporterPod *corev1.Pod, hostAndBase, cert string) (*exportv1.VirtualMachineExportLink, error) {
	exportLink := &exportv1.VirtualMachineExportLink{
		Volumes: []exportv1.VirtualMachineExportVolume{},
		Cert:    cert,
	}
	for _, pvc := range pvcs {
		if pvc != nil && exporterPod != nil && exporterPod.Status.Phase == corev1.PodRunning {
			const scheme = "https://"

			if ctrl.isKubevirtContentType(pvc) {
				exportLink.Volumes = append(exportLink.Volumes, exportv1.VirtualMachineExportVolume{
					Name: pvc.Name,
					Formats: []exportv1.VirtualMachineExportVolumeFormat{
						{
							Format: exportv1.KubeVirtRaw,
							Url:    scheme + path.Join(hostAndBase, rawURI(pvc)),
						},
						{
							Format: exportv1.KubeVirtGz,
							Url:    scheme + path.Join(hostAndBase, rawGzipURI(pvc)),
						},
					},
				})
			} else {
				exportLink.Volumes = append(exportLink.Volumes, exportv1.VirtualMachineExportVolume{
					Name: pvc.Name,
					Formats: []exportv1.VirtualMachineExportVolumeFormat{
						{
							Format: exportv1.Dir,
							Url:    scheme + path.Join(hostAndBase, dirURI(pvc)),
						},
						{
							Format: exportv1.ArchiveGz,
							Url:    scheme + path.Join(hostAndBase, archiveURI(pvc)),
						},
					},
				})
			}
		}
	}
	return exportLink, nil
}

func (ctrl *VMExportController) internalExportCa() (string, error) {
	key := controller.NamespacedKey(ctrl.KubevirtNamespace, components.KubeVirtExportCASecretName)
	ctrl.ConfigMapInformer.GetStore().GetByKey(key)
	obj, exists, err := ctrl.ConfigMapInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return "", err
	}
	cm := obj.(*corev1.ConfigMap).DeepCopy()
	bundle := cm.Data[caBundle]
	return strings.TrimSpace(bundle), nil
}

func (ctrl *VMExportController) isSourcePvc(source *exportv1.VirtualMachineExportSpec) bool {
	return source != nil && (source.Source.APIGroup == nil || *source.Source.APIGroup == corev1.SchemeGroupVersion.Group) && source.Source.Kind == "PersistentVolumeClaim"
}

func (ctrl *VMExportController) getPvc(namespace, name string) (*corev1.PersistentVolumeClaim, bool, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.PVCInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return obj.(*corev1.PersistentVolumeClaim).DeepCopy(), true, nil
}

func newReadyCondition(status corev1.ConditionStatus, reason string) exportv1.Condition {
	return exportv1.Condition{
		Type:               exportv1.ConditionReady,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func newPvcCondition(status corev1.ConditionStatus, reason string) exportv1.Condition {
	return exportv1.Condition{
		Type:               exportv1.ConditionPVC,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func updateCondition(conditions []exportv1.Condition, c exportv1.Condition, includeReason bool) []exportv1.Condition {
	found := false
	for i := range conditions {
		if conditions[i].Type == c.Type {
			if conditions[i].Status != c.Status || (includeReason && conditions[i].Reason != c.Reason) {
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
		} else if pvc.Status.Phase == corev1.ClaimLost {
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

func (ctrl *VMExportController) getExternalLinkHostAndCert() (string, string) {
	for _, obj := range ctrl.IngressCache.List() {
		if ingress, ok := obj.(*networkingv1.Ingress); ok {
			if host := getHostFromIngress(ingress); host != "" {
				cert, _ := ctrl.getIngressCert(host, ingress)
				return host, cert
			}
		}
	}
	for _, obj := range ctrl.RouteCache.List() {
		if route, ok := obj.(*routev1.Route); ok {
			if host := getHostFromRoute(route); host != "" {
				cert, _ := ctrl.getRouteCert(host)
				return host, cert
			}
		}
	}
	return "", ""
}

func (ctrl *VMExportController) getIngressCert(hostName string, ing *networkingv1.Ingress) (string, error) {
	secretName := ""
	for _, tls := range ing.Spec.TLS {
		if tls.SecretName != "" {
			secretName = tls.SecretName
			break
		}
	}
	key := controller.NamespacedKey(ctrl.KubevirtNamespace, secretName)
	obj, exists, err := ctrl.SecretInformer.GetStore().GetByKey(key)
	if err != nil {
		return "", err
	} else if !exists {
		return "", nil
	}
	if secret, ok := obj.(*corev1.Secret); ok {
		return ctrl.getIngressCertFromSecret(secret, hostName)
	}
	return "", nil
}

func (ctrl *VMExportController) getIngressCertFromSecret(secret *corev1.Secret, hostName string) (string, error) {
	certBytes := secret.Data["tls.crt"]
	certs, err := cert.ParseCertsPEM(certBytes)
	if err != nil {
		return "", err
	}
	return ctrl.findCertByHostName(hostName, certs)
}

func (ctrl *VMExportController) getRouteCert(hostName string) (string, error) {
	key := controller.NamespacedKey(ctrl.KubevirtNamespace, routeCAConfigMapName)
	obj, exists, err := ctrl.RouteConfigMapInformer.GetStore().GetByKey(key)
	if err != nil {
		return "", err
	} else if !exists {
		return "", nil
	}
	if cm, ok := obj.(*corev1.ConfigMap); ok {
		cmString := cm.Data[routeCaKey]
		certs, err := cert.ParseCertsPEM([]byte(cmString))
		if err != nil {
			return "", err
		}
		return ctrl.findCertByHostName(hostName, certs)
	}
	return "", fmt.Errorf("not a config map")
}

func (ctrl *VMExportController) findCertByHostName(hostName string, certs []*x509.Certificate) (string, error) {
	for _, cert := range certs {
		if ctrl.matchesOrWildCard(hostName, cert.Subject.CommonName) {
			return buildPemFromCert(cert), nil
		}
		for _, extension := range cert.Extensions {
			if extension.Id.String() == subjectAltNameId {
				value := strings.Map(func(r rune) rune {
					if unicode.IsPrint(r) && r <= unicode.MaxASCII {
						return r
					}
					return ' '
				}, string(extension.Value))
				names := strings.Split(value, " ")
				for _, name := range names {
					if ctrl.matchesOrWildCard(hostName, name) {
						return buildPemFromCert(cert), nil
					}
				}
			}
		}
	}
	return "", nil
}

func buildPemFromCert(cert *x509.Certificate) string {
	pemOut := strings.Builder{}
	pem.Encode(&pemOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	return strings.TrimSpace(pemOut.String())
}

func (ctrl *VMExportController) matchesOrWildCard(hostName, compare string) bool {
	wildCard := fmt.Sprintf("*.%s", getDomainFromHost(hostName, ctrl.KubevirtNamespace))
	return hostName == compare || wildCard == compare
}

func getDomainFromHost(host, namespace string) string {
	if index := strings.Index(host, "."); index != -1 {
		return host[index+1:]
	}
	return host
}

func getHostFromRoute(route *routev1.Route) string {
	if route.Spec.To.Name == components.VirtExportProxyServiceName {
		if len(route.Status.Ingress) > 0 {
			return route.Status.Ingress[0].Host
		}
	}
	return ""
}

func getHostFromIngress(ing *networkingv1.Ingress) string {
	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
		if ing.Spec.DefaultBackend.Service.Name != components.VirtExportProxyServiceName {
			return ""
		}
		return ing.Spec.Rules[0].Host
	}
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service != nil && path.Backend.Service.Name == components.VirtExportProxyServiceName {
				if rule.Host != "" {
					return rule.Host
				}
			}
		}
	}
	return ""
}
