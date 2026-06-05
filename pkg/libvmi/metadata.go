/*
Copyright The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package libvmi

import (
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

// WithLabel sets a label with specified value
func WithLabel(key, value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Labels == nil {
			vmi.Labels = map[string]string{}
		}
		vmi.Labels[key] = value
	}
}

// WithAnnotation adds an annotation with specified value
func WithAnnotation(key, value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Annotations == nil {
			vmi.Annotations = map[string]string{}
		}
		vmi.Annotations[key] = value
	}
}

func WithName(name string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Name = name
	}
}

func WithNamespace(namespace string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Namespace = namespace
	}
}

func WithUID(uid types.UID) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.ObjectMeta.UID = uid
	}
}
