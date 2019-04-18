package utils

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdiuploadv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/upload/v1alpha1"
	cdiClientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
)

const (
	// UploadFileMD5 is the expected MD5 of the uploaded file
	UploadFileMD5 = "bf07a12664935c64c472e907e5cbce7e"

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
