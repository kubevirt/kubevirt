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

package libvmops

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
)

const efiTestVariablePath = "/sys/firmware/efi/efivars/kvtest-12345678-1234-1234-1234-123456789abc"

// WriteEFITestVariable requires a guest console logged in as root.
func WriteEFITestVariable(vmi *v1.VirtualMachineInstance) {
	GinkgoHelper()
	By("Creating a nonvolatile EFI test variable")
	// Write four attribute bytes followed by the marker value 0x42.
	Expect(console.RunCommand(vmi, `printf '\007\000\000\000\102' > `+efiTestVariablePath, 10*time.Second)).To(Succeed())
}

// CheckEFITestVariable requires a guest console logged in as root.
func CheckEFITestVariable(vmi *v1.VirtualMachineInstance) {
	GinkgoHelper()
	By("Checking the nonvolatile EFI test variable")
	cmd := `test "$(od -An -tu1 -j4 -N1 ` + efiTestVariablePath + ` | tr -d '[:space:]')" = 66`
	Expect(console.RunCommand(vmi, cmd, 10*time.Second)).To(Succeed(), "expected EFI test variable is missing or has an unexpected value")
}
