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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package main

import (
	"fmt"
	"os"
	"strings"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"kubevirt.io/kubevirt/pkg/virtctl" // Import to initialize client auth plugins.
)

func main() {
	if err := virtctl.Execute(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, strings.TrimSpace(err.Error())+"\n")
		os.Exit(1)
	}
	os.Exit(0)
}
