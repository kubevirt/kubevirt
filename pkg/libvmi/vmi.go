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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package libvmi

import (
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
)

// Option represents an action that enables an option.
type Option func(vmi *v1.VirtualMachineInstance)

// New instantiates a new VMI configuration,
// building its properties based on the specified With* options.
func New(opts ...Option) *v1.VirtualMachineInstance {
	vmi := baseVmi(randName())

	WithTerminationGracePeriod(0)(vmi)
	for _, f := range opts {
		f(vmi)
	}

	return vmi
}

var defaultOptions []Option

func RegisterDefaultOption(opt Option) {
	defaultOptions = append(defaultOptions, opt)
}

// randName returns a random name for a virtual machine
func randName() string {
	const randomPostfixLen = 5
	return "testvmi" + "-" + rand.String(randomPostfixLen)
}

func baseVmi(name string) *v1.VirtualMachineInstance {
	vmi := v1.NewVMIReferenceFromNameWithNS("", name)
	vmi.Spec = v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}}
	vmi.TypeMeta = k8smetav1.TypeMeta{
		APIVersion: v1.GroupVersion.String(),
		Kind:       "VirtualMachineInstance",
	}

	for _, opt := range defaultOptions {
		opt(vmi)
	}

	return vmi
}
