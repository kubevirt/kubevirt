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

package mountrecord_test

import (
	"encoding/json"
	"errors"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/virt-handler/mountrecord"
)

const vmiUID = types.UID("1234")

type fakeCheckpointManager struct {
	checkpoints map[string][]byte
	stores      int
	storeErr    error
}

func (f *fakeCheckpointManager) Get(key string, value any) error {
	data, ok := f.checkpoints[key]
	if !ok {
		return os.ErrNotExist
	}
	return json.Unmarshal(data, value)
}

func (f *fakeCheckpointManager) Store(key string, value any) error {
	f.stores++
	if f.storeErr != nil {
		return f.storeErr
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.checkpoints[key] = data
	return nil
}

func (f *fakeCheckpointManager) Delete(key string) error {
	if _, ok := f.checkpoints[key]; !ok {
		return os.ErrNotExist
	}
	delete(f.checkpoints, key)
	return nil
}

var _ = Describe("Mount record store", func() {
	var (
		checkpoints *fakeCheckpointManager
		store       *mountrecord.Store
	)

	newStore := func() *mountrecord.Store {
		return mountrecord.NewStore(checkpoints)
	}

	BeforeEach(func() {
		checkpoints = &fakeCheckpointManager{checkpoints: map[string][]byte{}}
		store = newStore()
	})

	It("should return no entries for a VMI without a record", func() {
		Expect(store.Entries(vmiUID)).To(BeNil())
	})

	It("should reject a VMI without UID", func() {
		const noUID = "requires a VMI UID"
		Expect(store.Entries("")).Error().To(MatchError(ContainSubstring(noUID)))
		Expect(store.Add("", mountrecord.Entry{TargetFile: "/a"})).To(MatchError(ContainSubstring(noUID)))
		Expect(store.Replace("", nil)).To(MatchError(ContainSubstring(noUID)))
		Expect(store.Delete("")).To(MatchError(ContainSubstring(noUID)))
	})

	Context("Add", func() {
		It("should persist entries that survive a virt-handler restart", func() {
			Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/a", SocketFile: "/a.sock"})).To(Succeed())
			Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/b"})).To(Succeed())

			Expect(newStore().Entries(vmiUID)).To(Equal([]mountrecord.Entry{
				{TargetFile: "/a", SocketFile: "/a.sock"},
				{TargetFile: "/b"},
			}))
		})

		It("should skip targets that are already recorded without writing", func() {
			Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/a"})).To(Succeed())
			Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/a"}, mountrecord.Entry{TargetFile: "/a"})).To(Succeed())

			Expect(checkpoints.stores).To(Equal(1))
			Expect(store.Entries(vmiUID)).To(ConsistOf(mountrecord.Entry{TargetFile: "/a"}))
		})

		It("should not change the cached entries when the write fails", func() {
			Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/a"})).To(Succeed())
			checkpoints.storeErr = errors.New("disk full")

			Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/b"})).To(MatchError(ContainSubstring("disk full")))
			Expect(store.Entries(vmiUID)).To(ConsistOf(mountrecord.Entry{TargetFile: "/a"}))
		})
	})

	Context("Replace", func() {
		It("should record exactly the given entries", func() {
			Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/a"}, mountrecord.Entry{TargetFile: "/b"})).To(Succeed())
			Expect(store.Replace(vmiUID, []mountrecord.Entry{{TargetFile: "/b"}})).To(Succeed())

			Expect(newStore().Entries(vmiUID)).To(Equal([]mountrecord.Entry{{TargetFile: "/b"}}))
		})

		It("should not write unchanged entries again", func() {
			entries := []mountrecord.Entry{{TargetFile: "/a"}}
			Expect(store.Replace(vmiUID, entries)).To(Succeed())
			Expect(store.Replace(vmiUID, entries)).To(Succeed())

			Expect(checkpoints.stores).To(Equal(1))
		})
	})

	It("should not let callers modify the stored entries", func() {
		Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/a"})).To(Succeed())

		entries, err := store.Entries(vmiUID)
		Expect(err).ToNot(HaveOccurred())
		entries[0].TargetFile = "/modified"

		Expect(store.Entries(vmiUID)).To(ConsistOf(mountrecord.Entry{TargetFile: "/a"}))
	})

	Context("Delete", func() {
		It("should remove the record", func() {
			Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/a"})).To(Succeed())

			Expect(store.Delete(vmiUID)).To(Succeed())

			Expect(checkpoints.checkpoints).To(BeEmpty())
			Expect(store.Entries(vmiUID)).To(BeNil())
		})

		It("should succeed when there is no record", func() {
			Expect(store.Delete(vmiUID)).To(Succeed())
		})
	})

	It("should keep the on-disk record format", func() {
		checkpoints.checkpoints[string(vmiUID)] = []byte(`{"mountTargetEntries":[{"targetFile":"/t","socketFile":"/s"}],"usesSafePaths":true}`)
		Expect(store.Add(vmiUID, mountrecord.Entry{TargetFile: "/h"})).To(Succeed())

		Expect(checkpoints.checkpoints[string(vmiUID)]).To(MatchJSON(
			`{"mountTargetEntries":[{"targetFile":"/t","socketFile":"/s"},{"targetFile":"/h"}],"usesSafePaths":true}`,
		))
	})
})
