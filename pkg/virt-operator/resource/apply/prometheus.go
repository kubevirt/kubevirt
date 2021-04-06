package apply

import (
	"context"
	"encoding/json"
	"fmt"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"
)

func (r *Reconciler) createOrUpdateServiceMonitors() error {
	if !r.stores.ServiceMonitorEnabled {
		return nil
	}

	prometheusClient := r.clientset.PrometheusClient()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	for _, serviceMonitor := range r.targetStrategy.ServiceMonitors() {
		var cachedServiceMonitor *promv1.ServiceMonitor

		serviceMonitor := serviceMonitor.DeepCopy()
		obj, exists, _ := r.stores.ServiceMonitorCache.Get(serviceMonitor)
		if exists {
			cachedServiceMonitor = obj.(*promv1.ServiceMonitor)
		}

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

		} else if !objectMatchesVersion(&cachedServiceMonitor.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) {
			newSpec, err := json.Marshal(serviceMonitor.Spec)
			if err != nil {
				return err
			}

			ops, err := getPatchWithObjectMetaAndSpec([]string{}, &serviceMonitor.ObjectMeta, newSpec)
			if err != nil {
				return err
			}

			_, err = prometheusClient.MonitoringV1().ServiceMonitors(serviceMonitor.Namespace).Patch(context.Background(), serviceMonitor.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
			if err != nil {
				return fmt.Errorf("unable to patch serviceMonitor %+v: %v", serviceMonitor, err)
			}

			log.Log.V(2).Infof("serviceMonitor %v updated", serviceMonitor.GetName())

		} else {
			log.Log.V(4).Infof("serviceMonitor %v is up-to-date", serviceMonitor.GetName())
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdatePrometheusRules() error {

	if !r.stores.PrometheusRulesEnabled {
		return nil
	}

	prometheusClient := r.clientset.PrometheusClient()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	for _, prometheusRule := range r.targetStrategy.PrometheusRules() {
		var cachedPrometheusRule *promv1.PrometheusRule

		prometheusRule := prometheusRule.DeepCopy()
		obj, exists, _ := r.stores.PrometheusRuleCache.Get(prometheusRule)
		if exists {
			cachedPrometheusRule = obj.(*promv1.PrometheusRule)
		}

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

		} else if !objectMatchesVersion(&cachedPrometheusRule.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) {
			// Add Spec Patch
			newSpec, err := json.Marshal(prometheusRule.Spec)
			if err != nil {
				return err
			}

			ops, err := getPatchWithObjectMetaAndSpec([]string{}, &prometheusRule.ObjectMeta, newSpec)
			if err != nil {
				return err
			}

			_, err = prometheusClient.MonitoringV1().PrometheusRules(prometheusRule.Namespace).Patch(context.Background(), prometheusRule.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
			if err != nil {
				return fmt.Errorf("unable to patch PrometheusRule %+v: %v", prometheusRule, err)
			}

			log.Log.V(2).Infof("PrometheusRule %v updated", prometheusRule.GetName())

		} else {
			log.Log.V(4).Infof("PrometheusRule %v is up-to-date", prometheusRule.GetName())
		}
	}

	return nil
}
