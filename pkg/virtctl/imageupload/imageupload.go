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
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

const (
	// podPhaseAnnotation is the annotation on a PVC containing the upload pod phase
	podPhaseAnnotation = "cdi.kubevirt.io/storage.pod.phase"

	// podReadyAnnotation tells whether the uploadserver pod is ready
	podReadyAnnotation = "cdi.kubevirt.io/storage.pod.ready"

	uploadRequestAnnotation         = "cdi.kubevirt.io/storage.upload.target"
	forceImmediateBindingAnnotation = "cdi.kubevirt.io/storage.bind.immediate.requested"
	contentTypeAnnotation           = "cdi.kubevirt.io/storage.contentType"
	deleteAfterCompletionAnnotation = "cdi.kubevirt.io/storage.deleteAfterCompletion"
	usePopulatorAnnotation          = "cdi.kubevirt.io/storage.usePopulator"
	pvcPrimeNameAnnotation          = "cdi.kubevirt.io/storage.populator.pvcPrime"

	uploadReadyWaitInterval = 2 * time.Second

	processingWaitInterval = 2 * time.Second
	processingWaitTotal    = 24 * time.Hour

	//uploadProxyURIAsync is a URI of the upload proxy, the endpoint is asynchronous
	uploadProxyURIAsync = "/v1beta1/upload-async"

	//uploadProxyURI is a URI of the upload proxy, the endpoint is synchronous for backwards compatibility
	uploadProxyURI = "/v1beta1/upload"

	configName = "config"

	// provisioningFailed stores the 'provisioningFailed' event condition used for PVC error handling
	provisioningFailed = "provisioningFailed"
	// errClaimNotValid stores the 'errClaimNotValid' event condition used for DV error handling
	errClaimNotValid = "errClaimNotValid"

	// optimisticLockErrorMsg is returned by kube-apiserver when trying to update an old version of a resource
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
