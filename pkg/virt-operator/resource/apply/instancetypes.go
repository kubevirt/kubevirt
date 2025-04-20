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
 */

package apply

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/log"
)

func (r *Reconciler) createOrUpdateInstancetypes() error {
	for _, instancetype := range r.targetStrategy.Instancetypes() {
		if err := r.createOrUpdateInstancetype(instancetype.DeepCopy()); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) findInstancetype(name string) (*instancetypev1beta1.VirtualMachineClusterInstancetype, error) {
	obj, exists, err := r.stores.ClusterInstancetype.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("VirtualMachineClusterInstancetype"), name)
	}
	foundObj, ok := obj.(*instancetypev1beta1.VirtualMachineClusterInstancetype)
	if !ok {
		return nil, fmt.Errorf("unknown object within VirtualMachineClusterInstancetype store")
	}
	return foundObj, nil
}

func (r *Reconciler) createOrUpdateInstancetype(instancetype *instancetypev1beta1.VirtualMachineClusterInstancetype) error {
	foundObj, err := r.findInstancetype(instancetype.Name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	imageTag, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &instancetype.ObjectMeta, imageTag, imageRegistry, id, true)

	if errors.IsNotFound(err) {
		if _, err := r.clientset.VirtualMachineClusterInstancetype().Create(context.Background(), instancetype, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("unable to create instancetype %+v: %v", instancetype, err)
		}
		log.Log.V(2).Infof("instancetype %v created", instancetype.GetName())
		return nil
	}

	if equality.Semantic.DeepEqual(foundObj.Annotations, instancetype.Annotations) &&
		equality.Semantic.DeepEqual(foundObj.Labels, instancetype.Labels) &&
		equality.Semantic.DeepEqual(foundObj.Spec, instancetype.Spec) {
		log.Log.V(4).Infof("instancetype %v is up-to-date", instancetype.GetName())
		return nil
	}

	instancetype.ResourceVersion = foundObj.ResourceVersion
	if _, err := r.clientset.VirtualMachineClusterInstancetype().Update(context.Background(), instancetype, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("unable to update instancetype %+v: %v", instancetype, err)
	}
	log.Log.V(2).Infof("instancetype %v updated", instancetype.GetName())

	return nil
}

func (r *Reconciler) deleteInstancetypes() error {
	foundInstancetype := false
	for _, instancetype := range r.targetStrategy.Instancetypes() {
		_, exists, err := r.stores.ClusterInstancetype.GetByKey(instancetype.Name)
		if err != nil {
			return err
		}
		if exists {
			foundInstancetype = true
			break
		}
	}
	if !foundInstancetype {
		return nil
	}
	ls := labels.Set{
		v1.AppComponentLabel: GetAppComponent(r.kv),
		v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
	}

	if err := r.clientset.VirtualMachineClusterInstancetype().DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: ls.String(),
	}); err != nil {
		return fmt.Errorf("unable to delete preferences: %v", err)
	}

	return nil
}

func (r *Reconciler) createOrUpdatePreferences() error {
	for _, preference := range r.targetStrategy.Preferences() {
		if err := r.createOrUpdatePreference(preference.DeepCopy()); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) findPreference(name string) (*instancetypev1beta1.VirtualMachineClusterPreference, error) {
	obj, exists, err := r.stores.ClusterPreference.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("VirtualMachineClusterPreference"), name)
	}
	foundObj, ok := obj.(*instancetypev1beta1.VirtualMachineClusterPreference)
	if !ok {
		return nil, fmt.Errorf("unknown object within VirtualMachineClusterPreference store")
	}
	return foundObj, nil
}

func (r *Reconciler) createOrUpdatePreference(preference *instancetypev1beta1.VirtualMachineClusterPreference) error {
	foundObj, err := r.findPreference(preference.Name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	imageTag, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &preference.ObjectMeta, imageTag, imageRegistry, id, true)

	if errors.IsNotFound(err) {
		if _, err := r.clientset.VirtualMachineClusterPreference().Create(context.Background(), preference, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("unable to create preference %+v: %v", preference, err)
		}
		log.Log.V(2).Infof("preference %v created", preference.GetName())
		return nil
	}

	if equality.Semantic.DeepEqual(foundObj.Annotations, preference.Annotations) &&
		equality.Semantic.DeepEqual(foundObj.Labels, preference.Labels) &&
		equality.Semantic.DeepEqual(foundObj.Spec, preference.Spec) {
		log.Log.V(4).Infof("preference %v is up-to-date", preference.GetName())
		return nil
	}

	preference.ResourceVersion = foundObj.ResourceVersion
	if _, err := r.clientset.VirtualMachineClusterPreference().Update(context.Background(), preference, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("unable to update preference %+v: %v", preference, err)
	}
	log.Log.V(2).Infof("preference %v updated", preference.GetName())

	return nil
}

func (r *Reconciler) deletePreferences() error {
	foundPreference := false
	for _, preference := range r.targetStrategy.Preferences() {
		_, exists, err := r.stores.ClusterPreference.GetByKey(preference.Name)
		if err != nil {
			return err
		}
		if exists {
			foundPreference = true
			break
		}
	}
	if !foundPreference {
		return nil
	}
	ls := labels.Set{
		v1.AppComponentLabel: GetAppComponent(r.kv),
		v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
	}

	if err := r.clientset.VirtualMachineClusterPreference().DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: ls.String(),
	}); err != nil {
		return fmt.Errorf("unable to delete preferences: %v", err)
	}

	return nil
}
