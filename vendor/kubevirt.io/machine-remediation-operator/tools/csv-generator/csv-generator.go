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
	"fmt"
	"io/ioutil"
	"os"

	corev1 "k8s.io/api/core/v1"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/operator/components"
	"kubevirt.io/machine-remediation-operator/tools/utils"
)

func main() {
	namespace := flag.String("namespace", "opensgift-machine-api", "Namespace to use.")
	repository := flag.String("repository", "index.docker.io/kubevirt", "Image Repository to use.")
	version := flag.String("version", "latest", "version to use.")
	pullPolicy := flag.String("pullPolicy", "IfNotPresent", "ImagePullPolicy to use.")
	verbosity := flag.String("verbosity", "2", "Verbosity level to use.")
	csvVersion := flag.String("csv-version", "0.0.0", "ClusterServiceVersion version.")
	csvPreviousVersion := flag.String("csv-previous-version", "", "ClusterServiceVersion version to replace.")
	dumpCRD := flag.Bool("dump-crd", false, "Dump operator CRD together with CSV to the stdout.")

	flag.Parse()

	imagePullPolicy := corev1.PullPolicy(*pullPolicy)
	data := &components.ClusterServiceVersionData{
		CSVVersion:         *csvVersion,
		ContainerPrefix:    *repository,
		ContainerTag:       *version,
		ImagePullPolicy:    imagePullPolicy,
		Namespace:          *namespace,
		ReplacesCSVVersion: *csvPreviousVersion,
		Verbosity:          *verbosity,
	}
	csv, err := components.NewClusterServiceVersion(data)
	if err != nil {
		panic(fmt.Errorf("failed to get CSV component: %v", err))
	}
	utils.MarshallObject(csv, os.Stdout)

	if *dumpCRD {
		crdFilePath := fmt.Sprintf("/data/%s_%s_%s.yaml", "machineremediation", mrv1.SchemeGroupVersion.Version, "machineremediationoperator")
		crdFile, err := ioutil.ReadFile(crdFilePath)
		if err != nil {
			panic(fmt.Errorf("failed to read CRD file: %v", err))
		}
		if _, err := os.Stdout.Write(crdFile); err != nil {
			panic(fmt.Errorf("failed to write CRD to stdout: %v", err))
		}
	}
}
