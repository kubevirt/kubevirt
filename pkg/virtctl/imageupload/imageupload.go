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
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strconv"
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

	cdiClientset "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	uploadcdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/upload/v1alpha1"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	// PodPhaseAnnotation is the annotation on a PVC containing the upload pod phase
	PodPhaseAnnotation = "cdi.kubevirt.io/storage.pod.phase"

	// PodReadyAnnotation tells whether the uploadserver pod is ready
	PodReadyAnnotation = "cdi.kubevirt.io/storage.pod.ready"

	uploadRequestAnnotation = "cdi.kubevirt.io/storage.upload.target"

	uploadReadyWaitInterval = 2 * time.Second

	processingWaitInterval = 2 * time.Second
	processingWaitTotal    = 24 * time.Hour

	//UploadProxyURIAsync is a URI of the upload proxy, the endpoint is asynchronous
	UploadProxyURIAsync = "/v1alpha1/upload-async"

	//UploadProxyURI is a URI of the upload proxy, the endpoint is synchronous for backwards compatibility
	UploadProxyURI = "/v1alpha1/upload"

	configName = "config"
)

var (
	insecure       bool
	uploadProxyURL string
	name           string
	size           string
	pvcSize        string
	storageClass   string
	imagePath      string
	accessMode     string

	uploadPodWaitSecs uint
	blockVolume       bool
	noCreate          bool
	createPVC         bool
)

// HTTPClientCreator is a function that creates http clients
type HTTPClientCreator func(bool) *http.Client

var httpClientCreatorFunc HTTPClientCreator

type processingCompleteFunc func(kubernetes.Interface, string, string, time.Duration, time.Duration) error

// UploadProcessingCompleteFunc the function called while determining if post transfer processing is complete.
var UploadProcessingCompleteFunc processingCompleteFunc = waitUploadProcessingComplete

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

// NewImageUploadCommand returns a cobra.Command for handling the uploading of VM images
func NewImageUploadCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "image-upload",
		Short:   "Upload a VM image to a DataVolume/PersistentVolumeClaim.",
		Example: usage(),
		Args:    cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := command{clientConfig: clientConfig}
			return v.run(cmd, args)
		},
	}
	cmd.Flags().BoolVar(&insecure, "insecure", false, "Allow insecure server connections when using HTTPS.")
	cmd.Flags().StringVar(&uploadProxyURL, "uploadproxy-url", "", "The URL of the cdi-upload proxy service.")
	cmd.Flags().StringVar(&name, "pvc-name", "", "DEPRECATED - The destination DataVolume/PVC name.")
	cmd.Flags().StringVar(&pvcSize, "pvc-size", "", "DEPRECATED - The size of the PVC to create (ex. 10Gi, 500Mi).")
	cmd.Flags().StringVar(&size, "size", "", "The size of the DataVolume to create (ex. 10Gi, 500Mi).")
	cmd.Flags().StringVar(&storageClass, "storage-class", "", "The storage class for the PVC.")
	cmd.Flags().StringVar(&accessMode, "access-mode", "ReadWriteOnce", "The access mode for the PVC.")
	cmd.Flags().BoolVar(&blockVolume, "block-volume", false, "Create a PVC with VolumeMode=Block (default Filesystem).")
	cmd.Flags().StringVar(&imagePath, "image-path", "", "Path to the local VM image.")
	cmd.MarkFlagRequired("image-path")
	cmd.Flags().BoolVar(&noCreate, "no-create", false, "Don't attempt to create a new DataVolume/PVC.")
	cmd.Flags().UintVar(&uploadPodWaitSecs, "wait-secs", 300, "Seconds to wait for upload pod to start.")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := `  # Upload a local disk image to a newly created DataVolume:
  {{ProgramName}} image-upload dv fedora-dv --size=10Gi --image-path=/images/fedora30.qcow2

  # Upload a local disk image to an existing DataVolume
  {{ProgramName}} image-upload dv fedora-dv --no-create --image-path=/images/fedora30.qcow2

  # Upload a local disk image to a newly created PersistentVolumeClaim
  {{ProgramName}} image-upload pvc fedora-pvc --size=10Gi --image-path=/images/fedora30.qcow2

  # Upload a local disk image to an existing PersistentVolumeClaim
  {{ProgramName}} image-upload pvc fedora-pvc --no-create --image-path=/images/fedora30.qcow2

  # Upload to a DataVolume with explicit URL to CDI Upload Proxy
  {{ProgramName}} image-upload dv fedora-dv --uploadproxy-url=https://cdi-uploadproxy.mycluster.com --image-path=/images/fedora30.qcow2`
	return usage
}

type command struct {
	clientConfig clientcmd.ClientConfig
}

func parseArgs(args []string) error {
	if len(size) > 0 && len(pvcSize) > 0 && size != pvcSize {
		return fmt.Errorf("--pvc-size deprecated, use --size")
	}

	if len(pvcSize) > 0 {
		size = pvcSize
	}

	// check deprecated invocation
	if name != "" {
		if len(args) != 0 {
			return fmt.Errorf("cannot use --pvc-name and args")
		}

		createPVC = true

		return nil
	}

	if len(args) != 2 {
		return fmt.Errorf("expecting two args")
	}

	switch strings.ToLower(args[0]) {
	case "dv":
		createPVC = false
	case "pvc":
		createPVC = true
	default:
		return fmt.Errorf("invalid resource type %s", args[0])
	}

	name = args[1]

	return nil
}

func (c *command) run(cmd *cobra.Command, args []string) error {
	if err := parseArgs(args); err != nil {
		return err
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(file, nil)

	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	pvc, err := getAndValidateUploadPVC(virtClient, namespace, name, noCreate)
	if err != nil {
		if !(k8serrors.IsNotFound(err) && !noCreate) {
			return err
		}

		if !noCreate && len(size) == 0 {
			return fmt.Errorf("when creating a resource, the size must be specified")
		}

		var obj metav1.Object

		if createPVC {
			obj, err = createUploadPVC(virtClient, namespace, name, size, storageClass, accessMode, blockVolume)
			if err != nil {
				return err
			}
		} else {
			obj, err = createUploadDataVolume(virtClient, namespace, name, size, storageClass, accessMode, blockVolume)
			if err != nil {
				return err
			}
		}

		fmt.Printf("%s %s/%s created\n", reflect.TypeOf(obj).Elem().Name(), obj.GetNamespace(), obj.GetName())
	} else {
		pvc, err = ensurePVCSupportsUpload(virtClient, pvc)
		if err != nil {
			return err
		}

		fmt.Printf("Using existing PVC %s/%s\n", namespace, pvc.Name)
	}

	err = waitUploadServerReady(virtClient, namespace, name, uploadReadyWaitInterval, time.Duration(uploadPodWaitSecs)*time.Second)
	if err != nil {
		return err
	}

	if uploadProxyURL == "" {
		uploadProxyURL, err = getUploadProxyURL(virtClient.CdiClient())
		if err != nil {
			return err
		}
		if uploadProxyURL == "" {
			return fmt.Errorf("uploadproxy URL not found")
		}
	}

	u, err := url.Parse(uploadProxyURL)
	if err != nil {
		return err
	}

	if u.Scheme == "" {
		uploadProxyURL = fmt.Sprintf("https://%s", uploadProxyURL)
	}

	fmt.Printf("Uploading data to %s\n", uploadProxyURL)

	token, err := getUploadToken(virtClient.CdiClient(), namespace, name)
	if err != nil {
		return err
	}

	err = uploadData(uploadProxyURL, token, file, insecure)
	if err != nil {
		return err
	}

	fmt.Println("Uploading data completed successfully, waiting for processing to complete, you can hit ctrl-c without interrupting the progress")
	err = UploadProcessingCompleteFunc(virtClient, namespace, name, processingWaitInterval, processingWaitTotal)
	if err != nil {
		fmt.Printf("Timed out waiting for post upload processing to complete, please check upload pod status for progress\n")
	} else {
		fmt.Printf("Uploading %s completed successfully\n", imagePath)
	}

	return err
}

func getHTTPClient(insecure bool) *http.Client {
	client := &http.Client{}

	if insecure {
		// #nosec cause: InsecureSkipVerify: true resolution: this method explicitly ask for insecure http client
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return client
}

//ConstructUploadProxyPath - receives uploadproxy address and concatenates to it URI
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

//ConstructUploadProxyPathAsync - receives uploadproxy address and concatenates to it URI
func ConstructUploadProxyPathAsync(uploadProxyURL, token string, insecure bool) (string, error) {
	u, err := url.Parse(uploadProxyURL)

	if err != nil {
		return "", err
	}

	if !strings.Contains(uploadProxyURL, UploadProxyURIAsync) {
		u.Path = path.Join(u.Path, UploadProxyURIAsync)
	}

	// Attempt to discover async URL
	client := httpClientCreatorFunc(insecure)
	req, _ := http.NewRequest("HEAD", u.String(), nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Async not available, use regular upload url.
		return ConstructUploadProxyPath(uploadProxyURL)
	}

	return u.String(), nil
}

func uploadData(uploadProxyURL, token string, file *os.File, insecure bool) error {
	url, err := ConstructUploadProxyPathAsync(uploadProxyURL, token, insecure)
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
	req.ContentLength = fi.Size()

	fmt.Println()
	bar.Start()

	resp, err := client.Do(req)

	bar.Finish()
	fmt.Println()

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected return value %d, %s", resp.StatusCode, string(body))
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

func waitUploadServerReady(client kubernetes.Interface, namespace, name string, interval, timeout time.Duration) error {
	loggedStatus := false

	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			// DataVolume controller may not have created the PVC yet
			if k8serrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		// upload controler sets this to true when uploadserver pod is ready to receive data
		podReady := pvc.Annotations[PodReadyAnnotation]
		done, _ := strconv.ParseBool(podReady)

		if !done && !loggedStatus {
			fmt.Printf("Waiting for PVC %s upload pod to be ready...\n", name)
			loggedStatus = true
		}

		if done && loggedStatus {
			fmt.Printf("Pod now ready\n")
		}

		return done, nil
	})

	return err
}

func waitUploadProcessingComplete(client kubernetes.Interface, namespace, name string, interval, timeout time.Duration) error {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// upload controler sets this to true when uploadserver pod is ready to receive data
		podPhase := pvc.Annotations[PodPhaseAnnotation]

		if podPhase == string(v1.PodSucceeded) {
			fmt.Printf("Processing completed successfully\n")
		}

		return podPhase == string(v1.PodSucceeded), nil
	})

	return err
}

func createUploadDataVolume(client kubecli.KubevirtClient, namespace, name, size, storageClass, accessMode string, blockVolume bool) (*cdiv1.DataVolume, error) {
	pvcSpec, err := createPVCSpec(size, storageClass, accessMode, blockVolume)
	if err != nil {
		return nil, err
	}

	dv := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				Upload: &cdiv1.DataVolumeSourceUpload{},
			},
			PVC: pvcSpec,
		},
	}

	dv, err = client.CdiClient().CdiV1alpha1().DataVolumes(namespace).Create(dv)
	if err != nil {
		return nil, err
	}

	return dv, nil
}

func createUploadPVC(client kubernetes.Interface, namespace, name, size, storageClass, accessMode string, blockVolume bool) (*v1.PersistentVolumeClaim, error) {
	pvcSpec, err := createPVCSpec(size, storageClass, accessMode, blockVolume)
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
		Spec: *pvcSpec,
	}

	pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)
	if err != nil {
		return nil, err
	}

	return pvc, nil
}

func createPVCSpec(size, storageClass, accessMode string, blockVolume bool) (*v1.PersistentVolumeClaimSpec, error) {
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, fmt.Errorf("validation failed for size=%s: %s", size, err)
	}

	spec := &v1.PersistentVolumeClaimSpec{
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceStorage: quantity,
			},
		},
	}

	if storageClass != "" {
		spec.StorageClassName = &storageClass
	}

	if accessMode != "" {
		spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.PersistentVolumeAccessMode(accessMode)}
	}

	if blockVolume {
		volMode := v1.PersistentVolumeBlock
		spec.VolumeMode = &volMode
	}

	return spec, nil
}

func ensurePVCSupportsUpload(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	var err error
	_, hasAnnotation := pvc.Annotations[uploadRequestAnnotation]

	if !hasAnnotation {
		if pvc.GetAnnotations() == nil {
			pvc.SetAnnotations(make(map[string]string, 0))
		}
		pvc.Annotations[uploadRequestAnnotation] = ""
		pvc, err = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(pvc)
		if err != nil {
			return nil, err
		}
	}

	return pvc, nil
}

func getAndValidateUploadPVC(client kubernetes.Interface, namespace, name string, shouldExist bool) (*v1.PersistentVolumeClaim, error) {
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// for PVCs that exist, we ony want to use them if
	// 1. They have not already been used AND EITHER
	//   a. shouldExist is true
	//   b. shouldExist is false AND the upload annotation exists

	_, isUploadPVC := pvc.Annotations[uploadRequestAnnotation]
	podPhase := pvc.Annotations[PodPhaseAnnotation]

	if podPhase == string(v1.PodSucceeded) {
		return nil, fmt.Errorf("PVC %s already successfully imported/cloned/updated", name)
	}

	if !shouldExist && !isUploadPVC {
		return nil, fmt.Errorf("PVC %s not available for upload", name)
	}

	return pvc, nil
}

func getUploadProxyURL(client cdiClientset.Interface) (string, error) {
	cdiConfig, err := client.CdiV1alpha1().CDIConfigs().Get(configName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if cdiConfig.Spec.UploadProxyURLOverride != nil {
		return *cdiConfig.Spec.UploadProxyURLOverride, nil
	}
	if cdiConfig.Status.UploadProxyURL != nil {
		return *cdiConfig.Status.UploadProxyURL, nil
	}
	return "", nil
}
