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

package api

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// PendingPlatformTerminationIntent records an in-flight platform-initiated
// termination request. The zero value means the intent has been cleared.
type PendingPlatformTerminationIntent struct {
	Timestamp metav1.Time
}

type TerminationReason string

const (
	TerminationReasonGuestShutdown             TerminationReason = "GuestShutdown"
	TerminationReasonPlatformRequestedShutdown TerminationReason = "PlatformRequestedShutdown"
	TerminationReasonHostShutdown              TerminationReason = "HostShutdown"
	TerminationReasonHostStoppedFailed         TerminationReason = "HostStoppedFailed"
	TerminationReasonGuestCrashed              TerminationReason = "GuestCrashed"
)

func (reason TerminationReason) Message() string {
	switch reason {
	case TerminationReasonGuestShutdown:
		return "Guest requested shutdown of the virtual machine"
	case TerminationReasonPlatformRequestedShutdown:
		return "Platform requested shutdown of the virtual machine"
	case TerminationReasonHostShutdown:
		return "Host requested shutdown of the virtual machine"
	case TerminationReasonHostStoppedFailed:
		return "Host observed the virtual machine stop unexpectedly"
	case TerminationReasonGuestCrashed:
		return "Guest crashed, but the virtual machine might not be terminated yet"
	default:
		return "Guest terminated"
	}
}

func SupportedTerminationReasons() []TerminationReason {
	return []TerminationReason{
		TerminationReasonGuestShutdown,
		TerminationReasonPlatformRequestedShutdown,
		TerminationReasonHostShutdown,
		TerminationReasonHostStoppedFailed,
		TerminationReasonGuestCrashed,
	}
}

func IsSupportedTerminationReason(reason TerminationReason) bool {
	for _, supportedReason := range SupportedTerminationReasons() {
		if reason == supportedReason {
			return true
		}
	}
	return false
}

type TerminationEvent struct {
	Reason    TerminationReason
	Timestamp metav1.Time
}
