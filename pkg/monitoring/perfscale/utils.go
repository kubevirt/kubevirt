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

package perfscale

import "time"

const (
	transTimeErrFmt = "Error encountered during vmi transition time histogram calculation: %v"
	transTimeFail   = "Failed to get a histogram for a vmi lifecycle transition times"
)

func phaseTransitionTimeBuckets() []float64 {
	return []float64{
		(0.5 * time.Second.Seconds()),
		(1 * time.Second.Seconds()),
		(2 * time.Second.Seconds()),
		(5 * time.Second.Seconds()),
		(10 * time.Second.Seconds()),
		(20 * time.Second.Seconds()),
		(30 * time.Second.Seconds()),
		(40 * time.Second.Seconds()),
		(50 * time.Second.Seconds()),
		(60 * time.Second).Seconds(),
		(90 * time.Second).Seconds(),
		(2 * time.Minute).Seconds(),
		(3 * time.Minute).Seconds(),
		(5 * time.Minute).Seconds(),
		(10 * time.Minute).Seconds(),
		(20 * time.Minute).Seconds(),
		(30 * time.Minute).Seconds(),
		(1 * time.Hour).Seconds(),
	}
}
