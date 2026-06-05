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
var _ = ArchDefaulter(&defaulterS390X{})

type defaulterS390X struct{}

func (defaulterS390X) OSTypeArch() string {
	return "s390x"
}

func (defaulterS390X) OSTypeMachine() string {
	return "s390-ccw-virtio"
}

func (defaulterS390X) DeepCopy() ArchDefaulter {
	return defaulterS390X{}
}
