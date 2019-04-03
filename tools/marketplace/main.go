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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package main

import (
	"flag"
	"fmt"

	"kubevirt.io/kubevirt/tools/marketplace/helper"
)

func main() {

	repo := flag.String("quay-repository", "kubevirt", "the Quay.io repository name, defaults to kubevirt")
	flag.Parse()

	bh, err := helper.NewBundleHelper(*repo)
	if err != nil {
		panic(err)
	}

	fmt.Println("downloaded manifests:")

	for _, pkg := range bh.Pkgs {
		fmt.Printf("package: %+v\n", pkg)
	}
	for _, csv := range bh.CSVs {
		fmt.Printf("csv: %v\n", helper.GetCSVName(csv))
	}
	for _, crd := range bh.CRDs {
		fmt.Printf("crd: %v %v\n", crd.Name, crd.Spec.Version)
	}

}
