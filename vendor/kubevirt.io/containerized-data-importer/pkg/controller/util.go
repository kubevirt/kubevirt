package controller

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/cert/triple"
	"k8s.io/klog"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	clientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/keys"
	"kubevirt.io/containerized-data-importer/pkg/operator"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const (
	// DataVolName provides a const to use for creating volumes in pod specs
	DataVolName = "cdi-data-vol"

	// CertVolName is the name of the volumecontaining certs
	CertVolName = "cdi-cert-vol"

	// ScratchVolName provides a const to use for creating scratch pvc volumes in pod specs
	ScratchVolName = "cdi-scratch-vol"

	// ImagePathName provides a const to use for creating volumes in pod specs
	ImagePathName  = "image-path"
	socketPathName = "socket-path"

	// SourceHTTP is the source type HTTP, if unspecified or invalid, it defaults to SourceHTTP
	SourceHTTP = "http"
	// SourceS3 is the source type S3
	SourceS3 = "s3"
	// SourceGlance is the source type of glance
	SourceGlance = "glance"
	// SourceNone means there is no source.
	SourceNone = "none"
	// SourceRegistry is the source type of Registry
	SourceRegistry = "registry"
)

type podDeleteRequest struct {
	namespace string
	podName   string
	podLister corelisters.PodLister
	k8sClient kubernetes.Interface
}

type pvcDeleteRequest struct {
	namespace string
	pvcName   string
	pvcLister corelisters.PersistentVolumeClaimLister
	k8sClient kubernetes.Interface
}

func checkPVC(pvc *v1.PersistentVolumeClaim, annotation string) bool {
	if pvc.DeletionTimestamp != nil {
		return false
	}
	// check if we have proper annotation
	if !metav1.HasAnnotation(pvc.ObjectMeta, annotation) {
		klog.V(2).Infof("pvc annotation %q not found, skipping pvc \"%s/%s\"\n", annotation, pvc.Namespace, pvc.Name)
		return false
	}

	return true
}

// returns the endpoint string which contains the full path URI of the target object to be copied.
func getEndpoint(pvc *v1.PersistentVolumeClaim) (string, error) {
	ep, found := pvc.Annotations[AnnEndpoint]
	if !found || ep == "" {
		verb := "empty"
		if !found {
			verb = "missing"
		}
		return ep, errors.Errorf("annotation %q in pvc \"%s/%s\" is %s\n", AnnEndpoint, pvc.Namespace, pvc.Name, verb)
	}
	return ep, nil
}

func getRequestedImageSize(pvc *v1.PersistentVolumeClaim) (string, error) {
	pvcSize, found := pvc.Spec.Resources.Requests[v1.ResourceStorage]
	if !found {
		return "", errors.Errorf("storage request is missing in pvc \"%s/%s\"", pvc.Namespace, pvc.Name)
	}
	return pvcSize.String(), nil
}

// returns the source string which determines the type of source. If no source or invalid source found, default to http
func getSource(pvc *v1.PersistentVolumeClaim) string {
	source, found := pvc.Annotations[AnnSource]
	if !found {
		source = ""
	}
	switch source {
	case
		SourceHTTP,
		SourceS3,
		SourceGlance,
		SourceNone,
		SourceRegistry:
		klog.V(2).Infof("pvc source annotation found for pvc \"%s/%s\", value %s\n", pvc.Namespace, pvc.Name, source)
	default:
		klog.V(2).Infof("No valid source annotation found for pvc \"%s/%s\", default to http\n", pvc.Namespace, pvc.Name)
		source = SourceHTTP
	}
	return source
}

// returns the source string which determines the type of source. If no source or invalid source found, default to http
func getContentType(pvc *v1.PersistentVolumeClaim) string {
	contentType, found := pvc.Annotations[AnnContentType]
	if !found {
		contentType = ""
	}
	switch contentType {
	case
		string(cdiv1.DataVolumeKubeVirt),
		string(cdiv1.DataVolumeArchive):
		klog.V(2).Infof("pvc content type annotation found for pvc \"%s/%s\", value %s\n", pvc.Namespace, pvc.Name, contentType)
	default:
		klog.V(2).Infof("No content type annotation found for pvc \"%s/%s\", default to kubevirt\n", pvc.Namespace, pvc.Name)
		contentType = string(cdiv1.DataVolumeKubeVirt)
	}
	return contentType
}

// returns the volumeMode which determines if the PVC is block PVC or not.
func getVolumeMode(pvc *v1.PersistentVolumeClaim) v1.PersistentVolumeMode {
	if pvc.Spec.VolumeMode != nil {
		return *pvc.Spec.VolumeMode
	}
	return v1.PersistentVolumeFilesystem
}

// returns the name of the secret containing endpoint credentials consumed by the importer pod.
// A value of "" implies there are no credentials for the endpoint being used. A returned error
// causes processNextItem() to stop.
func getSecretName(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim) (string, error) {
	ns := pvc.Namespace
	name, found := pvc.Annotations[AnnSecret]
	if !found || name == "" {
		msg := "getEndpointSecret: "
		if !found {
			msg += "annotation %q is missing in pvc \"%s/%s\""
		} else {
			msg += "secret name is missing from annotation %q in pvc \"%s/%s\""
		}
		klog.V(2).Infof(msg+"\n", AnnSecret, ns, pvc.Name)
		return "", nil // importer pod will not contain secret credentials
	}
	klog.V(3).Infof("getEndpointSecret: retrieving Secret \"%s/%s\"\n", ns, name)
	_, err := client.CoreV1().Secrets(ns).Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		klog.V(1).Infof("secret %q defined in pvc \"%s/%s\" is missing. Importer pod will run once this secret is created\n", name, ns, pvc.Name)
		return name, nil
	}
	if err != nil {
		return "", errors.Wrapf(err, "error getting secret %q defined in pvc \"%s/%s\"", name, ns, pvc.Name)
	}
	klog.V(1).Infof("retrieved secret %q defined in pvc \"%s/%s\"\n", name, ns, pvc.Name)
	return name, nil
}

// Update and return a copy of the passed-in pvc. Only one of the annotation or label maps is required though
// both can be passed.
// Note: the only pvc changes supported are annotations and labels.
func updatePVC(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim, anno, label map[string]string) (*v1.PersistentVolumeClaim, error) {
	klog.V(3).Infof("updatePVC: updating pvc \"%s/%s\" with anno: %+v and label: %+v", pvc.Namespace, pvc.Name, anno, label)
	applyUpdt := func(claim *v1.PersistentVolumeClaim, a, l map[string]string) {
		if a != nil {
			claim.ObjectMeta.Annotations = addToMap(claim.ObjectMeta.Annotations, a)
		}
		if l != nil {
			claim.ObjectMeta.Labels = addToMap(claim.ObjectMeta.Labels, l)
		}
	}

	var updtPvc *v1.PersistentVolumeClaim
	nsName := fmt.Sprintf("%s/%s", pvc.Namespace, pvc.Name)
	// don't mutate the passed-in pvc since it's likely from the shared informer
	pvcCopy := pvc.DeepCopy()

	// loop a few times in case the pvc is stale
	err := wait.PollImmediate(time.Second*1, time.Second*10, func() (bool, error) {
		var e error
		applyUpdt(pvcCopy, anno, label)
		updtPvc, e = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(pvcCopy)
		if e == nil {
			return true, nil // successful update
		}
		if k8serrors.IsConflict(e) { // pvc is likely stale
			klog.V(3).Infof("pvc %q is stale, re-trying\n", nsName)
			pvcCopy, e = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name, metav1.GetOptions{})
			if e == nil {
				return false, nil // retry update
			}
			// Get failed, start over
			pvcCopy = pvc.DeepCopy()
		}
		klog.Errorf("%q update/get error: %v\n", nsName, e)
		return false, nil // retry
	})

	if err == nil {
		klog.V(3).Infof("updatePVC: pvc %q updated", nsName)
		return updtPvc, nil
	}
	return pvc, errors.Wrapf(err, "error updating pvc %q\n", nsName)
}

// Sets an annotation `key: val` in the given pvc. Returns the updated pvc.
func setPVCAnnotation(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim, key, val string) (*v1.PersistentVolumeClaim, error) {
	klog.V(3).Infof("setPVCAnnotation: adding annotation \"%s: %s\" to pvc \"%s/%s\"\n", key, val, pvc.Namespace, pvc.Name)
	return updatePVC(client, pvc, map[string]string{key: val}, nil)
}

// checks if annotation `key` has a value of `val`.
func checkIfAnnoExists(pvc *v1.PersistentVolumeClaim, key string, val string) bool {
	value, exists := pvc.ObjectMeta.Annotations[key]
	if exists && value == val {
		return true
	}
	return false
}

// checks if particular label exists in pvc
func checkIfLabelExists(pvc *v1.PersistentVolumeClaim, lbl string, val string) bool {
	value, exists := pvc.ObjectMeta.Labels[lbl]
	if exists && value == val {
		return true
	}
	return false
}

// newScratchPersistentVolumeClaimSpec creates a new PVC based on the size of the passed in PVC.
// It also sets the appropriate OwnerReferences on the resource
// which allows handleObject to discover the pod resource that 'owns' it, and clean up when needed.
func newScratchPersistentVolumeClaimSpec(pvc *v1.PersistentVolumeClaim, pod *v1.Pod, storageClassName string) *v1.PersistentVolumeClaim {
	labels := map[string]string{
		"cdi-controller": pod.Name,
		"app":            "containerized-data-importer",
		LabelImportPvc:   pvc.Name,
	}

	pvcDef := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvc.Name + "-scratch",
			Namespace: pvc.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       pod.Name,
					UID:        pod.GetUID(),
				},
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources:   pvc.Spec.Resources,
		},
	}
	if storageClassName != "" {
		pvcDef.Spec.StorageClassName = &storageClassName
	}
	return pvcDef
}

// CreateScratchPersistentVolumeClaim creates and returns a pointer to a scratch PVC which is created based on the passed-in pvc and storage class name.
func CreateScratchPersistentVolumeClaim(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim, pod *v1.Pod, storageClassName string) (*v1.PersistentVolumeClaim, error) {
	ns := pvc.Namespace
	scratchPvcSpec := newScratchPersistentVolumeClaimSpec(pvc, pod, storageClassName)
	scratchPvc, err := client.CoreV1().PersistentVolumeClaims(ns).Create(scratchPvcSpec)
	if err != nil {
		return nil, errors.Wrap(err, "scratch PVC API create errored")
	}
	klog.V(3).Infof("scratch PVC \"%s/%s\" created\n", scratchPvc.Namespace, scratchPvc.Name)
	return scratchPvc, nil
}

// GetScratchPvcStorageClass tries to determine which storage class to use for use with a scratch persistent
// volume claim. The order of preference is the following:
// 1. Defined value in CDI config map.
// 2. If 1 is not available use the 'default' storage class.
// 3. If 2 is not available use the storage class name of the original pvc that will own the scratch pvc.
// 4. If none of those are available, return blank.
func GetScratchPvcStorageClass(client kubernetes.Interface, cdiclient clientset.Interface, pvc *v1.PersistentVolumeClaim) string {
	config, err := cdiclient.CdiV1alpha1().CDIConfigs().Get(common.ConfigName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Unable to find CDI configuration, %v\n", err)
	}
	storageClassName := config.Status.ScratchSpaceStorageClass
	if storageClassName == "" {
		// Unable to determine default storage class, attempt to read the storage class from the pvc.
		if pvc.Spec.StorageClassName != nil {
			storageClassName = *pvc.Spec.StorageClassName
			if storageClassName != "" {
				return storageClassName
			}
		}
	} else {
		return storageClassName
	}
	return ""
}

// CreateImporterPod creates and returns a pointer to a pod which is created based on the passed-in endpoint, secret
// name, and pvc. A nil secret means the endpoint credentials are not passed to the
// importer pod.
func CreateImporterPod(client kubernetes.Interface, image, verbose, pullPolicy string, podEnvVar *importPodEnvVar, pvc *v1.PersistentVolumeClaim, scratchPvcName *string) (*v1.Pod, error) {
	ns := pvc.Namespace
	pod := MakeImporterPodSpec(image, verbose, pullPolicy, podEnvVar, pvc, scratchPvcName)

	pod, err := client.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		return nil, errors.Wrap(err, "importer pod API create errored")
	}
	klog.V(3).Infof("importer pod \"%s/%s\" (image: %q) created\n", pod.Namespace, pod.Name, image)
	return pod, nil
}

// MakeImporterPodSpec creates and return the importer pod spec based on the passed-in endpoint, secret and pvc.
func MakeImporterPodSpec(image, verbose, pullPolicy string, podEnvVar *importPodEnvVar, pvc *v1.PersistentVolumeClaim, scratchPvcName *string) *v1.Pod {
	// importer pod name contains the pvc name
	podName := fmt.Sprintf("%s-%s-", common.ImporterPodName, pvc.Name)

	blockOwnerDeletion := true
	isController := true

	volumes := []v1.Volume{
		{
			Name: DataVolName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
					ReadOnly:  false,
				},
			},
		},
	}

	if scratchPvcName != nil {
		volumes = append(volumes, v1.Volume{
			Name: ScratchVolName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: *scratchPvcName,
					ReadOnly:  false,
				},
			},
		})
	}

	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: podName,
			Annotations: map[string]string{
				AnnCreatedBy: "yes",
			},
			Labels: map[string]string{
				common.CDILabelKey:       common.CDILabelValue,
				common.CDIComponentLabel: common.ImporterPodName,
				// this label is used when searching for a pvc's import pod.
				LabelImportPvc:         pvc.Name,
				common.PrometheusLabel: "",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					Kind:               "PersistentVolumeClaim",
					Name:               pvc.Name,
					UID:                pvc.GetUID(),
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &isController,
				},
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            common.ImporterPodName,
					Image:           image,
					ImagePullPolicy: v1.PullPolicy(pullPolicy),
					Args:            []string{"-v=" + verbose},
					Ports: []v1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8443,
							Protocol:      v1.ProtocolTCP,
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyOnFailure,
			Volumes:       volumes,
		},
	}

	ownerUID := pvc.UID
	if len(pvc.OwnerReferences) == 1 {
		ownerUID = pvc.OwnerReferences[0].UID
	}

	if getVolumeMode(pvc) == v1.PersistentVolumeBlock {
		pod.Spec.Containers[0].VolumeDevices = addVolumeDevices()
	} else {
		pod.Spec.Containers[0].VolumeMounts = addVolumeMounts()
	}

	if scratchPvcName != nil {
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
			Name:      ScratchVolName,
			MountPath: common.ScratchDataDir,
		})
	}

	pod.Spec.Containers[0].Env = makeEnv(podEnvVar, ownerUID)

	if podEnvVar.certConfigMap != "" {
		vm := v1.VolumeMount{
			Name:      CertVolName,
			MountPath: common.ImporterCertDir,
		}

		vol := v1.Volume{
			Name: CertVolName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: podEnvVar.certConfigMap,
					},
				},
			},
		}

		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, vm)
		pod.Spec.Volumes = append(pod.Spec.Volumes, vol)
	}
	return pod
}

// this is being called for pods using PV with block volume mode
func addVolumeDevices() []v1.VolumeDevice {
	volumeDevices := []v1.VolumeDevice{
		{
			Name:       DataVolName,
			DevicePath: common.ImporterWriteBlockPath,
		},
	}
	return volumeDevices
}

// this is being called for pods using PV with filesystem volume mode
func addVolumeMounts() []v1.VolumeMount {
	volumeMounts := []v1.VolumeMount{
		{
			Name:      DataVolName,
			MountPath: common.ImporterDataDir,
		},
	}
	return volumeMounts
}

// return the Env portion for the importer container.
func makeEnv(podEnvVar *importPodEnvVar, uid types.UID) []v1.EnvVar {
	env := []v1.EnvVar{
		{
			Name:  common.ImporterSource,
			Value: podEnvVar.source,
		},
		{
			Name:  common.ImporterEndpoint,
			Value: podEnvVar.ep,
		},
		{
			Name:  common.ImporterContentType,
			Value: podEnvVar.contentType,
		},
		{
			Name:  common.ImporterImageSize,
			Value: podEnvVar.imageSize,
		},
		{
			Name:  common.OwnerUID,
			Value: string(uid),
		},
		{
			Name:  common.InsecureTLSVar,
			Value: strconv.FormatBool(podEnvVar.insecureTLS),
		},
	}
	if podEnvVar.secretName != "" {
		env = append(env, v1.EnvVar{
			Name: common.ImporterAccessKeyID,
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: podEnvVar.secretName,
					},
					Key: common.KeyAccess,
				},
			},
		}, v1.EnvVar{
			Name: common.ImporterSecretKey,
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: podEnvVar.secretName,
					},
					Key: common.KeySecret,
				},
			},
		})

	}
	if podEnvVar.certConfigMap != "" {
		env = append(env, v1.EnvVar{
			Name:  common.ImporterCertDirVar,
			Value: common.ImporterCertDir,
		})
	}
	return env
}

// Return a new map consisting of map1 with map2 added. In general, map2 is expected to have a single key. eg
// a single annotation or label. If map1 has the same key as map2 then map2's value is used.
func addToMap(m1, m2 map[string]string) map[string]string {
	if m1 == nil {
		m1 = make(map[string]string)
	}
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}

// returns the CloneRequest string which contains the pvc name (and namespace) from which we want to clone the image.
func getCloneRequestPVC(pvc *v1.PersistentVolumeClaim) (string, error) {
	cr, found := pvc.Annotations[AnnCloneRequest]
	if !found || cr == "" {
		verb := "empty"
		if !found {
			verb = "missing"
		}
		return cr, errors.Errorf("annotation %q in pvc \"%s/%s\" is %s\n", AnnCloneRequest, pvc.Namespace, pvc.Name, verb)
	}
	return cr, nil
}

// ParseSourcePvcAnnotation parses out the annotations for a CDI PVC
func ParseSourcePvcAnnotation(sourcePvcAnno, del string) (namespace, name string) {
	strArr := strings.Split(sourcePvcAnno, del)
	if strArr == nil || len(strArr) < 2 {
		klog.V(3).Infof("Bad CloneRequest Annotation")
		return "", ""
	}
	return strArr[0], strArr[1]
}

// CreateCloneSourcePod creates our cloning src pod which will be used for out of band cloning to read the contents of the src PVC
func CreateCloneSourcePod(client kubernetes.Interface, image string, pullPolicy string, cr string, pvc *v1.PersistentVolumeClaim) (*v1.Pod, error) {
	sourcePvcNamespace, sourcePvcName := ParseSourcePvcAnnotation(cr, "/")
	if sourcePvcNamespace == "" || sourcePvcName == "" {
		return nil, errors.Errorf("Bad CloneRequest Annotation")
	}
	pod := MakeCloneSourcePodSpec(image, pullPolicy, sourcePvcName, pvc)
	pod, err := client.CoreV1().Pods(sourcePvcNamespace).Create(pod)
	if err != nil {
		return nil, errors.Wrap(err, "source pod API create errored")
	}
	klog.V(1).Infof("cloning source pod \"%s/%s\" (image: %q) created\n", pod.Namespace, pod.Name, image)
	return pod, nil
}

// MakeCloneSourcePodSpec creates and returns the clone source pod spec based on the target pvc.
func MakeCloneSourcePodSpec(image, pullPolicy, sourcePvcName string, pvc *v1.PersistentVolumeClaim) *v1.Pod {
	// source pod name contains the pvc name
	podName := fmt.Sprintf("%s-", common.ClonerSourcePodName)
	id := string(pvc.GetUID())
	blockOwnerDeletion := true
	isController := true

	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: podName,
			Annotations: map[string]string{
				AnnCreatedBy:          "yes",
				AnnTargetPodNamespace: pvc.Namespace,
			},
			Labels: map[string]string{
				common.CDILabelKey:       common.CDILabelValue, //filtered by the podInformer
				common.CDIComponentLabel: common.ClonerSourcePodName,
				common.CloningLabelKey:   common.CloningLabelValue + "-" + id, //used by podAffity
				// this label is used when searching for a pvc's cloner source pod.
				CloneUniqueID: id + "-source-pod",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					Kind:               "PersistentVolumeClaim",
					Name:               pvc.Name,
					UID:                pvc.GetUID(),
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &isController,
				},
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            common.ClonerSourcePodName,
					Image:           image,
					ImagePullPolicy: v1.PullPolicy(pullPolicy),
					SecurityContext: &v1.SecurityContext{
						Privileged: &[]bool{true}[0],
						RunAsUser:  &[]int64{0}[0],
					},

					VolumeMounts: []v1.VolumeMount{
						{
							Name:      ImagePathName,
							MountPath: common.ClonerImagePath,
						},
						{
							Name:      socketPathName,
							MountPath: common.ClonerSocketPath + "/" + id,
						},
					},
					Args: []string{"source", id},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name: ImagePathName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: sourcePvcName,
							ReadOnly:  false,
						},
					},
				},
				{
					Name: socketPathName,
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: common.ClonerSocketPath + "/" + id,
						},
					},
				},
			},
		},
	}
	return pod
}

// CreateCloneTargetPod creates our cloning tgt pod which will be used for out of band cloning to write the contents of the tgt PVC
func CreateCloneTargetPod(client kubernetes.Interface, image string, pullPolicy string,
	pvc *v1.PersistentVolumeClaim, podAffinityNamespace string) (*v1.Pod, error) {
	ns := pvc.Namespace
	pod := MakeCloneTargetPodSpec(image, pullPolicy, podAffinityNamespace, pvc)

	pod, err := client.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		return nil, errors.Wrap(err, "clone target pod API create errored")
	}
	klog.V(1).Infof("cloning target pod \"%s/%s\" (image: %q) created\n", pod.Namespace, pod.Name, image)
	return pod, nil
}

// MakeCloneTargetPodSpec creates and returns the clone target pod spec based on the target pvc.
func MakeCloneTargetPodSpec(image, pullPolicy, podAffinityNamespace string, pvc *v1.PersistentVolumeClaim) *v1.Pod {
	// target pod name contains the pvc name
	podName := fmt.Sprintf("%s-", common.ClonerTargetPodName)
	id := string(pvc.GetUID())
	blockOwnerDeletion := true
	isController := true
	ownerUID := pvc.UID
	if len(pvc.OwnerReferences) == 1 {
		ownerUID = pvc.OwnerReferences[0].UID
	}

	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: podName,
			Annotations: map[string]string{
				AnnCreatedBy:          "yes",
				AnnTargetPodNamespace: pvc.Namespace,
			},
			Labels: map[string]string{
				common.CDILabelKey:       common.CDILabelValue, //filtered by the podInformer
				common.CDIComponentLabel: common.ClonerTargetPodName,
				// this label is used when searching for a pvc's cloner target pod.
				CloneUniqueID:          id + "-target-pod",
				common.PrometheusLabel: "",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					Kind:               "PersistentVolumeClaim",
					Name:               pvc.Name,
					UID:                pvc.GetUID(),
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &isController,
				},
			},
		},
		Spec: v1.PodSpec{
			Affinity: &v1.Affinity{
				PodAffinity: &v1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      common.CloningLabelKey,
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{common.CloningLabelValue + "-" + id},
									},
								},
							},
							Namespaces:  []string{podAffinityNamespace}, //the scheduler looks for the namespace of the source pod
							TopologyKey: common.CloningTopologyKey,
						},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:            common.ClonerTargetPodName,
					Image:           image,
					ImagePullPolicy: v1.PullPolicy(pullPolicy),
					SecurityContext: &v1.SecurityContext{
						Privileged: &[]bool{true}[0],
						RunAsUser:  &[]int64{0}[0],
					},

					VolumeMounts: []v1.VolumeMount{
						{
							Name:      ImagePathName,
							MountPath: common.ClonerImagePath,
						},
						{
							Name:      socketPathName,
							MountPath: common.ClonerSocketPath + "/" + id,
						},
					},
					Args: []string{"target", id},
					Ports: []v1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8443,
							Protocol:      v1.ProtocolTCP,
						},
					},
					Env: []v1.EnvVar{
						{
							Name:  common.OwnerUID,
							Value: string(ownerUID),
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name: ImagePathName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.Name,
							ReadOnly:  false,
						},
					},
				},
				{
					Name: socketPathName,
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: common.ClonerSocketPath + "/" + id,
						},
					},
				},
			},
		},
	}
	return pod
}

// CreateUploadPod creates upload service pod manifest and sends to server
func CreateUploadPod(client kubernetes.Interface,
	caKeyPair *triple.KeyPair,
	clientCACert *x509.Certificate,
	image string,
	verbose string,
	pullPolicy string,
	name string,
	pvc *v1.PersistentVolumeClaim,
	scratchPvcName string) (*v1.Pod, error) {
	ns := pvc.Namespace
	commonName := name + "." + ns
	secretName := name + "-server-tls"

	pod := MakeUploadPodSpec(image, verbose, pullPolicy, name, pvc, scratchPvcName, secretName)

	pod, err := client.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			pod, err = client.CoreV1().Pods(ns).Get(name, metav1.GetOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "upload pod should exist but couldn't retrieve it")
			}
		} else {
			return nil, errors.Wrap(err, "upload pod API create errored")
		}
	}

	podOwner := MakePodOwnerReference(pod)
	_, err = keys.GetOrCreateServerKeyPairAndCert(client, ns, secretName, caKeyPair, clientCACert, commonName, name, &podOwner)
	if err != nil {
		// try to clean up
		client.CoreV1().Pods(ns).Delete(pod.Name, &metav1.DeleteOptions{})

		return nil, errors.Wrap(err, "Error creating server key pair")
	}

	klog.V(1).Infof("upload pod \"%s/%s\" (image: %q) created\n", pod.Namespace, pod.Name, image)

	return pod, nil
}

// MakePVCOwnerReference makes owner reference from a PVC
func MakePVCOwnerReference(pvc *v1.PersistentVolumeClaim) metav1.OwnerReference {
	blockOwnerDeletion := true
	isController := true
	return metav1.OwnerReference{
		APIVersion:         "v1",
		Kind:               "PersistentVolumeClaim",
		Name:               pvc.Name,
		UID:                pvc.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}
}

// MakePodOwnerReference makes owner reference from a Pod
func MakePodOwnerReference(pod *v1.Pod) metav1.OwnerReference {
	blockOwnerDeletion := true
	isController := true
	return metav1.OwnerReference{
		APIVersion:         "v1",
		Kind:               "Pod",
		Name:               pod.Name,
		UID:                pod.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}
}

// MakeUploadPodSpec creates upload service pod manifest
func MakeUploadPodSpec(image, verbose, pullPolicy, name string, pvc *v1.PersistentVolumeClaim, scratchName, secretName string) *v1.Pod {
	requestImageSize, _ := getRequestedImageSize(pvc)
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				annCreatedByUpload: "yes",
			},
			Labels: map[string]string{
				common.CDILabelKey:              common.CDILabelValue,
				common.CDIComponentLabel:        common.UploadServerCDILabel,
				common.UploadServerServiceLabel: name,
			},
			OwnerReferences: []metav1.OwnerReference{
				MakePVCOwnerReference(pvc),
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            common.UploadServerPodname,
					Image:           image,
					ImagePullPolicy: v1.PullPolicy(pullPolicy),
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      DataVolName,
							MountPath: common.UploadServerDataDir,
						},
						{
							Name:      ScratchVolName,
							MountPath: common.ScratchDataDir,
						},
					},
					Env: []v1.EnvVar{
						{
							Name: "TLS_KEY",
							ValueFrom: &v1.EnvVarSource{
								SecretKeyRef: &v1.SecretKeySelector{
									LocalObjectReference: v1.LocalObjectReference{
										Name: secretName,
									},
									Key: keys.KeyStoreTLSKeyFile,
								},
							},
						},
						{
							Name: "TLS_CERT",
							ValueFrom: &v1.EnvVarSource{
								SecretKeyRef: &v1.SecretKeySelector{
									LocalObjectReference: v1.LocalObjectReference{
										Name: secretName,
									},
									Key: keys.KeyStoreTLSCertFile,
								},
							},
						},
						{
							Name: "CLIENT_CERT",
							ValueFrom: &v1.EnvVarSource{
								SecretKeyRef: &v1.SecretKeySelector{
									LocalObjectReference: v1.LocalObjectReference{
										Name: secretName,
									},
									Key: keys.KeyStoreTLSCAFile,
								},
							},
						},
						{
							Name:  common.UploadImageSize,
							Value: requestImageSize,
						},
					},
					Args: []string{"-v=" + verbose},
					ReadinessProbe: &v1.Probe{
						Handler: v1.Handler{
							HTTPGet: &v1.HTTPGetAction{
								Path: "/healthz",
								Port: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 8080,
								},
							},
						},
						InitialDelaySeconds: 2,
						PeriodSeconds:       5,
					},
				},
			},
			RestartPolicy: v1.RestartPolicyOnFailure,
			Volumes: []v1.Volume{
				{
					Name: DataVolName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.Name,
							ReadOnly:  false,
						},
					},
				},
				{
					Name: ScratchVolName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: scratchName,
							ReadOnly:  false,
						},
					},
				},
			},
		},
	}
	return pod
}

// CreateUploadService creates upload service service manifest and sends to server
func CreateUploadService(client kubernetes.Interface, name string, pvc *v1.PersistentVolumeClaim) (*v1.Service, error) {
	ns := pvc.Namespace
	service := MakeUploadServiceSpec(name, pvc)

	service, err := client.CoreV1().Services(ns).Create(service)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			service, err = client.CoreV1().Services(ns).Get(name, metav1.GetOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "upload service should exist but couldn't retrieve it")
			}
		} else {
			return nil, errors.Wrap(err, "upload pod API create errored")
		}
	}
	klog.V(1).Infof("upload service \"%s/%s\" created\n", service.Namespace, service.Name)
	return service, nil
}

// MakeUploadServiceSpec creates upload service service manifest
func MakeUploadServiceSpec(name string, pvc *v1.PersistentVolumeClaim) *v1.Service {
	blockOwnerDeletion := true
	isController := true
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				annCreatedByUpload: "yes",
			},
			Labels: map[string]string{
				common.CDILabelKey:       common.CDILabelValue,
				common.CDIComponentLabel: common.UploadServerCDILabel,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					Kind:               "PersistentVolumeClaim",
					Name:               pvc.Name,
					UID:                pvc.GetUID(),
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &isController,
				},
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
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
				common.UploadServerServiceLabel: name,
			},
		},
	}
	return service
}

// EnsureCDIConfigExists creates an empty CDIConfig if necessary
func EnsureCDIConfigExists(client kubernetes.Interface, cdiClient clientset.Interface, name string) error {
	cfg := MakeEmptyCDIConfigSpec(name)

	err := operator.SetOwner(client, cfg)
	if err != nil {
		return errors.Wrap(err, "Error setting CDI config owner ref")
	}

	config, err := cdiClient.CdiV1alpha1().CDIConfigs().Create(cfg)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}

		return errors.Wrap(err, "CDI config create errored")
	}

	klog.V(1).Infof("CDI config \"%s\" created\n", config.Name)
	return nil
}

// MakeEmptyCDIConfigSpec creates cdi config manifest
func MakeEmptyCDIConfigSpec(name string) *cdiv1.CDIConfig {
	config := &cdiv1.CDIConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				common.CDILabelKey:       common.CDILabelValue,
				common.CDIComponentLabel: "",
			},
		},
	}
	return config

}

func updateCDIConfig(cdiClient clientset.Interface, config *cdiv1.CDIConfig) error {
	_, err := cdiClient.CdiV1alpha1().CDIConfigs().Update(config)
	if err != nil {
		return err
	}
	return nil
}

func deletePod(req podDeleteRequest) error {
	pod, err := req.podLister.Pods(req.namespace).Get(req.podName)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err == nil && pod.DeletionTimestamp == nil {
		err = req.k8sClient.CoreV1().Pods(req.namespace).Delete(req.podName, &metav1.DeleteOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		}
	}
	if err != nil {
		klog.V(1).Infof("error encountered deleting pod (%s): %s", req.podName, err.Error())
	}
	return err
}

func createImportEnvVar(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim) (*importPodEnvVar, error) {
	podEnvVar := &importPodEnvVar{}
	podEnvVar.source = getSource(pvc)
	podEnvVar.contentType = getContentType(pvc)

	var err error
	if podEnvVar.source != SourceNone {
		podEnvVar.ep, err = getEndpoint(pvc)
		if err != nil {
			return nil, err
		}
		podEnvVar.secretName, err = getSecretName(client, pvc)
		if err != nil {
			return nil, err
		}
		if podEnvVar.secretName == "" {
			klog.V(2).Infof("no secret will be supplied to endpoint %q\n", podEnvVar.ep)
		}
		podEnvVar.certConfigMap, err = getCertConfigMap(client, pvc)
		if err != nil {
			return nil, err
		}
		podEnvVar.insecureTLS, err = isInsecureTLS(client, pvc)
		if err != nil {
			return nil, err
		}
	}
	//get the requested image size.
	podEnvVar.imageSize, err = getRequestedImageSize(pvc)
	if err != nil {
		return nil, err
	}
	return podEnvVar, nil
}

func getCertConfigMap(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim) (string, error) {
	value, ok := pvc.Annotations[AnnCertConfigMap]
	if !ok || value == "" {
		return "", nil
	}

	_, err := client.CoreV1().ConfigMaps(pvc.Namespace).Get(value, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			klog.Warningf("Configmap %s does not exist, pod will not start until it does", value)
			return value, nil
		}

		return "", err
	}

	return value, nil
}

func getURLFromIngress(ing *extensionsv1beta1.Ingress, uploadProxyServiceName string) string {
	if ing.Spec.Backend != nil {
		if ing.Spec.Backend.ServiceName != uploadProxyServiceName {
			return ""
		}
		return ing.Spec.Rules[0].Host
	}
	for _, rule := range ing.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			if path.Backend.ServiceName == uploadProxyServiceName {
				if rule.Host != "" {
					return rule.Host
				}
			}
		}
	}
	return ""

}

func getURLFromRoute(route *routev1.Route, uploadProxyServiceName string) string {
	if route.Spec.To.Name == uploadProxyServiceName {
		if len(route.Status.Ingress) > 0 {
			return route.Status.Ingress[0].Host
		}
	}
	return ""

}

//IsOpenshift checks if we are on OpenShift platform
func IsOpenshift(client kubernetes.Interface) bool {
	result := client.Discovery().RESTClient().Get().AbsPath("/oapi/v1").Do()
	var statusCode int
	result.StatusCode(&statusCode)

	if result.Error() == nil {
		// It is OpenShift
		if statusCode == http.StatusOK {
			return true
		}
	} else {
		// Got 404 so this is not Openshift
		if statusCode == http.StatusNotFound {
			return false
		}
	}
	return false
}

func isInsecureTLS(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim) (bool, error) {
	var configMapName string

	value, ok := pvc.Annotations[AnnEndpoint]
	if !ok || value == "" {
		return false, nil
	}

	url, err := url.Parse(value)
	if err != nil {
		return false, err
	}

	switch url.Scheme {
	case "docker":
		configMapName = common.InsecureRegistryConfigMap
	default:
		return false, nil
	}

	cm, err := client.CoreV1().ConfigMaps(util.GetNamespace()).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	for host := range cm.Data {
		if host == url.Host {
			return true, nil
		}
	}

	return false, nil
}
