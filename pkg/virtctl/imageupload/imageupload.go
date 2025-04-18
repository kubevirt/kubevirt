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
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
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
	usePopulatorAnnotation          = "cdi.kubevirt.io/storage.usePopulator"
	pvcPrimeNameAnnotation          = "cdi.kubevirt.io/storage.populator.pvcPrime"

	uploadReadyWaitInterval = 2 * time.Second

	processingWaitInterval = 2 * time.Second
	processingWaitTotal    = 24 * time.Hour

	//UploadProxyURIAsync is a URI of the upload proxy, the endpoint is asynchronous
	uploadProxyURIAsync = "/v1beta1/upload-async"

	//UploadProxyURI is a URI of the upload proxy, the endpoint is synchronous for backwards compatibility
	uploadProxyURI = "/v1beta1/upload"

	configName = "config"

	// ProvisioningFailed stores the 'ProvisioningFailed' event condition used for PVC error handling
	provisioningFailed = "ProvisioningFailed"
	// ErrClaimNotValid stores the 'ErrClaimNotValid' event condition used for DV error handling
	errClaimNotValid = "ErrClaimNotValid"

	// OptimisticLockErrorMsg is returned by kube-apiserver when trying to update an old version of a resource
	// https://github.com/kubernetes/kubernetes/blob/b89f564539fad77cd22de1b155d84638daf8c83f/staging/src/k8s.io/apiserver/pkg/registry/generic/registry/store.go#L240
	optimisticLockErrorMsg = "the object has been modified; please apply your changes to the latest version and try again"
)

// UploadProcessingCompleteFunc the function called while determining if post transfer processing is complete.
var UploadProcessingCompleteFunc = waitUploadProcessingComplete

// GetHTTPClientFn allows overriding the default http client (useful for unit testing)
var GetHTTPClientFn = GetHTTPClient

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
