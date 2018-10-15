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

package featuregates

import (
	"os"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/util"
)

const featureGateEnvVar = "FEATURE_GATES"

const (
	dataVolumesGate   = "DataVolumes"
	cpuManager        = "CPUManager"
	liveMigrationGate = "LiveMigration"
)

func ParseFeatureGatesFromConfigMap() {
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

		cfgMap, curErr = virtClient.CoreV1().ConfigMaps(namespace).Get("kubevirt-config", metav1.GetOptions{})

		if curErr != nil {
			if errors.IsNotFound(curErr) {
				// ignore if config map does not exist
				return true, nil
			}
			return false, curErr
		}

		val, ok := cfgMap.Data["feature-gates"]
		if !ok {
			// no feature gates set
			return true, nil
		}

		os.Setenv(featureGateEnvVar, val)
		return true, nil
	})

	if err != nil {
		panic(err)
	}
}

func DataVolumesEnabled() bool {
	return strings.Contains(os.Getenv(featureGateEnvVar), dataVolumesGate)
}

func CPUManagerEnabled() bool {
	return strings.Contains(os.Getenv(featureGateEnvVar), cpuManager)
}

func LiveMigrationEnabled() bool {
	return strings.Contains(os.Getenv(featureGateEnvVar), liveMigrationGate)
}
