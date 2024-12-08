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
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	pb "github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	uploadcdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/upload/v1beta1"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"

	instancetypeapi "kubevirt.io/api/instancetype"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
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
	UploadProxyURIAsync = "/v1beta1/upload-async"

	//UploadProxyURI is a URI of the upload proxy, the endpoint is synchronous for backwards compatibility
	UploadProxyURI = "/v1beta1/upload"

	configName = "config"

	// ProvisioningFailed stores the 'ProvisioningFailed' event condition used for PVC error handling
	ProvisioningFailed = "ProvisioningFailed"
	// ErrClaimNotValid stores the 'ErrClaimNotValid' event condition used for DV error handling
	ErrClaimNotValid = "ErrClaimNotValid"

	// OptimisticLockErrorMsg is returned by kube-apiserver when trying to update an old version of a resource
	// https://github.com/kubernetes/kubernetes/blob/b89f564539fad77cd22de1b155d84638daf8c83f/staging/src/k8s.io/apiserver/pkg/registry/generic/registry/store.go#L240
	OptimisticLockErrorMsg = "the object has been modified; please apply your changes to the latest version and try again"
)

type processingCompleteFunc func(kubernetes.Interface, *cobra.Command, string, string, time.Duration, time.Duration) error

// UploadProcessingCompleteFunc the function called while determining if post transfer processing is complete.
var UploadProcessingCompleteFunc processingCompleteFunc = waitUploadProcessingComplete

// GetHTTPClientFn allows overriding the default http client (useful for unit testing)
var GetHTTPClientFn = GetHTTPClient

// NewImageUploadCommand returns a cobra.Command for handling the uploading of VM images
func NewImageUploadCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	c := command{}
	cmd := &cobra.Command{
		Use:     "image-upload",
		Short:   "Upload a VM image to a DataVolume/PersistentVolumeClaim.",
		Example: usage(),
		Args:    cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, _, err := clientConfig.Namespace()
			if err != nil {
				return err
			}

			client, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
			if err != nil {
				return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
			}

			c.cmd = cmd
			c.client = client
			c.namespace = namespace
			return c.run(args)
		},
	}
	cmd.Flags().BoolVar(&c.insecure, "insecure", false, "Allow insecure server connections when using HTTPS.")
	cmd.Flags().StringVar(&c.uploadProxyURL, "uploadproxy-url", "", "The URL of the cdi-upload proxy service.")
	cmd.Flags().StringVar(&c.name, "pvc-name", "", "The destination DataVolume/PVC name.")
	cmd.Flags().StringVar(&c.pvcSize, "pvc-size", "", "The size of the PVC to create (ex. 10Gi, 500Mi).")
	cmd.Flags().StringVar(&c.size, "size", "", "The size of the DataVolume to create (ex. 10Gi, 500Mi).")
	cmd.Flags().StringVar(&c.storageClass, "storage-class", "", "The storage class for the PVC.")
	cmd.Flags().StringVar(&c.accessMode, "access-mode", "", "The access mode for the PVC.")
	cmd.Flags().BoolVar(&c.blockVolume, "block-volume", false, "Create a PVC with VolumeMode=Block (default is the storageProfile default. for archive upload default is filesystem).")
	cmd.Flags().StringVar(&c.volumeMode, "volume-mode", "", "Specify the VolumeMode (block/filesystem) used to create the PVC. Default is the storageProfile default. For archive upload default is filesystem.")
	cmd.Flags().StringVar(&c.imagePath, "image-path", "", "Path to the local VM image.")
	cmd.Flags().StringVar(&c.archivePath, "archive-path", "", "Path to the local archive.")
	cmd.Flags().BoolVar(&c.noCreate, "no-create", false, "Don't attempt to create a new DataVolume/PVC.")
	cmd.Flags().UintVar(&c.uploadPodWaitSecs, "wait-secs", 300, "Seconds to wait for upload pod to start.")
	cmd.Flags().UintVar(&c.uploadRetries, "retry", 5, "When upload server returns a transient error, we retry this number of times before giving up")
	cmd.Flags().BoolVar(&c.forceBind, "force-bind", false, "Force bind the PVC, ignoring the WaitForFirstConsumer logic.")
	cmd.Flags().BoolVar(&c.dataSource, "datasource", false, "Create a DataSource pointing to the created DataVolume/PVC.")
	cmd.Flags().StringVar(&c.defaultInstancetype, "default-instancetype", "", "The default instance type to associate with the image.")
	cmd.Flags().StringVar(&c.defaultInstancetypeKind, "default-instancetype-kind", "", "The default instance type kind to associate with the image.")
	cmd.Flags().StringVar(&c.defaultPreference, "default-preference", "", "The default preference to associate with the image.")
	cmd.Flags().StringVar(&c.defaultPreferenceKind, "default-preference-kind", "", "The default preference kind to associate with the image.")
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
	cmd                     *cobra.Command
	client                  kubecli.KubevirtClient
	insecure                bool
	uploadProxyURL          string
	name                    string
	namespace               string
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
	uploadPodWaitSecs       uint
	uploadRetries           uint
	blockVolume             bool
	noCreate                bool
	createPVC               bool
	forceBind               bool
	dataSource              bool
	archiveUpload           bool
}

func (c *command) parseArgs(args []string) error {
	if len(c.size) > 0 && len(c.pvcSize) > 0 && c.size != c.pvcSize {
		return fmt.Errorf("--pvc-size and --size can not be specified at the same time")
	}

	if len(c.pvcSize) > 0 {
		c.size = c.pvcSize
	}

	// check deprecated invocation
	if c.name != "" {
		if len(args) != 0 {
			return fmt.Errorf("cannot use --pvc-name and args")
		}

		c.createPVC = true

		return nil
	}

	// check deprecated blockVolume flag
	if c.blockVolume {
		if c.volumeMode == "" {
			c.volumeMode = "block"
		} else if c.volumeMode != "block" {
			return fmt.Errorf("incompatible --volume-mode '%s' and --block-volume", c.volumeMode)
		}
	}
	if c.volumeMode != "block" && c.volumeMode != "filesystem" && c.volumeMode != "" {
		return fmt.Errorf("Invalid volume mode '%s'. Valid values are 'block' and 'filesystem'.", c.volumeMode)
	}

	c.archiveUpload = false
	if c.imagePath == "" && c.archivePath == "" {
		return fmt.Errorf("either image-path or archive-path must be provided")
	} else if c.imagePath != "" && c.archivePath != "" {
		return fmt.Errorf("cannot handle both image-path and archive-path, provide only one")
	} else if c.archivePath != "" {
		c.archiveUpload = true
		c.imagePath = c.archivePath
		if c.volumeMode == "block" {
			return fmt.Errorf("In archive upload the volume mode should always be filesystem")
		}
	}

	if len(args) != 2 {
		return fmt.Errorf("expecting two args")
	}

	switch strings.ToLower(args[0]) {
	case "dv":
		c.createPVC = false
	case "pvc":
		c.createPVC = true
	default:
		return fmt.Errorf("invalid resource type %s", args[0])
	}

	c.name = args[1]

	return nil
}

func (c *command) validateDefaultInstancetypeArgs() error {
	if c.defaultInstancetype == "" && c.defaultInstancetypeKind != "" {
		return fmt.Errorf("--default-instancetype must be provided with --default-instancetype-kind")
	}
	if c.defaultPreference == "" && c.defaultPreferenceKind != "" {
		return fmt.Errorf("--default-preference must be provided with --default-preference-kind")
	}
	if (c.defaultInstancetype != "" || c.defaultPreference != "") && c.noCreate {
		return fmt.Errorf("--default-instancetype and --default-preference cannot be used with --no-create")
	}
	return nil
}

func (c *command) run(args []string) error {
	if err := c.parseArgs(args); err != nil {
		return err
	}

	if err := c.validateDefaultInstancetypeArgs(); err != nil {
		return err
	}

	// #nosec G304 No risk for path injection as this function executes with
	// the same privileges as those of virtctl user who supplies imagePath
	file, err := os.Open(c.imagePath)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(file, nil)

	pvc, err := c.getAndValidateUploadPVC()
	if err != nil {
		if !(k8serrors.IsNotFound(err) && !c.noCreate) {
			return err
		}

		if !c.noCreate && len(c.size) == 0 {
			return fmt.Errorf("when creating a resource, the size must be specified")
		}

		var obj metav1.Object

		if c.createPVC {
			obj, err = c.createUploadPVC()
			if err != nil {
				return err
			}
		} else {
			obj, err = c.createUploadDataVolume()
			if err != nil {
				return err
			}
		}

		c.cmd.Printf("%s %s/%s created\n", reflect.TypeOf(obj).Elem().Name(), obj.GetNamespace(), obj.GetName())
	} else {
		pvc, err = c.ensurePVCSupportsUpload(pvc)
		if err != nil {
			return err
		}

		c.cmd.Printf("Using existing PVC %s/%s\n", c.namespace, pvc.Name)
	}

	if c.createPVC {
		if err := c.waitUploadServerReady(); err != nil {
			return err
		}
	} else {
		if err := c.waitDvUploadScheduled(); err != nil {
			return err
		}
	}
	if c.uploadProxyURL == "" {
		c.uploadProxyURL, err = c.getUploadProxyURL()
		if err != nil {
			return err
		}
		if c.uploadProxyURL == "" {
			return fmt.Errorf("uploadproxy URL not found")
		}
	}

	u, err := url.Parse(c.uploadProxyURL)
	if err != nil {
		return err
	}

	if u.Scheme == "" {
		c.uploadProxyURL = fmt.Sprintf("https://%s", c.uploadProxyURL)
	}

	c.cmd.Printf("Uploading data to %s\n", c.uploadProxyURL)

	token, err := c.getUploadToken()
	if err != nil {
		return err
	}

	if err := c.uploadData(token, file); err != nil {
		return err
	}

	if c.dataSource {
		if err := c.handleDataSource(); err != nil {
			return err
		}
	}

	c.cmd.Println("Uploading data completed successfully, waiting for processing to complete, you can hit ctrl-c without interrupting the progress")
	err = UploadProcessingCompleteFunc(c.client, c.cmd, c.namespace, c.name, processingWaitInterval, processingWaitTotal)
	if err != nil {
		c.cmd.Printf("Timed out waiting for post upload processing to complete, please check upload pod status for progress\n")
	} else {
		c.cmd.Printf("Uploading %s completed successfully\n", c.imagePath)
	}

	return err
}

func GetHTTPClient(insecure bool) *http.Client {
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
	client := GetHTTPClientFn(insecure)
	req, _ := http.NewRequest("HEAD", u.String(), nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Async not available, use regular upload url.
		return ConstructUploadProxyPath(uploadProxyURL)
	}

	return u.String(), nil
}

func (c *command) uploadData(token string, file *os.File) error {
	uploadURL, err := ConstructUploadProxyPathAsync(c.uploadProxyURL, token, c.insecure)
	if err != nil {
		return err
	}

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	bar := pb.Full.Start64(fi.Size())
	bar.SetWriter(os.Stdout)
	bar.Set(pb.Bytes, true)
	reader := bar.NewProxyReader(file)

	client := GetHTTPClientFn(c.insecure)
	req, _ := http.NewRequest("POST", uploadURL, io.NopCloser(reader))

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/octet-stream")
	req.ContentLength = fi.Size()

	clientDo := func() error {
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		resp, err := client.Do(req)
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

	c.cmd.Println()
	bar.Start()

	retry := uint(0)
	for retry < c.uploadRetries {
		if err = clientDo(); err == nil {
			break
		}
		retry++
		if retry < c.uploadRetries {
			time.Sleep(time.Duration(retry*rand.UintN(50)) * time.Millisecond)
		}
	}

	bar.Finish()
	c.cmd.Println()

	if err != nil && retry == c.uploadRetries {
		return fmt.Errorf("error uploading image after %d retries: %w", c.uploadRetries, err)
	}

	return nil
}

func (c *command) getUploadToken() (string, error) {
	request := &uploadcdiv1.UploadTokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "token-for-virtctl",
		},
		Spec: uploadcdiv1.UploadTokenRequestSpec{
			PvcName: c.name,
		},
	}

	response, err := c.client.CdiClient().UploadV1beta1().UploadTokenRequests(c.namespace).Create(context.Background(), request, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return response.Status.Token, nil
}

func (c *command) waitDvUploadScheduled() error {
	loggedStatus := false
	//
	err := virtwait.PollImmediately(uploadReadyWaitInterval, time.Duration(c.uploadPodWaitSecs)*time.Second, func(ctx context.Context) (bool, error) {
		dv, err := c.client.CdiClient().CdiV1beta1().DataVolumes(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
		if err != nil {
			// DataVolume controller may not have created the DV yet ? TODO:
			if k8serrors.IsNotFound(err) {
				c.cmd.Printf("DV %s not found... \n", c.name)
				return false, nil
			}

			return false, err
		}

		if (dv.Status.Phase == cdiv1.WaitForFirstConsumer || dv.Status.Phase == cdiv1.PendingPopulation) && !c.forceBind {
			return false, fmt.Errorf("cannot upload to DataVolume in %s phase, make sure the PVC is Bound, or use force-bind flag", string(dv.Status.Phase))
		}

		done := dv.Status.Phase == cdiv1.UploadReady
		if !done {
			// We check events to provide user with pertinent error messages
			if err := c.handleEventErrors(dv.Status.ClaimName, c.name); err != nil {
				return false, err
			}
			if !loggedStatus {
				c.cmd.Printf("Waiting for PVC %s upload pod to be ready...\n", c.name)
				loggedStatus = true
			}
		}

		if done && loggedStatus {
			c.cmd.Printf("Pod now ready\n")
		}

		return done, nil
	})

	return err
}

func (c *command) waitUploadServerReady() error {
	loggedStatus := false

	err := virtwait.PollImmediately(uploadReadyWaitInterval, time.Duration(c.uploadPodWaitSecs)*time.Second, func(ctx context.Context) (bool, error) {
		pvc, err := c.client.CoreV1().PersistentVolumeClaims(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
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
			if err := c.handleEventErrors(c.name, c.name); err != nil {
				return false, err
			}
			if !loggedStatus {
				c.cmd.Printf("Waiting for PVC %s upload pod to be ready...\n", c.name)
				loggedStatus = true
			}
		}

		if done && loggedStatus {
			c.cmd.Printf("Pod now ready\n")
		}

		return done, nil
	})

	return err
}

func waitUploadProcessingComplete(client kubernetes.Interface, cmd *cobra.Command, namespace, name string, interval, timeout time.Duration) error {
	err := virtwait.PollImmediately(interval, timeout, func(ctx context.Context) (bool, error) {
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// upload controller sets this to true when uploadserver pod is ready to receive data
		podPhase := pvc.Annotations[PodPhaseAnnotation]

		if podPhase == string(v1.PodSucceeded) {
			cmd.Printf("Processing completed successfully\n")
		}

		return podPhase == string(v1.PodSucceeded), nil
	})

	return err
}

func (c *command) setDefaultInstancetypeLabels(target metav1.Object) {
	if target.GetLabels() == nil {
		target.SetLabels(make(map[string]string))
	}

	if c.defaultInstancetype != "" {
		target.GetLabels()[instancetypeapi.DefaultInstancetypeLabel] = c.defaultInstancetype
	}
	if c.defaultInstancetypeKind != "" {
		target.GetLabels()[instancetypeapi.DefaultInstancetypeKindLabel] = c.defaultInstancetypeKind
	}
	if c.defaultPreference != "" {
		target.GetLabels()[instancetypeapi.DefaultPreferenceLabel] = c.defaultPreference
	}
	if c.defaultPreferenceKind != "" {
		target.GetLabels()[instancetypeapi.DefaultPreferenceKindLabel] = c.defaultPreferenceKind
	}
}

func (c *command) createUploadDataVolume() (*cdiv1.DataVolume, error) {
	pvcSpec, err := c.createStorageSpec()
	if err != nil {
		return nil, err
	}

	// We check if the user-defined storageClass exists before attempting to create the dataVolume
	if c.storageClass != "" {
		_, err = c.client.StorageV1().StorageClasses().Get(context.Background(), c.storageClass, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	annotations := map[string]string{}
	if c.forceBind {
		annotations[forceImmediateBindingAnnotation] = ""
	}

	contentType := cdiv1.DataVolumeKubeVirt
	if c.archiveUpload {
		contentType = cdiv1.DataVolumeArchive
	}

	dv := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.name,
			Namespace:   c.namespace,
			Annotations: annotations,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: &cdiv1.DataVolumeSource{
				Upload: &cdiv1.DataVolumeSourceUpload{},
			},
			ContentType: contentType,
			Storage:     pvcSpec,
		},
	}
	c.setDefaultInstancetypeLabels(&dv.ObjectMeta)

	dv, err = c.client.CdiClient().CdiV1beta1().DataVolumes(c.namespace).Create(context.Background(), dv, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return dv, nil
}

func (c *command) createStorageSpec() (*cdiv1.StorageSpec, error) {
	quantity, err := resource.ParseQuantity(c.size)
	if err != nil {
		return nil, fmt.Errorf("validation failed for size=%s: %s", c.size, err)
	}

	spec := &cdiv1.StorageSpec{
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceStorage: quantity,
			},
		},
	}

	if c.storageClass != "" {
		spec.StorageClassName = &c.storageClass
	}

	if c.accessMode != "" {
		if c.accessMode == string(v1.ReadOnlyMany) {
			return nil, fmt.Errorf("cannot upload to a readonly volume, use either ReadWriteOnce or ReadWriteMany if supported")
		}
		spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.PersistentVolumeAccessMode(c.accessMode)}
	}

	switch c.volumeMode {
	case "block":
		volMode := v1.PersistentVolumeBlock
		spec.VolumeMode = &volMode
	case "filesystem":
		volMode := v1.PersistentVolumeFilesystem
		spec.VolumeMode = &volMode
	}

	return spec, nil
}

func (c *command) createUploadPVC() (*v1.PersistentVolumeClaim, error) {
	if c.accessMode == string(v1.ReadOnlyMany) {
		return nil, fmt.Errorf("cannot upload to a readonly volume, use either ReadWriteOnce or ReadWriteMany if supported")
	}

	// We check if the user-defined storageClass exists before attempting to create the PVC
	if c.storageClass != "" {
		_, err := c.client.StorageV1().StorageClasses().Get(context.Background(), c.storageClass, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	quantity, err := resource.ParseQuantity(c.size)
	if err != nil {
		return nil, fmt.Errorf("validation failed for size=%s: %s", c.size, err)
	}
	pvc := storagetypes.RenderPVC(&quantity, c.name, c.namespace, c.storageClass, c.accessMode, c.volumeMode == "block")
	if c.volumeMode == "filesystem" {
		volMode := v1.PersistentVolumeFilesystem
		pvc.Spec.VolumeMode = &volMode
	}

	contentType := string(cdiv1.DataVolumeKubeVirt)
	if c.archiveUpload {
		contentType = string(cdiv1.DataVolumeArchive)
	}

	annotations := map[string]string{
		uploadRequestAnnotation: "",
		contentTypeAnnotation:   contentType,
	}

	if c.forceBind {
		annotations[forceImmediateBindingAnnotation] = ""
	}

	pvc.ObjectMeta.Annotations = annotations
	c.setDefaultInstancetypeLabels(&pvc.ObjectMeta)

	pvc, err = c.client.CoreV1().PersistentVolumeClaims(c.namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return pvc, nil
}

func (c *command) ensurePVCSupportsUpload(pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	var err error
	_, hasAnnotation := pvc.Annotations[uploadRequestAnnotation]

	if !hasAnnotation {
		if pvc.GetAnnotations() == nil {
			pvc.SetAnnotations(make(map[string]string, 0))
		}
		pvc.Annotations[uploadRequestAnnotation] = ""
		pvc, err = c.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(context.Background(), pvc, metav1.UpdateOptions{})
		if err != nil {
			return nil, err
		}
	}

	return pvc, nil
}

func (c *command) getAndValidateUploadPVC() (*v1.PersistentVolumeClaim, error) {
	pvc, err := c.client.CoreV1().PersistentVolumeClaims(c.namespace).Get(context.Background(), c.name, metav1.GetOptions{})
	if err != nil {
		c.cmd.Printf("PVC %s/%s not found \n", c.namespace, c.name)
		return nil, err
	}

	if !c.createPVC {
		pvc, err = c.validateUploadDataVolume(pvc)
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
		return nil, fmt.Errorf("PVC %s already successfully imported/cloned/updated", c.name)
	}

	if !c.noCreate && !isUploadPVC {
		return nil, fmt.Errorf("PVC %s not available for upload", c.name)
	}

	// for PVCs that exist and the user wants to upload archive
	// the pvc has to have the contentType archive annotation
	if c.archiveUpload {
		contentType, found := pvc.Annotations[contentTypeAnnotation]
		if !found || contentType != string(cdiv1.DataVolumeArchive) {
			return nil, fmt.Errorf("PVC %s doesn't have archive contentType annotation", c.name)
		}
	}

	return pvc, nil
}

func (c *command) validateUploadDataVolume(pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	dv, err := c.client.CdiClient().CdiV1beta1().DataVolumes(pvc.Namespace).Get(context.Background(), c.name, metav1.GetOptions{})
	if err != nil {
		// When the PVC exists but the DV doesn't, there are two possible outcomes:
		if k8serrors.IsNotFound(err) {
			// 1. The DV was already garbage-collected. The PVC was created and populated by CDI as expected.
			if dvGarbageCollected := pvc.Annotations[deleteAfterCompletionAnnotation] == "true" &&
				pvc.Annotations[PodPhaseAnnotation] == string(v1.PodSucceeded); dvGarbageCollected {
				return nil, fmt.Errorf("DataVolume already garbage-collected: Assuming PVC %s/%s is successfully populated", pvc.Namespace, c.name)
			}
			// 2. The PVC was created independently of a DV.
			return nil, fmt.Errorf("No DataVolume is associated with the existing PVC %s/%s", pvc.Namespace, c.name)
		}
		return nil, err
	}

	// When using populators, the upload happens on the PVC Prime. We need to check it instead.
	if dv.Annotations[UsePopulatorAnnotation] == "true" {
		// We can assume the PVC is populated once it's bound
		if pvc.Status.Phase == v1.ClaimBound {
			return nil, fmt.Errorf("PVC %s already successfully populated", c.name)
		}
		// Get the PVC Prime since the upload is happening there
		pvcPrimeName, ok := pvc.Annotations[PVCPrimeNameAnnotation]
		if !ok {
			return nil, fmt.Errorf("Unable to get PVC Prime name from PVC %s/%s", pvc.Namespace, c.name)
		}
		pvc, err = c.client.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), pvcPrimeName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("Unable to get PVC Prime %s/%s", dv.Namespace, c.name)
		}
	}

	return pvc, nil
}

func (c *command) getUploadProxyURL() (string, error) {
	cdiConfig, err := c.client.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), configName, metav1.GetOptions{})
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
func (c *command) handleEventErrors(pvcName, dvName string) error {
	var pvcUID types.UID
	var dvUID types.UID

	eventList, err := c.client.CoreV1().Events(c.namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	if pvcName != "" {
		pvc, err := c.client.CoreV1().PersistentVolumeClaims(c.namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
		if !k8serrors.IsNotFound(err) {
			if err != nil {
				return err
			}
			pvcUID = pvc.GetUID()
		}
	}

	if dvName != "" {
		dv, err := c.client.CdiClient().CdiV1beta1().DataVolumes(c.namespace).Get(context.Background(), dvName, metav1.GetOptions{})
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

func (c *command) handleDataSource() error {
	ds, err := c.client.CdiClient().CdiV1beta1().DataSources(c.namespace).Get(context.Background(), c.name, metav1.GetOptions{})
	if err == nil {
		return c.updateExistingDataSource(ds)
	}

	if k8serrors.IsNotFound(err) {
		return c.createNewDataSource()
	}

	return err
}

func (c *command) createNewDataSource() error {
	ds := &cdiv1.DataSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.name,
			Namespace: c.namespace,
			Labels:    map[string]string{},
		},
		Spec: cdiv1.DataSourceSpec{
			Source: cdiv1.DataSourceSource{
				PVC: &cdiv1.DataVolumeSourcePVC{
					Name:      c.name,
					Namespace: c.namespace,
				},
			},
		},
	}
	c.setDefaultInstancetypeLabels(&ds.ObjectMeta)

	_, err := c.client.CdiClient().CdiV1beta1().DataSources(c.namespace).Create(context.Background(), ds, metav1.CreateOptions{})
	if err == nil {
		c.cmd.Printf("Created a new DataSource %s/%s\n", c.namespace, c.name)
	}
	return err
}

func (c *command) updateExistingDataSource(ds *cdiv1.DataSource) error {
	c.setDefaultInstancetypeLabels(&ds.ObjectMeta)

	patchBytes, err := patch.GeneratePatchPayload(
		patch.PatchOperation{
			Op:    patch.PatchReplaceOp,
			Path:  "/metadata/labels",
			Value: ds.Labels,
		},
		patch.PatchOperation{
			Op:   patch.PatchReplaceOp,
			Path: "/spec/source/pvc",
			Value: map[string]string{
				"name":      c.name,
				"namespace": c.namespace,
			},
		},
	)
	if err != nil {
		return err
	}

	if _, err = c.client.CdiClient().CdiV1beta1().DataSources(ds.Namespace).Patch(context.Background(), ds.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err == nil {
		c.cmd.Printf("Updated an existing DataSource %s/%s\n", ds.Namespace, ds.Name)
	}
	return err
}
