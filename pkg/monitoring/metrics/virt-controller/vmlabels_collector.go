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

package virt_controller

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"
)

var (
	// Default allowlist for VM labels ("*" means all labels exposed by default)
	vmLabelsAllowlist = []string{"*"}

	// Default ignorelist for VM labels (empty by default)
	vmLabelsIgnorelist = []string{}

	// Protects updates and reads of vmLabelsAllowlist/vmLabelsIgnorelist
	vmLabelsConfigMu sync.RWMutex

	// ConfigMap name for VM labels configuration
	vmLabelsConfigMapName = "kubevirt-vm-labels-config"

	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

	vmLabels = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_labels",
			Help: "The metric exposes the VM labels as Prometheus labels. Configure allowed and ignored labels via the 'kubevirt-vm-labels-config' ConfigMap.",
		},
		labels,
	)
)

func loadVMLabelsConfiguration() {
	if kubevirtClient == nil {
		resetVMLabelsConfigToDefaults()
		return
	}

	namespace, err := clientutil.GetNamespace()
	if err != nil {
		log.Log.Errorf("vm-labels: failed to determine namespace: %v", err)
		resetVMLabelsConfigToDefaults()
		return
	}

	configMap, err := kubevirtClient.CoreV1().ConfigMaps(namespace).Get(
		context.TODO(), vmLabelsConfigMapName, metav1.GetOptions{})
	if err != nil {
		resetVMLabelsConfigToDefaults()
		return
	}

	updateVMLabelsConfigFromConfigMap(configMap)
}

func updateVMLabelsConfigFromConfigMap(configMap *k8sv1.ConfigMap) {
	if configMap.Data == nil {
		resetVMLabelsConfigToDefaults()
		return
	}

	// Update under lock to avoid races
	vmLabelsConfigMu.Lock()
	defer vmLabelsConfigMu.Unlock()

	if allowlistData, exists := configMap.Data["allowlist"]; exists {
		allowlist := parseLabelsFromString(allowlistData)
		vmLabelsAllowlist = allowlist
	} else {
		vmLabelsAllowlist = []string{"*"}
	}

	if ignorelistData, exists := configMap.Data["ignorelist"]; exists {
		ignorelist := parseLabelsFromString(ignorelistData)
		vmLabelsIgnorelist = ignorelist
	} else {
		vmLabelsIgnorelist = []string{}
	}

}

func parseLabelsFromString(data string) []string {
	if strings.TrimSpace(data) == "" {
		return []string{}
	}

	labels := strings.Split(data, ",")
	result := make([]string, 0, len(labels))

	for _, label := range labels {
		trimmed := strings.TrimSpace(label)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func reportVmLabels(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	vmLabelsConfigMu.RLock()
	allowlist := vmLabelsAllowlist
	ignorelist := vmLabelsIgnorelist
	vmLabelsConfigMu.RUnlock()

	if len(allowlist) == 0 {
		log.Log.Infof("kubevirt_vm_labels skipping vm %s/%s, allowlist empty", vm.Namespace, vm.Name)
		return cr
	}

	if len(vm.Labels) == 0 {
		return cr
	}

	vmLabelsMap := filterVMLabels(vm.Labels, allowlist, ignorelist)

	if len(vmLabelsMap) == 0 {
		log.Log.Infof("kubevirt_vm_labels skipping vm %s/%s, no allowlist keys found", vm.Namespace, vm.Name)
		return cr
	}

	constLabels := make(map[string]string)

	for labelKey, labelValue := range vmLabelsMap {
		sanitizedLabelName := sanitizeLabelName(labelKey)
		prometheusLabelName := "label_" + sanitizedLabelName
		constLabels[prometheusLabelName] = labelValue
	}

	cr = append(cr, operatormetrics.CollectorResult{
		Metric:      vmLabels,
		Labels:      []string{vm.Name, vm.Namespace},
		ConstLabels: constLabels,
		Value:       1.0,
	})

	return cr
}

func filterVMLabels(vmLabels map[string]string, allowlist []string, ignorelist []string) map[string]string {
	if len(allowlist) == 0 {
		return nil
	}

	filteredLabels := make(map[string]string)
	allowMap := make(map[string]bool)
	allowAll := false

	for _, allowed := range allowlist {
		if allowed == "*" {
			allowAll = true
			break
		}
		allowMap[allowed] = true
	}

	ignoreMap := make(map[string]bool)
	for _, ignored := range ignorelist {
		ignoreMap[ignored] = true
	}

	for key, value := range vmLabels {
		// Ignore takes precedence over allow
		if ignoreMap[key] {
			continue
		}

		if allowAll || allowMap[key] {
			filteredLabels[key] = value
		}
	}

	return filteredLabels
}

func sanitizeLabelName(name string) string {
	sanitized := invalidLabelCharRE.ReplaceAllString(name, "_")
	if len(sanitized) == 0 || !((sanitized[0] >= 'a' && sanitized[0] <= 'z') || (sanitized[0] >= 'A' && sanitized[0] <= 'Z') || sanitized[0] == '_') {
		sanitized = "_" + sanitized
	}
	return sanitized
}

func resetVMLabelsConfigToDefaults() {
	vmLabelsConfigMu.Lock()
	vmLabelsAllowlist = []string{"*"}
	vmLabelsIgnorelist = []string{}
	vmLabelsConfigMu.Unlock()
}
