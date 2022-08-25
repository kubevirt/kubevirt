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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package oper

import (
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/client-go/kubecli"
)

// waitRemoval deletes the specified VMI and waits for its absence.
func waitRemoval(namespace, name string, timeout time.Duration) error {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	return wait.PollImmediate(1*time.Second, timeout, func() (done bool, err error) {
		_, err = virtClient.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}
