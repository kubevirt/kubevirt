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

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/flags"
	objUtil "kubevirt.io/kubevirt/tools/perfscale-load-generator/object"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/watcher"
)

// BurstLoadGenerator generates VMI workloads
type BurstLoadGenerator struct {
	Workload   *config.Workload
	virtClient kubecli.KubevirtClient
	UUID       string
	objType    string
}

// NewBurstLoadGenerator
func NewBurstLoadGenerator(virtClient kubecli.KubevirtClient, workload *config.Workload) *BurstLoadGenerator {
	uid, _ := uuid.NewUUID()
	return &BurstLoadGenerator{
		virtClient: virtClient,
		Workload:   workload,
		UUID:       uid.String(),
	}
}

func (b *BurstLoadGenerator) CreateWorkload() {
	var wg sync.WaitGroup

	obj := b.Workload.Object
	for replica := 0; replica < b.Workload.Count; replica++ {
		log.Log.V(2).Infof("Replica %d of %d", replica, b.Workload.Count)
		templateData := objUtil.GenerateObjectTemplateData(obj, replica)

		newObject, err := objUtil.RenderObject(templateData, obj.ObjectTemplate)
		if err != nil {
			log.Log.Errorf("error rendering obj: %v", err)
		}
		config.AddLabels(newObject, b.UUID)
		if b.objType == "" {
			b.objType = objUtil.GetObjectResource(newObject)
		}

		if _, err := objUtil.CreateObject(b.virtClient, newObject); err != nil {
			log.Log.Errorf("error creating obj %s: %v", newObject.GroupVersionKind().Kind, err)
		}

		wg.Add(1)
		go func(newObject *unstructured.Unstructured) {
			defer wg.Done()
			b.Watch(newObject)
			log.Log.Infof("obj %s is available", newObject.GroupVersionKind().Kind)
		}(newObject)
	}
	wg.Wait()
}

func (b *BurstLoadGenerator) DeleteWorkload() {
	obj := b.Workload.Object
	getObject, objType := objUtil.GetObject(b.virtClient, obj, b.Workload.Count)
	b.objType = objType
	if getObject != nil {
		labels := getObject.GetLabels()
		jobUUID := labels[config.WorkloadLabel]
		log.Log.V(2).Infof("Deleting all workloads for job %s", jobUUID)
		objUtil.DeleteAllObjectsInNamespaces(b.virtClient, objType, config.GetListOpts(config.WorkloadLabel, jobUUID))
		b.Watch(getObject)
	}
	log.Log.V(2).Infof("All workloads for job have been deleted")
}

func (b *BurstLoadGenerator) Watch(obj *unstructured.Unstructured) {
	objWatcher := watcher.NewObjListWatcher(
		b.virtClient,
		b.objType,
		b.Workload.Count,
		*config.GetListOpts(config.WorkloadLabel, b.UUID))
	objWatcher.Run()
	if flags.Delete {
		log.Log.Infof("Wait for obj(s) %s to be deleted", b.objType)
		objWatcher.WaitDeletion(b.Workload.Timeout.Duration)
	} else {
		log.Log.Infof("Wait for obj(s) %s to be available", b.objType)
		objWatcher.WaitRunning(b.Workload.Timeout.Duration)
	}
	objWatcher.Stop()
}
