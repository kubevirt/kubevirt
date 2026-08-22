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
 * Copyright 2025 Red Hat, Inc.
 *
 */

package capability

import (
	"fmt"
	"unsafe"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"golang.org/x/sys/unix"
)

func mockCapUserData(data *unix.CapUserData) *[2]unix.CapUserData {
	return (*[2]unix.CapUserData)(unsafe.Pointer(data))
}

var _ = Describe("Capability propagation", func() {
	var (
		origCapget func(*unix.CapUserHeader, *unix.CapUserData) error
		origCapset func(*unix.CapUserHeader, *unix.CapUserData) error
	)

	BeforeEach(func() {
		origCapget = capget
		origCapset = capset
	})

	AfterEach(func() {
		capget = origCapget
		capset = origCapset
	})

	Describe("GetPermittedCaps", func() {
		It("should return capabilities from the low word", func() {
			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				d[0].Permitted = (1 << unix.CAP_NET_BIND_SERVICE) | (1 << unix.CAP_SYS_NICE)
				return nil
			}

			caps, err := GetPermittedCaps()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(ConsistOf(uintptr(unix.CAP_NET_BIND_SERVICE), uintptr(unix.CAP_SYS_NICE)))
		})

		It("should return capabilities from the high word", func() {
			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				d[0].Permitted = 1 << unix.CAP_NET_BIND_SERVICE
				d[1].Permitted = 1 << 3 // capability 35
				return nil
			}

			caps, err := GetPermittedCaps()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(ConsistOf(uintptr(unix.CAP_NET_BIND_SERVICE), uintptr(35)))
		})

		It("should return empty list when no capabilities are permitted", func() {
			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				d[0].Permitted = 0
				d[1].Permitted = 0
				return nil
			}

			caps, err := GetPermittedCaps()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(BeEmpty())
		})

		It("should return error when capget fails", func() {
			capget = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return fmt.Errorf("permission denied")
			}

			_, err := GetPermittedCaps()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("capget"))
		})
	})

	Describe("EnsureInheritable", func() {
		It("should leave inheritable bits unchanged when caps is empty", func() {
			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				d[0].Inheritable = 0x3
				d[1].Inheritable = 0x4
				return nil
			}
			capset = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				Expect(d[0].Inheritable).To(Equal(uint32(0x3)))
				Expect(d[1].Inheritable).To(Equal(uint32(0x4)))
				return nil
			}

			Expect(EnsureInheritable(nil)).To(Succeed())
			Expect(EnsureInheritable([]uintptr{})).To(Succeed())
		})

		It("should set inheritable bits for capabilities", func() {
			var savedData [2]unix.CapUserData

			capget = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return nil
			}
			capset = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				savedData = *d
				return nil
			}

			err := EnsureInheritable([]uintptr{unix.CAP_NET_BIND_SERVICE, unix.CAP_SYS_NICE})
			Expect(err).ToNot(HaveOccurred())
			Expect(savedData[0].Inheritable & (1 << unix.CAP_NET_BIND_SERVICE)).ToNot(BeZero())
			Expect(savedData[0].Inheritable & (1 << unix.CAP_SYS_NICE)).ToNot(BeZero())
		})

		It("should set inheritable bits in the high word for caps >= 32", func() {
			var savedData [2]unix.CapUserData

			capget = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return nil
			}
			capset = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				savedData = *d
				return nil
			}

			err := EnsureInheritable([]uintptr{35})
			Expect(err).ToNot(HaveOccurred())
			Expect(savedData[1].Inheritable & (1 << 3)).ToNot(BeZero()) // 35 - 32 = 3
		})

		It("should return error when capget fails", func() {
			capget = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return fmt.Errorf("permission denied")
			}

			err := EnsureInheritable([]uintptr{unix.CAP_NET_BIND_SERVICE})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("capget"))
		})

		It("should return error when capset fails", func() {
			capget = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return nil
			}
			capset = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return fmt.Errorf("operation not permitted")
			}

			err := EnsureInheritable([]uintptr{unix.CAP_NET_BIND_SERVICE})
			Expect(err).To(HaveOccurred())
		})

		It("should preserve existing inheritable bits", func() {
			var savedData [2]unix.CapUserData

			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				d[0].Inheritable = 1 << unix.CAP_NET_BIND_SERVICE
				return nil
			}
			capset = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				savedData = *d
				return nil
			}

			err := EnsureInheritable([]uintptr{unix.CAP_SYS_NICE})
			Expect(err).ToNot(HaveOccurred())
			Expect(savedData[0].Inheritable & (1 << unix.CAP_NET_BIND_SERVICE)).ToNot(BeZero())
			Expect(savedData[0].Inheritable & (1 << unix.CAP_SYS_NICE)).ToNot(BeZero())
		})
	})

	Describe("BuildAmbientCaps", func() {
		It("should return all permitted capabilities", func() {
			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				d[0].Permitted = (1 << unix.CAP_NET_BIND_SERVICE) | (1 << unix.CAP_SYS_NICE) | (1 << unix.CAP_SYS_RAWIO)
				return nil
			}
			capset = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return nil
			}

			caps := BuildAmbientCaps()
			Expect(caps).To(ContainElements(
				uintptr(unix.CAP_NET_BIND_SERVICE),
				uintptr(unix.CAP_SYS_NICE),
				uintptr(unix.CAP_SYS_RAWIO),
			))
		})

		It("should fall back to CAP_NET_BIND_SERVICE when capget fails", func() {
			capget = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return fmt.Errorf("permission denied")
			}

			caps := BuildAmbientCaps()
			Expect(caps).To(Equal([]uintptr{unix.CAP_NET_BIND_SERVICE}))
		})

		It("should fall back to CAP_NET_BIND_SERVICE when no caps are permitted", func() {
			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				d[0].Permitted = 0
				d[1].Permitted = 0
				return nil
			}

			caps := BuildAmbientCaps()
			Expect(caps).To(Equal([]uintptr{unix.CAP_NET_BIND_SERVICE}))
		})

		It("should still return caps even when EnsureInheritable fails", func() {
			capgetCount := 0
			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				capgetCount++
				if capgetCount == 1 {
					d := mockCapUserData(data)
					d[0].Permitted = (1 << unix.CAP_NET_BIND_SERVICE) | (1 << unix.CAP_SYS_NICE)
					return nil
				}
				return nil
			}
			capset = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return fmt.Errorf("operation not permitted")
			}

			caps := BuildAmbientCaps()
			Expect(caps).To(ContainElements(
				uintptr(unix.CAP_NET_BIND_SERVICE),
				uintptr(unix.CAP_SYS_NICE),
			))
		})
	})

	Describe("Existing capability propagation (regression)", func() {
		It("should propagate NET_BIND_SERVICE for non-root pods (existing behavior)", func() {
			var inheritableSet [2]unix.CapUserData

			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				// Non-root pod: only NET_BIND_SERVICE in Permitted
				d[0].Permitted = 1 << unix.CAP_NET_BIND_SERVICE
				return nil
			}
			capset = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				inheritableSet = *d
				return nil
			}

			caps := BuildAmbientCaps()

			By("returning NET_BIND_SERVICE as ambient cap")
			Expect(caps).To(ConsistOf(uintptr(unix.CAP_NET_BIND_SERVICE)))

			By("raising NET_BIND_SERVICE into the Inheritable set")
			Expect(inheritableSet[0].Inheritable & (1 << unix.CAP_NET_BIND_SERVICE)).ToNot(BeZero())
		})

	})

	Describe("Plugin-injected capability propagation", func() {
		It("should propagate SYS_RAWIO when a plugin adds it to the pod", func() {
			var inheritableSet [2]unix.CapUserData

			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				// Plugin added SYS_RAWIO to the pod's securityContext.capabilities.add,
				// so the container runtime puts it into the Permitted set alongside
				// the existing caps.
				d[0].Permitted = (1 << unix.CAP_NET_BIND_SERVICE) | (1 << unix.CAP_SYS_RAWIO)
				return nil
			}
			capset = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				inheritableSet = *d
				return nil
			}

			caps := BuildAmbientCaps()

			By("including SYS_RAWIO in the ambient cap list")
			Expect(caps).To(ContainElement(uintptr(unix.CAP_SYS_RAWIO)))

			By("also preserving the existing NET_BIND_SERVICE")
			Expect(caps).To(ContainElement(uintptr(unix.CAP_NET_BIND_SERVICE)))

			By("raising SYS_RAWIO into the Inheritable set for the exec chain")
			Expect(inheritableSet[0].Inheritable & (1 << unix.CAP_SYS_RAWIO)).ToNot(BeZero())
		})

		It("should propagate multiple plugin-injected capabilities", func() {
			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				// Multiple plugins each add their own caps
				d[0].Permitted = (1 << unix.CAP_NET_BIND_SERVICE) |
					(1 << unix.CAP_SYS_NICE) |
					(1 << unix.CAP_SYS_RAWIO) |
					(1 << unix.CAP_NET_ADMIN)
				return nil
			}
			capset = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return nil
			}

			caps := BuildAmbientCaps()

			By("propagating all capabilities from all plugins")
			Expect(caps).To(ConsistOf(
				uintptr(unix.CAP_NET_BIND_SERVICE),
				uintptr(unix.CAP_SYS_NICE),
				uintptr(unix.CAP_SYS_RAWIO),
				uintptr(unix.CAP_NET_ADMIN),
			))
		})

		It("should propagate caps without requiring any core code change", func() {
			// This test demonstrates that a hypothetical future capability
			// (e.g. CAP_SYS_PTRACE=19) can be propagated purely by having
			// the plugin add it to the pod — no change to KubeVirt core needed.
			capget = func(_ *unix.CapUserHeader, data *unix.CapUserData) error {
				d := mockCapUserData(data)
				d[0].Permitted = (1 << unix.CAP_NET_BIND_SERVICE) | (1 << unix.CAP_SYS_PTRACE)
				return nil
			}
			capset = func(_ *unix.CapUserHeader, _ *unix.CapUserData) error {
				return nil
			}

			caps := BuildAmbientCaps()
			Expect(caps).To(ContainElement(uintptr(unix.CAP_SYS_PTRACE)))
		})
	})
})
