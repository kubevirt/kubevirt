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

package validating_webhook

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	v1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"k8s.io/apimachinery/pkg/labels"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	cloudInitMaxLen = 2048
	arrayLenMax     = 256
	maxStrLen       = 256
)

var validInterfaceModels = []string{"e1000", "e1000e", "ne2k_pci", "pcnet", "rtl8139", "virtio"}

func getAdmissionReview(r *http.Request) (*v1beta1.AdmissionReview, error) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return nil, fmt.Errorf("contentType=%s, expect application/json", contentType)
	}

	ar := &v1beta1.AdmissionReview{}
	err := json.Unmarshal(body, ar)
	return ar, err
}

func toAdmissionResponseError(err error) *v1beta1.AdmissionResponse {
	log.Log.Reason(err).Error("admitting vmis with generic error")

	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		},
	}
}
func toAdmissionResponse(causes []metav1.StatusCause) *v1beta1.AdmissionResponse {
	log.Log.Infof("rejected vmi admission")

	globalMessage := ""
	for _, cause := range causes {
		globalMessage = fmt.Sprintf("%s %s", globalMessage, cause.Message)
	}

	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: globalMessage,
			Code:    http.StatusUnprocessableEntity,
			Details: &metav1.StatusDetails{
				Causes: causes,
			},
		},
	}
}

type admitFunc func(*v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

func serve(resp http.ResponseWriter, req *http.Request, admit admitFunc) {
	response := v1beta1.AdmissionReview{}
	review, err := getAdmissionReview(req)

	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	reviewResponse := admit(review)
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = review.Request.UID
	}
	// reset the Object and OldObject, they are not needed in a response.
	review.Request.Object = runtime.RawExtension{}
	review.Request.OldObject = runtime.RawExtension{}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Log.Reason(err).Errorf("failed json encode webhook response")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := resp.Write(responseBytes); err != nil {
		log.Log.Reason(err).Errorf("failed to write webhook response")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	resp.WriteHeader(http.StatusOK)
}

func validateDisks(field *k8sfield.Path, disks []v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	nameMap := make(map[string]int)

	if len(disks) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s list exceeds the %d element limit in length", field.String(), arrayLenMax),
			Field:   field.String(),
		})
		// We won't process anything over the limit
		return causes
	}

	for idx, disk := range disks {
		// verify name is unique
		otherIdx, ok := nameMap[disk.Name]
		if !ok {
			nameMap[disk.Name] = idx
		} else {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s and %s must not have the same Name.", field.Index(idx).String(), field.Index(otherIdx).String()),
				Field:   field.Index(idx).Child("name").String(),
			})
		}
		// Verify only a single device type is set.
		deviceTargetSetCount := 0
		if disk.Disk != nil {
			deviceTargetSetCount++
		}
		if disk.LUN != nil {
			deviceTargetSetCount++
		}
		if disk.Floppy != nil {
			deviceTargetSetCount++
		}
		if disk.CDRom != nil {
			deviceTargetSetCount++
		}

		// NOTE: not setting a device target is okay. We default to Disk.
		// However, only a single device target is allowed to be set at a time.
		if deviceTargetSetCount > 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s can only have a single target type defined", field.Index(idx).String()),
				Field:   field.Index(idx).String(),
			})
		}

		// Verify boot order is greater than 0, if provided
		if disk.BootOrder != nil && *disk.BootOrder < 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must have a boot order > 0, if supplied", field.Index(idx).String()),
				Field:   field.Index(idx).Child("bootOrder").String(),
			})
		}

		// Verify serial number is made up of valid characters for libvirt, if provided
		isValid := regexp.MustCompile(`^[A-Za-z0-9_.+-]+$`).MatchString
		if disk.Serial != "" && !isValid(disk.Serial) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must be made up of the following characters [A-Za-z0-9_.+-], if specified", field.Index(idx).String()),
				Field:   field.Index(idx).Child("serial").String(),
			})
		}

		// Verify serial number is within valid length, if provided
		if disk.Serial != "" && len([]rune(disk.Serial)) > maxStrLen {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must be less than or equal to %d in length, if specified", field.Index(idx).String(), maxStrLen),
				Field:   field.Index(idx).Child("serial").String(),
			})
		}
	}

	return causes
}

func validateVolumes(field *k8sfield.Path, volumes []v1.Volume) []metav1.StatusCause {
	var causes []metav1.StatusCause
	nameMap := make(map[string]int)

	if len(volumes) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s list exceeds the %d element limit in length", field.String(), arrayLenMax),
			Field:   field.String(),
		})
		// We won't process anything over the limit
		return causes
	}
	for idx, volume := range volumes {
		// verify name is unique
		otherIdx, ok := nameMap[volume.Name]
		if !ok {
			nameMap[volume.Name] = idx
		} else {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s and %s must not have the same Name.", field.Index(idx).String(), field.Index(otherIdx).String()),
				Field:   field.Index(idx).Child("name").String(),
			})
		}

		// Verify exactly one source is set
		volumeSourceSetCount := 0
		if volume.PersistentVolumeClaim != nil {
			volumeSourceSetCount++
		}
		if volume.CloudInitNoCloud != nil {
			volumeSourceSetCount++
		}
		if volume.RegistryDisk != nil {
			volumeSourceSetCount++
		}
		if volume.Ephemeral != nil {
			volumeSourceSetCount++
		}
		if volume.EmptyDisk != nil {
			volumeSourceSetCount++
		}

		if volumeSourceSetCount != 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must have exactly one source type set", field.Index(idx).String()),
				Field:   field.Index(idx).String(),
			})
		}

		// Verify cloud init data is within size limits
		if volume.CloudInitNoCloud != nil {
			noCloud := volume.CloudInitNoCloud
			userDataLen := 0

			userDataSourceCount := 0
			if noCloud.UserDataSecretRef != nil && noCloud.UserDataSecretRef.Name != "" {
				userDataSourceCount++
			}
			if noCloud.UserDataBase64 != "" {
				userDataSourceCount++
				userData, err := base64.StdEncoding.DecodeString(noCloud.UserDataBase64)
				if err != nil {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s.cloudInitNoCloud.userDataBase64 is not a valid base64 value.", field.Index(idx).Child("cloudInitNoCloud", "userDataBase64").String()),
						Field:   field.Index(idx).Child("cloudInitNoCloud", "userDataBase64").String(),
					})
				}
				userDataLen = len(userData)
			}
			if noCloud.UserData != "" {
				userDataSourceCount++
				userDataLen = len(noCloud.UserData)
			}

			if userDataSourceCount != 1 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s must have one exactly one userdata source set.", field.Index(idx).Child("cloudInitNoCloud").String()),
					Field:   field.Index(idx).Child("cloudInitNoCloud").String(),
				})
			}

			if userDataLen > cloudInitMaxLen {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s userdata exceeds %d byte limit", field.Index(idx).Child("cloudInitNoCloud").String(), cloudInitMaxLen),
					Field:   field.Index(idx).Child("cloudInitNoCloud").String(),
				})
			}
		}
	}
	return causes
}

func validateDevices(field *k8sfield.Path, devices *v1.Devices) []metav1.StatusCause {
	var causes []metav1.StatusCause
	causes = append(causes, validateDisks(field.Child("disks"), devices.Disks)...)
	return causes
}

func validateDomainPresetSpec(field *k8sfield.Path, spec *v1.DomainPresetSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	causes = append(causes, validateDevices(field.Child("devices"), &spec.Devices)...)
	return causes
}

func validateDomainSpec(field *k8sfield.Path, spec *v1.DomainSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	causes = append(causes, validateDevices(field.Child("devices"), &spec.Devices)...)
	return causes
}

func getNumberOfPodInterfaces(spec *v1.VirtualMachineInstanceSpec) int {
	nPodInterfaces := 0
	for _, net := range spec.Networks {
		if net.Pod != nil {
			for _, iface := range spec.Domain.Devices.Interfaces {
				if iface.Name == net.Name {
					nPodInterfaces++
					break // we maintain 1-to-1 relationship between networks and interfaces
				}
			}
		}
	}
	return nPodInterfaces
}

func ValidateVirtualMachineInstanceSpec(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	volumeToDiskIndexMap := make(map[string]int)
	volumeNameMap := make(map[string]*v1.Volume)
	networkNameMap := make(map[string]*v1.Network)

	if len(spec.Domain.Devices.Disks) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s list exceeds the %d element limit in length", field.Child("domain", "devices", "disks").String(), arrayLenMax),
			Field:   field.Child("domain", "devices", "disks").String(),
		})
		// We won't process anything over the limit
		return causes
	} else if len(spec.Volumes) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s list exceeds the %d element limit in length", field.Child("volumes").String(), arrayLenMax),
			Field:   field.Child("volumes").String(),
		})
		// We won't process anything over the limit
		return causes
	}
	// Validate hostname according to DNS label rules
	if spec.Hostname != "" {
		errors := validation.IsDNS1123Label(spec.Hostname)
		if len(errors) != 0 {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s does not conform to the kubernetes DNS_LABEL rules : %s",
					field.Child("hostname").String(), strings.Join(errors, ", ")),
				Field: field.Child("hostname").String(),
			})
		}
	}

	// Validate subdomain according to DNS subdomain rules
	if spec.Subdomain != "" {
		errors := validation.IsDNS1123Subdomain(spec.Subdomain)
		if len(errors) != 0 {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s does not conform to the kubernetes DNS_SUBDOMAIN rules : %s",
					field.Child("subdomain").String(), strings.Join(errors, ", ")),
				Field: field.Child("subdomain").String(),
			})
		}
	}

	// Validate memory size if values are not negative
	if spec.Domain.Resources.Requests.Memory().Value() < 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s '%s': must be greater than or equal to 0.", field.Child("domain", "resources", "requests", "memory").String(),
				spec.Domain.Resources.Requests.Memory()),
			Field: field.Child("domain", "resources", "requests", "memory").String(),
		})
	}

	if spec.Domain.Resources.Limits.Memory().Value() < 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s '%s': must be greater than or equal to 0.", field.Child("domain", "resources", "limits", "memory").String(),
				spec.Domain.Resources.Limits.Memory()),
			Field: field.Child("domain", "resources", "limits", "memory").String(),
		})
	}

	if spec.Domain.Resources.Limits.Memory().Value() > 0 &&
		spec.Domain.Resources.Requests.Memory().Value() > spec.Domain.Resources.Limits.Memory().Value() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s '%s' is greater than %s '%s'", field.Child("domain", "resources", "requests", "memory").String(),
				spec.Domain.Resources.Requests.Memory(),
				field.Child("domain", "resources", "limits", "memory").String(),
				spec.Domain.Resources.Limits.Memory()),
			Field: field.Child("domain", "resources", "requests", "memory").String(),
		})
	}

	if spec.Domain.Memory != nil && spec.Domain.Memory.Hugepages != nil && spec.Domain.Memory.Guest != nil {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("'%s' and '%s' must not be set at the same time",
				field.Child("domain", "memory", "guest").String(),
				field.Child("domain", "memory", "hugepages", "size").String()),
			Field: field.Child("domain", "resources", "requests", "memory").String(),
		})
	}

	// Validate hugepages
	if spec.Domain.Memory != nil && spec.Domain.Memory.Hugepages != nil {
		hugepagesSize, err := resource.ParseQuantity(spec.Domain.Memory.Hugepages.PageSize)
		if err != nil {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s '%s': %s",
					field.Child("domain", "hugepages", "size").String(),
					spec.Domain.Memory.Hugepages.PageSize,
					resource.ErrFormatWrong,
				),
				Field: field.Child("domain", "hugepages", "size").String(),
			})
		} else {
			vmMemory := spec.Domain.Resources.Requests.Memory().Value()
			if vmMemory < hugepagesSize.Value() {
				causes = append(causes, metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s '%s' must be equal to or larger than page size %s '%s'",
						field.Child("domain", "resources", "requests", "memory").String(),
						spec.Domain.Resources.Requests.Memory(),
						field.Child("domain", "hugepages", "size").String(),
						spec.Domain.Memory.Hugepages.PageSize,
					),
					Field: field.Child("domain", "resources", "requests", "memory").String(),
				})
			} else if vmMemory%hugepagesSize.Value() != 0 {
				causes = append(causes, metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s '%s' is not a multiple of the page size %s '%s'",
						field.Child("domain", "resources", "requests", "memory").String(),
						spec.Domain.Resources.Requests.Memory(),
						field.Child("domain", "hugepages", "size").String(),
						spec.Domain.Memory.Hugepages.PageSize,
					),
					Field: field.Child("domain", "resources", "requests", "memory").String(),
				})
			}
		}
	}
	// Validate hugepages
	if spec.Domain.Memory != nil && spec.Domain.Memory.Guest != nil {
		requests := spec.Domain.Resources.Requests.Memory().Value()
		limits := spec.Domain.Resources.Limits.Memory().Value()
		guest := spec.Domain.Memory.Guest.Value()
		if requests > guest {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s '%s' must be equal to or larger than the requested memory %s '%s'",
					field.Child("domain", "memory", "guest").String(),
					spec.Domain.Memory.Guest,
					field.Child("domain", "resources", "requests", "memory").String(),
					spec.Domain.Resources.Requests.Memory(),
				),
				Field: field.Child("domain", "memory", "guest").String(),
			})
		}
		if limits < guest && limits != 0 {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s '%s' must be equal to or less than the memory limit %s '%s'",
					field.Child("domain", "memory", "guest").String(),
					spec.Domain.Memory.Guest,
					field.Child("domain", "resources", "limits", "memory").String(),
					spec.Domain.Resources.Limits.Memory(),
				),
				Field: field.Child("domain", "memory", "guest").String(),
			})
		}
	}

	if len(spec.Domain.Devices.Interfaces) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s list exceeds the %d element limit in length", field.Child("domain", "devices", "interfaces").String(), arrayLenMax),
			Field:   field.Child("domain", "devices", "interfaces").String(),
		})
		return causes
	} else if len(spec.Networks) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s list exceeds the %d element limit in length", field.Child("networks").String(), arrayLenMax),
			Field:   field.Child("networks").String(),
		})
		return causes
	} else if getNumberOfPodInterfaces(spec) > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueDuplicate,
			Message: fmt.Sprintf("more than one interface is connected to a pod network in %s", field.Child("interfaces").String()),
			Field:   field.Child("interfaces").String(),
		})
		return causes
	}

	for _, volume := range spec.Volumes {
		volumeNameMap[volume.Name] = &volume
	}

	// used to validate uniqueness of boot orders among disks and interfaces
	bootOrderMap := make(map[uint]bool)

	// Validate disks and VolumeNames match up correctly
	for idx, disk := range spec.Domain.Devices.Disks {
		var matchingVolume *v1.Volume

		matchingVolume, volumeExists := volumeNameMap[disk.VolumeName]

		if !volumeExists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s '%s' not found.", field.Child("domain", "devices", "disks").Index(idx).Child("volumeName").String(), disk.VolumeName),
				Field:   field.Child("domain", "devices", "disks").Index(idx).Child("volumeName").String(),
			})
		}

		// verify no other disk maps to this volume
		otherIdx, ok := volumeToDiskIndexMap[disk.VolumeName]
		if !ok {
			volumeToDiskIndexMap[disk.VolumeName] = idx
		} else {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s and %s reference the same volumeName.", field.Child("domain", "devices", "disks").Index(idx).String(), field.Child("domain", "devices", "disks").Index(otherIdx).String()),
				Field:   field.Child("domain", "devices", "disks").Index(idx).Child("volumeName").String(),
			})
		}

		// Verify Lun disks are only mapped to network/block devices.
		if disk.LUN != nil && volumeExists && matchingVolume.PersistentVolumeClaim == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s can only be mapped to a PersistentVolumeClaim volume.", field.Child("domain", "devices", "disks").Index(idx).Child("lun").String()),
				Field:   field.Child("domain", "devices", "disks").Index(idx).Child("lun").String(),
			})
		}

		// verify that there are no duplicate boot orders
		if disk.BootOrder != nil {
			order := *disk.BootOrder
			if bootOrderMap[order] {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("Boot order for %s already set for a different device.", field.Child("domain", "devices", "disks").Index(idx).Child("bootOrder").String()),
					Field:   field.Child("domain", "devices", "disks").Index(idx).Child("bootOrder").String(),
				})
			}
			bootOrderMap[order] = true
		}
	}

	if len(spec.Networks) > 0 && len(spec.Domain.Devices.Interfaces) > 0 {
		for idx, network := range spec.Networks {
			if network.Pod == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueRequired,
					Message: fmt.Sprintf("should only accept networks with a pod network source"),
					Field:   field.Child("networks").Index(idx).Child("pod").String(),
				})
			}
			networkNameMap[network.Name] = &network
		}

		// Make sure interfaces and networks are 1to1 related
		networkInterfaceMap := make(map[string]struct{})

		// Make sure the port name is unique across all the interfaces
		portForwardMap := make(map[string]struct{})

		// Validate that each interface has a matching network
		for idx, iface := range spec.Domain.Devices.Interfaces {

			networkData, networkExists := networkNameMap[iface.Name]

			if !networkExists {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s '%s' not found.", field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.Name),
					Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
				})
			} else if iface.Bridge != nil && networkData.Pod == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("Bridge interface only implemented with pod network"),
					Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
				})
			} else if iface.Slirp != nil && networkData.Pod == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("Slirp interface only implemented with pod network"),
					Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
				})
			}

			// Check if the interface name is unique
			if _, networkAlreadyUsed := networkInterfaceMap[iface.Name]; networkAlreadyUsed {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueDuplicate,
					Message: fmt.Sprintf("Only one interface can be connected to one specific network"),
					Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
				})
			}

			networkInterfaceMap[iface.Name] = struct{}{}

			// Check only ports configured on interfaces connected to a pod network
			if networkExists && networkData.Pod != nil && iface.Ports != nil {
				for portIdx, forwardPort := range iface.Ports {

					if forwardPort.Port == 0 {
						causes = append(causes, metav1.StatusCause{
							Type:    metav1.CauseTypeFieldValueRequired,
							Message: fmt.Sprintf("Port field is mandatory in every Port"),
							Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("ports").Index(portIdx).String(),
						})
					}

					if forwardPort.Port < 0 || forwardPort.Port > 65536 {
						causes = append(causes, metav1.StatusCause{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("Port field must be in range 0 < x < 65536."),
							Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("ports").Index(portIdx).String(),
						})
					}

					if forwardPort.Protocol != "" {
						if forwardPort.Protocol != "TCP" && forwardPort.Protocol != "UDP" {
							causes = append(causes, metav1.StatusCause{
								Type:    metav1.CauseTypeFieldValueInvalid,
								Message: fmt.Sprintf("Unknown protocol, only TCP or UDP allowed"),
								Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("ports").Index(portIdx).Child("protocol").String(),
							})
						}
					} else {
						forwardPort.Protocol = "TCP"
					}

					if forwardPort.Name != "" {
						if _, ok := portForwardMap[forwardPort.Name]; ok {
							causes = append(causes, metav1.StatusCause{
								Type:    metav1.CauseTypeFieldValueDuplicate,
								Message: fmt.Sprintf("Duplicate name of the port: %s", forwardPort.Name),
								Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("ports").Index(portIdx).Child("name").String(),
							})
						}

						portForwardMap[forwardPort.Name] = struct{}{}
					}
				}
			}

			// verify that selected model is supported
			if iface.Model != "" {
				isModelSupported := func(model string) bool {
					for _, m := range validInterfaceModels {
						if m == model {
							return true
						}
					}
					return false
				}
				if !isModelSupported(iface.Model) {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueNotSupported,
						Message: fmt.Sprintf("interface %s uses model %s that is not supported.", field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.Model),
						Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("model").String(),
					})
				}
			}

			// verify that selected macAddress is valid
			if iface.MacAddress != "" {
				mac, err := net.ParseMAC(iface.MacAddress)
				if err != nil {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("interface %s has malformed MAC address (%s).", field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.MacAddress),
						Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("macAddress").String(),
					})
				}
				if len(mac) > 6 {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("interface %s has MAC address (%s) that is too long.", field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.MacAddress),
						Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("macAddress").String(),
					})
				}
			}

			if iface.BootOrder != nil {
				order := *iface.BootOrder
				// Verify boot order is greater than 0, if provided
				if order < 1 {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s must have a boot order > 0, if supplied", field.Index(idx).String()),
						Field:   field.Index(idx).Child("bootOrder").String(),
					})
				} else {
					// verify that there are no duplicate boot orders
					if bootOrderMap[order] {
						causes = append(causes, metav1.StatusCause{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("Boot order for %s already set for a different device.", field.Child("domain", "devices", "interfaces").Index(idx).Child("bootOrder").String()),
							Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("bootOrder").String(),
						})
					}
					bootOrderMap[order] = true
				}
			}
			// verify that the specified pci address is valid
			if iface.PciAddress != "" {
				_, err := util.ParsePciAddress(iface.PciAddress)
				if err != nil {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("interface %s has malformed PCI address (%s).", field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.PciAddress),
						Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("pciAddress").String(),
					})
				}
			}
		}
	}

	causes = append(causes, validateDomainSpec(field.Child("domain"), &spec.Domain)...)
	causes = append(causes, validateVolumes(field.Child("volumes"), spec.Volumes)...)
	return causes
}

func ValidateVirtualMachineSpec(field *k8sfield.Path, spec *v1.VirtualMachineSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Template == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing virtual machine template."),
			Field:   field.Child("template").String(),
		})
	}

	causes = append(causes, ValidateVirtualMachineInstanceSpec(field.Child("template", "spec"), &spec.Template.Spec)...)
	return causes
}

func ValidateVMIPresetSpec(field *k8sfield.Path, spec *v1.VirtualMachineInstancePresetSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Domain == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing domain."),
			Field:   field.Child("domain").String(),
		})
	}

	causes = append(causes, validateDomainPresetSpec(field.Child("domain"), spec.Domain)...)
	return causes
}

func ValidateVMIRSSpec(field *k8sfield.Path, spec *v1.VirtualMachineInstanceReplicaSetSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Template == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing virtual machine template."),
			Field:   field.Child("template").String(),
		})
	}
	causes = append(causes, ValidateVirtualMachineInstanceSpec(field.Child("template", "spec"), &spec.Template.Spec)...)

	selector, err := metav1.LabelSelectorAsSelector(spec.Selector)
	if err != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err.Error(),
			Field:   field.Child("selector").String(),
		})
	} else if !selector.Matches(labels.Set(spec.Template.ObjectMeta.Labels)) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("selector does not match labels."),
			Field:   field.Child("selector").String(),
		})
	}

	return causes
}

func admitVMIs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	vmiResource := metav1.GroupVersionResource{
		Group:    v1.VirtualMachineInstanceGroupVersionKind.Group,
		Version:  v1.VirtualMachineInstanceGroupVersionKind.Version,
		Resource: "virtualmachineinstances",
	}
	if ar.Request.Resource != vmiResource {
		err := fmt.Errorf("expect resource to be '%s'", vmiResource.Resource)
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	vmi := v1.VirtualMachineInstance{}

	err := json.Unmarshal(raw, &vmi)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &vmi.Spec)
	if len(causes) > 0 {
		return toAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMIs(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMIs)
}

func admitVMs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	resource := metav1.GroupVersionResource{
		Group:    v1.VirtualMachineGroupVersionKind.Group,
		Version:  v1.VirtualMachineGroupVersionKind.Version,
		Resource: "virtualmachines",
	}
	if ar.Request.Resource != resource {
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	vm := v1.VirtualMachine{}

	err := json.Unmarshal(raw, &vm)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	causes := ValidateVirtualMachineSpec(k8sfield.NewPath("spec"), &vm.Spec)
	if len(causes) > 0 {
		return toAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMs(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMs)
}

func admitVMIRS(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	resource := metav1.GroupVersionResource{
		Group:    v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group,
		Version:  v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Version,
		Resource: "virtualmachineinstancereplicasets",
	}
	if ar.Request.Resource != resource {
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	vmirs := v1.VirtualMachineInstanceReplicaSet{}

	err := json.Unmarshal(raw, &vmirs)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	causes := ValidateVMIRSSpec(k8sfield.NewPath("spec"), &vmirs.Spec)
	if len(causes) > 0 {
		return toAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMIRS(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMIRS)
}
func admitVMIPreset(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	resource := metav1.GroupVersionResource{
		Group:    v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group,
		Version:  v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Version,
		Resource: "virtualmachineinstancepresets",
	}
	if ar.Request.Resource != resource {
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	vmipreset := v1.VirtualMachineInstancePreset{}

	err := json.Unmarshal(raw, &vmipreset)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	causes := ValidateVMIPresetSpec(k8sfield.NewPath("spec"), &vmipreset.Spec)
	if len(causes) > 0 {
		return toAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMIPreset(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMIPreset)
}
