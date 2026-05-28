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
	"strconv"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/virt-template-api/core/v1beta1"

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

func getVMTemplate() *v1beta1.VirtualMachineTemplate {
	data, err := os.ReadFile(vmTemplateManifestPath)
	if err != nil {
		log.Log.Reason(err).Info("Unable to load VMTemplate manifest data")
		return nil
	}
	tpl := &v1beta1.VirtualMachineTemplate{}
	if err := json.Unmarshal(data, tpl); err != nil {
		log.Log.Reason(err).Info("Unable to parse VMTemplate manifest data")
		return nil
	}
	return tpl
}

func newVMTemplateOCIBuilder(tpl *v1beta1.VirtualMachineTemplate, disks []oci.DiskInfo) (*oci.Builder, error) {
	configJSON, err := prepareVMTemplateConfig(tpl)
	if err != nil {
		return nil, fmt.Errorf("error preparing VMTemplate config: %w", err)
	}

	architecture := extractArchitectureFromVMTemplate(tpl)

	return oci.NewVMTemplateBuilder(configJSON, architecture, disks), nil
}

func prepareVMTemplateConfig(tpl *v1beta1.VirtualMachineTemplate) ([]byte, error) {
	out := tpl.DeepCopy()
	out.APIVersion = v1beta1.GroupVersion.String()
	out.Kind = "VirtualMachineTemplate"
	return json.Marshal(out)
}

func extractArchitectureFromVMTemplate(tpl *v1beta1.VirtualMachineTemplate) string {
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
