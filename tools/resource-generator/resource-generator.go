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

	v1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/creation/rbac"
	"kubevirt.io/kubevirt/tools/util"
)

func main() {
	resourceType := flag.String("type", "", "Type of resource to generate. vmi | vmipreset | vmirs | vm | vmim | kv | rbac | priorityclass")
	namespace := flag.String("namespace", "kube-system", "Namespace to use.")
	pullPolicy := flag.String("pullPolicy", "IfNotPresent", "ImagePullPolicy to use.")

	flag.Parse()

	imagePullPolicy := v1.PullPolicy(*pullPolicy)

	switch *resourceType {
	case "kv":
		kv, err := components.NewKubeVirtCrd()
		if err != nil {
			panic(fmt.Errorf("This should not happen, %v", err))
		}
		util.MarshallObject(kv, os.Stdout)
	case "kv-cr":
		util.MarshallObject(components.NewKubeVirtCR(*namespace, imagePullPolicy), os.Stdout)
	case "operator-rbac":
		all := rbac.GetAllOperator(*namespace)
		for _, r := range all {
			util.MarshallObject(r, os.Stdout)
		}
	case "priorityclass":
		priorityClass := components.NewKubeVirtPriorityClassCR()
		util.MarshallObject(priorityClass, os.Stdout)
	default:
		panic(fmt.Errorf("unknown resource type %s", *resourceType))
	}
}
