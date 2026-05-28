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

package export

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/log"
	templateapi "kubevirt.io/virt-template-api/core"
	"kubevirt.io/virt-template-api/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	vmTemplateKind     = "VirtualMachineTemplate"
	vmTemplateManifest = "virtualmachinetemplate-manifest"

	vmTemplateNotFoundReason = "VMTemplateNotFound"
	vmTemplateNotReadyReason = "VMTemplateNotReady"
)

type VMTemplateSource struct {
	tpl           *v1beta1.VirtualMachineTemplate
	sourceVolumes *sourceVolumes
}

func NewVMTemplateSource(tpl *v1beta1.VirtualMachineTemplate, sourceVolumes *sourceVolumes) *VMTemplateSource {
	return &VMTemplateSource{
		tpl:           tpl,
		sourceVolumes: sourceVolumes,
	}
}

func (s *VMTemplateSource) IsSourceAvailable() bool {
	return s.sourceVolumes.isSourceAvailable()
}

func (s *VMTemplateSource) HasContent() bool {
	return s.sourceVolumes.hasContent()
}

func (s *VMTemplateSource) SourceCondition() exportv1.Condition {
	return s.sourceVolumes.sourceCondition
}

func (s *VMTemplateSource) ReadyCondition() exportv1.Condition {
	return s.sourceVolumes.readyCondition
}

func (s *VMTemplateSource) ServicePorts() []corev1.ServicePort {
	return []corev1.ServicePort{exportPort()}
}

func (s *VMTemplateSource) ConfigurePod(pod *corev1.Pod) {
	s.sourceVolumes.configurePodVolumes(pod)
}

func (s *VMTemplateSource) ConfigureExportLink(_ *exportv1.VirtualMachineExportLink, _ *ServerPaths, _ *exportv1.VirtualMachineExport, _ *corev1.Pod, _, _ string) {
}

func (s *VMTemplateSource) UpdateStatus(vmExport *exportv1.VirtualMachineExport, _ *corev1.Pod, _ *corev1.Service) (time.Duration, error) {
	if !s.HasContent() {
		vmExport.Status.Phase = exportv1.Skipped
	}

	if !s.sourceVolumes.isPopulated &&
		s.ReadyCondition().Reason != vmTemplateNotFoundReason {
		return requeueTime, nil
	}

	return 0, nil
}

func (s *VMTemplateSource) ManifestData() (key string, data []byte, extra map[string]string, err error) {
	if s.tpl == nil {
		return "", nil, nil, nil
	}

	out := s.tpl.DeepCopy()
	out.Status = v1beta1.VirtualMachineTemplateStatus{}
	out.ManagedFields = nil
	cleanedObjectMeta := metav1.ObjectMeta{}
	cleanedObjectMeta.Name = out.ObjectMeta.Name
	cleanedObjectMeta.Labels = out.ObjectMeta.Labels
	cleanedObjectMeta.Annotations = out.ObjectMeta.Annotations
	out.ObjectMeta = cleanedObjectMeta

	if out.Spec.VirtualMachine != nil && out.Spec.VirtualMachine.Raw != nil {
		rewritten, err := rewriteEmbeddedVM(&out.Spec, s.tpl.Namespace)
		if err != nil {
			return "", nil, nil, fmt.Errorf("error rewriting embedded VM: %w", err)
		}
		out.Spec.VirtualMachine = rewritten
	}

	tplBytes, err := json.Marshal(out)
	if err != nil {
		return "", nil, nil, err
	}

	return vmTemplateManifest, tplBytes, nil, nil
}

func (ctrl *VMExportController) handleVMTemplate(obj any) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if tpl, ok := obj.(*v1beta1.VirtualMachineTemplate); ok {
		tplKey, _ := cache.MetaNamespaceKeyFunc(tpl)
		keys, err := ctrl.VMExportInformer.GetIndexer().IndexKeys(templateapi.SingularResourceName, tplKey)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, key := range keys {
			log.Log.V(3).Infof("Adding VMExport due to template %s", tplKey)
			ctrl.vmExportQueue.Add(key)
		}
	}
}

func (ctrl *VMExportController) getPVCFromSourceVMTemplate(vmExport *exportv1.VirtualMachineExport) (*v1beta1.VirtualMachineTemplate, *sourceVolumes, error) {
	sourceVolumes := &sourceVolumes{
		volumes:         []sourceVolume{},
		inUse:           false,
		isPopulated:     false,
		readyCondition:  newReadyCondition(corev1.ConditionFalse, initializingReason, ""),
		sourceCondition: exportv1.Condition{},
	}

	tpl, exists, err := ctrl.getVMTemplate(vmExport.Namespace, vmExport.Spec.Source.Name)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		sourceVolumes.readyCondition = newReadyCondition(corev1.ConditionFalse, vmTemplateNotFoundReason,
			fmt.Sprintf("VirtualMachineTemplate %s/%s not found", vmExport.Namespace, vmExport.Spec.Source.Name))
		return nil, sourceVolumes, nil
	}

	if !meta.IsStatusConditionTrue(tpl.Status.Conditions, v1beta1.ConditionReady) {
		sourceVolumes.readyCondition = newReadyCondition(corev1.ConditionFalse, vmTemplateNotReadyReason,
			fmt.Sprintf("VirtualMachineTemplate %s/%s is not ready", vmExport.Namespace, vmExport.Spec.Source.Name))
		return nil, sourceVolumes, nil
	}

	pvcs, allPopulated, err := ctrl.getPVCsFromVMTemplate(tpl)
	if err != nil {
		return nil, nil, err
	}
	log.Log.V(3).Infof("Number of volumes found for VMTemplate %s/%s: %d, allPopulated %t", vmExport.Namespace, vmExport.Spec.Source.Name, len(pvcs), allPopulated)

	sourceVolumes.isPopulated = allPopulated

	if len(pvcs) == 0 && allPopulated {
		sourceVolumes.isPopulated = true
		sourceVolumes.readyCondition = newReadyCondition(corev1.ConditionFalse, noVolumeVMReason,
			fmt.Sprintf("VirtualMachineTemplate %s/%s has no volumes", vmExport.Namespace, vmExport.Spec.Source.Name))
	} else if !allPopulated {
		sourceVolumes.readyCondition = newReadyCondition(corev1.ConditionFalse, volumesNotPopulatedReason,
			fmt.Sprintf("Not all volumes in VirtualMachineTemplate %s/%s are populated", vmExport.Namespace, vmExport.Spec.Source.Name))
	}

	sourceVolumes.volumes = ctrl.pvcsToSourceVolumes(pvcs...)

	return tpl, sourceVolumes, nil
}

func (ctrl *VMExportController) getPVCsFromVMTemplate(tpl *v1beta1.VirtualMachineTemplate) ([]*corev1.PersistentVolumeClaim, bool, error) {
	if tpl.Spec.VirtualMachine == nil || tpl.Spec.VirtualMachine.Raw == nil {
		return nil, true, nil
	}

	var pvcs []*corev1.PersistentVolumeClaim
	allPopulated := true

	addPVC := func(name string) error {
		pvc, exists, err := ctrl.getPvc(tpl.Namespace, name)
		if err != nil {
			return err
		}
		if !exists {
			allPopulated = false
			return nil
		}
		populated, err := ctrl.isPVCPopulated(pvc)
		if err != nil {
			return err
		}
		pvcs = append(pvcs, pvc)
		if !populated {
			allPopulated = false
		}
		return nil
	}

	var obj map[string]any
	if err := json.Unmarshal(tpl.Spec.VirtualMachine.Raw, &obj); err != nil {
		return nil, false, fmt.Errorf("failed to parse embedded VM: %w", err)
	}

	dvtPVCNames, dvtNames := extractLocalDVTPVCNames(obj, tpl.Spec.Parameters, tpl.Namespace)
	for _, pvcName := range dvtPVCNames {
		if err := addPVC(pvcName); err != nil {
			return nil, false, err
		}
	}

	volPVCNames := extractVolumePVCNames(obj, tpl.Spec.Parameters)
	for rawName, resolvedName := range volPVCNames {
		if slices.Contains(dvtNames, rawName) {
			continue
		}
		if err := addPVC(resolvedName); err != nil {
			return nil, false, err
		}
	}

	return pvcs, allPopulated, nil
}

func (ctrl *VMExportController) isSourceVMTemplate(source *exportv1.VirtualMachineExportSpec) bool {
	return ctrl.VMTemplateInformer != nil &&
		source != nil && source.Source.APIGroup != nil &&
		*source.Source.APIGroup == templateapi.GroupName &&
		source.Source.Kind == vmTemplateKind
}

func (ctrl *VMExportController) getVMTemplate(namespace, name string) (*v1beta1.VirtualMachineTemplate, bool, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.VMTemplateInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return obj.(*v1beta1.VirtualMachineTemplate).DeepCopy(), true, nil
}

func extractLocalDVTPVCNames(obj map[string]any, params []v1beta1.Parameter, namespace string) (pvcNames, dvtNames []string) {
	for _, dvt := range findLocalDVTPVCs(obj, params, namespace) {
		pvcNames = append(pvcNames, dvt.ResolvedName)
		dvtNames = append(dvtNames, dvt.DVTName)
	}
	return pvcNames, dvtNames
}

// localDVTPVC describes a DataVolumeTemplate with a local PVC source.
type localDVTPVC struct {
	Index        int
	DVTName      string
	ResolvedName string
}

// findLocalDVTPVCs returns DVTs that reference a PVC in the same namespace
// (or no namespace). DVTs with cross-namespace sources, missing metadata
// names, or unresolvable parameter placeholders are skipped.
func findLocalDVTPVCs(obj map[string]any, params []v1beta1.Parameter, namespace string) []localDVTPVC {
	dvts, found, _ := unstructured.NestedSlice(obj, "spec", "dataVolumeTemplates")
	if !found {
		return nil
	}

	var results []localDVTPVC
	for i, dvt := range dvts {
		dvtMap, ok := dvt.(map[string]any)
		if !ok {
			continue
		}
		pvcSource, hasPVC, _ := unstructured.NestedMap(dvtMap, "spec", "source", "pvc")
		if !hasPVC {
			continue
		}
		srcNs, _, _ := unstructured.NestedString(pvcSource, "namespace")
		if srcNs != "" && srcNs != namespace {
			continue
		}
		srcName, _, _ := unstructured.NestedString(pvcSource, "name")
		if srcName == "" {
			continue
		}
		resolved, ok := ResolveParameterValue(srcName, params)
		if !ok {
			continue
		}
		dvtName, _, _ := unstructured.NestedString(dvtMap, "metadata", "name")
		if dvtName == "" {
			continue
		}
		results = append(results, localDVTPVC{Index: i, DVTName: dvtName, ResolvedName: resolved})
	}
	return results
}

// extractVolumePVCNames returns a map from raw volume PVC name to resolved
// PVC name for volumes that reference a PersistentVolumeClaim or DataVolume.
// The raw name is useful for dedup against DVT names (which may contain
// parameter placeholders). Volumes with unresolvable placeholders are skipped.
func extractVolumePVCNames(obj map[string]any, params []v1beta1.Parameter) map[string]string {
	volumes, found, _ := unstructured.NestedSlice(obj, "spec", "template", "spec", "volumes")
	if !found {
		return nil
	}

	names := make(map[string]string)
	for _, vol := range volumes {
		volMap, ok := vol.(map[string]any)
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(volMap, "persistentVolumeClaim", "claimName")
		if name == "" {
			name, _, _ = unstructured.NestedString(volMap, "dataVolume", "name")
		}
		if name == "" {
			continue
		}
		resolved, ok := ResolveParameterValue(name, params)
		if !ok {
			continue
		}
		names[name] = resolved
	}

	return names
}

// rewriteEmbeddedVM rewrites the embedded VM's DataVolumeTemplates and
// Volumes on a per-element basis. Each DVT and volume is individually
// unmarshalled into a concrete type; if that fails (e.g. due to template
// parameter placeholders without default values), the element is kept as-is.
func rewriteEmbeddedVM(spec *v1beta1.VirtualMachineTemplateSpec, namespace string) (*k8sruntime.RawExtension, error) {
	var obj map[string]any
	if err := json.Unmarshal(spec.VirtualMachine.Raw, &obj); err != nil {
		return nil, fmt.Errorf("error unmarshalling embedded VM: %w", err)
	}

	if err := rewriteDataVolumeTemplates(obj, spec.Parameters, namespace); err != nil {
		return nil, fmt.Errorf("error rewriting dataVolumeTemplates: %w", err)
	}

	if err := rewriteVolumes(obj, spec.Parameters); err != nil {
		return nil, fmt.Errorf("error rewriting volumes: %w", err)
	}

	rewritten, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error marshalling rewritten VM: %w", err)
	}

	return &k8sruntime.RawExtension{Raw: rewritten}, nil
}

// rewriteDataVolumeTemplates rewrites resolved DVTs (those with a local
// PVC source) to reference the exported PVC by name. Parameter placeholders
// are resolved using default values; elements with unresolvable placeholders
// or cross-namespace sources are kept as-is.
func rewriteDataVolumeTemplates(obj map[string]any, params []v1beta1.Parameter, namespace string) error {
	matches := findLocalDVTPVCs(obj, params, namespace)
	if len(matches) == 0 {
		return nil
	}

	dvts, _, _ := unstructured.NestedSlice(obj, "spec", "dataVolumeTemplates")
	for _, match := range matches {
		dvtMap, ok := dvts[match.Index].(map[string]any)
		if !ok {
			continue
		}
		if err := unstructured.SetNestedField(dvtMap, map[string]any{
			"pvc": map[string]any{"name": match.ResolvedName},
		}, "spec", "source"); err != nil {
			return fmt.Errorf("error setting source for DVT %s: %w", match.DVTName, err)
		}
		unstructured.RemoveNestedField(dvtMap, "spec", "sourceRef")
	}

	return unstructured.SetNestedSlice(obj, dvts, "spec", "dataVolumeTemplates")
}

// rewriteVolumes converts DataVolume volume sources to
// PersistentVolumeClaim volume sources. Parameter placeholders are
// resolved using default values; elements with unresolvable placeholders
// are kept as-is.
func rewriteVolumes(obj map[string]any, params []v1beta1.Parameter) error {
	volumes, found, _ := unstructured.NestedSlice(obj, "spec", "template", "spec", "volumes")
	if !found {
		return nil
	}

	dvtNames := extractDVTNames(obj)

	for i, item := range volumes {
		volMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		dvName, _, _ := unstructured.NestedString(volMap, "dataVolume", "name")
		if dvName == "" {
			continue
		}
		if slices.Contains(dvtNames, dvName) {
			continue
		}
		resolvedName, ok := ResolveParameterValue(dvName, params)
		if !ok {
			continue
		}
		volName, _, _ := unstructured.NestedString(volMap, "name")
		volumes[i] = map[string]any{
			"name": volName,
			"persistentVolumeClaim": map[string]any{
				"claimName": resolvedName,
			},
		}
	}

	return unstructured.SetNestedSlice(obj, volumes, "spec", "template", "spec", "volumes")
}

func extractDVTNames(obj map[string]any) []string {
	dvts, found, _ := unstructured.NestedSlice(obj, "spec", "dataVolumeTemplates")
	if !found {
		return nil
	}
	var names []string
	for _, item := range dvts {
		dvtMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(dvtMap, "metadata", "name")
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// ResolveParameterValue substitutes ${PARAM} placeholders in s using the
// template's parameter default values. Returns the resolved string and
// true if all placeholders were resolved.
func ResolveParameterValue(s string, params []v1beta1.Parameter) (string, bool) {
	for _, p := range params {
		if p.Value == "" {
			continue
		}
		s = strings.ReplaceAll(s, fmt.Sprintf("${%s}", p.Name), p.Value)
	}
	return s, !strings.Contains(s, "${")
}
