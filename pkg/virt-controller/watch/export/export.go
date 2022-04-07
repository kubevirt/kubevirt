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
	"fmt"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	exportv1 "kubevirt.io/api/export/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/pkg/controller"
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
)

// variable so can be overridden in tests
var currentTime = func() *metav1.Time {
	t := metav1.Now()
	return &t
}

// VMExportController is resonsible for exporting VMs
type VMExportController struct {
	Client kubecli.KubevirtClient

	TemplateService services.TemplateService

	VMExportInformer cache.SharedIndexInformer
	PVCInformer      cache.SharedIndexInformer
	PodInformer      cache.SharedIndexInformer

	Recorder record.EventRecorder

	ResyncPeriod time.Duration

	vmExportQueue workqueue.RateLimitingInterface
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
	})
	ctrl.PVCInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePVC,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePVC(newObj) },
			DeleteFunc: ctrl.handlePVC,
		},
		ctrl.ResyncPeriod,
	)
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
			log.Log.V(1).Infof("Found key: %s", k)
			ctrl.vmExportQueue.Add(k)
		}
	}
}

func (ctrl *VMExportController) updateVMExport(vmExport *exportv1.VirtualMachineExport) (time.Duration, error) {
	log.Log.V(1).Infof("Updating VirtualMachineExport %s/%s", vmExport.Namespace, vmExport.Name)
	var retry time.Duration

	service, err := ctrl.getOrCreateExportService(vmExport)
	if err != nil {
		return 0, err
	}
	certSecret, err := ctrl.getOrCreateTokenCertSecret(vmExport)
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
		} else {
			pvcs = append(pvcs, &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmExport.Spec.Source.Name,
					Namespace: vmExport.Namespace,
				},
			})
		}
		var pod *corev1.Pod
		inUse, err := ctrl.isPVCInUse(vmExport, pvc)
		if err != nil {
			return retry, err
		}
		if !inUse {
			pod, err = ctrl.getOrCreateExporterPod(vmExport, pvcs, certSecret)
			if err != nil {
				return 0, err
			}
		}
		return ctrl.updateVMExportPvcStatus(vmExport, pvc, pod, service, certSecret, inUse)
	}
	return retry, nil
}

func (ctrl *VMExportController) getOrCreateTokenCertSecret(vmExport *exportv1.VirtualMachineExport) (*corev1.Secret, error) {
	secret, err := ctrl.Client.CoreV1().Secrets(vmExport.Namespace).Create(context.Background(), ctrl.createCertSecretManifest(vmExport), metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return nil, err
	} else if err != nil && errors.IsAlreadyExists(err) {
		// Secret already exists, set the name since we use it in other places.
		secret.Name = ctrl.getExportSecretName(vmExport)
		secret.Namespace = vmExport.Namespace
	}
	return secret, nil
}

func (ctrl *VMExportController) createCertSecretManifest(vmExport *exportv1.VirtualMachineExport) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrl.getExportSecretName(vmExport),
			Namespace: vmExport.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmExport, schema.GroupVersionKind{
					Group:   exportv1.SchemeGroupVersion.Group,
					Version: exportv1.SchemeGroupVersion.Version,
					Kind:    "VirtualMachineExport",
				}),
			},
		},
		StringData: map[string]string{
			"tls.crt": `-----BEGIN CERTIFICATE-----
MIIC8DCCAdigAwIBAgIUHFV1fE0pjwSu10Jsu0qCERKxzq0wDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTIyMDQwNTIwNDA1MFoXDTIyMDUw
NTIwNDA1MFowFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAvnGus/3zqeMj7F3kIm1IEeZYQ19CnAT1A1NzGRVpD4nu
NP7LEnYx3ZFhOmRxMvS+1Q7QccOTkO8YBS3Cx178DO96lpXOldZFgbK5iWsVbaCc
pWaGHV47t4XvBZwIe8I7ze3ucs6w/6gaQGTctDTeNSHwuJ8M20aPUVA2/T/Wgby8
lElCBVnWM/C+VmuhfylyfHfZdD2gHJof87EzDKkIWf9F1/tt+ba78iZr+17y3z4R
44CLjGX8LWhYGCVvZwkZH9ncRnf0MLJOlFXdcXjhPdJbfHiobL3OdLW542N3XeIo
cOwWy/q4pR8O7cCENBKI/pnIi7j5wwfFScaXcGRA4wIDAQABozowODAUBgNVHREE
DTALgglsb2NhbGhvc3QwCwYDVR0PBAQDAgeAMBMGA1UdJQQMMAoGCCsGAQUFBwMB
MA0GCSqGSIb3DQEBCwUAA4IBAQB26fqJwlc3a06+84aWjKOQ0TeENX+c7+Aw/ux+
D7HUKAxuxuU5GEXQuCFEnbi32FFHDxBE98bHeI9U0R2qfP067WGOqcHSfnNikZT1
CFU9T+iWaSB5EDi8nPUC2Mi7R14l36kdvwi9KNpg6WTXA37weXtfKNR9EtbIxS1x
DNry6TB9JbxMLgWKckbUP2X43aX8apMm+Hfk8/xl2mz0jy8fkalfB0/hvi2dL634
L9R/x4pLMIbVXMQabiK4W/G0LkKZBEi1paBmBR4Iwj0PMgj9wz6Ipz2eCNjCmdOM
zEIRWVbxcyj/1uiQ3mhEcqgRzplklmVpCQi0x1Bp2x43XTXC
-----END CERTIFICATE-----`,
			"tls.key": `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC+ca6z/fOp4yPs
XeQibUgR5lhDX0KcBPUDU3MZFWkPie40/ssSdjHdkWE6ZHEy9L7VDtBxw5OQ7xgF
LcLHXvwM73qWlc6V1kWBsrmJaxVtoJylZoYdXju3he8FnAh7wjvN7e5yzrD/qBpA
ZNy0NN41IfC4nwzbRo9RUDb9P9aBvLyUSUIFWdYz8L5Wa6F/KXJ8d9l0PaAcmh/z
sTMMqQhZ/0XX+235trvyJmv7XvLfPhHjgIuMZfwtaFgYJW9nCRkf2dxGd/Qwsk6U
Vd1xeOE90lt8eKhsvc50tbnjY3dd4ihw7BbL+rilHw7twIQ0Eoj+mciLuPnDB8VJ
xpdwZEDjAgMBAAECggEATupiv3krQCm8WBTsFQv9wlUWHAzcWDSBpvgsiKdjmqnI
SLOQSL0rmqnEhWLbuYbLkRQLcijd/D/nTzYQMXd9sIqH3OCE83gP41fBJF14Sq40
WyGpz3+d9UWNr2Bh746kI4hFt9NIaxgokKh7AD2sGo5O5uIZfL+3YbWAo96RL77j
O0fTYHs9Jeny97bKItt+I7drLCmuaSd48JYulLzW73pORe7tFeOtOwjVVzSgXJmF
/A5qpYhx2BPqkK28ytMsq+dXxngo4mJrWtT1RGkD1C9h5XwWDENKHAR29vGIls/b
JYUkaggI1Nqi/8c1SfGlDkty+nW077QPzhhQj92soQKBgQDvWA4ujidOk59u3Of1
dPUkrhZQ9pHixplI6qNjnf/rt9deo8KMZH/ys/KPEboILSA5J+t0lt5qZ10aseud
rWkRbhxP1xq1sDlbMVJdW+dBtkobbqKZCULLAclm2C56K7s8FzSrt4qNfHJJ6Dsg
keAZEleqlO/l2yPzv9WkL1ZLMwKBgQDLsngKoUNGaT8BRjKmt5VEQ7DLK28aE7m5
bPdOp8NsvP3HGDHggo0rw4hjaXhaAPPwh5TDNotfzq5Vf2k7Ho6HjCWQwRNAOzFq
CbFr4SKTJBHI1hPdoaCaf7qDmMDM4/G4MtaS8/fuyDnqnuGs9lI7vMfjaApYIdXt
oPqWZlWzkQKBgHkCrVDugIMi8jYMLJ8WvicIebH/qGze+ns6Xter98vHDHYGGAQB
gAtG3fll/gfKQQOE4m/1I4jqr9Eiab00Au5UHK5lVFTOP4GS41DeeYLo1nkeK8ly
PDoFsj10SbNtTuIn3XKAfuXgKKyjZNmnx4UFmBtf6BbwADJqKGs1n8yvAoGBAJJ/
krIidR4Ix5WFBRy+YA4umNImNMuOcD6Zzeu14GkuK16rWgPcIOfewxKsYjBpCwhs
mmMjsW2AWgWHkwk/2sZF1yaaldvWNp3Kxt2Nl643fMrynGsDuVwkjOHkVJWHQut1
NLmP2TrUqkLBbhFVPqNUDHbS9s2X2CIFavQMOYrhAoGBAIcysv8rhglALKkG62Fn
+FtHMoKlyw+vrqQETXd36EkH/umxXgCPioJhFuU7dc7/L8mEa/23Ft114wxa0tY4
xESkznjB8l9rc26g/VNYcjvu3IYDC/liE7OOWNmVpp7GZAAo1/Qf4JSwjX8slspH
Da+rJEHvvG7EFYTOvuqLD/jf
-----END PRIVATE KEY-----
`,
		},
	}
}

func (ctrl *VMExportController) isPVCInUse(vmExport *exportv1.VirtualMachineExport, pvc *corev1.PersistentVolumeClaim) (bool, error) {
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

func (ctrl *VMExportController) getExportSecretName(vmExport *exportv1.VirtualMachineExport) string {
	// TODO: Ensure name is not too long
	return fmt.Sprintf("%s-%s", exportPrefix, vmExport.Name)
}

func (ctrl *VMExportController) getExportServiceName(vmExport *exportv1.VirtualMachineExport) string {
	// TODO: Ensure name is not too long
	return fmt.Sprintf("%s-%s", exportPrefix, vmExport.Name)
}

func (ctrl *VMExportController) getOrCreateExportService(vmExport *exportv1.VirtualMachineExport) (*corev1.Service, error) {
	if service, err := ctrl.Client.CoreV1().Services(vmExport.Namespace).Get(context.Background(), ctrl.getExportServiceName(vmExport), metav1.GetOptions{}); err != nil && !errors.IsNotFound(err) {
		return nil, err
	} else if service != nil && err == nil {
		return service, nil
	}
	return ctrl.Client.CoreV1().Services(vmExport.Namespace).Create(context.Background(), ctrl.createServiceManifest(vmExport), metav1.CreateOptions{})
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

func (ctrl *VMExportController) getOrCreateExporterPod(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim, secret *corev1.Secret) (*corev1.Pod, error) {
	manifest := ctrl.createExporterPodManifest(vmExport, pvcs, secret)
	log.Log.V(3).Infof("Checking if pod exist: %s/%s", manifest.Namespace, manifest.Name)
	if pod, err := ctrl.Client.CoreV1().Pods(vmExport.Namespace).Get(context.Background(), manifest.Name, metav1.GetOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Log.V(3).Errorf("error %v", err)
		return nil, err
	} else if pod != nil && err == nil {
		log.Log.V(1).Infof("Found pod %s/%s", pod.Namespace, pod.Name)
		return pod, nil
	}
	log.Log.V(3).Infof("Creating new exporter pod %s/%s", manifest.Namespace, manifest.Name)
	return ctrl.Client.CoreV1().Pods(vmExport.Namespace).Create(context.Background(), manifest, metav1.CreateOptions{})
}

func (ctrl *VMExportController) createExporterPodManifest(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim, secret *corev1.Secret) *corev1.Pod {
	labels := make(map[string]string)
	labels[exportServiceLabel] = vmExport.Name
	podManifest := ctrl.TemplateService.RenderExporterManifest(vmExport, exportPrefix)
	podManifest.ObjectMeta.Labels = labels
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
		}
		ctrl.addVolumeEnvironmentVariables(&podManifest.Spec.Containers[0], pvc, i, mountPoint)
		podManifest.Spec.Volumes = append(podManifest.Spec.Volumes, corev1.Volume{
			Name: pvc.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		})
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
	})

	podManifest.Spec.Volumes = append(podManifest.Spec.Volumes, corev1.Volume{
		Name: ctrl.getExportSecretName(vmExport),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secret.Name,
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
		Name:      ctrl.getExportSecretName(vmExport),
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
			Value: filepath.Join(fmt.Sprintf("%s/%s/disk.img", urlBasePath, pvc.Name)),
		}, corev1.EnvVar{
			Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_GZIP_URI", index),
			Value: filepath.Join(fmt.Sprintf("%s/%s/disk.img.gz", urlBasePath, pvc.Name)),
		})
	} else {
		if ctrl.isKubevirtContentType(pvc) {
			exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_URI", index),
				Value: filepath.Join(fmt.Sprintf("%s/%s/disk.img", urlBasePath, pvc.Name)),
			}, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_GZIP_URI", index),
				Value: filepath.Join(fmt.Sprintf("%s/%s/disk.img.gz", urlBasePath, pvc.Name)),
			})
		} else {
			exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_ARCHIVE_URI", index),
				Value: filepath.Join(fmt.Sprintf("%s/%s/disk.tar.gz", urlBasePath, pvc.Name)),
			}, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_DIR_URI", index),
				Value: filepath.Join(fmt.Sprintf("%s/%s/dir", urlBasePath, pvc.Name)) + "/",
			})
		}
	}
}

func (ctrl *VMExportController) isKubevirtContentType(pvc *corev1.PersistentVolumeClaim) bool {
	ann := pvc.GetAnnotations()
	if ann == nil {
		return false
	}
	return ann[annContentType] == string(cdiv1.DataVolumeKubeVirt)
}

func (ctrl *VMExportController) updateVMExportPvcStatus(vmExport *exportv1.VirtualMachineExport, pvc *corev1.PersistentVolumeClaim, exporterPod *corev1.Pod, service *corev1.Service, secret *corev1.Secret, pvcInUse bool) (time.Duration, error) {
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

	if pvc == nil {
		log.Log.V(1).Info("PVC not found, updating status to not found")
		updateCondition(vmExportCopy.Status.Conditions, newPvcCondition(corev1.ConditionFalse, pvcNotFoundReason), true)
	} else {
		updateCondition(vmExportCopy.Status.Conditions, ctrl.pvcConditionFromPVC(pvc), true)
	}
	vmExportCopy.Status.Links = &exportv1.VirtualMachineExportLinks{}
	vmExportCopy.Status.Links.Internal = &exportv1.VirtualMachineExportLink{
		Volumes: []exportv1.VirtualMachineExportVolume{},
	}
	if ctrl.isKubevirtContentType(pvc) {
		vmExportCopy.Status.Links.Internal.Volumes = append(vmExportCopy.Status.Links.Internal.Volumes, exportv1.VirtualMachineExportVolume{
			Name: pvc.Name,
			Formats: []exportv1.VirtualMachineExportVolumeFormat{
				{
					Format: exportv1.KubeVirtRaw,
					Url:    fmt.Sprintf("https://%s.%s.svc/volumes/export-fs0/disk.img", service.Name, vmExport.Namespace),
				},
				{
					Format: exportv1.KubeVirtGz,
					Url:    fmt.Sprintf("https://%s.%s.svc/volumes/export-fs0/disk.img.gz", service.Name, vmExport.Namespace),
				},
			},
		})
	} else {
		vmExportCopy.Status.Links.Internal.Volumes = append(vmExportCopy.Status.Links.Internal.Volumes, exportv1.VirtualMachineExportVolume{
			Name: pvc.Name,
			Formats: []exportv1.VirtualMachineExportVolumeFormat{
				{
					Format: exportv1.Archive,
					Url:    fmt.Sprintf("https://%s.%s.svc/volumes/export-fs0/dir.tar.gz", service.Name, vmExport.Namespace),
				},
				{
					Format: exportv1.ArchiveGz,
					Url:    fmt.Sprintf("https://%s.%s.svc/volumes/export-fs0/dir/", service.Name, vmExport.Namespace),
				},
			},
		})
	}
	//	updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "Source does not exist"))
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

func (ctrl *VMExportController) isSourcePvc(source *exportv1.VirtualMachineExportSpec) bool {
	return source != nil && source.Source.APIGroup != nil && *source.Source.APIGroup == "v1" && source.Source.Kind == "PersistentVolumeClaim"
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

func (ctrl *VMExportController) pvcConditionFromPVC(pvc *corev1.PersistentVolumeClaim) exportv1.Condition {
	cond := exportv1.Condition{
		Type:               exportv1.ConditionPVC,
		LastTransitionTime: *currentTime(),
	}
	switch pvc.Status.Phase {
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
