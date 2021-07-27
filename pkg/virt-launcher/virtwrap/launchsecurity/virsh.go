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
 * Copyright 2021
 *
 */

package launchsecurity

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"os/exec"
)

type Virsh interface {
	Domcapabilities() ([]byte, error)
}

type virsh struct {
}

func NewVirsh() Virsh {
	return &virsh{}
}

func (v *virsh) Domcapabilities() ([]byte, error) {
	return exec.Command("virsh",
		"domcapabilities",
		"--machine", "q35",
		"--arch", "x86_64",
		"--virttype", "kvm",
	).Output()
}
