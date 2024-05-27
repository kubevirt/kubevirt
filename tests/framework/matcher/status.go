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

package matcher

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
)

func BeCreated() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Created": BeTrue(),
		}),
	}))
}

func BeReady() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Ready": BeTrue(),
		}),
	}))
}

func BeRestarted(oldUID types.UID) gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"UID": Not(Equal(oldUID)),
		}),
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Phase": Equal(v1.Running),
		}),
	}))
}

func BeInCrashLoop() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"PrintableStatus": Equal(v1.VirtualMachineStatusCrashLoopBackOff),
			"StartFailure": gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"ConsecutiveFailCount": BeNumerically(">", 0),
			})),
		}),
	}))
}

func NotBeInCrashLoop() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"StartFailure": BeNil(),
		}),
	}))
}

func HaveStateChangeRequests() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"StateChangeRequests": Not(BeEmpty()),
		}),
	}))
}

func InterfaceIPs(networkInterface *v1.VirtualMachineInstanceNetworkInterface) []string {
	if networkInterface == nil {
		return nil
	}
	return networkInterface.IPs
}

func MatchIPsAtInterfaceByName(interfaceName string, ipsMatcher gomegatypes.GomegaMatcher) gomegatypes.GomegaMatcher {
	return WithTransform(
		func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstanceNetworkInterface {
			return vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, interfaceName)
		},
		SatisfyAll(
			Not(BeNil()),
			WithTransform(InterfaceIPs, ipsMatcher)))
}
