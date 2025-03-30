/*
Copyright 2024 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package checkpoint

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type record struct {
	Name string `json:"name"`
}

var _ = Describe("Simple checkpoint manager", func() {
	It("should be able to check and retrieve", func() {
		r := &record{"Hi"}
		cp := NewSimpleCheckpointManager(GinkgoT().TempDir())

		Expect(cp.Store("win", r)).To(Succeed())

		newR := &record{}
		Expect(cp.Get("win", newR)).To(Succeed())

		Expect(newR.Name).To(Equal("Hi"))
	})

	It("should override if checkpoint already exists", func() {
		r := &record{"Hi"}
		cp := NewSimpleCheckpointManager(GinkgoT().TempDir())

		Expect(cp.Store("win", r)).To(Succeed())

		newRecord := &record{"300"}
		Expect(cp.Store("win", newRecord)).To(Succeed())

		Expect(cp.Get("win", r)).To(Succeed())
		Expect(r.Name).To(Equal("300"))
	})

	It("should return ErrNotExist when asked for non-existing key", func() {
		cp := NewSimpleCheckpointManager(GinkgoT().TempDir())

		r := &record{}
		Expect(cp.Get("win", r)).To(MatchError(os.ErrNotExist))
	})

	It("should remove key", func() {
		r := &record{"Hi"}
		cp := NewSimpleCheckpointManager(GinkgoT().TempDir())

		Expect(cp.Store("win", r)).To(Succeed())

		r = &record{}
		Expect(cp.Get("win", r)).To(Succeed())
		Expect(r.Name).To(Equal("Hi"))

		Expect(cp.Delete("win")).To(Succeed())

		r = &record{}
		Expect(cp.Get("win", r)).To(MatchError(os.ErrNotExist))
	})

	It("remove non-existing key should return not found", func() {
		cp := NewSimpleCheckpointManager(GinkgoT().TempDir())

		Expect(cp.Delete("win")).To(MatchError(os.ErrNotExist))
	})
})
