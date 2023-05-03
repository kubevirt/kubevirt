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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/client-go/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
)

// KubeVirtUpdateAdmitter validates KubeVirt updates
type KubeVirtUpdateAdmitter struct {
	Client        kubecli.KubevirtClient
	ClusterConfig *virtconfig.ClusterConfig
}

// NewKubeVirtUpdateAdmitter creates a KubeVirtUpdateAdmitter
func NewKubeVirtUpdateAdmitter(client kubecli.KubevirtClient, clusterConfig *virtconfig.ClusterConfig) *KubeVirtUpdateAdmitter {
	return &KubeVirtUpdateAdmitter{
		Client:        client,
		ClusterConfig: clusterConfig,
	}
}

func (admitter *KubeVirtUpdateAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	// Get new and old KubeVirt from admission response
	newKV, currKV, err := getAdmissionReviewKubeVirt(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.KubeVirtGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	var results []metav1.StatusCause

	results = append(results, validateCustomizeComponents(newKV.Spec.CustomizeComponents)...)
	results = append(results, validateCertificates(newKV.Spec.CertificateRotationStrategy.SelfSigned)...)
	results = append(results, validateGuestToRequestHeadroom(newKV.Spec.Configuration.AdditionalGuestMemoryOverheadRatio)...)

	if !equality.Semantic.DeepEqual(currKV.Spec.Configuration.TLSConfiguration, newKV.Spec.Configuration.TLSConfiguration) {
		if newKV.Spec.Configuration.TLSConfiguration != nil {
			results = append(results,
				validateTLSConfiguration(newKV.Spec.Configuration.TLSConfiguration)...)
		}
	}

	if !equality.Semantic.DeepEqual(currKV.Spec.Infra, newKV.Spec.Infra) {
		if newKV.Spec.Infra != nil && newKV.Spec.Infra.NodePlacement != nil {
			results = append(results,
				validateInfraPlacement(newKV.Namespace, newKV.Spec.Infra.NodePlacement, admitter.Client)...)
		}
	}

	if !equality.Semantic.DeepEqual(currKV.Spec.Workloads, newKV.Spec.Workloads) {
		if newKV.Spec.Workloads != nil && newKV.Spec.Workloads.NodePlacement != nil {
			results = append(results,
				validateWorkloadPlacement(newKV.Namespace, newKV.Spec.Workloads.NodePlacement, admitter.Client)...)
		}
	}

	if !equality.Semantic.DeepEqual(currKV.Spec.Configuration.SeccompConfiguration, newKV.Spec.Configuration.SeccompConfiguration) {
		results = append(results,
			validateSeccompConfiguration(field.NewPath("spec").Child("configuration", "seccompConfiguration"), newKV.Spec.Configuration.SeccompConfiguration)...)

	}

	if newKV.Spec.Infra != nil {
		results = append(results, validateInfraReplicas(newKV.Spec.Infra.Replicas)...)
	}

	response := validating_webhooks.NewAdmissionResponse(results)

	if featureGatesChanged(&currKV.Spec, &newKV.Spec) {
		featureGates := newKV.Spec.Configuration.DeveloperConfiguration.FeatureGates
		response.Warnings = append(response.Warnings, warnDeprecatedFeatureGates(featureGates, admitter.ClusterConfig)...)
	}

	const mdevWarningfmt = "%s is deprecated, use mediatedDeviceTypes"
	if mdev := newKV.Spec.Configuration.MediatedDevicesConfiguration; mdev != nil {
		f := field.NewPath("spec", "configuration", "mediatedDevicesConfiguration")
		if mdev.MediatedDevicesTypes != nil {
			f := f.Child("mediatedDevicesTypes")
			response.Warnings = append(response.Warnings, fmt.Sprintf(mdevWarningfmt, f.String()))
		}

		f = f.Child("nodeMediatedDeviceTypes")
		for i, mdevType := range mdev.NodeMediatedDeviceTypes {
			f := f.Index(i).Child("mediatedDevicesTypes")
			if mdevType.MediatedDevicesTypes != nil {
				response.Warnings = append(response.Warnings, fmt.Sprintf(mdevWarningfmt, f.String()))
			}
		}
	}

	return response
}

func getAdmissionReviewKubeVirt(ar *admissionv1.AdmissionReview) (new *v1.KubeVirt, old *v1.KubeVirt, err error) {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, KubeVirtGroupVersionResource.Group, KubeVirtGroupVersionResource.Resource) {
		return nil, nil, fmt.Errorf("expect resource to be '%s'", KubeVirtGroupVersionResource)
	}

	raw := ar.Request.Object.Raw
	newKV := v1.KubeVirt{}

	err = json.Unmarshal(raw, &newKV)
	if err != nil {
		return nil, nil, err
	}

	if ar.Request.Operation == admissionv1.Update {
		raw := ar.Request.OldObject.Raw
		oldKV := v1.KubeVirt{}
		err = json.Unmarshal(raw, &oldKV)
		if err != nil {
			return nil, nil, err
		}

		return &newKV, &oldKV, nil
	}

	return &newKV, nil, nil
}

func validateCustomizeComponents(customization v1.CustomizeComponents) []metav1.StatusCause {
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

func validateCertificates(certConfig *v1.KubeVirtSelfSignConfiguration) []metav1.StatusCause {
	statuses := []metav1.StatusCause{}

	if certConfig == nil {
		return statuses
	}

	deprecatedApi := false
	if certConfig.CARotateInterval != nil || certConfig.CertRotateInterval != nil || certConfig.CAOverlapInterval != nil {
		deprecatedApi = true
	}

	currentApi := false
	if certConfig.CA != nil || certConfig.Server != nil {
		currentApi = true
	}

	if deprecatedApi && currentApi {
		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("caRotateInterval, certRotateInterval and caOverlapInterval are deprecated and conflict with CertConfig defined rotation parameters"),
		})
	}

	caDuration := apply.GetCADuration(certConfig)
	caRenewBefore := apply.GetCARenewBefore(certConfig)
	certDuration := apply.GetCertDuration(certConfig)
	certRenewBefore := apply.GetCertRenewBefore(certConfig)

	if caDuration.Duration < caRenewBefore.Duration {
		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("CA RenewBefore cannot exceed Duration (spec.certificateRotationStrategy.selfSigned.ca.duration < spec.certificateRotationStrategy.selfSigned.ca.renewBefore)"),
		})

	}

	if certDuration.Duration < certRenewBefore.Duration {
		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Cert RenewBefore cannot exceed Duration (spec.certificateRotationStrategy.selfSigned.server.duration < spec.certificateRotationStrategy.selfSigned.server.renewBefore)"),
		})
	}

	if certDuration.Duration > caDuration.Duration {
		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Certificate duration cannot exceed CA (spec.certificateRotationStrategy.selfSigned.server.duration > spec.certificateRotationStrategy.selfSigned.ca.duration)"),
		})
	}

	return statuses
}

func validateTLSConfiguration(tlsConfiguration *v1.TLSConfiguration) []metav1.StatusCause {
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

func validateSeccompConfiguration(field *field.Path, seccompConf *v1.SeccompConfiguration) []metav1.StatusCause {
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

func validateWorkloadPlacement(namespace string, placementConfig *v1.NodePlacement, client kubecli.KubevirtClient) []metav1.StatusCause {
	statuses := []metav1.StatusCause{}

	const (
		dsName    = "placement-validation-webhook"
		mockLabel = "kubevirt.io/choose-me"
		podName   = "placement-verification-pod"
		mockUrl   = "test.only:latest"
	)

	mockDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: dsName,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					mockLabel: "",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: podName,
					Labels: map[string]string{
						mockLabel: "",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  podName,
							Image: mockUrl,
						},
					},
					// Inject placement fields here
					NodeSelector: placementConfig.NodeSelector,
					Affinity:     placementConfig.Affinity,
					Tolerations:  placementConfig.Tolerations,
				},
			},
		},
	}

	_, err := client.AppsV1().DaemonSets(namespace).Create(context.Background(), mockDaemonSet, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})

	if err != nil {
		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err.Error(),
		})
	}
	return statuses
}

func validateInfraPlacement(namespace string, placementConfig *v1.NodePlacement, client kubecli.KubevirtClient) []metav1.StatusCause {
	statuses := []metav1.StatusCause{}

	const (
		deploymentName = "placement-validation-webhook"
		mockLabel      = "kubevirt.io/choose-me"
		podName        = "placement-verification-pod"
		mockUrl        = "test.only:latest"
	)

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					mockLabel: "",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: podName,
					Labels: map[string]string{
						mockLabel: "",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  podName,
							Image: mockUrl,
						},
					},
					// Inject placement fields here
					NodeSelector: placementConfig.NodeSelector,
					Affinity:     placementConfig.Affinity,
					Tolerations:  placementConfig.Tolerations,
				},
			},
		},
	}

	_, err := client.AppsV1().Deployments(namespace).Create(context.Background(), mockDeployment, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})

	if err != nil {
		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err.Error(),
		})
	}

	return statuses
}

func validateInfraReplicas(replicas *uint8) []metav1.StatusCause {
	statuses := []metav1.StatusCause{}

	if replicas != nil && *replicas == 0 {
		statuses = append(statuses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "infra replica count can't be 0",
		})
	}

	return statuses
}

func featureGatesChanged(currKVSpec, newKVSpec *v1.KubeVirtSpec) bool {
	currDevConfig := currKVSpec.Configuration.DeveloperConfiguration
	newDevConfig := newKVSpec.Configuration.DeveloperConfiguration

	if (currDevConfig == nil && newDevConfig == nil) || (currDevConfig != nil && newDevConfig == nil) {
		return false
	}

	if currDevConfig == nil && newDevConfig != nil {
		return len(newDevConfig.FeatureGates) > 0
	}

	return !equality.Semantic.DeepEqual(currDevConfig.FeatureGates, newDevConfig.FeatureGates)
}

func warnDeprecatedFeatureGates(featureGates []string, config *virtconfig.ClusterConfig) (warnings []string) {
	for _, featureGate := range featureGates {
		if config.IsFeatureGateDeprecated(featureGate) {
			const warningPattern = "feature gate %s is deprecated, therefore it can be safely removed and is redundant. " +
				"For more info, please look at: https://github.com/kubevirt/kubevirt/blob/main/docs/deprecation.md"
			warnings = append(warnings, fmt.Sprintf(warningPattern, featureGate))

			log.Log.Warningf(warningPattern, featureGate)
		}
	}

	return warnings
}

func validateGuestToRequestHeadroom(ratioStrPtr *string) (causes []metav1.StatusCause) {
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
