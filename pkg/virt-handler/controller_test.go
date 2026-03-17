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

package virthandler

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Controller", func() {

	Context("isMigrationInProgress", func() {
		var now metav1.Time

		BeforeEach(func() {
			now = metav1.NewTime(time.Now())
		})

		It("should return false when both vmi and domain are nil", func() {
			Expect(isMigrationInProgress(nil, nil)).To(BeFalse())
		})

		It("should return false when VMI has no migration state", func() {
			vmi := &v1.VirtualMachineInstance{}
			Expect(isMigrationInProgress(vmi, nil)).To(BeFalse())
		})

		It("should return true when VMI migration has started but not ended", func() {
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						StartTimestamp: &now,
					},
				},
			}
			Expect(isMigrationInProgress(vmi, nil)).To(BeTrue())
		})

		It("should return false when VMI migration has both start and end timestamps", func() {
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						StartTimestamp: &now,
						EndTimestamp:   &now,
					},
				},
			}
			Expect(isMigrationInProgress(vmi, nil)).To(BeFalse())
		})

		It("should return true when domain has migration metadata with start but no end", func() {
			domain := &api.Domain{
				Spec: api.DomainSpec{
					Metadata: api.Metadata{
						KubeVirt: api.KubeVirtMetadata{
							Migration: &api.MigrationMetadata{
								StartTimestamp: &now,
							},
						},
					},
				},
			}
			Expect(isMigrationInProgress(nil, domain)).To(BeTrue())
		})

		It("should return false when domain has migration metadata with start and end", func() {
			domain := &api.Domain{
				Spec: api.DomainSpec{
					Metadata: api.Metadata{
						KubeVirt: api.KubeVirtMetadata{
							Migration: &api.MigrationMetadata{
								StartTimestamp: &now,
								EndTimestamp:   &now,
							},
						},
					},
				},
			}
			Expect(isMigrationInProgress(nil, domain)).To(BeFalse())
		})

		DescribeTable("should return true when domain is paused for migration reasons",
			func(reason api.StateChangeReason) {
				domain := &api.Domain{
					Status: api.DomainStatus{
						Status: api.Paused,
						Reason: reason,
					},
				}
				Expect(isMigrationInProgress(nil, domain)).To(BeTrue())
			},
			Entry("paused for migration", api.ReasonPausedMigration),
			Entry("paused for starting up", api.ReasonPausedStartingUp),
			Entry("paused for postcopy", api.ReasonPausedPostcopy),
		)

		It("should return false when domain is paused for a non-migration reason", func() {
			domain := &api.Domain{
				Status: api.DomainStatus{
					Status: api.Paused,
					Reason: api.ReasonPausedUser,
				},
			}
			Expect(isMigrationInProgress(nil, domain)).To(BeFalse())
		})

		It("should return true when VMI is a migration target and migration is not completed", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						v1.CreateMigrationTarget: "true",
					},
				},
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						Completed: false,
					},
				},
			}
			Expect(isMigrationInProgress(vmi, nil)).To(BeTrue())
		})

		It("should return false when VMI is a migration target and migration is completed", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						v1.CreateMigrationTarget: "true",
					},
				},
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						Completed: true,
					},
				},
			}
			Expect(isMigrationInProgress(vmi, nil)).To(BeFalse())
		})
	})

	Context("isMigrationSource", func() {
		const host = "node1"

		It("should return false when migration state is nil", func() {
			c := &BaseController{host: host}
			vmi := &v1.VirtualMachineInstance{}
			Expect(c.isMigrationSource(vmi)).To(BeFalse())
		})

		It("should return false when source node does not match host", func() {
			c := &BaseController{host: host}
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						SourceNode:         "other-node",
						TargetNodeAddress:  "10.0.0.1",
						Completed:          false,
					},
				},
			}
			Expect(c.isMigrationSource(vmi)).To(BeFalse())
		})

		It("should return false when target node address is empty", func() {
			c := &BaseController{host: host}
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						SourceNode:        host,
						TargetNodeAddress: "",
						Completed:         false,
					},
				},
			}
			Expect(c.isMigrationSource(vmi)).To(BeFalse())
		})

		It("should return false when migration is completed", func() {
			c := &BaseController{host: host}
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						SourceNode:        host,
						TargetNodeAddress: "10.0.0.1",
						Completed:         true,
					},
				},
			}
			Expect(c.isMigrationSource(vmi)).To(BeFalse())
		})

		It("should return true when all conditions are met for non-decentralized migration", func() {
			c := &BaseController{host: host}
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						SourceNode:        host,
						TargetNodeAddress: "10.0.0.1",
						Completed:         false,
					},
				},
			}
			Expect(c.isMigrationSource(vmi)).To(BeTrue())
		})
	})

	Context("getVMIFromCache", func() {
		It("should return VMI when it exists in the store", func() {
			store := cache.NewStore(cache.MetaNamespaceKeyFunc)
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
				},
			}
			Expect(store.Add(vmi)).To(Succeed())

			c := &BaseController{vmiStore: store}
			result, exists, err := c.getVMIFromCache("default/test-vmi")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(result.Name).To(Equal("test-vmi"))
			Expect(result.Namespace).To(Equal("default"))
		})

		It("should return a new VMI with name and namespace when not found in store", func() {
			store := cache.NewStore(cache.MetaNamespaceKeyFunc)
			c := &BaseController{vmiStore: store}

			result, exists, err := c.getVMIFromCache("default/test-vmi")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(result.Name).To(Equal("test-vmi"))
			Expect(result.Namespace).To(Equal("default"))
		})
	})

	Context("getDomainFromCache", func() {
		It("should return domain when it exists and is not deleted", func() {
			store := cache.NewStore(cache.MetaNamespaceKeyFunc)
			domain := &api.Domain{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-domain",
					Namespace: "default",
				},
				Spec: api.DomainSpec{
					Metadata: api.Metadata{
						KubeVirt: api.KubeVirtMetadata{
							UID: "test-uid",
						},
					},
				},
			}
			Expect(store.Add(domain)).To(Succeed())

			c := &BaseController{domainStore: store}
			result, exists, uid, err := c.getDomainFromCache("default/test-domain")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(result).ToNot(BeNil())
			Expect(uid).To(BeEquivalentTo("test-uid"))
		})

		It("should treat domain with DeletionTimestamp as non-existent", func() {
			store := cache.NewStore(cache.MetaNamespaceKeyFunc)
			now := metav1.NewTime(time.Now())
			domain := &api.Domain{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-domain",
					Namespace:         "default",
					DeletionTimestamp: &now,
				},
				Spec: api.DomainSpec{
					Metadata: api.Metadata{
						KubeVirt: api.KubeVirtMetadata{
							UID: "test-uid",
						},
					},
				},
			}
			Expect(store.Add(domain)).To(Succeed())

			c := &BaseController{domainStore: store}
			result, exists, uid, err := c.getDomainFromCache("default/test-domain")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(result).To(BeNil())
			Expect(uid).To(BeEquivalentTo("test-uid"))
		})

		It("should return not found when domain is not in the store", func() {
			store := cache.NewStore(cache.MetaNamespaceKeyFunc)
			c := &BaseController{domainStore: store}

			result, exists, uid, err := c.getDomainFromCache("default/missing")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(result).To(BeNil())
			Expect(uid).To(BeEquivalentTo(""))
		})
	})

	Context("setupNetwork", func() {
		It("should return nil when networks list is empty", func() {
			c := &BaseController{}
			vmi := &v1.VirtualMachineInstance{}
			err := c.setupNetwork(vmi, []v1.Network{}, nil)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
