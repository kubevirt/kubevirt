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
	"encoding/base64"
	"fmt"
	"path"
	"time"

	corev1 "k8s.io/api/core/v1"
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

	exportServiceLabel = "export-service"

	exportPrefix = "virt-export"

	blockVolumeMountPath = "/dev/export-volumes"
	fileSystemMountPath  = "/export-volumes"
	urlBasePath          = "/volumes"

	// annContentType is an annotation on a PVC indicating the content type. This is populated by CDI.
	annContentType = "cdi.kubevirt.io/storage.contentType"

	caDefaultPath = "/etc/virt-controller/exportca"
	caCertFile    = caDefaultPath + "/tls.crt"
	caKeyFile     = caDefaultPath + "/tls.key"
	caBundle      = "ca-bundle"
	// name of certificate secret volume in pod
	certificates = "certificates"

	exporterPodFailedOrCompletedEvent = "ExporterPodFailedOrCompleted"
	exporterPodCreatedEvent           = "ExporterPodCreated"
	secretCreatedEvent                = "SecretCreated"
	serviceCreatedEvent               = "ServiceCreated"

	certExpiry = 30 // 30 hours
	deadline   = 24 // 24 hours
)

// variable so can be overridden in tests
var currentTime = func() *metav1.Time {
	t := metav1.Now()
	return &t
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

	VMExportInformer  cache.SharedIndexInformer
	PVCInformer       cache.SharedIndexInformer
	PodInformer       cache.SharedIndexInformer
	VMInformer        cache.SharedIndexInformer
	ConfigMapInformer cache.SharedIndexInformer

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
	ctrl.VMExportInformer.AddIndexers(cache.Indexers{
		"pvc": func(obj interface{}) ([]string, error) {
			vmExport, isObj := obj.(*exportv1.VirtualMachineExport)
			if !isObj {
				return nil, fmt.Errorf("object of type %T is not a VirtualMachineExport", obj)
			}
			return []string{controller.NamespacedKey(vmExport.Namespace, vmExport.Spec.Source.Name)}, nil
		},
		"pod": func(obj interface{}) ([]string, error) {
			vmExport, isObj := obj.(*exportv1.VirtualMachineExport)
			if !isObj {
				return nil, fmt.Errorf("object of type %T is not a VirtualMachineExport", obj)
			}
			return []string{controller.NamespacedKey(vmExport.Namespace, ctrl.getExportPodName(vmExport))}, nil
		},
	})
	ctrl.PVCInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePVC,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePVC(newObj) },
			DeleteFunc: ctrl.handlePVC,
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

func (ctrl *VMExportController) handlePVC(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		key, _ := cache.MetaNamespaceKeyFunc(pvc)
		log.Log.V(3).Infof("Processing PVC %s", key)
		keys, err := ctrl.VMExportInformer.GetIndexer().IndexKeys("pvc", key)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		for _, k := range keys {
			ctrl.vmExportQueue.Add(k)
		}
	}
}

func (ctrl *VMExportController) handlePod(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if pod, ok := obj.(*corev1.Pod); ok {
		key, _ := cache.MetaNamespaceKeyFunc(pod)
		log.Log.V(3).Infof("Processing POD %s", key)
		keys, err := ctrl.VMExportInformer.GetIndexer().IndexKeys("pod", key)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		for _, k := range keys {
			log.Log.V(1).Infof("Found key: %s", k)
			ctrl.vmExportQueue.Add(k)
		}
	}
}

func (ctrl *VMExportController) updateVMExport(vmExport *exportv1.VirtualMachineExport) (time.Duration, error) {
	log.Log.V(3).Infof("Updating VirtualMachineExport %s/%s", vmExport.Namespace, vmExport.Name)
	var retry time.Duration

	service, err := ctrl.getOrCreateExportService(vmExport)
	if err != nil {
		return 0, err
	}

	if ctrl.isSourcePvc(&vmExport.Spec) {
		pvc, err := ctrl.getPvc(vmExport.Namespace, vmExport.Spec.Source.Name)
		if err != nil {
			return 0, err
		}
		pvcs := make([]*corev1.PersistentVolumeClaim, 0)
		if pvc != nil {
			pvcs = append(pvcs, pvc)
		}
		inUse, err := ctrl.isPVCInUse(vmExport, pvc)
		if err != nil {
			return retry, err
		}
		var pod *corev1.Pod
		if !inUse && len(pvcs) > 0 {
			pod, err = ctrl.getOrCreateExporterPod(vmExport, pvcs)
			if err != nil {
				return 0, err
			} else if pod == nil {
				return retry, nil
			}

			_, err := ctrl.getOrCreateTokenCertSecret(vmExport, pod)
			if err != nil {
				return 0, err
			}
		}
		return ctrl.updateVMExportPvcStatus(vmExport, pvcs, pod, service, inUse)
	} else if ctrl.isSourceVM(&vmExport.Spec) {
		pvcs, err := ctrl.getPvcsFromVm(vmExport)
		if err != nil {
			return 0, err
		}
		anyInUse := false
		for _, pvc := range pvcs {
			inUse, err := ctrl.isPVCInUse(vmExport, pvc)
			if err != nil {
				return retry, err
			}
			if inUse {
				anyInUse = true
			}
		}
		var pod *corev1.Pod
		if !anyInUse && len(pvcs) > 0 {
			pod, err = ctrl.getOrCreateExporterPod(vmExport, pvcs)
			if err != nil {
				return 0, err
			} else if pod == nil {
				return retry, nil
			}
			_, err := ctrl.getOrCreateTokenCertSecret(vmExport, pod)
			if err != nil {
				return 0, err
			}
		}
		return ctrl.updateVMExportPvcStatus(vmExport, pvcs, pod, service, anyInUse)
	}
	return retry, nil
}

func (ctrl *VMExportController) getPvcsFromVm(vmExport *exportv1.VirtualMachineExport) ([]*corev1.PersistentVolumeClaim, error) {
	res := make([]*corev1.PersistentVolumeClaim, 0)
	item, exists, err := ctrl.VMInformer.GetStore().GetByKey(controller.NamespacedKey(vmExport.Namespace, vmExport.Spec.Source.Name))
	if err != nil {
		return res, err
	}
	if !exists {
		return res, fmt.Errorf("virtual machine %s/%s not found", vmExport.Namespace, vmExport.Spec.Source.Name)
	}
	vm, ok := item.(*virtv1.VirtualMachine)
	if !ok {
		return res, fmt.Errorf("%v not a Virtual Machine", item)
	}
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			pvc, err := ctrl.getPvc(vmExport.Namespace, volume.VolumeSource.PersistentVolumeClaim.ClaimName)
			if err != nil {
				return make([]*corev1.PersistentVolumeClaim, 0), err
			}
			res = append(res, pvc)
		} else if volume.VolumeSource.DataVolume != nil {
			pvc, err := ctrl.getPvc(vmExport.Namespace, volume.VolumeSource.DataVolume.Name)
			if err != nil {
				return make([]*corev1.PersistentVolumeClaim, 0), err
			}
			res = append(res, pvc)
		}
	}
	return res, nil
}

func (ctrl *VMExportController) getOrCreateTokenCertSecret(vmExport *exportv1.VirtualMachineExport, ownerPod *corev1.Pod) (*corev1.Secret, error) {
	secret, err := ctrl.Client.CoreV1().Secrets(vmExport.Namespace).Create(context.Background(), ctrl.createCertSecretManifest(vmExport, ownerPod), metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return nil, err
	} else if err != nil && errors.IsAlreadyExists(err) {
		// Secret already exists, set the name since we use it in other places.
		secret.Name = ctrl.getExportSecretName(ownerPod)
		secret.Namespace = vmExport.Namespace
	} else {
		log.Log.V(3).Infof("Created new exporter pod secret")
		ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, secretCreatedEvent, "Created exporter pod secret")
	}
	return secret, nil
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
		time.Hour*certExpiry,
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
	// TODO: Ensure name is not too long
	return fmt.Sprintf("%s-%s", exportPrefix, vmExport.Name)
}

func (ctrl *VMExportController) getExportPodName(vmExport *exportv1.VirtualMachineExport) string {
	// TODO: Ensure name is not too long
	return fmt.Sprintf("%s-%s", exportPrefix, vmExport.Name)
}

func (ctrl *VMExportController) getOrCreateExportService(vmExport *exportv1.VirtualMachineExport) (*corev1.Service, error) {
	if service, err := ctrl.Client.CoreV1().Services(vmExport.Namespace).Get(context.Background(), ctrl.getExportServiceName(vmExport), metav1.GetOptions{}); err != nil && !errors.IsNotFound(err) {
		return nil, err
	} else if service != nil && err == nil {
		return service, nil
	}
	service := ctrl.createServiceManifest(vmExport)
	log.Log.V(3).Infof("Creating new exporter service %s/%s", service.Namespace, service.Name)
	ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, serviceCreatedEvent, "Created service %s/%s", service.Namespace, service.Name)
	return ctrl.Client.CoreV1().Services(vmExport.Namespace).Create(context.Background(), service, metav1.CreateOptions{})
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

func (ctrl *VMExportController) getOrCreateExporterPod(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrl.getExportPodName(vmExport),
			Namespace: vmExport.Namespace,
		},
	}
	log.Log.V(3).Infof("Checking if pod exist: %s/%s", pod.Namespace, pod.Name)
	if pod, err := ctrl.Client.CoreV1().Pods(vmExport.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Log.V(3).Errorf("error %v", err)
		return nil, err
	} else if pod != nil && err == nil {
		log.Log.V(1).Infof("Found pod %s/%s", pod.Namespace, pod.Name)
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			// The server died or completed, delete the pod.
			ctrl.Recorder.Eventf(vmExport, corev1.EventTypeWarning, exporterPodFailedOrCompletedEvent, "Exporter pod %s/%s succeeded or failed", pod.Namespace, pod.Name)
			if err := ctrl.Client.CoreV1().Pods(vmExport.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{}); err != nil {
				return nil, err
			} else {
				return nil, nil
			}
		}
		return pod, nil
	}
	manifest := ctrl.createExporterPodManifest(vmExport, pvcs)

	log.Log.V(3).Infof("Creating new exporter pod %s/%s", manifest.Namespace, manifest.Name)
	ctrl.Recorder.Eventf(vmExport, corev1.EventTypeNormal, exporterPodCreatedEvent, "Created exporter pod %s/%s", pod.Namespace, pod.Name)
	return ctrl.Client.CoreV1().Pods(vmExport.Namespace).Create(context.Background(), manifest, metav1.CreateOptions{})
}

func (ctrl *VMExportController) createExporterPodManifest(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim) *corev1.Pod {
	podManifest := ctrl.TemplateService.RenderExporterManifest(vmExport, exportPrefix)
	podManifest.ObjectMeta.Labels = map[string]string{exportServiceLabel: vmExport.Name}
	podManifest.Spec.SecurityContext = &corev1.PodSecurityContext{
		RunAsUser: &[]int64{0}[0],
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
			// TODO should be outside else, once we support block volumes
			podManifest.Spec.Volumes = append(podManifest.Spec.Volumes, corev1.Volume{
				Name: pvc.Name,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvc.Name,
					},
				},
			})
		}
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
		Value: currentTime().Add(time.Hour * deadline).Format(time.RFC3339),
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
	ann := pvc.GetAnnotations()
	if ann == nil {
		return false
	}
	return ann[annContentType] == string(cdiv1.DataVolumeKubeVirt) || ann[annContentType] == ""
}

func (ctrl *VMExportController) updateVMExportPvcStatus(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim, exporterPod *corev1.Pod, service *corev1.Service, pvcInUse bool) (time.Duration, error) {
	var retry time.Duration
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
		log.Log.V(1).Info("PVC(s) not found, updating status to not found")
		updateCondition(vmExportCopy.Status.Conditions, newPvcCondition(corev1.ConditionFalse, pvcNotFoundReason), true)
	} else {
		updateCondition(vmExportCopy.Status.Conditions, ctrl.pvcConditionFromPVC(pvcs), true)
	}
	internalCert, err := ctrl.base64EncodeExportCa()
	if err != nil {
		return retry, err
	}
	vmExportCopy.Status.ServiceName = service.Name
	vmExportCopy.Status.Links = &exportv1.VirtualMachineExportLinks{}
	vmExportCopy.Status.Links.Internal = &exportv1.VirtualMachineExportLink{
		Volumes: []exportv1.VirtualMachineExportVolume{},
		Cert:    internalCert,
	}
	for _, pvc := range pvcs {
		if pvc != nil && exporterPod != nil && exporterPod.Status.Phase == corev1.PodRunning {
			const scheme = "https://"
			host := fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace)
			if ctrl.isKubevirtContentType(pvc) {
				vmExportCopy.Status.Links.Internal.Volumes = append(vmExportCopy.Status.Links.Internal.Volumes, exportv1.VirtualMachineExportVolume{
					Name: pvc.Name,
					Formats: []exportv1.VirtualMachineExportVolumeFormat{
						{
							Format: exportv1.KubeVirtRaw,
							Url:    scheme + path.Join(host, rawURI(pvc)),
						},
						{
							Format: exportv1.KubeVirtGz,
							Url:    scheme + path.Join(host, rawGzipURI(pvc)),
						},
					},
				})
			} else {
				vmExportCopy.Status.Links.Internal.Volumes = append(vmExportCopy.Status.Links.Internal.Volumes, exportv1.VirtualMachineExportVolume{
					Name: pvc.Name,
					Formats: []exportv1.VirtualMachineExportVolumeFormat{
						{
							Format: exportv1.Archive,
							Url:    scheme + path.Join(host, dirURI(pvc)),
						},
						{
							Format: exportv1.ArchiveGz,
							Url:    scheme + path.Join(host, archiveURI(pvc)),
						},
					},
				})
			}
		}
	}
	if !equality.Semantic.DeepEqual(vmExport, vmExportCopy) {
		if _, err := ctrl.Client.VirtualMachineExport(vmExportCopy.Namespace).Update(context.Background(), vmExportCopy, metav1.UpdateOptions{}); err != nil {
			return retry, err
		}
	}
	if vmExportCopy.Status.Phase == exportv1.Pending {
		log.Log.V(1).Info("Not ready requeueing")
		retry = time.Second
	}
	return retry, nil
}

func (ctrl *VMExportController) base64EncodeExportCa() (string, error) {
	key := controller.NamespacedKey(ctrl.KubevirtNamespace, components.KubeVirtExportCASecretName)
	ctrl.ConfigMapInformer.GetStore().GetByKey(key)
	obj, exists, err := ctrl.ConfigMapInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		log.DefaultLogger().Infof("CA Config Map not found, %v", ctrl.ConfigMapInformer.GetStore().ListKeys())
		return "", err
	}
	cm := obj.(*corev1.ConfigMap).DeepCopy()
	bundle := cm.Data[caBundle]
	return base64.StdEncoding.EncodeToString([]byte(bundle)), nil
}

func (ctrl *VMExportController) isSourcePvc(source *exportv1.VirtualMachineExportSpec) bool {
	return source != nil && source.Source.APIGroup != nil && *source.Source.APIGroup == corev1.SchemeGroupVersion.Group && source.Source.Kind == "PersistentVolumeClaim"
}

func (ctrl *VMExportController) isSourceVM(source *exportv1.VirtualMachineExportSpec) bool {
	return source != nil && source.Source.APIGroup != nil && *source.Source.APIGroup == virtv1.SchemeGroupVersion.Group && source.Source.Kind == "VirtualMachine"
}

func (ctrl *VMExportController) getPvc(namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.PVCInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, err
	}
	return obj.(*corev1.PersistentVolumeClaim).DeepCopy(), nil
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
