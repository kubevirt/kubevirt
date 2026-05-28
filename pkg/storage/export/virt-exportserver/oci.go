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

package virtexportserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"slices"
	"strconv"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/virt-template-api/core/v1alpha1"

	"kubevirt.io/kubevirt/pkg/storage/export/export"
	"kubevirt.io/kubevirt/pkg/storage/oci"
)

func ociHTTPHandler(builder *oci.Builder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !builder.Ready() {
			http.Error(w, "OCI export not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/x-tar")
		w.Header().Set("Content-Disposition", `attachment; filename="export.oci.tar"`)
		if size := builder.Size(); size >= 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}
		if err := builder.WriteTar(req.Context(), w); err != nil {
			log.Log.Reason(err).Error("error writing OCI TAR")
		}
	})
}

func newOCIBuilder(paths *export.ServerPaths) (*oci.Builder, error) {
	disks, err := collectDiskInfo(paths)
	if err != nil {
		return nil, err
	}

	if tpl := getVMTemplate(); tpl != nil {
		return newVMTemplateOCIBuilder(tpl, disks)
	}

	if vm := getExpandedVM(); vm != nil {
		return newVMOCIBuilder(vm, disks)
	}

	return nil, nil
}

func collectDiskInfo(paths *export.ServerPaths) ([]oci.DiskInfo, error) {
	var disks []oci.DiskInfo
	for _, vi := range paths.Volumes {
		p := vi.Path

		fi, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("error statting %s: %w", p, err)
		}
		if fi.IsDir() {
			p = path.Join(p, "disk.img")
		}

		disks = append(disks, oci.DiskInfo{
			FilePath:   p,
			VolumeName: path.Base(vi.Path),
		})
	}
	return disks, nil
}

func getVMTemplate() *v1alpha1.VirtualMachineTemplate {
	data, err := os.ReadFile(vmTemplateManifestPath)
	if err != nil {
		log.Log.Reason(err).Info("Unable to load VMTemplate manifest data")
		return nil
	}
	tpl := &v1alpha1.VirtualMachineTemplate{}
	if err := json.Unmarshal(data, tpl); err != nil {
		log.Log.Reason(err).Info("Unable to parse VMTemplate manifest data")
		return nil
	}
	return tpl
}

func newVMTemplateOCIBuilder(tpl *v1alpha1.VirtualMachineTemplate, disks []oci.DiskInfo) (*oci.Builder, error) {
	configJSON, err := prepareVMTemplateConfig(tpl)
	if err != nil {
		return nil, fmt.Errorf("error preparing VMTemplate config: %w", err)
	}

	architecture := extractArchitectureFromVMTemplate(tpl)

	return oci.NewVMTemplateBuilder(configJSON, architecture, disks), nil
}

func prepareVMTemplateConfig(tpl *v1alpha1.VirtualMachineTemplate) ([]byte, error) {
	out := tpl.DeepCopy()

	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = "VirtualMachineTemplate"
	out.Namespace = ""
	out.UID = ""
	out.ResourceVersion = ""
	out.CreationTimestamp.Reset()
	out.Generation = 0
	out.ManagedFields = nil
	out.OwnerReferences = nil
	out.Finalizers = nil
	out.Status = v1alpha1.VirtualMachineTemplateStatus{}

	if out.Spec.VirtualMachine != nil && out.Spec.VirtualMachine.Raw != nil {
		rewritten, err := rewriteEmbeddedVM(&out.Spec, tpl.Namespace)
		if err != nil {
			return nil, fmt.Errorf("error rewriting embedded VM: %w", err)
		}
		out.Spec.VirtualMachine = rewritten
	}

	return json.Marshal(out)
}

// rewriteEmbeddedVM rewrites the embedded VM's DataVolumeTemplates and
// Volumes on a per-element basis. Each DVT and volume is individually
// unmarshalled into a concrete type; if that fails (e.g. due to template
// parameter placeholders without default values), the element is kept as-is.
func rewriteEmbeddedVM(spec *v1alpha1.VirtualMachineTemplateSpec, namespace string) (*k8sruntime.RawExtension, error) {
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
func rewriteDataVolumeTemplates(obj map[string]any, params []v1alpha1.Parameter, namespace string) error {
	matches := export.FindLocalDVTPVCs(obj, params, namespace)
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
func rewriteVolumes(obj map[string]any, params []v1alpha1.Parameter) error {
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
		resolvedName, ok := export.ResolveParameterValue(dvName, params)
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

func extractArchitectureFromVMTemplate(tpl *v1alpha1.VirtualMachineTemplate) string {
	if tpl.Spec.VirtualMachine == nil || tpl.Spec.VirtualMachine.Raw == nil {
		return ""
	}

	var obj map[string]any
	if err := json.Unmarshal(tpl.Spec.VirtualMachine.Raw, &obj); err != nil {
		return ""
	}

	arch, _, _ := unstructured.NestedString(obj, "spec", "template", "spec", "architecture")
	if arch == "" {
		return ""
	}
	resolved, ok := export.ResolveParameterValue(arch, tpl.Spec.Parameters)
	if !ok {
		return ""
	}

	return resolved
}

func newVMOCIBuilder(vm *virtv1.VirtualMachine, disks []oci.DiskInfo) (*oci.Builder, error) {
	configJSON, err := prepareVMConfig(vm)
	if err != nil {
		return nil, fmt.Errorf("error preparing VM config: %w", err)
	}

	architecture := ""
	if vm.Spec.Template != nil {
		architecture = vm.Spec.Template.Spec.Architecture
	}

	return oci.NewVMBuilder(configJSON, architecture, disks), nil
}

func prepareVMConfig(vm *virtv1.VirtualMachine) ([]byte, error) {
	out := vm.DeepCopy()

	gvk := virtv1.VirtualMachineGroupVersionKind
	out.APIVersion, out.Kind = gvk.ToAPIVersionAndKind()
	out.Namespace = ""
	out.UID = ""
	out.ResourceVersion = ""
	out.CreationTimestamp.Reset()
	out.Generation = 0
	out.ManagedFields = nil
	out.OwnerReferences = nil
	out.Finalizers = nil
	out.Status = virtv1.VirtualMachineStatus{}

	out.Spec.DataVolumeTemplates = nil
	if out.Spec.Template != nil {
		for i, vol := range out.Spec.Template.Spec.Volumes {
			if vol.DataVolume != nil {
				out.Spec.Template.Spec.Volumes[i] = virtv1.Volume{
					Name: vol.Name,
					VolumeSource: virtv1.VolumeSource{
						PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: vol.DataVolume.Name,
							},
						},
					},
				}
			}
		}
	}

	return json.Marshal(out)
}
