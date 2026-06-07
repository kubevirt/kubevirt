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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
// +genclient:nonNamespaced

// Plugin defines a KubeVirt extension that can modify VM domain XML,
// hook into VM lifecycle events, and reference admission objects.
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines the plugin's hooks and admission references.
	Spec PluginSpec `json:"spec"`
	// Status reflects the observed state of the plugin.
	// +optional
	Status PluginStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=atomic
	Items []Plugin `json:"items"`
}

type PluginSpec struct {
	// Condition is a CEL expression that determines whether this plugin applies to a given VM.
	// When set, this acts as a baseline filter for all hooks in the plugin.
	// Individual hooks may further narrow the scope with their own Condition fields.
	// +optional
	Condition string `json:"condition,omitempty"`

	// FailureStrategy specifies the default behavior when the plugin itself is unhealthy
	// (e.g. a referenced webhook is not ready, or a sidecar socket is unreachable).
	// Individual hooks may override this with their own FailureStrategy.
	// +optional
	FailureStrategy FailureStrategy `json:"failureStrategy,omitempty"`

	// DomainHooks defines hooks that modify the libvirt domain XML.
	// Hooks are applied in declaration order within each plugin.
	// Across plugins, hooks are applied in alphabetical order by plugin name.
	// +optional
	// +listType=atomic
	DomainHooks []DomainHook `json:"domainHooks,omitempty"`

	// NodeHooks defines hooks that execute during VM lifecycle events.
	// +optional
	// +listType=atomic
	NodeHooks []NodeHook `json:"nodeHooks,omitempty"`

	// MutatingAdmissionPolicies references MutatingAdmissionPolicy objects managed by the plugin.
	// +optional
	// +listType=atomic
	MutatingAdmissionPolicies []AdmissionReference `json:"mutatingAdmissionPolicies,omitempty"`
	// ValidatingAdmissionPolicies references ValidatingAdmissionPolicy objects managed by the plugin.
	// +optional
	// +listType=atomic
	ValidatingAdmissionPolicies []AdmissionReference `json:"validatingAdmissionPolicies,omitempty"`
	// MutatingAdmissionWebhooks references MutatingWebhookConfiguration objects managed by the plugin.
	// +optional
	// +listType=atomic
	MutatingAdmissionWebhooks []AdmissionReference `json:"mutatingAdmissionWebhooks,omitempty"`
	// ValidatingAdmissionWebhooks references ValidatingWebhookConfiguration objects managed by the plugin.
	// +optional
	// +listType=atomic
	ValidatingAdmissionWebhooks []AdmissionReference `json:"validatingAdmissionWebhooks,omitempty"`
}

// FailureStrategy specifies how hook failures are handled. Defaults to Fail if not specified.
// +enum
type FailureStrategy string

const (
	FailureStrategyFail   FailureStrategy = "Fail"
	FailureStrategyIgnore FailureStrategy = "Ignore"
)

// DomainHook defines a hook that modifies the libvirt domain XML.
// Exactly one of cel or sidecar must be specified.
type DomainHook struct {
	// CEL defines a CEL expression that transforms the domain XML.
	// +optional
	CEL *CELDomainHook `json:"cel,omitempty"`
	// Sidecar defines a sidecar-based hook that transforms the domain XML via a Unix socket.
	// +optional
	Sidecar *SidecarDomainHook `json:"sidecar,omitempty"`
	// Condition is a CEL expression that determines whether this hook applies to a given VM.
	// +optional
	Condition string `json:"condition,omitempty"`
	// FailureStrategy specifies how to handle hook failures (Fail or Ignore).
	// +optional
	FailureStrategy FailureStrategy `json:"failureStrategy,omitempty"`
	// Timeout specifies the maximum duration to wait for the hook to complete.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

type CELDomainHook struct {
	// Expression is the CEL expression applied to the domain XML.
	// +kubebuilder:validation:MinLength=1
	Expression string `json:"expression"`
}

type SidecarDomainHook struct {
	// SocketPath is the path to the Unix socket used to communicate with the sidecar.
	// +kubebuilder:validation:MinLength=1
	SocketPath string `json:"socketPath"`
}

// NodeHook defines a hook that runs an executable on the hosting node during VM lifecycle events.
// Unlike DomainHooks which modify the libvirt domain XML, NodeHooks perform node-level operations
// such as configuring networking, storage preparation, or device management.
type NodeHook struct {
	// Socket is the path to the Unix socket for hook communication.
	// +kubebuilder:validation:MinLength=1
	Socket string `json:"socket"`
	// PermittedHooks lists the VM lifecycle events this hook handles.
	// +kubebuilder:validation:MinItems=1
	// +listType=atomic
	PermittedHooks []NodeHookPoint `json:"permittedHooks"`
	// Condition is a CEL expression that determines whether this hook applies to a given VM.
	// +optional
	Condition string `json:"condition,omitempty"`
	// FailureStrategy specifies how to handle hook failures (Fail or Ignore).
	// +optional
	FailureStrategy FailureStrategy `json:"failureStrategy,omitempty"`
	// Timeout specifies the maximum duration to wait for the hook to complete.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// NodeHookPoint identifies a VM lifecycle event for node-level hooks.
// +enum
type NodeHookPoint string

const (
	NodeHookPreVMStart          NodeHookPoint = "PreVMStart"
	NodeHookPostVMStart         NodeHookPoint = "PostVMStart"
	NodeHookPreVMStop           NodeHookPoint = "PreVMStop"
	NodeHookPostVMStop          NodeHookPoint = "PostVMStop"
	NodeHookPreMigrationSource  NodeHookPoint = "PreMigrationSource"
	NodeHookPostMigrationSource NodeHookPoint = "PostMigrationSource"
	NodeHookPreMigrationTarget  NodeHookPoint = "PreMigrationTarget"
	NodeHookPostMigrationTarget NodeHookPoint = "PostMigrationTarget"
)

// AdmissionReference is a reference to an admission object by name.
type AdmissionReference struct {
	// Name is the name of the admission object.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

type PluginStatus struct{}
