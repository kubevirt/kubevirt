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
	"os"

	"k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
	"kubevirt.io/kubevirt/tools/util"
)

func main() {
	resourceType := flag.String("type", "", "Type of resource to generate. vmi | vmipreset | vmirs | vm | vmim | kv | rbac")
	namespace := flag.String("namespace", "kube-system", "Namespace to use.")
	repository := flag.String("repository", "kubevirt", "Image Repository to use.")
	version := flag.String("version", "latest", "version to use.")
	pullPolicy := flag.String("pullPolicy", "IfNotPresent", "ImagePullPolicy to use.")
	verbosity := flag.String("verbosity", "2", "Verbosity level to use.")

	flag.Parse()

	imagePullPolicy := v1.PullPolicy(*pullPolicy)

	switch *resourceType {
	case "vmi":
		util.MarshallObject(components.NewVirtualMachineInstanceCrd(), os.Stdout)
	case "vmipreset":
		util.MarshallObject(components.NewPresetCrd(), os.Stdout)
	case "vmirs":
		util.MarshallObject(components.NewReplicaSetCrd(), os.Stdout)
	case "vm":
		util.MarshallObject(components.NewVirtualMachineCrd(), os.Stdout)
	case "vmim":
		util.MarshallObject(components.NewVirtualMachineInstanceMigrationCrd(), os.Stdout)
	case "kv":
		util.MarshallObject(components.NewKubeVirtCrd(), os.Stdout)
	case "kv-cr":
		util.MarshallObject(components.NewKubeVirtCR(*namespace, imagePullPolicy), os.Stdout)
	case "rbac":
		all := make([]interface{}, 0)
		all = append(all, rbac.GetAllCluster(*namespace)...)
		all = append(all, rbac.GetAllApiServer(*namespace)...)
		all = append(all, rbac.GetAllController(*namespace)...)
		for _, r := range all {
			util.MarshallObject(r, os.Stdout)
		}
	case "prometheus":
		util.MarshallObject(components.NewPrometheusService(*namespace), os.Stdout)
	case "virt-api":
		apisService := components.NewApiServerService(*namespace)
		apiDeployment, err := components.NewApiServerDeployment(*namespace, *repository, *version, imagePullPolicy, *verbosity)
		if err != nil {
			panic(fmt.Errorf("error generating virt-apiserver deployment %v", err))

		}
		all := []interface{}{apisService, apiDeployment}
		for _, r := range all {
			util.MarshallObject(r, os.Stdout)
		}
	case "virt-controller":
		controller, err := components.NewControllerDeployment(*namespace, *repository, *version, imagePullPolicy, *verbosity)
		if err != nil {
			panic(fmt.Errorf("error generating virt-controller deployment %v", err))

		}
		util.MarshallObject(controller, os.Stdout)
	case "virt-handler":
		handler, err := components.NewHandlerDaemonSet(*namespace, *repository, *version, imagePullPolicy, *verbosity)
		if err != nil {
			panic(fmt.Errorf("error generating virt-handler deployment %v", err))
		}
		util.MarshallObject(handler, os.Stdout)
	default:
		panic(fmt.Errorf("unknown resource type %s", *resourceType))
	}
}
