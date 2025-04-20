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
 * Copyright The KubeVirt Authors.
 */

package main

import (
	"os"
	"time"

	"github.com/spf13/pflag"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/reporter"
)

func main() {
	var duration time.Duration
	pflag.CommandLine.AddGoFlagSet(kubecli.FlagSet())
	pflag.DurationVarP(&duration, "since", "s", 10*time.Minute, "collection window, defaults to 10 minutes")
	pflag.Parse()

	// Hardcoding maxFails to 1 since the purpouse here is just to dump the state once
	reporter := reporter.NewKubernetesReporter(os.Getenv("ARTIFACTS"), 1)
	reporter.Cleanup()
	reporter.DumpTestObjects(duration)
}
