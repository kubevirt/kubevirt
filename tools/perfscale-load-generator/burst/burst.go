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
 * Copyright 2021 Nvidia
 *
 */

package burst

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/flags"
	objUtil "kubevirt.io/kubevirt/tools/perfscale-load-generator/object"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/watcher"
)

type BurstLoadGenerator struct {
	Done <-chan time.Time
	UUID string
}

type BurstJob struct {
	Workload   *config.Workload
	virtClient kubecli.KubevirtClient
	k8sClient  kubernetes.Interface
	UUID       string
	objType    string
	done       <-chan time.Time
	watchers   map[string]*watcher.ObjListWatcher
}

// NewBurstJob
func newBurstJob(virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, workload *config.Workload, uuid string, d <-chan time.Time) *BurstJob {
	return &BurstJob{
		virtClient: virtClient,
		k8sClient:  k8sClient,
		Workload:   workload,
		UUID:       uuid,
		done:       d,
		watchers:   map[string]*watcher.ObjListWatcher{},
	}
}

func (b *BurstLoadGenerator) Delete(virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, workload *config.Workload) {
	j := newBurstJob(virtClient, k8sClient, workload, b.UUID, b.Done)
	j.DeleteWorkloads()
	j.stopAllWatchers()
}

func (b *BurstLoadGenerator) Run(virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, workload *config.Workload) {
	j := newBurstJob(virtClient, k8sClient, workload, b.UUID, b.Done)
	j.CreateWorkloads()
	// only stop watchers if the test will not delete the objs, otherwise the watchers will watch for the obj deletions
	if !flags.Delete {
		j.stopAllWatchers()
	}
}

func (b *BurstJob) CreateWorkloads() {
	log.Log.V(1).Infof("Burst Load Generator CreateWorkloads")

	// The watcher must be created before to be able to watch all events related to the objs.
	// This is important because before creating each obj we need to create a obj watcher for each obj type.
	objSpec := b.Workload.Object
	objSample := renderObjSpecTemplate(objSpec, b.UUID)
	b.createWatcherIfNotExist(objSample)
	objUtil.CreateNamespaceIfNotExist(b.k8sClient, objSample.GetNamespace(), config.WorkloadUUIDLabel, b.UUID)

	// Create all replicas
	for r := 1; r <= b.Workload.Count; r++ {
		log.Log.V(2).Infof("Replica %d of %d", r, b.Workload.Count)
		idx := r
		_, err := objUtil.CreateObjectReplica(b.virtClient, objSpec, &idx, b.UUID)
		if err != nil {
			continue
		}
	}

	// Wait all objects be Running
	for objType, objWatcher := range b.watchers {
		log.Log.Infof("Wait for obj(s) %s to be available", objType)
		objWatcher.WaitRunning(b.Workload.Timeout.Duration)
	}
}

func (b *BurstJob) DeleteWorkloads() {
	log.Log.V(1).Infof("Burst Load Generator DeleteWorkloads")
	objSpec := b.Workload.Object
	objSample := renderObjSpecTemplate(objSpec, b.UUID)
	b.createWatcherIfNotExist(objSample)

	// Delete objects that match labels up to the count
	objType := objUtil.GetObjectResource(objSample)
	labels := objSample.GetLabels()
	jobUUID := labels[config.WorkloadUUIDLabel]
	log.Log.V(2).Infof("Deleting %d objects for job %s", b.Workload.Count, jobUUID)
	objUtil.DeleteNObjectsInNamespaces(b.virtClient, objType, config.GetListOpts(config.WorkloadUUIDLabel, jobUUID), b.Workload.Count)

	// Wait all objects be Deleted. In the case of VMI, deleted means the succeded phase.
	for objType, objWatcher := range b.watchers {
		log.Log.Infof("Wait for obj(s) %s to be deleted", objType)
		objWatcher.WaitDeletion(b.Workload.Timeout.Duration)
	}

	log.Log.V(2).Infof("All workloads for job have been deleted")
}

func (b *BurstJob) createWatcherIfNotExist(obj *unstructured.Unstructured) {
	objType := objUtil.GetObjectResource(obj)
	if _, exist := b.watchers[objType]; !exist {
		objWatcher := watcher.NewObjListWatcher(
			b.virtClient,
			objType,
			b.Workload.Count,
			*config.GetListOpts(config.WorkloadUUIDLabel, b.UUID))
		objWatcher.Run()
		b.watchers[objType] = objWatcher
	}
}

func (b *BurstJob) Wait() {
	log.Log.V(1).Infof("Burst Load Generator Wait")
	return
}

func (b *BurstJob) stopAllWatchers() {
	for objType, objWatcher := range b.watchers {
		log.Log.Infof("Stopping obj %s watcher", objType)
		objWatcher.Stop()
	}
}

// Render objSpec template to be able to parse a sample of the object info
func renderObjSpecTemplate(objSpec *config.ObjectSpec, uuid string) *unstructured.Unstructured {
	var err error
	var obj *unstructured.Unstructured
	idx := 0
	if obj, err = objUtil.CreateObjectReplicaSpec(objSpec, &idx, uuid); err != nil {
		panic(err)
	}
	return obj
}
