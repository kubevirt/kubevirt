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
 *
 */

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package snapshot

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	virtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/instancetype/revision"
)

type Controller interface {
	Sync(*snapshotv1.VirtualMachineSnapshot) error
}

type revisionHandler interface {
	StoreSnapshot(snapshot *snapshotv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) error
}

type controller struct {
	clientset       kubecli.KubevirtClient
	recorder        record.EventRecorder
	revisionHandler revisionHandler
}

func New(
	instancetypeStore, clusterInstancetypeStore, preferenceStore, clusterPreferenceStore cache.Store,
	virtClient kubecli.KubevirtClient, recorder record.EventRecorder,
) Controller {
	return &controller{
		revisionHandler: revision.New(instancetypeStore,
			clusterInstancetypeStore,
			preferenceStore,
			clusterPreferenceStore,
			virtClient),
		clientset: virtClient,
		recorder:  recorder,
	}
}

func (c *controller) Sync(snapshot *snapshotv1.VirtualMachineSnapshot) error {
	vm, err := c.clientset.VirtualMachine(snapshot.Namespace).
		Get(context.Background(), snapshot.Spec.Source.Name, metav1.GetOptions{})
	if err != nil {
		log.Log.Errorf("Failed to get VM %s/%s: %v", snapshot.Namespace, snapshot.Spec.Source.Name, err)
		return err
	}

	if err := c.revisionHandler.StoreSnapshot(snapshot, vm); err != nil {
		log.Log.Errorf("Failed to patch InstanceType ControllerRevision %s: %v",
			vm.Spec.Instancetype.RevisionName, err)
		return err
	}

	return nil
}
