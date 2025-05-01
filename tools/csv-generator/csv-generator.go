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

package main

import (
	"flag"
	"os"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/csv"
	"kubevirt.io/kubevirt/tools/util"
)

const (
	customImageExample   = "Examples: some.registry.com@sha256:abcdefghijklmnop, other.registry.com:tag1"
	shaEnvDeprecationMsg = "This argument is deprecated. Please use virt-*-image instead"
)

func main() {
	namespace := flag.String("namespace", "placeholder", "Namespace to use.")
	operatorImageVersion := flag.String("operatorImageVersion", "latest", "Image sha256 hash or image tag used to uniquely identify the operator container image to use in the CSV")
	imagePrefix := flag.String("imagePrefix", "", "Optional prefix for virt-* image names.")
	dockerPrefix := flag.String("dockerPrefix", "kubevirt", "Image Repository to use.")
	kubeVirtVersion := flag.String("kubeVirtVersion", "", "represents the KubeVirt releaseassociated with this CSV. Required when image SHAs are used.")
	pullPolicy := flag.String("pullPolicy", "IfNotPresent", "ImagePullPolicy to use.")
	verbosity := flag.String("verbosity", "2", "Verbosity level to use.")
	apiSha := flag.String("apiSha", "", "virt-api image sha. "+shaEnvDeprecationMsg)
	controllerSha := flag.String("controllerSha", "", "virt-controller image sha. "+shaEnvDeprecationMsg)
	handlerSha := flag.String("handlerSha", "", "virt-handler image sha. "+shaEnvDeprecationMsg)
	launcherSha := flag.String("launcherSha", "", "virt-launcher image sha. "+shaEnvDeprecationMsg)
	exportProxySha := flag.String("exportProxySha", "", "virt-exportproxy image sha. "+shaEnvDeprecationMsg)
	exportServerSha := flag.String("exportServerSha", "", "virt-exportserver image sha. "+shaEnvDeprecationMsg)
	synchronizationControllerSha := flag.String("synchronizationControllerSha", "", "virt-synchronization-controller image sha. "+shaEnvDeprecationMsg)
	gsSha := flag.String("gsSha", "", "libguestfs-tools image sha")
	prHelperSha := flag.String("prHelperSha", "", "pr-helper image sha")
	sidecarShimSha := flag.String("sidecarShimSha", "", "sidecar-shim image sha")
	runbookURLTemplate := flag.String("", "", "")
	kubeVirtLogo := flag.String("kubevirtLogo", "", "kubevirt logo data in base64")
	csvVersion := flag.String("csvVersion", "", "the CSV version being generated")
	replacesCsvVersion := flag.String("replacesCsvVersion", "", "the CSV version being replaced by this generated CSV")
	csvCreatedAtTimestamp := flag.String("csvCreatedAtTimestamp", "", "creation timestamp set in the 'createdAt' annotation on the CSV")
	dumpCRDs := flag.Bool("dumpCRDs", false, "dump CRDs along with CSV manifests to stdout")
	virtOperatorImage := flag.String("virt-operator-image", "", "custom image for virt-operator")
	virtApiImage := flag.String("virt-api-image", "", "custom image for virt-api. "+customImageExample)
	virtControllerImage := flag.String("virt-controller-image", "", "custom image for virt-controller. "+customImageExample)
	virtHandlerImage := flag.String("virt-handler-image", "", "custom image for virt-handler. "+customImageExample)
	virtLauncherImage := flag.String("virt-launcher-image", "", "custom image for virt-launcher. "+customImageExample)
	virtExportProxyImage := flag.String("virt-export-proxy-image", "", "custom image for virt-export-proxy. "+customImageExample)
	virtExportServerImage := flag.String("virt-export-server-image", "", "custom image for virt-export-server. "+customImageExample)
	virtSynchronizationControllerImage := flag.String("virt-synchronization-controller-image", "", "custom image for virt-synchronization-controller. "+customImageExample)
	gsImage := flag.String("gs-image", "", "custom image for gs. "+customImageExample)
	prHelperImage := flag.String("pr-helper-image", "", "custom image for pr-helper. "+customImageExample)
	sidecarShimImage := flag.String("sidecar-shim-image", "", "custom image for sidecar-shim. "+customImageExample)

	flag.Parse()

	csvData := csv.NewClusterServiceVersionData{
		Namespace:                          *namespace,
		KubeVirtVersion:                    *kubeVirtVersion,
		OperatorImageVersion:               *operatorImageVersion,
		DockerPrefix:                       *dockerPrefix,
		ImagePrefix:                        *imagePrefix,
		ImagePullPolicy:                    *pullPolicy,
		Verbosity:                          *verbosity,
		CsvVersion:                         *csvVersion,
		VirtApiSha:                         *apiSha,
		VirtControllerSha:                  *controllerSha,
		VirtHandlerSha:                     *handlerSha,
		VirtLauncherSha:                    *launcherSha,
		VirtExportProxySha:                 *exportProxySha,
		VirtExportServerSha:                *exportServerSha,
		VirtSynchronizationControllerSha:   *synchronizationControllerSha,
		GsSha:                              *gsSha,
		PrHelperSha:                        *prHelperSha,
		SidecarShimSha:                     *sidecarShimSha,
		RunbookURLTemplate:                 *runbookURLTemplate,
		ReplacesCsvVersion:                 *replacesCsvVersion,
		IconBase64:                         *kubeVirtLogo,
		Replicas:                           2,
		CreatedAtTimestamp:                 *csvCreatedAtTimestamp,
		VirtOperatorImage:                  *virtOperatorImage,
		VirtApiImage:                       *virtApiImage,
		VirtControllerImage:                *virtControllerImage,
		VirtHandlerImage:                   *virtHandlerImage,
		VirtLauncherImage:                  *virtLauncherImage,
		VirtExportProxyImage:               *virtExportProxyImage,
		VirtExportServerImage:              *virtExportServerImage,
		VirtSynchronizationControllerImage: *virtSynchronizationControllerImage,
		GsImage:                            *gsImage,
		PrHelperImage:                      *prHelperImage,
		SidecarShimImage:                   *sidecarShimImage,
	}

	operatorCsv, err := csv.NewClusterServiceVersion(&csvData)
	if err != nil {
		panic(err)
	}

	util.MarshallObject(operatorCsv, os.Stdout)

	if *dumpCRDs {
		kvCRD, err := components.NewKubeVirtCrd()
		if err != nil {
			panic(err)
		}
		util.MarshallObject(kvCRD, os.Stdout)
	}
}
