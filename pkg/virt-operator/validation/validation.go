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

// Package validation holds the semantic validation rules for the KubeVirt CR
// spec that can be evaluated without access to a live cluster
package validation

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

func ValidateKubeVirtSpec(kv *v1.KubeVirt) []metav1.StatusCause {
	config := &kv.Spec.Configuration

	var causes []metav1.StatusCause
	causes = append(causes, ValidateCustomizeComponents(kv.Spec.CustomizeComponents)...)
	causes = append(causes, ValidateGuestToRequestHeadroom(config.AdditionalGuestMemoryOverheadRatio)...)

	if config.TLSConfiguration != nil {
		causes = append(causes, ValidateTLSConfiguration(config.TLSConfiguration)...)
	}

	causes = append(causes, ValidateSeccompConfiguration(
		field.NewPath("spec").Child("configuration", "seccompConfiguration"),
		config.SeccompConfiguration)...)

	if kv.Spec.Infra != nil {
		causes = append(causes, ValidateInfraReplicas(kv.Spec.Infra.Replicas)...)
	}

	causes = append(causes, ValidateFeatureGates(config.DeveloperConfiguration)...)

	return causes
}

// CausesToError formats the given validation causes into a single error or
// returns nil when there are no causes
func CausesToError(causes []metav1.StatusCause) error {
	if len(causes) == 0 {
		return nil
	}

	messages := make([]string, 0, len(causes))
	for _, cause := range causes {
		if cause.Field != "" {
			messages = append(messages, fmt.Sprintf("%s: %s", cause.Field, cause.Message))
			continue
		}
		messages = append(messages, cause.Message)
	}

	return fmt.Errorf("invalid KubeVirt spec: %s", strings.Join(messages, "; "))
}

func ValidateCustomizeComponents(customization v1.CustomizeComponents) []metav1.StatusCause {
	patches := customization.Patches
	statuses := []metav1.StatusCause{}

	for _, patch := range patches {
		if json.Valid([]byte(patch.Patch)) {
			continue
		}

		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("patch %q is not valid JSON", patch.Patch),
		})
	}

	return statuses
}

func ValidateTLSConfiguration(tlsConfiguration *v1.TLSConfiguration) []metav1.StatusCause {
	var statuses []metav1.StatusCause

	if tlsConfiguration == nil {
		return statuses
	}

	if tlsConfiguration.MinTLSVersion == v1.VersionTLS13 || tlsConfiguration.MinTLSVersion == "" {
		if len(tlsConfiguration.Ciphers) > 0 {
			statuses = append(statuses, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "You cannot specify ciphers when spec.configuration.tlsConfiguration.minTLSVersion is empty or VersionTLS13",
				Field:   "spec.configuration.tlsConfiguration.ciphers",
			})
		}
		return statuses
	}

	if len(tlsConfiguration.Ciphers) > 0 {
		var idByName = kvtls.CipherSuiteNameMap()
		for index, cipher := range tlsConfiguration.Ciphers {
			if _, exists := idByName[cipher]; !exists {
				statuses = append(statuses, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: fmt.Sprintf("%s is not a valid cipher", cipher),
					Field:   fmt.Sprintf("spec.configuration.tlsConfiguration.ciphers#%d", index),
				})
			}
		}

		return statuses
	}

	return statuses
}

func ValidateSeccompConfiguration(field *field.Path, seccompConf *v1.SeccompConfiguration) []metav1.StatusCause {
	statuses := []metav1.StatusCause{}
	if seccompConf == nil || seccompConf.VirtualMachineInstanceProfile == nil {
		return statuses
	}

	customProfile := seccompConf.VirtualMachineInstanceProfile.CustomProfile
	customProfileField := field.Child("virtualMachineInstanceProfile").Child("customProfile")

	if customProfile != nil {
		if customProfile.LocalhostProfile != nil && customProfile.RuntimeDefaultProfile {
			localhostProfileField := customProfileField.Child("localhostProfile")
			runtimeDefaultProfileField := customProfileField.Child("runtimeDefaultProfile")
			statuses = append(statuses, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   localhostProfileField.String(),
				Message: fmt.Sprintf("%s cannot be set when %s is set", localhostProfileField.String(), runtimeDefaultProfileField.String()),
			})
			statuses = append(statuses, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   runtimeDefaultProfileField.String(),
				Message: fmt.Sprintf("%s cannot be set when %s is set", runtimeDefaultProfileField.String(), localhostProfileField.String()),
			})
		}
	} else {
		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   customProfileField.String(),
			Message: fmt.Sprintf("%s needs to be set", customProfileField.String()),
		})
	}

	return statuses
}

func ValidateInfraReplicas(replicas *uint8) []metav1.StatusCause {
	statuses := []metav1.StatusCause{}

	if replicas != nil && *replicas == 0 {
		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "infra replica count can't be 0",
		})
	}

	return statuses
}

func ValidateGuestToRequestHeadroom(ratioStrPtr *string) (causes []metav1.StatusCause) {
	if ratioStrPtr == nil {
		return
	}

	ratioStr := *ratioStrPtr

	ratio, err := strconv.ParseFloat(ratioStr, 64)
	if err != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("ratio provided, %s, cannot be parsed into float: %v", ratioStr, err),
		})
		return
	}

	if ratio < 1.0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("ratio provided, %s, cannot be smaller than 1.0", ratioStr),
		})
	}

	return
}

func ValidateFeatureGates(devConfig *v1.DeveloperConfiguration) (causes []metav1.StatusCause) {
	if devConfig == nil {
		return
	}

	enabledFGs := devConfig.FeatureGates
	disabledFGs := devConfig.DisabledFeatureGates

	if len(enabledFGs) == 0 || len(disabledFGs) == 0 {
		return
	}

	// check that the same feature doesn't appear in both FeatureGates and DisabledFeatureGates, emit error otherwise
	for _, enabledFG := range enabledFGs {
		if slices.Contains(disabledFGs, enabledFG) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeForbidden,
				Message: fmt.Sprintf(`feature gate "%s" exists on both "FeatureGates" and "DisabledFeatureGates"`, enabledFG),
				Field:   field.NewPath("spec", "configuration", "developerConfiguration", "featureGates").String(),
			})
		}
	}

	return causes
}

func ValidateVirtTemplateDeployment(config *v1.KubeVirtConfiguration) []metav1.StatusCause {
	virtTemplateDeployment := config.VirtTemplateDeployment
	if virtTemplateDeployment == nil || virtTemplateDeployment.Enabled == nil || !*virtTemplateDeployment.Enabled {
		return nil
	}

	if hasFeatureGateEnabled(config, featuregate.Template) {
		return nil
	}

	return []metav1.StatusCause{{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Field:   "spec.configuration.virtTemplateDeployment.enabled",
		Message: fmt.Sprintf("VirtTemplateDeployment cannot be enabled without enabling the %s feature gate", featuregate.Template),
	}}
}

func ValidateRoleAggregationStrategy(config *v1.KubeVirtConfiguration) []metav1.StatusCause {
	if config.RoleAggregationStrategy == nil || *config.RoleAggregationStrategy == v1.RoleAggregationStrategyAggregateToDefault {
		return nil
	}

	if hasFeatureGateEnabled(config, featuregate.OptOutRoleAggregation) {
		return nil
	}

	return []metav1.StatusCause{{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Field:   "spec.configuration.roleAggregationStrategy",
		Message: fmt.Sprintf("RoleAggregationStrategy cannot be set to Manual without enabling the %s feature gate", featuregate.OptOutRoleAggregation),
	}}
}

func ValidateMigrationConfiguration(oldConfig, newConfig *v1.KubeVirtConfiguration) []metav1.StatusCause {
	if newConfig.MigrationConfiguration == nil {
		return nil
	}

	var causes []metav1.StatusCause
	newMigrationConfig := newConfig.MigrationConfiguration

	if newMigrationConfig.MaxDowntimeMs != nil {
		var oldMaxDowntimeMs *uint64
		if oldConfig.MigrationConfiguration != nil {
			oldMaxDowntimeMs = oldConfig.MigrationConfiguration.MaxDowntimeMs
		}
		if !equality.Semantic.DeepEqual(oldMaxDowntimeMs, newMigrationConfig.MaxDowntimeMs) &&
			!hasFeatureGateEnabled(newConfig, featuregate.MigrationStallDetection) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "spec.configuration.migrationConfiguration.maxDowntimeMs",
				Message: fmt.Sprintf("maxDowntimeMs cannot be modified without enabling the %s feature gate", featuregate.MigrationStallDetection),
			})
		}
	}

	return causes
}

func hasFeatureGateEnabled(config *v1.KubeVirtConfiguration, gate string) bool {
	return featuregate.IsEnabled(gate, config.DeveloperConfiguration)
}
