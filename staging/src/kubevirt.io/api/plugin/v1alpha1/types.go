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
// +kubebuilder:validation:XValidation:rule="!has(self.spec.launcherHooks) || self.spec.launcherHooks.all(lh, !has(lh.sidecar) || lh.sidecar.socketPath.startsWith('/var/run/kubevirt-plugin/' + self.metadata.name + '/'))",message="sidecar socketPath must start with /var/run/kubevirt-plugin/<plugin-name>/"
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
	// Condition expressions are evaluated once per pipeline invocation, against the VMI/domain
	// state as it existed before any hook in that invocation ran. They are not re-evaluated to
	// reflect mutations made by other hooks applied earlier in the same invocation.
	// +optional
	Condition string `json:"condition,omitempty"`

	// FailureStrategy specifies the default behavior when the plugin itself is unhealthy
	// (e.g. a referenced webhook is not ready, or a sidecar socket is unreachable).
	// Individual hooks may override this with their own FailureStrategy.
	// +optional
	FailureStrategy FailureStrategy `json:"failureStrategy,omitempty"`

	// LauncherHooks defines hooks that run inside the virt-launcher pod at well-defined
	// points in the VM lifecycle.
	// Hooks are applied in declaration order within each plugin.
	// Across plugins, hooks are applied in alphabetical order by plugin name.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=32
	LauncherHooks []LauncherHook `json:"launcherHooks,omitempty"`

	// NodeHooks defines hooks that execute during VM lifecycle events.
	// Hooks are applied in declaration order within each plugin.
	// Across plugins, hooks are applied in alphabetical order by plugin name.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=32
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
// MaxLength bounds the CEL cost estimate of the value-set rule; it must stay
// at or above the longest member of the set.
// +enum
// +kubebuilder:validation:MaxLength=32
// +kubebuilder:validation:XValidation:rule="self in ['Fail','Ignore']",message="failureStrategy must be one of: Fail, Ignore"
type FailureStrategy string

const (
	FailureStrategyFail   FailureStrategy = "Fail"
	FailureStrategyIgnore FailureStrategy = "Ignore"
)

// InvocationContext identifies why a domain hook is being invoked.
// +enum
type InvocationContext string

const (
	InvocationContextBoot            InvocationContext = "Boot"
	InvocationContextMigrationSource InvocationContext = "MigrationSource"
	InvocationContextMigrationTarget InvocationContext = "MigrationTarget"
)

// LauncherHook defines a hook that runs inside the virt-launcher pod at a specific point in the VM lifecycle.
// Exactly one of cel or sidecar must be specified.
// +kubebuilder:validation:XValidation:rule="has(self.cel) != has(self.sidecar)",message="a launcher hook must define exactly one of cel or sidecar"
// +kubebuilder:validation:XValidation:rule="!has(self.timeout) || duration(self.timeout) > duration('0s')",message="timeout must be greater than zero"
type LauncherHook struct {
	// CEL defines a CEL expression hook.
	// +optional
	CEL *CELLauncherHook `json:"cel,omitempty"`
	// Sidecar defines a sidecar-based hook that communicates via a Unix socket.
	// +optional
	Sidecar *SidecarLauncherHook `json:"sidecar,omitempty"`
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

// LauncherHookPoint identifies a point in the VM lifecycle where a launcher hook can run.
// The set is closed and validated at the API boundary. It is append-only across versions:
// new hook points may be added, existing ones must never be removed.
// MaxLength bounds the CEL cost estimate of the value-set rule; it must stay
// at or above the longest member of the set.
// +enum
// +kubebuilder:validation:MaxLength=32
// +kubebuilder:validation:XValidation:rule="self in ['GuestDefinition','PreBoot','PreMigrationSource']",message="hook point must be one of: GuestDefinition, PreBoot, PreMigrationSource"
type LauncherHookPoint string

const (
	LauncherHookGuestDefinition    LauncherHookPoint = "GuestDefinition"
	LauncherHookPreBoot            LauncherHookPoint = "PreBoot"
	LauncherHookPreMigrationSource LauncherHookPoint = "PreMigrationSource"
)

type CELLauncherHook struct {
	// HookPoint specifies which launcher hook point this CEL expression applies to.
	// GuestDefinition is the only hook point that supports CEL today; a CEL hook declaring any
	// other hook point is ignored.
	HookPoint LauncherHookPoint `json:"hookPoint"`
	// Expression is the CEL expression applied at the specified hook point.
	// +kubebuilder:validation:MinLength=1
	Expression string `json:"expression"`
}

// +kubebuilder:validation:XValidation:rule="!self.socketPath.contains('..') && !self.socketPath.contains('//') && !self.socketPath.endsWith('/')",message="socketPath must be a clean path: no '..' segments, repeated separators or trailing separator"
type SidecarLauncherHook struct {
	// SocketPath is the path to the Unix socket used to communicate with the sidecar.
	// MaxLength is bounded by the sockaddr_un sun_path limit (108 bytes), an absolute
	// platform invariant.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=108
	SocketPath string `json:"socketPath"`
	// PermittedHooks lists the launcher hook points this sidecar handles.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=32
	// +listType=set
	PermittedHooks []LauncherHookPoint `json:"permittedHooks"`
}

// NodeHook defines a hook that runs an executable on the hosting node during VM lifecycle events.
// Unlike LauncherHooks which run inside the virt-launcher pod, NodeHooks perform node-level operations
// such as configuring networking, storage preparation, or device management.
// Hooks may fire multiple times for the same lifecycle event due to reconciliation retries.
// Implementations must be idempotent.
// +kubebuilder:validation:XValidation:rule="!self.socket.contains('..') && !self.socket.contains('//') && !self.socket.endsWith('/')",message="socket must be a clean path: no '..' segments, repeated separators or trailing separator"
// +kubebuilder:validation:XValidation:rule="!has(self.timeout) || duration(self.timeout) > duration('0s')",message="timeout must be greater than zero"
type NodeHook struct {
	// Socket is the path to the Unix socket for hook communication.
	// MaxLength is bounded by the sockaddr_un sun_path limit (108 bytes), an absolute
	// platform invariant.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=108
	Socket string `json:"socket"`
	// PermittedHooks lists the VM lifecycle events this hook handles.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=32
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
// The set is closed and validated at the API boundary. It is append-only across versions:
// new hook points may be added, existing ones must never be removed.
// MaxLength bounds the CEL cost estimate of the value-set rule; it must stay
// at or above the longest member of the set.
// +enum
// +kubebuilder:validation:MaxLength=32
// +kubebuilder:validation:XValidation:rule="self in ['PreVMStart','PostVMStart','OnVMStop','PostVMStop','PreMigrationSource','PreMigrationTarget','PostMigrationTarget']",message="hook point must be one of: PreVMStart, PostVMStart, OnVMStop, PostVMStop, PreMigrationSource, PreMigrationTarget, PostMigrationTarget"
type NodeHookPoint string

const (
	NodeHookPreVMStart          NodeHookPoint = "PreVMStart"
	NodeHookPostVMStart         NodeHookPoint = "PostVMStart"
	NodeHookOnVMStop            NodeHookPoint = "OnVMStop"
	NodeHookPostVMStop          NodeHookPoint = "PostVMStop"
	NodeHookPreMigrationSource  NodeHookPoint = "PreMigrationSource"
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
