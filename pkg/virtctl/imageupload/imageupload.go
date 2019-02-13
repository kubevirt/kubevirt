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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package imageupload

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
	pb "gopkg.in/cheggaaa/pb.v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	uploadcdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/upload/v1alpha1"
	cdiClientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	// PodPhaseAnnotation is the annotation on a PVC containing the upload pod phase
	PodPhaseAnnotation = "cdi.kubevirt.io/storage.pod.phase"

	uploadRequestAnnotation = "cdi.kubevirt.io/storage.upload.target"

	uploadPodWaitInterval = 2 * time.Second

	//UploadProxyURI is a URI of the upoad proxy
	UploadProxyURI = "/v1alpha1/upload"
)

var (
	insecure       bool
	uploadProxyURL string
	pvcName        string
	pvcSize        string
	storageClass   string
	imagePath      string

	uploadPodWaitSecs uint = 60
	accessMode             = "ReadWriteOnce"
	noCreate               = false
)

// HTTPClientCreator is a function that creates http clients
type HTTPClientCreator func(bool) *http.Client

var httpClientCreatorFunc HTTPClientCreator

// SetHTTPClientCreator allows overriding the default http client
// useful for unit tests
func SetHTTPClientCreator(f HTTPClientCreator) {
	httpClientCreatorFunc = f
}

// SetDefaultHTTPClientCreator sets the http client creator back to default
func SetDefaultHTTPClientCreator() {
	httpClientCreatorFunc = getHTTPClient
}

func init() {
	SetDefaultHTTPClientCreator()
}

// NewImageUploadCommand returns a cobra.Command for handling the the uploading of VM images
func NewImageUploadCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "image-upload",
		Short:   "Upload a VM image to a PersistentVolumeClaim.",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := command{clientConfig: clientConfig}
			return v.run(cmd, args)
		},
	}
	cmd.Flags().BoolVar(&insecure, "insecure", insecure, "Allow insecure server connections when using HTTPS.")
	cmd.Flags().StringVar(&uploadProxyURL, "uploadproxy-url", "", "The URL of the cdi-upload proxy service.")
	cmd.MarkFlagRequired("uploadproxy-url")
	cmd.Flags().StringVar(&pvcName, "pvc-name", "", "The destination PVC.")
	cmd.MarkFlagRequired("pvc-name")
	cmd.Flags().StringVar(&pvcSize, "pvc-size", "", "The size of the PVC to create (ex. 10Gi, 500Mi).")
	cmd.Flags().StringVar(&storageClass, "storage-class", "", "The storage class for the PVC.")
	cmd.Flags().StringVar(&accessMode, "access-mode", accessMode, "The access mode for the PVC.")
	cmd.Flags().StringVar(&imagePath, "image-path", "", "Path to the local VM image.")
	cmd.MarkFlagRequired("image-path")
	cmd.Flags().BoolVar(&noCreate, "no-create", noCreate, "Don't attempt to create a new PVC.")
	cmd.Flags().UintVar(&uploadPodWaitSecs, "wait-secs", uploadPodWaitSecs, "Seconds to wait for upload pod to start.")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := `  # Upload a local disk image to a newly created PersistentVolumeClaim:
	virtctl image-upload --uploadproxy-url=https://cdi-uploadproxy.mycluster.com --pvc-name=upload-pvc --pvc-size=10Gi --image-path=/images/fedora28.qcow2`
	return usage
}

type command struct {
	clientConfig clientcmd.ClientConfig
}

func (c *command) run(cmd *cobra.Command, args []string) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	pvc, err := getAndValidateUploadPVC(virtClient, namespace, pvcName, noCreate)
	if err != nil {
		if !(k8serrors.IsNotFound(err) && !noCreate) {
			return err
		}

		pvc, err = createUploadPVC(virtClient, namespace, pvcName, pvcSize, storageClass, accessMode)
		if err != nil {
			return err
		}

		fmt.Printf("PVC %s/%s created\n", namespace, pvc.Name)
	} else {
		pvc, err = ensurePVCSupportsUpload(virtClient, pvc)
		if err != nil {
			return err
		}

		fmt.Printf("Using existing PVC %s/%s\n", namespace, pvc.Name)
	}

	err = waitUploadPodRunning(virtClient, namespace, pvcName, uploadPodWaitInterval, time.Duration(uploadPodWaitSecs)*time.Second)
	if err != nil {
		return err
	}

	token, err := getUploadToken(virtClient.CdiClient(), namespace, pvcName)
	if err != nil {
		return err
	}
	u, err := url.Parse(uploadProxyURL)
	if err != nil {
		return err
	} else if u.Scheme == "" {
		uploadProxyURL = fmt.Sprintf("http://%s", uploadProxyURL)
	}
	fmt.Printf("Uploading data to %s\n", uploadProxyURL)

	err = uploadData(uploadProxyURL, token, file, insecure)
	if err != nil {
		return err
	}

	fmt.Printf("Uploading %s completed successfully\n", imagePath)

	return nil
}

func getHTTPClient(insecure bool) *http.Client {
	client := &http.Client{}

	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return client
}

//ConstructUploadProxyPath - receives uploadproxy adress and concatenates to it URI
func ConstructUploadProxyPath(uploadProxyURL string) (string, error) {
	u, err := url.Parse(uploadProxyURL)

	if err != nil {
		return "", err
	}

	if !strings.Contains(uploadProxyURL, UploadProxyURI) {
		u.Path = path.Join(u.Path, UploadProxyURI)
	}

	return u.String(), nil
}

func uploadData(uploadProxyURL, token string, file *os.File, insecure bool) error {

	url, err := ConstructUploadProxyPath(uploadProxyURL)
	if err != nil {
		return err
	}

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	bar := pb.New64(fi.Size()).SetUnits(pb.U_BYTES)
	reader := bar.NewProxyReader(file)

	client := httpClientCreatorFunc(insecure)
	req, _ := http.NewRequest("POST", url, reader)

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/octet-stream")

	fmt.Println()
	bar.Start()

	resp, err := client.Do(req)

	bar.Finish()
	fmt.Println()

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected return value %d", resp.StatusCode)
	}

	return nil
}

func getUploadToken(client cdiClientset.Interface, namespace, name string) (string, error) {
	request := &uploadcdiv1.UploadTokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "token-for-virtctl",
		},
		Spec: uploadcdiv1.UploadTokenRequestSpec{
			PvcName: name,
		},
	}

	response, err := client.UploadV1alpha1().UploadTokenRequests(namespace).Create(request)
	if err != nil {
		return "", err
	}

	return response.Status.Token, nil
}

func waitUploadPodRunning(client kubernetes.Interface, namespace, name string, interval, timeout time.Duration) error {
	serviceName := "cdi-upload-" + name
	loggedStatus := false

	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		endpoints, err := client.CoreV1().Endpoints(namespace).Get(serviceName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		podPhase, _ := pvc.Annotations[PodPhaseAnnotation]

		done := false
		availableEndpoint := false
		for _, subset := range endpoints.Subsets {
			if len(subset.Addresses) > 0 {
				// we're looking to make sure the service endpoint has
				// the upload pod marked as being available, which means
				// that it is ready to accept connections
				availableEndpoint = true
				break
			}
		}
		running := (podPhase == string(v1.PodRunning))

		if running && availableEndpoint {
			done = true
		}

		if !done && !loggedStatus {
			fmt.Printf("Waiting for PVC %s upload pod to be running...\n", name)
			loggedStatus = true
		}

		if done && loggedStatus {
			// be really sure
			time.Sleep(interval)
			fmt.Printf("Pod now running\n")
		}

		return done, nil
	})

	return err
}

func createUploadPVC(client kubernetes.Interface, namespace, name, size, storageClass, accessMode string) (*v1.PersistentVolumeClaim, error) {
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}

	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				uploadRequestAnnotation: "",
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: quantity,
				},
			},
		},
	}

	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	if accessMode != "" {
		pvc.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.PersistentVolumeAccessMode(accessMode)}
	}

	pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)
	if err != nil {
		return nil, err
	}

	return pvc, nil
}

func ensurePVCSupportsUpload(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	var err error
	_, hasAnnotation := pvc.Annotations[uploadRequestAnnotation]

	if !hasAnnotation {
		pvc.Annotations[uploadRequestAnnotation] = ""
		pvc, err = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(pvc)
		if err != nil {
			return nil, err
		}
	}

	return pvc, nil
}

func getAndValidateUploadPVC(client kubernetes.Interface, namespace, name string, shouldExist bool) (*v1.PersistentVolumeClaim, error) {
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// for PVCs that exist, we ony want to use them if
	// 1. They have not already been used AND EITHER
	//   a. shouldExist is true
	//   b. shouldExist is false AND the upload annotation exists

	_, isUploadPVC := pvc.Annotations[uploadRequestAnnotation]
	podPhase, _ := pvc.Annotations[PodPhaseAnnotation]

	if podPhase == string(v1.PodSucceeded) {
		return nil, fmt.Errorf("PVC %s already successfully imported/cloned/updated", name)
	}

	if !shouldExist && !isUploadPVC {
		return nil, fmt.Errorf("PVC %s not available for upload", name)
	}

	return pvc, nil
}
