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

	"github.com/imdario/mergo"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func (r *Reconciler) createOrUpdateServiceMonitors() error {
	if !r.config.ServiceMonitorEnabled {
		return nil
	}

	for _, serviceMonitor := range r.targetStrategy.ServiceMonitors() {
		if err := r.createOrUpdateServiceMonitor(serviceMonitor.DeepCopy()); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdateServiceMonitor(serviceMonitor *promv1.ServiceMonitor) error {
	prometheusClient := r.clientset.PrometheusClient()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	obj, exists, _ := r.stores.ServiceMonitorCache.Get(serviceMonitor)

	injectOperatorMetadata(r.kv, &serviceMonitor.ObjectMeta, version, imageRegistry, id, true)
	if !exists {
		// Create non existent
		r.expectations.ServiceMonitor.RaiseExpectations(r.kvKey, 1, 0)
		_, err := prometheusClient.MonitoringV1().ServiceMonitors(serviceMonitor.Namespace).Create(context.Background(), serviceMonitor, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ServiceMonitor.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create serviceMonitor %+v: %v", serviceMonitor, err)
		}

		log.Log.V(2).Infof("serviceMonitor %v created", serviceMonitor.GetName())
		return nil
	}

	cachedServiceMonitor := obj.(*promv1.ServiceMonitor)
	endpointsModified, err := ensureServiceMonitorSpec(serviceMonitor, cachedServiceMonitor)
	if err != nil {
		return err
	}

	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &cachedServiceMonitor.ObjectMeta, serviceMonitor.ObjectMeta)

	// there was no change to metadata and the spec fields are equal
	if !*modified && !endpointsModified {
		log.Log.V(4).Infof("serviceMonitor %v is up-to-date", serviceMonitor.GetName())
		return nil
	}
	patchBytes, err := patch.New(getPatchWithObjectMetaAndSpec([]patch.PatchOption{}, &serviceMonitor.ObjectMeta, serviceMonitor.Spec)...).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = prometheusClient.MonitoringV1().ServiceMonitors(serviceMonitor.Namespace).Patch(context.Background(), serviceMonitor.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch serviceMonitor %+v: %v", serviceMonitor, err)
	}

	log.Log.V(2).Infof("serviceMonitor %v updated", serviceMonitor.GetName())

	return nil
}

func ensureServiceMonitorSpec(required, existing *promv1.ServiceMonitor) (bool, error) {
	if err := mergo.Merge(&existing.Spec, &required.Spec); err != nil {
		return false, err
	}

	if equality.Semantic.DeepEqual(existing.Spec, required.Spec) {
		return false, nil
	}

	return true, nil
}

func (r *Reconciler) createOrUpdatePrometheusRules() error {
	if !r.config.PrometheusRulesEnabled {
		return nil
	}

	for _, prometheusRule := range r.targetStrategy.PrometheusRules() {
		if err := r.createOrUpdatePrometheusRule(prometheusRule.DeepCopy()); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdatePrometheusRule(prometheusRule *promv1.PrometheusRule) error {
	prometheusClient := r.clientset.PrometheusClient()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	obj, exists, _ := r.stores.PrometheusRuleCache.Get(prometheusRule)

	injectOperatorMetadata(r.kv, &prometheusRule.ObjectMeta, version, imageRegistry, id, true)
	if !exists {
		// Create non existent
		r.expectations.PrometheusRule.RaiseExpectations(r.kvKey, 1, 0)
		_, err := prometheusClient.MonitoringV1().PrometheusRules(prometheusRule.Namespace).Create(context.Background(), prometheusRule, metav1.CreateOptions{})
		if err != nil {
			r.expectations.PrometheusRule.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create PrometheusRule %+v: %v", prometheusRule, err)
		}

		log.Log.V(2).Infof("PrometheusRule %v created", prometheusRule.GetName())
		return nil
	}

	cachedPrometheusRule := obj.(*promv1.PrometheusRule)
	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedPrometheusRule.DeepCopy()

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, prometheusRule.ObjectMeta)

	if !*modified && equality.Semantic.DeepEqual(cachedPrometheusRule.Spec, prometheusRule.Spec) {
		log.Log.V(4).Infof("PrometheusRule %v is up-to-date", prometheusRule.GetName())
		return nil
	}
	patchBytes, err := patch.New(getPatchWithObjectMetaAndSpec([]patch.PatchOption{}, &prometheusRule.ObjectMeta, prometheusRule.Spec)...).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = prometheusClient.MonitoringV1().PrometheusRules(prometheusRule.Namespace).Patch(context.Background(), prometheusRule.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch PrometheusRule %+v: %v", prometheusRule, err)
	}

	log.Log.V(2).Infof("PrometheusRule %v updated", prometheusRule.GetName())

	return nil
}
