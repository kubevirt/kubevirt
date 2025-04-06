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

package featuregate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
)

func ValidateFeatureGates(featureGates []string, vmiSpec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	featureGates, _ = ParseEnableFeatureGates(featureGates)

	var causes []metav1.StatusCause
	for _, fgName := range featureGates {
		fg := FeatureGateInfo(fgName)
		if fg != nil && fg.State == Discontinued && fg.VmiSpecUsed != nil {
			if used := fg.VmiSpecUsed(vmiSpec); used {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: fg.Message,
				})
			}
		}
	}
	return causes
}

// ParseEnableFeatureGates parses the enabled feature gates from a slice of strings,
// as described by FeatureGates's field documentation in kubevirt CR.
// While the function will report errors, it will return a valid list of enabled feature gates.
func ParseEnableFeatureGates(featureGates []string) (enabledFeatureGates []string, err error) {
	if featureGates == nil {
		return
	}

	var errs []error

	type fgState struct {
		name                   string
		isEnabled              bool
		isConfiguredExplicitly bool
	}
	fgStates := map[string]fgState{}

	for _, fgStrConfig := range featureGates {
		var isEnabled, isConfiguredExplicitly bool
		var featureGateName string

		if splitFg := strings.SplitN(fgStrConfig, "=", 2); len(splitFg) == 2 {
			// This means the feature gate was explicitly configured.
			isConfiguredExplicitly = true

			featureGateName = splitFg[0]
			isEnabled, err = strconv.ParseBool(splitFg[1])
			if err != nil {
				errs = append(errs, errors.New(fmt.Sprintf(`invalid feature gate value: "%s". must be "true" or "false". error: %v`, splitFg[1], err)))
				continue
			}

			if fg, isFgConfigured := fgStates[featureGateName]; isFgConfigured && fg.isConfiguredExplicitly && fg.isEnabled != isEnabled {
				errs = append(errs, errors.New("feature gate "+featureGateName+" is configured multiple with contradicting values"))
				continue
			}
		} else {
			featureGateName = fgStrConfig
			isConfiguredExplicitly = false

			if _, isFgConfigured := fgStates[featureGateName]; isFgConfigured {
				// FGs that are configured explicitly take precedence over the default value.
				continue
			}

			isEnabled = true
		}

		fgStates[featureGateName] = fgState{
			name:                   featureGateName,
			isEnabled:              isEnabled,
			isConfiguredExplicitly: isConfiguredExplicitly,
		}
	}

	enabledFeatureGates = make([]string, 0, len(fgStates))
	for _, curFgState := range fgStates {
		if curFgState.isEnabled {
			enabledFeatureGates = append(enabledFeatureGates, curFgState.name)
		}
	}

	return enabledFeatureGates, errors.Join(errs...)
}
