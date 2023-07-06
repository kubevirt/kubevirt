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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libinfra

import (
	"context"
	"encoding/json"
	"fmt"

	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	v13 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/util"
)

func GetLeader() string {
	virtClient := kubevirt.Client()

	controllerEndpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), leaderelectionconfig.DefaultEndpointName, v1.GetOptions{})
	util.PanicOnError(err)

	var record resourcelock.LeaderElectionRecord
	if recordBytes, found := controllerEndpoint.Annotations[resourcelock.LeaderElectionRecordAnnotationKey]; found {
		err := json.Unmarshal([]byte(recordBytes), &record)
		util.PanicOnError(err)
	}
	return record.HolderIdentity
}

func GetNewLeaderPod(virtClient kubecli.KubevirtClient) *v12.Pod {
	labelSelector, err := labels.Parse(fmt.Sprint(v13.AppLabel + "=virt-controller"))
	util.PanicOnError(err)
	fieldSelector := fields.ParseSelectorOrDie("status.phase=" + string(v12.PodRunning))
	controllerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(),
		v1.ListOptions{LabelSelector: labelSelector.String(), FieldSelector: fieldSelector.String()})
	util.PanicOnError(err)
	leaderPodName := GetLeader()
	for _, pod := range controllerPods.Items {
		if pod.Name != leaderPodName {
			return &pod
		}
	}
	return nil
}
