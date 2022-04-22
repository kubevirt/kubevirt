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
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
	objUtil "kubevirt.io/kubevirt/tools/perfscale-load-generator/object"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/utils"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/watcher"
)

type BurstLoadGenerator struct {
	Done <-chan time.Time
}

type BurstJob struct {
	Workload   *config.Workload
	virtClient kubecli.KubevirtClient
	UUID       string
	objType    string
	done       <-chan time.Time
}

// NewBurstJob
func newBurstJob(virtClient kubecli.KubevirtClient, workload *config.Workload, d <-chan time.Time) *BurstJob {
	uid, _ := uuid.NewUUID()
	return &BurstJob{
		virtClient: virtClient,
		Workload:   workload,
		UUID:       uid.String(),
		done:       d,
	}
}

func (b *BurstLoadGenerator) Delete(virtClient kubecli.KubevirtClient, workload *config.Workload) {
	j := newBurstJob(virtClient, workload, b.Done)
	j.DeleteWorkload()
}

func (b *BurstLoadGenerator) Run(virtClient kubecli.KubevirtClient, workload *config.Workload) {
	j := newBurstJob(virtClient, workload, b.Done)
	j.CreateWorkload()
}

func (b *BurstJob) CreateWorkload() {
	log.Log.V(1).Infof("Burst Load Generator CreateWorkload")

	var wg sync.WaitGroup
	obj := b.Workload.Object
	for replica := 1; replica <= b.Workload.Count; replica++ {
		select {
		case <-b.done:
			log.Log.V(1).Infof("Burst Load Generator duration has timed out")
			return
		default:
		}

		log.Log.V(2).Infof("Replica %d of %d", replica, b.Workload.Count)
		newObject, err := utils.Create(b.virtClient, replica, obj, b.UUID)
		if err != nil {
			continue
		}
		if b.objType == "" {
			b.objType = objUtil.GetObjectResource(newObject)
		}

		wg.Add(1)
		go func(newObject *unstructured.Unstructured) {
			defer wg.Done()
			b.watchCreate(newObject)
			log.Log.Infof("obj %s is available", newObject.GroupVersionKind().Kind)
		}(newObject)
	}
	wg.Wait()
}

func (b *BurstJob) DeleteWorkload() {
	log.Log.V(1).Infof("Burst Load Generator DeleteWorkload")

	obj := b.Workload.Object
	getObject, objType := objUtil.FindObject(b.virtClient, obj, b.Workload.Count)
	b.objType = objType
	if getObject != nil {
		labels := getObject.GetLabels()
		jobUUID := labels[config.WorkloadLabel]
		log.Log.V(2).Infof("Deleting all workloads for job %s", jobUUID)
		objUtil.DeleteAllObjectsInNamespaces(b.virtClient, objType, config.GetListOpts(config.WorkloadLabel, jobUUID))
		b.watchDelete(getObject)
	}
	log.Log.V(2).Infof("All workloads for job have been deleted")
}

func (b *BurstJob) watchDelete(obj *unstructured.Unstructured) {
	objWatcher := watcher.NewObjListWatcher(
		b.virtClient,
		b.objType,
		b.Workload.Count,
		*config.GetListOpts(config.WorkloadLabel, b.UUID))
	objWatcher.Run()

	log.Log.Infof("Wait for obj(s) %s to be deleted", b.objType)
	objWatcher.WaitDeletion(b.Workload.Timeout.Duration)
	objWatcher.Stop()
}

func (b *BurstJob) watchCreate(obj *unstructured.Unstructured) {
	objWatcher := watcher.NewObjListWatcher(
		b.virtClient,
		b.objType,
		b.Workload.Count,
		*config.GetListOpts(config.WorkloadLabel, b.UUID))
	objWatcher.Run()

	log.Log.Infof("Wait for obj(s) %s to be available", b.objType)
	objWatcher.WaitRunning(b.Workload.Timeout.Duration)
	objWatcher.Stop()
}

func (b *BurstJob) Wait() {
	log.Log.V(1).Infof("Burst Load Generator Wait")
	return
}
