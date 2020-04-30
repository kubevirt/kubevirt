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

	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/csv"
	"kubevirt.io/kubevirt/tools/util"
)

func main() {
	namespace := flag.String("namespace", "placeholder", "Namespace to use.")
	operatorImageVersion := flag.String("operatorImageVersion", "latest", "Image sha256 hash or image tag used to uniquely identify the operator container image to use in the CSV")
	imagePrefix := flag.String("imagePrefix", "", "Optional prefix for virt-* image names.")
	dockerPrefix := flag.String("dockerPrefix", "kubevirt", "Image Repository to use.")
	kubeVirtVersion := flag.String("kubeVirtVersion", "", "represents the KubeVirt releaseassociated with this CSV. Required when image SHAs are used.")
	pullPolicy := flag.String("pullPolicy", "IfNotPresent", "ImagePullPolicy to use.")
	verbosity := flag.String("verbosity", "2", "Verbosity level to use.")
	apiSha := flag.String("apiSha", "", "virt-api image sha")
	controllerSha := flag.String("controllerSha", "", "virt-controller image sha")
	handlerSha := flag.String("handlerSha", "", "virt-handler image sha")
	launcherSha := flag.String("launcherSha", "", "virt-launcher image sha")
	kubeVirtLogo := flag.String("kubevirtLogo", "", "kubevirt logo data in base64")
	csvVersion := flag.String("csvVersion", "", "the CSV version being generated")
	replacesCsvVersion := flag.String("replacesCsvVersion", "", "the CSV version being replaced by this generated CSV")
	csvCreatedAtTimestamp := flag.String("csvCreatedAtTimestamp", "", "creation timestamp set in the 'createdAt' annotation on the CSV")
	dumpCRDs := flag.Bool("dumpCRDs", false, "dump CRDs along with CSV manifests to stdout")

	flag.Parse()

	csvData := csv.NewClusterServiceVersionData{
		Namespace:            *namespace,
		KubeVirtVersion:      *kubeVirtVersion,
		OperatorImageVersion: *operatorImageVersion,
		DockerPrefix:         *dockerPrefix,
		ImagePrefix:          *imagePrefix,
		ImagePullPolicy:      *pullPolicy,
		Verbosity:            *verbosity,
		CsvVersion:           *csvVersion,
		VirtApiSha:           *apiSha,
		VirtControllerSha:    *controllerSha,
		VirtHandlerSha:       *handlerSha,
		VirtLauncherSha:      *launcherSha,
		ReplacesCsvVersion:   *replacesCsvVersion,
		IconBase64:           *kubeVirtLogo,
		Replicas:             2,
		CreatedAtTimestamp:   *csvCreatedAtTimestamp,
	}

	operatorCsv, err := csv.NewClusterServiceVersion(&csvData)
	if err != nil {
		panic(nil)
	}

	util.MarshallObject(operatorCsv, os.Stdout)

	if *dumpCRDs {
		kvCRD := components.NewKubeVirtCrd()
		util.MarshallObject(kvCRD, os.Stdout)
	}
}
