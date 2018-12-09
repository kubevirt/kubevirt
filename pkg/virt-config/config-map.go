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

package virtconfig

import (
	"os"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	configMapName          = "kubevirt-config"
	featureGateEnvVar      = "FEATURE_GATES"
	FeatureGatesKey        = "feature-gates"
	emulatedMachinesEnvVar = "VIRT_EMULATED_MACHINES"
	emulatedMachinesKey    = "emulated-machines"
)

// We cannot rely on automatic invocation of 'init' method because this initialization
// code assumes a cluster is available to pull the configmap from
func Init() {
	cfgMap := getConfigMap()
	if val, ok := cfgMap.Data[FeatureGatesKey]; ok {
		os.Setenv(featureGateEnvVar, val)
	}
	if val, ok := cfgMap.Data[emulatedMachinesKey]; ok {
		os.Setenv(emulatedMachinesEnvVar, val)
	}
}

func getConfigMap() *k8sv1.ConfigMap {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	var cfgMap *k8sv1.ConfigMap
	err = wait.PollImmediate(time.Second*1, time.Second*10, func() (bool, error) {

		namespace, curErr := util.GetNamespace()
		if err != nil {
			return false, err
		}

		cfgMap, curErr = virtClient.CoreV1().ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})

		if curErr != nil {
			if errors.IsNotFound(curErr) {
				logger := log.DefaultLogger()
				logger.Infof("%s ConfigMap does not exist. Using defaults.", configMapName)
				cfgMap = &k8sv1.ConfigMap{}
				return true, nil
			}
			return false, curErr
		}

		return true, nil
	})

	if err != nil {
		panic(err)
	}

	return cfgMap
}
