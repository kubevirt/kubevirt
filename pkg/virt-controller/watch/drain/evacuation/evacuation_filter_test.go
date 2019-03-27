package evacuation

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v12 "kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Evacuation", func() {

	Context("filtering VMIs", func() {

		var taints []v1.Taint
		var evictionStrategies *v12.EvictionPolicy

		BeforeEach(func() {
			taints = []v1.Taint{
				{
					Key:    "key",
					Effect: "effect",
				},
			}
			evictionStrategies = &v12.EvictionPolicy{

				Taints: []v12.TaintEvictionPolicy{
					{
						Toleration: v1.Toleration{
							Key:    "key1",
							Effect: "effect1",
						},
					},
				},
			}
		})

		It("should ignore taints if they don't have an eviction policy of LiveMigrate set", func() {
			policy := v12.EvictionStrategyNone
			evictionStrategies.Taints = append(evictionStrategies.Taints, v12.TaintEvictionPolicy{
				Toleration: v1.Toleration{
					Key:    "key",
					Effect: "effect",
				},
				Strategy: &policy,
			})
			notTolerated, temporaryTolerated, retryTime := findNotToleratedTaints(time.Now(), evictionStrategies, taints)
			Expect(notTolerated).To(BeEmpty())
			Expect(temporaryTolerated).To(BeEmpty())
			Expect(retryTime).To(BeNil())
		})

		It("should ignore taints if they is no eviction policy set", func() {
			evictionStrategies.Taints = append(evictionStrategies.Taints, v12.TaintEvictionPolicy{
				Toleration: v1.Toleration{
					Key:    "key",
					Effect: "effect",
				},
				Strategy: nil,
			})
			notTolerated, temporaryTolerated, retryTime := findNotToleratedTaints(time.Now(), evictionStrategies, taints)
			Expect(notTolerated).To(BeEmpty())
			Expect(temporaryTolerated).To(BeEmpty())
			Expect(retryTime).To(BeNil())
		})

		It("should not tolerate taints if they have an eviction policy of LiveMigrate set", func() {
			policy := v12.EvictionStrategyLiveMigrate
			evictionStrategies.Taints = append(evictionStrategies.Taints, v12.TaintEvictionPolicy{
				Toleration: v1.Toleration{
					Key:    "key",
					Effect: "effect",
				},
				Strategy: &policy,
			})
			notTolerated, temporaryTolerated, retryTime := findNotToleratedTaints(time.Now(), evictionStrategies, taints)
			Expect(notTolerated).To(HaveLen(1))
			Expect(temporaryTolerated).To(BeEmpty())
			Expect(retryTime).To(BeNil())
		})

		It("should detect if a taint is only temporarily tolerated", func() {
			now := v13.Now()
			var tolerationSeconds int64 = 10
			policy := v12.EvictionStrategyLiveMigrate
			taints = append(taints, v1.Taint{
				Key:       "key2",
				Effect:    "effect2",
				TimeAdded: &now,
			})
			evictionStrategies.Taints = append(evictionStrategies.Taints, v12.TaintEvictionPolicy{
				Toleration: v1.Toleration{
					Key:               "key2",
					Effect:            "effect2",
					TolerationSeconds: &tolerationSeconds,
				},
				Strategy: &policy,
			})
			notTolerated, temporaryTolerated, retryTime := findNotToleratedTaints(now.Time, evictionStrategies, taints)
			Expect(notTolerated).To(BeEmpty())
			Expect(temporaryTolerated).To(HaveLen(1))
			expectedRetryTime := now.Add(time.Duration(tolerationSeconds) * time.Second)
			Expect(retryTime).To(Equal(&expectedRetryTime))
		})

		It("should detect the earliest retry delay", func() {
			now := v13.Now()
			var tolerationSeconds int64 = 10
			var shortTolerationSeconds int64 = 5
			policy := v12.EvictionStrategyLiveMigrate
			taints = append(taints, []v1.Taint{
				{
					Key:       "key2",
					Effect:    "effect2",
					TimeAdded: &now,
				},
				{
					Key:       "key3",
					Effect:    "effect3",
					TimeAdded: &now,
				},
			}...)
			evictionStrategies.Taints = append(evictionStrategies.Taints, []v12.TaintEvictionPolicy{
				{
					Toleration: v1.Toleration{
						Key:               "key2",
						Effect:            "effect2",
						TolerationSeconds: &tolerationSeconds,
					},
					Strategy: &policy,
				},
				{
					Toleration: v1.Toleration{
						Key:               "key3",
						Effect:            "effect3",
						TolerationSeconds: &shortTolerationSeconds,
					},
					Strategy: &policy,
				},
			}...)
			notTolerated, temporaryTolerated, retryTime := findNotToleratedTaints(now.Time, evictionStrategies, taints)
			Expect(notTolerated).To(BeEmpty())
			Expect(temporaryTolerated).To(HaveLen(2))
			expectedRetryTime := now.Add(time.Duration(shortTolerationSeconds) * time.Second)
			Expect(retryTime).To(Equal(&expectedRetryTime))
		})

		It("should detect if a temporary taint toleration expired ", func() {
			now := v13.Now()
			var tolerationSeconds int64 = 10
			policy := v12.EvictionStrategyLiveMigrate
			taints = append(taints, v1.Taint{
				Key:       "key2",
				Effect:    "effect2",
				TimeAdded: &now,
			})
			evictionStrategies.Taints = append(evictionStrategies.Taints, v12.TaintEvictionPolicy{
				Toleration: v1.Toleration{
					Key:               "key2",
					Effect:            "effect2",
					TolerationSeconds: &tolerationSeconds,
				},
				Strategy: &policy,
			})
			notTolerated, temporaryTolerated, retryTime := findNotToleratedTaints(now.Time.Add(-11*time.Second), evictionStrategies, taints)
			Expect(notTolerated).To(HaveLen(1))
			Expect(temporaryTolerated).To(BeEmpty())
			Expect(retryTime).To(BeNil())
		})
	})
})
