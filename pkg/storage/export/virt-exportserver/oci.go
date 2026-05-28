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

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

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
	vm := getExpandedVM()
	if vm == nil {
		return nil, nil
	}

	configJSON, err := prepareVMConfig(vm)
	if err != nil {
		return nil, fmt.Errorf("error preparing VM config: %w", err)
	}

	architecture := ""
	if vm.Spec.Template != nil {
		architecture = vm.Spec.Template.Spec.Architecture
	}

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
