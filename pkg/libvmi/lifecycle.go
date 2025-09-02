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

import v1 "kubevirt.io/api/core/v1"

// WithTerminationGracePeriod specifies the termination grace period in seconds.
func WithTerminationGracePeriod(seconds int64) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.TerminationGracePeriodSeconds = &seconds
	}
}

func WithEvictionStrategy(evictionStrategy v1.EvictionStrategy) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.EvictionStrategy = &evictionStrategy
	}
}

func WithStartStrategy(startStrategy v1.StartStrategy) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.StartStrategy = &startStrategy
	}
}
