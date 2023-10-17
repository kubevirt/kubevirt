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

func (r *Reconciler) createOrUpdateInstancetype(instancetype *instancetypev1beta1.VirtualMachineClusterInstancetype) error {
	instancetypeClient := r.clientset.VirtualMachineClusterInstancetype()

	foundObj, err := instancetypeClient.Get(context.Background(), instancetype.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	imageTag, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &instancetype.ObjectMeta, imageTag, imageRegistry, id, true)

	if errors.IsNotFound(err) {
		if _, err := instancetypeClient.Create(context.Background(), instancetype, metav1.CreateOptions{}); err != nil {
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
	if _, err := instancetypeClient.Update(context.Background(), instancetype, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("unable to update instancetype %+v: %v", instancetype, err)
	}
	log.Log.V(2).Infof("instancetype %v updated", instancetype.GetName())

	return nil
}

func (r *Reconciler) deleteInstancetypes() error {
	ls := labels.Set{
		v1.AppComponentLabel: v1.AppComponent,
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

func (r *Reconciler) createOrUpdatePreference(preference *instancetypev1beta1.VirtualMachineClusterPreference) error {
	preferenceClient := r.clientset.VirtualMachineClusterPreference()

	foundObj, err := preferenceClient.Get(context.Background(), preference.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	imageTag, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &preference.ObjectMeta, imageTag, imageRegistry, id, true)

	if errors.IsNotFound(err) {
		if _, err := preferenceClient.Create(context.Background(), preference, metav1.CreateOptions{}); err != nil {
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
	if _, err := preferenceClient.Update(context.Background(), preference, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("unable to update preference %+v: %v", preference, err)
	}
	log.Log.V(2).Infof("preference %v updated", preference.GetName())

	return nil
}

func (r *Reconciler) deletePreferences() error {
	ls := labels.Set{
		v1.AppComponentLabel: v1.AppComponent,
		v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
	}

	if err := r.clientset.VirtualMachineClusterPreference().DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: ls.String(),
	}); err != nil {
		return fmt.Errorf("unable to delete preferences: %v", err)
	}

	return nil
}
