package utils

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	uploadcdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/uploadcontroller/v1alpha1"
	cdiClientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
)

const (
	// UploadFileMD5 is the expected MD5 of the uploaded file
	UploadFileMD5 = "2a7a52285c846314d1dbd79e9818270d"

	uploadTargetAnnotation = "cdi.kubevirt.io/storage.upload.target"
	uploadStatusAnnotation = "cdi.kubevirt.io/storage.pod.phase"

	tmpDir            = "/tmp/cdi-upload-test"
	imageFile         = "tinyCore.iso"
	imageDownloadPath = tmpDir + "/" + imageFile
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

// WaitPVCUploadPodStatusRunning waits for the upload server pod status annotation to be Running
func WaitPVCUploadPodStatusRunning(clientSet *kubernetes.Clientset, pvc *k8sv1.PersistentVolumeClaim) (bool, error) {
	return WaitForPVCAnnotationWithValue(clientSet, pvc.Namespace, pvc, uploadStatusAnnotation, string(k8sv1.PodRunning))
}

// RequestUploadToken sends an upload token request to the server
func RequestUploadToken(clientSet *cdiClientset.Clientset, pvc *k8sv1.PersistentVolumeClaim) (string, error) {
	request := &uploadcdiv1.UploadTokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-token",
			Namespace: pvc.Namespace,
		},
		Spec: uploadcdiv1.UploadTokenRequestSpec{
			PvcName: pvc.Name,
		},
	}

	response, err := clientSet.UploadV1alpha1().UploadTokenRequests(pvc.Namespace).Create(request)
	if err != nil {
		return "", err
	}

	return response.Status.Token, nil
}

// DownloadImageToNode downloads an image file to node01 in the cluster
func DownloadImageToNode(clientSet *kubernetes.Clientset, cliCommandPath string) error {
	RunGoCLICommand(cliCommandPath, "ssh", "node01", "rm -rf "+tmpDir)
	_, err := RunGoCLICommand(cliCommandPath, "ssh", "node01", "mkdir "+tmpDir)
	if err != nil {
		return err
	}

	fileServerService, err := GetServiceInNamespace(clientSet, FileHostNs, FileHostName)
	if err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("http://%s/%s", fileServerService.Spec.ClusterIP, imageFile)
	_, err = RunGoCLICommand(cliCommandPath, "ssh", "node01", fmt.Sprintf("curl -o %s %s", imageDownloadPath, downloadURL))
	if err != nil {
		return err
	}

	return nil
}

// UploadImageFromNode uploads the image to the upload proxy
func UploadImageFromNode(clientSet *kubernetes.Clientset, cliCommandPath, token string) error {
	uploadProxyService, err := GetServiceInNamespace(clientSet, "kube-system", "cdi-uploadproxy")
	if err != nil {
		return err
	}

	authHeader := "Authorization: Bearer " + token
	curlCommand := fmt.Sprintf("curl -v --insecure -H \"%s\" --data-binary @%s https://%s/v1alpha1/upload",
		authHeader, imageDownloadPath, uploadProxyService.Spec.ClusterIP)

	_, err = RunGoCLICommand(cliCommandPath, "ssh", "node01", curlCommand)
	if err != nil {
		return err
	}

	return nil
}
