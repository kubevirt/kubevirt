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
 * Copyright 2021 IBM, Inc.
 *
 */

package flags

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

var UUID string
var Name string
var Namespace string
var NamespacedIterations bool
var VMImage string
var VMIImgTag string
var VMIImgRepo string
var CPULimit string
var MEMLimit string
var VMICount int
var IterationCount int
var IterationInterval time.Duration
var IterationWaitForDeletion bool
var IterationVMIWait bool
var IterationCleanup bool
var MaxWaitTimeout time.Duration
var QPS float64
var Burst int
var WaitWhenFinished time.Duration
var Kubeconfig string
var Kubemaster string
var Verbosity int

func init() {
	uid, _ := uuid.NewUUID()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flag.StringVar(&UUID, "uuid", uid.String(), "scenario uuid")
	flag.StringVar(&Name, "name", "kubevirt-test-default", "scenario name")
	flag.StringVar(&Namespace, "namespace", "kubevirt-test-default", "namespace base name to use")
	flag.BoolVar(&NamespacedIterations, "namespaced-iterations", false, "create a namespace per scenario iteration")
	flag.IntVar(&VMICount, "vmi-count", 100, "total number of VMs to be created")
	flag.StringVar(&VMImage, "vmi-img", "cirros", "vmi image name (cirros, alpine, fedora-cloud)")
	flag.StringVar(&VMIImgTag, "img-tag", "latest", "Set the image tag or digest to use")
	flag.StringVar(&VMIImgRepo, "img-prefix", "quay.io/kubevirt", "Set the repository prefix for all images")
	flag.StringVar(&MEMLimit, "vmi-mem-limit", "90Mi", "vmi memory request and limit (MEM overhead ~ +170Mi)")
	flag.StringVar(&CPULimit, "vmi-cpu-limit", "100m", "vmi CPU request and limit (1 CPU = 1000m)")
	flag.IntVar(&IterationCount, "iteration-count", 1, "how many times to execute the scenario")
	flag.DurationVar(&IterationInterval, "iteration-interval", 0, "how much time to wait between each scenario iteration")
	flag.BoolVar(&IterationWaitForDeletion, "iteration-wait-for-deletion", true, "wait for VMIs to be deleted and all objects disapear in each iteration")
	flag.BoolVar(&IterationVMIWait, "iteration-vmi-wait", true, "wait for all vmis to be running before moving forward to the next iteration")
	flag.BoolVar(&IterationCleanup, "iteration-cleanup", true, "clean up old tests, delete all created VMIs and namespaces before moving forward to the next iteration")
	flag.DurationVar(&MaxWaitTimeout, "max-wait-timeout", 30*time.Minute, "maximum wait period")
	flag.Float64Var(&QPS, "qps", 20, "number of queries per second for VMI creation")
	flag.IntVar(&Burst, "burst", 20, "maximum burst for throttle the VMI creation")
	flag.DurationVar(&WaitWhenFinished, "wait-when-finished", 30*time.Second, "delays the termination of the scenario")
	flag.StringVar(&Kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&Kubemaster, "master", "", "kubernetes master url")
	flag.IntVar(&Verbosity, "v", 2, "log level for V logs")

	if Kubeconfig == "" {
		if os.Getenv("KUBECONFIG") != "" {
			Kubeconfig = os.Getenv("KUBECONFIG")
		} else {
			_, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".kube", "config"))
			if !os.IsNotExist(err) {
				Kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
			}
		}
	}

	flag.Parse()
}
