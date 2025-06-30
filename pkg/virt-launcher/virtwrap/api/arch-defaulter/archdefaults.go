/* Licensed under the Apache License, Version 2.0 (the "License");
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
 * Copyright the KubeVirt Authors.
 *
 */

package archdefaulter

import "kubevirt.io/client-go/log"

type ArchDefaulter interface {
	OSTypeArch() string
	OSTypeMachine() string
	DeepCopy() ArchDefaulter
}

func NewArchDefaulter(arch string) ArchDefaulter {
	switch arch {
	case "arm64":
		return defaulterARM64{}
	case "s390x":
		return defaulterS390X{}
	case "amd64":
		return defaulterAMD64{}
	default:
		log.Log.Warning("Trying to create an arch defaulter from an unknown arch: " + arch + ". Falling back to AMD64")
		return defaulterAMD64{}
	}
}
