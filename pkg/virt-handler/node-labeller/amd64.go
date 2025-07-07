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
 * Copyright the KubeVirt Authors.
 *
 */

package nodelabeller

// Ensure that there is a compile error should the struct not implement the archLabeller interface anymore.
var _ = archLabeller(&archLabellerAMD64{})

type archLabellerAMD64 struct {
	defaultArchLabeller
}

func (archLabellerAMD64) shouldLabelNodes() bool {
	return true
}

func (archLabellerAMD64) hasHostSupportedFeatures() bool {
	return true
}

func (archLabellerAMD64) supportsHostModel() bool {
	return true
}

func (archLabellerAMD64) arch() string {
	return amd64
}
