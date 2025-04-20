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
 */

package domainstats

func nanosecondsToSeconds(ns uint64) float64 {
	return float64(ns) / 1000000000
}

func microsecondsToSeconds(us uint64) float64 {
	return float64(us) / 1000000
}

func kibibytesToBytes(kibibytes uint64) float64 {
	return float64(kibibytes) * 1024
}
