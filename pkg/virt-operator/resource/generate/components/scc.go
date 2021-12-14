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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package components

import (
	"fmt"

	secv1 "github.com/openshift/api/security/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetAllSCC(namespace string, managed bool) []*secv1.SecurityContextConstraints {
	sccs := []*secv1.SecurityContextConstraints{
		NewKubeVirtHandlerSCC(namespace),
		newBaseVirtSCC(),
	}

	base := newBaseVirtSCC
	sccs = append(sccs, name(cpuPinningSCC(base()), "kubevirt-cpu-pinning"))
	sccs = append(sccs, name(hostDiskSCC(base()), "kubevirt-host-disk"))

	sccs = append(sccs, name(hostDiskSCC(cpuPinningSCC(base())), "kubevirt-cpu-pinning-and-host-disk"))

	if managed {
		sccs = append(sccs, NewKubeVirtControllerSCC(namespace))
	}
	return sccs

}

func newBlankSCC() *secv1.SecurityContextConstraints {
	return &secv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.openshift.io/v1",
			Kind:       "SecurityContextConstraints",
		},
	}
}

func NewKubeVirtHandlerSCC(namespace string) *secv1.SecurityContextConstraints {
	scc := newBlankSCC()

	scc.Name = "kubevirt-handler"
	scc.AllowPrivilegedContainer = true
	scc.AllowHostPID = true
	scc.AllowHostPorts = true
	scc.AllowHostIPC = true
	scc.RunAsUser = secv1.RunAsUserStrategyOptions{
		Type: secv1.RunAsUserStrategyRunAsAny,
	}
	scc.SELinuxContext = secv1.SELinuxContextStrategyOptions{
		Type: secv1.SELinuxStrategyRunAsAny,
	}
	scc.Volumes = []secv1.FSType{secv1.FSTypeAll}
	scc.AllowHostDirVolumePlugin = true
	scc.Users = []string{fmt.Sprintf("system:serviceaccount:%s:kubevirt-handler", namespace)}

	return scc
}

func NewKubeVirtControllerSCC(namespace string) *secv1.SecurityContextConstraints {
	scc := newBlankSCC()

	scc.Name = "kubevirt-controller"
	scc.AllowPrivilegedContainer = false
	scc.RunAsUser = secv1.RunAsUserStrategyOptions{
		Type: secv1.RunAsUserStrategyRunAsAny,
	}
	scc.SELinuxContext = secv1.SELinuxContextStrategyOptions{
		Type: secv1.SELinuxStrategyRunAsAny,
	}
	scc.AllowedCapabilities = []corev1.Capability{
		// add a CAP_SYS_NICE capability to allow setting cpu affinity
		"SYS_NICE",
		// add CAP_NET_BIND_SERVICE capability to allow dhcp and slirp operations
		"NET_BIND_SERVICE",
	}
	scc.AllowHostDirVolumePlugin = true
	scc.Users = []string{fmt.Sprintf("system:serviceaccount:%s:kubevirt-controller", namespace)}

	return scc
}

func newBaseVirtSCC() *secv1.SecurityContextConstraints {
	scc := newBlankSCC()

	scc.Name = "kubevirt-base"
	scc.AllowPrivilegedContainer = false
	scc.RunAsUser = secv1.RunAsUserStrategyOptions{
		Type: secv1.RunAsUserStrategyRunAsAny,
	}
	scc.SELinuxContext = secv1.SELinuxContextStrategyOptions{
		Type: secv1.SELinuxStrategyRunAsAny,
	}

	scc.AllowedCapabilities = []corev1.Capability{
		// add CAP_NET_BIND_SERVICE capability to allow dhcp and slirp operations
		"NET_BIND_SERVICE",
	}
	scc.AllowHostDirVolumePlugin = false
	scc.AllowHostNetwork = false
	scc.FSGroup = secv1.FSGroupStrategyOptions{
		// This is default - avaible for all policies except restricted
		Type: secv1.FSGroupStrategyRunAsAny,
	}

	return scc
}

func cpuPinningSCC(scc *secv1.SecurityContextConstraints) *secv1.SecurityContextConstraints {
	scc.AllowedCapabilities = append(scc.AllowedCapabilities, "SYS_NICE")
	return scc
}

func hostDiskSCC(scc *secv1.SecurityContextConstraints) *secv1.SecurityContextConstraints {
	scc.AllowHostDirVolumePlugin = true
	return scc
}

func name(scc *secv1.SecurityContextConstraints, name string) *secv1.SecurityContextConstraints {
	scc.Name = name
	return scc
}
