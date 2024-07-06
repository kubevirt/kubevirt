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

package util

import (
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

func DaemonsetIsReady(kv *v1.KubeVirt, daemonset *appsv1.DaemonSet, stores Stores) bool {

	// ensure we're looking at the latest daemonset from cache
	obj, exists, _ := stores.DaemonSetCache.Get(daemonset)
	if exists {
		daemonset = obj.(*appsv1.DaemonSet)
	} else {
		// not in cache yet
		return false
	}

	if daemonset.Status.DesiredNumberScheduled == 0 ||
		daemonset.Status.DesiredNumberScheduled != daemonset.Status.NumberReady {

		log.Log.V(4).Infof("DaemonSet %v not ready yet", daemonset.Name)
		return false
	}

	// cross check that we have 'daemonset.Status.NumberReady' pods with
	// the desired version tag. This ensures we wait for rolling update to complete
	// before marking the infrastructure as 100% ready.
	var podsReady int32
	for _, obj := range stores.InfrastructurePodCache.List() {
		if pod, ok := obj.(*k8sv1.Pod); ok {
			if !podIsRunning(pod) {
				continue
			} else if !podHasNamePrefix(pod, daemonset.Name) {
				continue
			}

			if !PodIsUpToDate(pod, kv) {
				log.Log.Infof("DaemonSet %v waiting for out of date pods to terminate.", daemonset.Name)
				return false
			}

			if PodIsReady(pod) {
				podsReady++
			}
		}
	}

	if podsReady == 0 {
		log.Log.Infof("DaemonSet %v not ready yet. Waiting for all pods to be ready", daemonset.Name)
		return false
	}

	// Misscheduled but up to date daemonset pods will not be evicted unless manually deleted or the daemonset gets updated.
	// Don't force the Available condition to false or block the upgrade on up-to-date misscheduled pods.
	return podsReady >= daemonset.Status.DesiredNumberScheduled
}

func DeploymentIsReady(kv *v1.KubeVirt, deployment *appsv1.Deployment, stores Stores) bool {
	// ensure we're looking at the latest deployment from cache
	obj, exists, _ := stores.DeploymentCache.Get(deployment)
	if exists {
		deployment = obj.(*appsv1.Deployment)
	} else {
		// not in cache yet
		return false
	}

	if deployment.Status.Replicas == 0 || deployment.Status.ReadyReplicas == 0 {
		log.Log.V(4).Infof("Deployment %v not ready yet", deployment.Name)
		return false
	}

	// cross check that we have 'deployment.Status.ReadyReplicas' pods with
	// the desired version tag. This ensures we wait for rolling update to complete
	// before marking the infrastructure as 100% ready.
	var podsReady int32
	for _, obj := range stores.InfrastructurePodCache.List() {
		if pod, ok := obj.(*k8sv1.Pod); ok {
			if !podIsRunning(pod) {
				continue
			} else if !podHasNamePrefix(pod, deployment.Name) {
				continue
			}

			if !PodIsUpToDate(pod, kv) {
				log.Log.Infof("Deployment %v waiting for out of date pods to terminate.", deployment.Name)
				return false
			}

			if PodIsReady(pod) {
				podsReady++
			}
		}
	}

	if podsReady == 0 {
		log.Log.Infof("Deployment %v not ready yet. Waiting for at least one pod to become ready", deployment.Name)
		return false
	}
	return true
}

func DaemonSetIsUpToDate(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) bool {
	version := kv.Status.TargetKubeVirtVersion
	registry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	return daemonSet.Annotations[v1.InstallStrategyVersionAnnotation] == version &&
		daemonSet.Annotations[v1.InstallStrategyRegistryAnnotation] == registry &&
		daemonSet.Annotations[v1.InstallStrategyIdentifierAnnotation] == id
}

func podIsRunning(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodRunning
}

func podHasNamePrefix(pod *k8sv1.Pod, namePrefix string) bool {
	return strings.Contains(pod.Name, namePrefix)
}

func PodIsUpToDate(pod *k8sv1.Pod, kv *v1.KubeVirt) bool {
	if pod.Annotations == nil {
		return false
	}

	version, ok := pod.Annotations[v1.InstallStrategyVersionAnnotation]
	if !ok || version != kv.Status.TargetKubeVirtVersion {
		return false
	}

	imageRegistry, ok := pod.Annotations[v1.InstallStrategyRegistryAnnotation]
	if !ok || imageRegistry != kv.Status.TargetKubeVirtRegistry {
		return false
	}

	id, ok := pod.Annotations[v1.InstallStrategyIdentifierAnnotation]
	if !ok || id != kv.Status.TargetDeploymentID {
		return false
	}

	return true
}

func PodIsReady(pod *k8sv1.Pod) bool {
	if pod.Status.Phase != k8sv1.PodRunning {
		return false
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			return false
		}
	}
	return true
}

func PodIsCrashLooping(pod *k8sv1.Pod) bool {
	haveContainersCrashed := func(cs []k8sv1.ContainerStatus) bool {
		for i := range cs {
			if cs[i].State.Terminated != nil ||
				cs[i].LastTerminationState.Terminated != nil ||
				cs[i].RestartCount > 0 {
				return true
			}
		}
		return false
	}

	return haveContainersCrashed(pod.Status.ContainerStatuses)
}
