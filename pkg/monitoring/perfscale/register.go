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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package perfscale

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"
)

func RegisterPerfScaleMetrics(vmInformer cache.SharedIndexInformer, vmiInformer cache.SharedIndexInformer) {
	log.Log.Infof("Starting performance and scale metrics")
	// VM metrics
	prometheus.MustRegister(newVMStatusTransitionTimeFromCreationHistogramVec(vmInformer))
	// VMI metrics
	prometheus.MustRegister(newVMIPhaseTransitionTimeHistogramVec(vmiInformer))
	prometheus.MustRegister(newVMIPhaseTransitionTimeFromCreationHistogramVec(vmiInformer))
	prometheus.MustRegister(newVMIPhaseTransitionTimeFromDeletionHistogramVec(vmiInformer))
}
