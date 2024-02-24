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
	"context"
	"crypto/tls"
	"fmt"
	"io"
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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	cdiClientset "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	uploadcdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/upload/v1beta1"

	instancetypeapi "kubevirt.io/api/instancetype"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	// PodPhaseAnnotation is the annotation on a PVC containing the upload pod phase
	PodPhaseAnnotation = "cdi.kubevirt.io/storage.pod.phase"

	// PodReadyAnnotation tells whether the uploadserver pod is ready
	PodReadyAnnotation = "cdi.kubevirt.io/storage.pod.ready"

	uploadRequestAnnotation         = "cdi.kubevirt.io/storage.upload.target"
	forceImmediateBindingAnnotation = "cdi.kubevirt.io/storage.bind.immediate.requested"
	contentTypeAnnotation           = "cdi.kubevirt.io/storage.contentType"
	deleteAfterCompletionAnnotation = "cdi.kubevirt.io/storage.deleteAfterCompletion"
	UsePopulatorAnnotation          = "cdi.kubevirt.io/storage.usePopulator"
	PVCPrimeNameAnnotation          = "cdi.kubevirt.io/storage.populator.pvcPrime"

	uploadReadyWaitInterval = 2 * time.Second

	processingWaitInterval = 2 * time.Second
	processingWaitTotal    = 24 * time.Hour

	//UploadProxyURIAsync is a URI of the upload proxy, the endpoint is asynchronous
	UploadProxyURIAsync = "/v1alpha1/upload-async"

	//UploadProxyURI is a URI of the upload proxy, the endpoint is synchronous for backwards compatibility
	UploadProxyURI = "/v1alpha1/upload"

	configName = "config"

	// ProvisioningFailed stores the 'ProvisioningFailed' event condition used for PVC error handling
	ProvisioningFailed = "ProvisioningFailed"
	// ErrClaimNotValid stores the 'ErrClaimNotValid' event condition used for DV error handling
	ErrClaimNotValid = "ErrClaimNotValid"

	// OptimisticLockErrorMsg is returned by kube-apiserver when trying to update an old version of a resource
	// https://github.com/kubernetes/kubernetes/blob/b89f564539fad77cd22de1b155d84638daf8c83f/staging/src/k8s.io/apiserver/pkg/registry/generic/registry/store.go#L240
	OptimisticLockErrorMsg = "the object has been modified; please apply your changes to the latest version and try again"
)

var (
	insecure                bool
	uploadProxyURL          string
	name                    string
	size                    string
	pvcSize                 string
	storageClass            string
	imagePath               string
	volumeMode              string
	archivePath             string
	accessMode              string
	defaultInstancetype     string
	defaultInstancetypeKind string
	defaultPreference       string
	defaultPreferenceKind   string

	uploadPodWaitSecs uint
	blockVolume       bool
	noCreate          bool
	createPVC         bool
	forceBind         bool
	archiveUpload     bool
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
			return v.run(args)
		},
	}
	cmd.Flags().BoolVar(&insecure, "insecure", false, "Allow insecure server connections when using HTTPS.")
	cmd.Flags().StringVar(&uploadProxyURL, "uploadproxy-url", "", "The URL of the cdi-upload proxy service.")
	cmd.Flags().StringVar(&name, "pvc-name", "", "The destination DataVolume/PVC name.")
	cmd.Flags().StringVar(&pvcSize, "pvc-size", "", "The size of the PVC to create (ex. 10Gi, 500Mi).")
	cmd.Flags().StringVar(&size, "size", "", "The size of the DataVolume to create (ex. 10Gi, 500Mi).")
	cmd.Flags().StringVar(&storageClass, "storage-class", "", "The storage class for the PVC.")
	cmd.Flags().StringVar(&accessMode, "access-mode", "", "The access mode for the PVC.")
	cmd.Flags().BoolVar(&blockVolume, "block-volume", false, "Create a PVC with VolumeMode=Block (default is the storageProfile default. for archive upload default is filesystem).")
	cmd.Flags().StringVar(&volumeMode, "volume-mode", "", "Specify the VolumeMode (block/filesystem) used to create the PVC. Default is the storageProfile default. For archive upload default is filesystem.")
	cmd.Flags().StringVar(&imagePath, "image-path", "", "Path to the local VM image.")
	cmd.Flags().StringVar(&archivePath, "archive-path", "", "Path to the local archive.")
	cmd.Flags().BoolVar(&noCreate, "no-create", false, "Don't attempt to create a new DataVolume/PVC.")
	cmd.Flags().UintVar(&uploadPodWaitSecs, "wait-secs", 300, "Seconds to wait for upload pod to start.")
	cmd.Flags().BoolVar(&forceBind, "force-bind", false, "Force bind the PVC, ignoring the WaitForFirstConsumer logic.")
	cmd.Flags().StringVar(&defaultInstancetype, "default-instancetype", "", "The default instance type to associate with the image.")
	cmd.Flags().StringVar(&defaultInstancetypeKind, "default-instancetype-kind", "", "The default instance type kind to associate with the image.")
	cmd.Flags().StringVar(&defaultPreference, "default-preference", "", "The default preference to associate with the image.")
	cmd.Flags().StringVar(&defaultPreferenceKind, "default-preference-kind", "", "The default preference kind to associate with the image.")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().MarkDeprecated("pvc-name", "specify the name as the second argument instead.")
	cmd.Flags().MarkDeprecated("pvc-size", "use --size instead.")
	cmd.Flags().MarkDeprecated("block-volume", "specify volume mode (filesystem/block) with --volume-mode instead.")
	return cmd
}

func usage() string {
	usage := `  # Upload a local disk image to a newly created DataVolume:
  {{ProgramName}} image-upload dv fedora-dv --size=10Gi --image-path=/images/fedora30.qcow2

  # Upload a local disk image to an existing DataVolume
  {{ProgramName}} image-upload dv fedora-dv --no-create --image-path=/images/fedora30.qcow2

  # Upload a local disk image to a newly created PersistentVolumeClaim
  {{ProgramName}} image-upload pvc fedora-pvc --size=10Gi --image-path=/images/fedora30.qcow2

  # Upload a local disk image to a newly created PersistentVolumeClaim and label it with a default instance type and preference
  {{ProgramName}} image-upload pvc fedora-pvc --size=10Gi --image-path=/images/fedora30.qcow2 --default-instancetype=n1.medium --default-preference=fedora

  # Upload a local disk image to an existing PersistentVolumeClaim
  {{ProgramName}} image-upload pvc fedora-pvc --no-create --image-path=/images/fedora30.qcow2

  # Upload to a DataVolume with explicit URL to CDI Upload Proxy
  {{ProgramName}} image-upload dv fedora-dv --uploadproxy-url=https://cdi-uploadproxy.mycluster.com --image-path=/images/fedora30.qcow2

  # Upload a local disk archive to a newly created DataVolume:
  {{ProgramName}} image-upload dv fedora-dv --size=10Gi --archive-path=/images/fedora30.tar`
	return usage
}

type command struct {
	clientConfig clientcmd.ClientConfig
}

func parseArgs(args []string) error {
	if len(size) > 0 && len(pvcSize) > 0 && size != pvcSize {
		return fmt.Errorf("--pvc-size and --size can not be specified at the same time")
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

	// check deprecated blockVolume flag
	if blockVolume {
		if volumeMode == "" {
			volumeMode = "block"
		} else if volumeMode != "block" {
			return fmt.Errorf("incompatible --volume-mode '%s' and --block-volume", volumeMode)
		}
	}
	if volumeMode != "block" && volumeMode != "filesystem" && volumeMode != "" {
		return fmt.Errorf("Invalid volume mode '%s'. Valid values are 'block' and 'filesystem'.", volumeMode)
	}

	archiveUpload = false
	if imagePath == "" && archivePath == "" {
		return fmt.Errorf("either image-path or archive-path must be provided")
	} else if imagePath != "" && archivePath != "" {
		return fmt.Errorf("cannot handle both image-path and archive-path, provide only one")
	} else if archivePath != "" {
		archiveUpload = true
		imagePath = archivePath
		if volumeMode == "block" {
			return fmt.Errorf("In archive upload the volume mode should always be filesystem")
		}
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

func validateDefaultInstancetypeArgs() error {
	if defaultInstancetype == "" && defaultInstancetypeKind != "" {
		return fmt.Errorf("--default-instancetype must be provided with --default-instancetype-kind")
	}
	if defaultPreference == "" && defaultPreferenceKind != "" {
		return fmt.Errorf("--default-preference must be provided with --default-preference-kind")
	}
	if (defaultInstancetype != "" || defaultPreference != "") && noCreate {
		return fmt.Errorf("--default-instancetype and --default-preference cannot be used with --no-create")
	}
	return nil
}

func (c *command) run(args []string) error {
	if err := parseArgs(args); err != nil {
		return err
	}

	if err := validateDefaultInstancetypeArgs(); err != nil {
		return err
	}

	// #nosec G304 No risk for path injection as this function executes with
	// the same privileges as those of virtctl user who supplies imagePath
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

	pvc, err := getAndValidateUploadPVC(virtClient, namespace, name, noCreate, archiveUpload)
	if err != nil {
		if !(k8serrors.IsNotFound(err) && !noCreate) {
			return err
		}

		if !noCreate && len(size) == 0 {
			return fmt.Errorf("when creating a resource, the size must be specified")
		}

		var obj metav1.Object

		if createPVC {
			obj, err = createUploadPVC(virtClient, namespace, name, size, storageClass, accessMode, volumeMode, archiveUpload)
			if err != nil {
				return err
			}
		} else {
			obj, err = createUploadDataVolume(virtClient, namespace, name, size, storageClass, accessMode, volumeMode, archiveUpload)
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

	if createPVC {
		err = waitUploadServerReady(virtClient, namespace, name, uploadReadyWaitInterval, time.Duration(uploadPodWaitSecs)*time.Second)
		if err != nil {
			return err
		}
	} else {
		err = waitDvUploadScheduled(virtClient, namespace, name, uploadReadyWaitInterval, time.Duration(uploadPodWaitSecs)*time.Second)
		if err != nil {
			return err
		}
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

// ConstructUploadProxyPath - receives uploadproxy address and concatenates to it URI
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

// ConstructUploadProxyPathAsync - receives uploadproxy address and concatenates to it URI
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
	req, _ := http.NewRequest("POST", url, io.NopCloser(reader))

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
		body, err := io.ReadAll(resp.Body)
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

	response, err := client.UploadV1beta1().UploadTokenRequests(namespace).Create(context.Background(), request, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return response.Status.Token, nil
}

func waitDvUploadScheduled(client kubecli.KubevirtClient, namespace, name string, interval, timeout time.Duration) error {
	loggedStatus := false
	//
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		dv, err := client.CdiClient().CdiV1beta1().DataVolumes(namespace).Get(context.Background(), name, metav1.GetOptions{})

		if err != nil {
			// DataVolume controller may not have created the DV yet ? TODO:
			if k8serrors.IsNotFound(err) {
				fmt.Printf("DV %s not found... \n", name)
				return false, nil
			}

			return false, err
		}

		if (dv.Status.Phase == cdiv1.WaitForFirstConsumer || dv.Status.Phase == cdiv1.PendingPopulation) && !forceBind {
			return false, fmt.Errorf("cannot upload to DataVolume in %s phase, make sure the PVC is Bound, or use force-bind flag", string(dv.Status.Phase))
		}

		done := dv.Status.Phase == cdiv1.UploadReady
		if !done {
			// We check events to provide user with pertinent error messages
			if err := handleEventErrors(client, dv.Status.ClaimName, name, namespace); err != nil {
				return false, err
			}
			if !loggedStatus {
				fmt.Printf("Waiting for PVC %s upload pod to be ready...\n", name)
				loggedStatus = true
			}
		}

		if done && loggedStatus {
			fmt.Printf("Pod now ready\n")
		}

		return done, nil
	})

	return err
}

func waitUploadServerReady(client kubecli.KubevirtClient, namespace, name string, interval, timeout time.Duration) error {
	loggedStatus := false

	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			// DataVolume controller may not have created the PVC yet
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		// upload controller sets this to true when uploadserver pod is ready to receive data
		podReady := pvc.Annotations[PodReadyAnnotation]
		done, _ := strconv.ParseBool(podReady)

		if !done {
			// We check events to provide user with pertinent error messages
			if err := handleEventErrors(client, name, name, namespace); err != nil {
				return false, err
			}
			if !loggedStatus {
				fmt.Printf("Waiting for PVC %s upload pod to be ready...\n", name)
				loggedStatus = true
			}
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
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// upload controller sets this to true when uploadserver pod is ready to receive data
		podPhase := pvc.Annotations[PodPhaseAnnotation]

		if podPhase == string(v1.PodSucceeded) {
			fmt.Printf("Processing completed successfully\n")
		}

		return podPhase == string(v1.PodSucceeded), nil
	})

	return err
}

func setDefaultInstancetypeLabels(labels map[string]string) {
	if defaultInstancetype != "" {
		labels[instancetypeapi.DefaultInstancetypeLabel] = defaultInstancetype
	}
	if defaultInstancetypeKind != "" {
		labels[instancetypeapi.DefaultInstancetypeKindLabel] = defaultInstancetypeKind
	}
	if defaultPreference != "" {
		labels[instancetypeapi.DefaultPreferenceLabel] = defaultPreference
	}
	if defaultPreferenceKind != "" {
		labels[instancetypeapi.DefaultPreferenceKindLabel] = defaultPreferenceKind
	}
}

func createUploadDataVolume(client kubecli.KubevirtClient, namespace, name, size, storageClass, accessMode, volumeMode string, archiveUpload bool) (*cdiv1.DataVolume, error) {
	pvcSpec, err := createStorageSpec(client, size, storageClass, accessMode, volumeMode)
	if err != nil {
		return nil, err
	}

	// We check if the user-defined storageClass exists before attempting to create the dataVolume
	if storageClass != "" {
		_, err = client.StorageV1().StorageClasses().Get(context.Background(), storageClass, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	annotations := map[string]string{}
	if forceBind {
		annotations[forceImmediateBindingAnnotation] = ""
	}

	labels := make(map[string]string)
	setDefaultInstancetypeLabels(labels)

	contentType := cdiv1.DataVolumeKubeVirt
	if archiveUpload {
		contentType = cdiv1.DataVolumeArchive
	}

	dv := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: &cdiv1.DataVolumeSource{
				Upload: &cdiv1.DataVolumeSourceUpload{},
			},
			ContentType: contentType,
			Storage:     pvcSpec,
		},
	}

	dv, err = client.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), dv, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return dv, nil
}

func createStorageSpec(client kubecli.KubevirtClient, size, storageClass, accessMode, volumeMode string) (*cdiv1.StorageSpec, error) {
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, fmt.Errorf("validation failed for size=%s: %s", size, err)
	}

	spec := &cdiv1.StorageSpec{
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
		if accessMode == string(v1.ReadOnlyMany) {
			return nil, fmt.Errorf("cannot upload to a readonly volume, use either ReadWriteOnce or ReadWriteMany if supported")
		}
		spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.PersistentVolumeAccessMode(accessMode)}
	}

	switch volumeMode {
	case "block":
		volMode := v1.PersistentVolumeBlock
		spec.VolumeMode = &volMode
	case "filesystem":
		volMode := v1.PersistentVolumeFilesystem
		spec.VolumeMode = &volMode
	}

	return spec, nil
}

func createUploadPVC(client kubecli.KubevirtClient, namespace, name, size, storageClass, accessMode, volumeMode string, archiveUpload bool) (*v1.PersistentVolumeClaim, error) {
	if accessMode == string(v1.ReadOnlyMany) {
		return nil, fmt.Errorf("cannot upload to a readonly volume, use either ReadWriteOnce or ReadWriteMany if supported")
	}

	// We check if the user-defined storageClass exists before attempting to create the PVC
	if storageClass != "" {
		_, err := client.StorageV1().StorageClasses().Get(context.Background(), storageClass, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, fmt.Errorf("validation failed for size=%s: %s", size, err)
	}
	pvc := storagetypes.RenderPVC(&quantity, name, namespace, storageClass, accessMode, volumeMode == "block")
	if volumeMode == "filesystem" {
		volMode := v1.PersistentVolumeFilesystem
		pvc.Spec.VolumeMode = &volMode
	}

	contentType := string(cdiv1.DataVolumeKubeVirt)
	if archiveUpload {
		contentType = string(cdiv1.DataVolumeArchive)
	}

	annotations := map[string]string{
		uploadRequestAnnotation: "",
		contentTypeAnnotation:   contentType,
	}

	if forceBind {
		annotations[forceImmediateBindingAnnotation] = ""
	}

	pvc.ObjectMeta.Annotations = annotations

	labels := make(map[string]string)
	setDefaultInstancetypeLabels(labels)
	pvc.ObjectMeta.Labels = labels

	pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return pvc, nil
}

func ensurePVCSupportsUpload(client kubernetes.Interface, pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	var err error
	_, hasAnnotation := pvc.Annotations[uploadRequestAnnotation]

	if !hasAnnotation {
		if pvc.GetAnnotations() == nil {
			pvc.SetAnnotations(make(map[string]string, 0))
		}
		pvc.Annotations[uploadRequestAnnotation] = ""
		pvc, err = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(context.Background(), pvc, metav1.UpdateOptions{})
		if err != nil {
			return nil, err
		}
	}

	return pvc, nil
}

func getAndValidateUploadPVC(client kubecli.KubevirtClient, namespace, name string, shouldExist, archiveUpload bool) (*v1.PersistentVolumeClaim, error) {
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("PVC %s/%s not found \n", namespace, name)
		return nil, err
	}

	if !createPVC {
		pvc, err = validateUploadDataVolume(client, pvc)
		if err != nil {
			return nil, err
		}
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

	// for PVCs that exist and the user wants to upload archive
	// the pvc has to have the contentType archive annotation
	if archiveUpload {
		contentType, found := pvc.Annotations[contentTypeAnnotation]
		if !found || contentType != string(cdiv1.DataVolumeArchive) {
			return nil, fmt.Errorf("PVC %s doesn't have archive contentType annotation", name)
		}
	}

	return pvc, nil
}

func validateUploadDataVolume(client kubecli.KubevirtClient, pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	dv, err := client.CdiClient().CdiV1beta1().DataVolumes(pvc.Namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		// When the PVC exists but the DV doesn't, there are two possible outcomes:
		if k8serrors.IsNotFound(err) {
			// 1. The DV was already garbage-collected. The PVC was created and populated by CDI as expected.
			if dvGarbageCollected := pvc.Annotations[deleteAfterCompletionAnnotation] == "true" &&
				pvc.Annotations[PodPhaseAnnotation] == string(v1.PodSucceeded); dvGarbageCollected {
				return nil, fmt.Errorf("DataVolume already garbage-collected: Assuming PVC %s/%s is successfully populated", pvc.Namespace, name)
			}
			// 2. The PVC was created independently of a DV.
			return nil, fmt.Errorf("No DataVolume is associated with the existing PVC %s/%s", pvc.Namespace, name)
		}
		return nil, err
	}

	// When using populators, the upload happens on the PVC Prime. We need to check it instead.
	if dv.Annotations[UsePopulatorAnnotation] == "true" {
		// We can assume the PVC is populated once it's bound
		if pvc.Status.Phase == v1.ClaimBound {
			return nil, fmt.Errorf("PVC %s already successfully populated", name)
		}
		// Get the PVC Prime since the upload is happening there
		pvcPrimeName, ok := pvc.Annotations[PVCPrimeNameAnnotation]
		if !ok {
			return nil, fmt.Errorf("Unable to get PVC Prime name from PVC %s/%s", pvc.Namespace, name)
		}
		pvc, err = client.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), pvcPrimeName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("Unable to get PVC Prime %s/%s", dv.Namespace, name)
		}
	}

	return pvc, nil
}

func getUploadProxyURL(client cdiClientset.Interface) (string, error) {
	cdiConfig, err := client.CdiV1beta1().CDIConfigs().Get(context.Background(), configName, metav1.GetOptions{})
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

// handleEventErrors checks PVC and DV-related events and, when encountered, returns appropriate errors
func handleEventErrors(client kubecli.KubevirtClient, pvcName, dvName, namespace string) error {
	var pvcUID types.UID
	var dvUID types.UID

	eventList, err := client.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	if pvcName != "" {
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
		if !k8serrors.IsNotFound(err) {
			if err != nil {
				return err
			}
			pvcUID = pvc.GetUID()
		}
	}

	if dvName != "" {
		dv, err := client.CdiClient().CdiV1beta1().DataVolumes(namespace).Get(context.Background(), dvName, metav1.GetOptions{})
		if !k8serrors.IsNotFound(err) {
			if err != nil {
				return err
			}
			dvUID = dv.GetUID()
		}
	}

	// TODO: Currently, we only check 'ProvisioningFailed' and 'ErrClaimNotValid' events.
	// If necessary, support more relevant errors
	for _, event := range eventList.Items {
		if event.InvolvedObject.Kind == "PersistentVolumeClaim" && event.InvolvedObject.UID == pvcUID {
			if event.Reason == ProvisioningFailed {
				if !strings.Contains(event.Message, OptimisticLockErrorMsg) {
					return fmt.Errorf("Provisioning failed: %s", event.Message)
				}
			}
		}
		if event.InvolvedObject.Kind == "DataVolume" && event.InvolvedObject.UID == dvUID {
			if event.Reason == ErrClaimNotValid {
				return fmt.Errorf("Claim not valid: %s", event.Message)
			}
		}
	}

	return nil
}
