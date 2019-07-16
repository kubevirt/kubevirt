package utils

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	cdiuploadv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/upload/v1alpha1"
	cdiClientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
)

const (
	// UploadFile is the file to upload
	UploadFile = "./images/tinyCore.iso"

	// UploadFileSize is the size of UploadFile
	UploadFileSize = 18874368

	// UploadFileMD5 is the expected MD5 of the uploaded file
	UploadFileMD5 = "2a7a52285c846314d1dbd79e9818270d"

	// UploadFileMD5Extended is the size of the image after being extended
	UploadFileMD5Extended = "bbd634a97ffa672834717993b40e0ab7"

	uploadTargetAnnotation = "cdi.kubevirt.io/storage.upload.target"
	uploadStatusAnnotation = "cdi.kubevirt.io/storage.pod.phase"
)

// UploadPodName returns the name of the upload server pod associated with a PVC
func UploadPodName(pvc *k8sv1.PersistentVolumeClaim) string {
	return "cdi-upload-" + pvc.Name
}

// UploadPVCDefinition creates a PVC with the upload target annotation
func UploadPVCDefinition() *k8sv1.PersistentVolumeClaim {
	annotations := map[string]string{uploadTargetAnnotation: ""}
	return NewPVCDefinition("upload-test", "1G", annotations, nil)
}

// UploadBlockPVCDefinition creates a PVC with the upload target annotation for block PV
func UploadBlockPVCDefinition() *k8sv1.PersistentVolumeClaim {
	annotations := map[string]string{uploadTargetAnnotation: ""}
	return NewBlockPVCDefinition("upload-test", "500M", annotations, nil, "manual")
}

// WaitPVCUploadPodStatusRunning waits for the upload server pod status annotation to be Running
func WaitPVCUploadPodStatusRunning(clientSet *kubernetes.Clientset, pvc *k8sv1.PersistentVolumeClaim) (bool, error) {
	return WaitForPVCAnnotationWithValue(clientSet, pvc.Namespace, pvc, uploadStatusAnnotation, string(k8sv1.PodRunning))
}

// RequestUploadToken sends an upload token request to the server
func RequestUploadToken(clientSet *cdiClientset.Clientset, pvc *k8sv1.PersistentVolumeClaim) (string, error) {
	request := &cdiuploadv1alpha1.UploadTokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-token",
			Namespace: pvc.Namespace,
		},
		Spec: cdiuploadv1alpha1.UploadTokenRequestSpec{
			PvcName: pvc.Name,
		},
	}

	response, err := clientSet.UploadV1alpha1().UploadTokenRequests(pvc.Namespace).Create(request)
	if err != nil {
		return "", err
	}

	return response.Status.Token, nil
}
