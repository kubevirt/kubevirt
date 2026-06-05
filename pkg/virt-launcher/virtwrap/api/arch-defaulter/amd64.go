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

// This file is build on all arches. Golang only filters files ending with _<arch>.go
package archdefaulter

// Ensure that there is a compile error should the struct not implement the ArchDefaulter interface anymore.
var _ = ArchDefaulter(&defaulterAMD64{})

type defaulterAMD64 struct{}

func (defaulterAMD64) OSTypeArch() string {
	return "x86_64"
}

func (defaulterAMD64) OSTypeMachine() string {
	// q35 is an alias of the newest q35 machine type.
	return "q35"
}

func (defaulterAMD64) DeepCopy() ArchDefaulter {
	return defaulterAMD64{}
}
