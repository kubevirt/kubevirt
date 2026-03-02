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

package services

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
)

// req is a tiny constructor to keep the table entries readable.
func req(key string, op k8sv1.NodeSelectorOperator, vals ...string) k8sv1.NodeSelectorRequirement {
	return k8sv1.NodeSelectorRequirement{Key: key, Operator: op, Values: vals}
}

func term(matchExpressions []k8sv1.NodeSelectorRequirement, matchFields []k8sv1.NodeSelectorRequirement) k8sv1.NodeSelectorTerm {
	return k8sv1.NodeSelectorTerm{MatchExpressions: matchExpressions, MatchFields: matchFields}
}

var _ = Describe("Node affinity merge algorithm", func() {

	Describe("simplifyNodeSelectorRequirements", func() {
		DescribeTable("canonicalises and detects unsatisfiability",
			func(input []k8sv1.NodeSelectorRequirement, expected []k8sv1.NodeSelectorRequirement, expectedSat bool, expectErr bool) {
				out, sat, err := simplifyNodeSelectorRequirements(input)
				if expectErr {
					Expect(err).To(HaveOccurred())
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(sat).To(Equal(expectedSat))
				if !expectedSat {
					return
				}
				Expect(out).To(Equal(expected))
			},
			Entry("empty input is satisfiable and produces no requirements",
				[]k8sv1.NodeSelectorRequirement{},
				[]k8sv1.NodeSelectorRequirement{},
				true, false,
			),
			Entry("two In requirements on the same key intersect",
				[]k8sv1.NodeSelectorRequirement{
					req("zone", k8sv1.NodeSelectorOpIn, "a", "b", "c"),
					req("zone", k8sv1.NodeSelectorOpIn, "b", "c", "d"),
				},
				[]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "b", "c")},
				true, false,
			),
			Entry("disjoint In requirements are unsatisfiable",
				[]k8sv1.NodeSelectorRequirement{
					req("zone", k8sv1.NodeSelectorOpIn, "a"),
					req("zone", k8sv1.NodeSelectorOpIn, "b"),
				},
				nil, false, false,
			),
			Entry("NotIn requirements union and prune In",
				[]k8sv1.NodeSelectorRequirement{
					req("zone", k8sv1.NodeSelectorOpIn, "a", "b", "c"),
					req("zone", k8sv1.NodeSelectorOpNotIn, "b"),
					req("zone", k8sv1.NodeSelectorOpNotIn, "c"),
				},
				[]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a")},
				true, false,
			),
			Entry("In fully consumed by NotIn is unsatisfiable",
				[]k8sv1.NodeSelectorRequirement{
					req("zone", k8sv1.NodeSelectorOpIn, "a"),
					req("zone", k8sv1.NodeSelectorOpNotIn, "a"),
				},
				nil, false, false,
			),
			Entry("Gt and Lt with no integer between them are unsatisfiable",
				[]k8sv1.NodeSelectorRequirement{
					req("rank", k8sv1.NodeSelectorOpGt, "5"),
					req("rank", k8sv1.NodeSelectorOpLt, "6"),
				},
				nil, false, false,
			),
			Entry("Gt and Lt with at least one integer between them are kept",
				[]k8sv1.NodeSelectorRequirement{
					req("rank", k8sv1.NodeSelectorOpGt, "5"),
					req("rank", k8sv1.NodeSelectorOpLt, "7"),
				},
				[]k8sv1.NodeSelectorRequirement{
					req("rank", k8sv1.NodeSelectorOpGt, "5"),
					req("rank", k8sv1.NodeSelectorOpLt, "7"),
				},
				true, false,
			),
			Entry("In with Gt drops non-conforming integers and non-integers",
				[]k8sv1.NodeSelectorRequirement{
					req("rank", k8sv1.NodeSelectorOpIn, "1", "5", "10", "foo"),
					req("rank", k8sv1.NodeSelectorOpGt, "4"),
				},
				[]k8sv1.NodeSelectorRequirement{req("rank", k8sv1.NodeSelectorOpIn, "10", "5")},
				true, false,
			),
			Entry("Exists with In is collapsed to In only",
				[]k8sv1.NodeSelectorRequirement{
					req("zone", k8sv1.NodeSelectorOpExists),
					req("zone", k8sv1.NodeSelectorOpIn, "a"),
				},
				[]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a")},
				true, false,
			),
			Entry("Exists with DoesNotExist is unsatisfiable",
				[]k8sv1.NodeSelectorRequirement{
					req("zone", k8sv1.NodeSelectorOpExists),
					req("zone", k8sv1.NodeSelectorOpDoesNotExist),
				},
				nil, false, false,
			),
			Entry("Gt with no value returns an error",
				[]k8sv1.NodeSelectorRequirement{
					{Key: "rank", Operator: k8sv1.NodeSelectorOpGt},
				},
				nil, false, true,
			),
		)
	})

	Describe("mergeNestedNodeSelectorTerms", func() {
		It("returns a single empty term for an empty input (matches all nodes)", func() {
			out, err := mergeNestedNodeSelectorTerms(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(k8sv1.NodeSelectorTerm{}))
		})

		It("preserves a single PV's OR'd terms", func() {
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{
					term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "us-east-1a")}, nil),
					term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "us-east-1b")}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(
				term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "us-east-1a")}, []k8sv1.NodeSelectorRequirement{}),
				term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "us-east-1b")}, []k8sv1.NodeSelectorRequirement{}),
			))
		})

		It("intersects two PVs sharing a single key", func() {
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a", "b")}, nil)},
				{term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "b", "c")}, nil)},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(
				term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "b")}, []k8sv1.NodeSelectorRequirement{}),
			))
		})

		It("returns an error when no cross-product is satisfiable", func() {
			_, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a")}, nil)},
				{term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "b")}, nil)},
			})
			Expect(err).To(HaveOccurred())
		})

		It("drops unsatisfiable cross-products and keeps the satisfiable ones", func() {
			// PV1 requires zone in {a,b}; PV2 allows zone=a OR zone=c.
			// Cross-products: (a∩{a,b})=a is satisfiable, (c∩{a,b}) is unsatisfiable.
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a", "b")}, nil)},
				{
					term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a")}, nil),
					term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "c")}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(
				term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a")}, []k8sv1.NodeSelectorRequirement{}),
			))
		})

		It("merges MatchFields and MatchExpressions on the same term", func() {
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{term(
					[]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a")},
					nil,
				)},
				{term(
					nil,
					[]k8sv1.NodeSelectorRequirement{req("metadata.name", k8sv1.NodeSelectorOpIn, "n1", "n2")},
				)},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(term(
				[]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a")},
				[]k8sv1.NodeSelectorRequirement{req("metadata.name", k8sv1.NodeSelectorOpIn, "n1", "n2")},
			)))
		})

		It("prunes a disjunct fully subsumed by another", func() {
			// {zone In [a]} is a subset of {zone In [a, b]} so the more-specific
			// term is redundant in a disjunction and gets dropped.
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{
					term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a", "b")}, nil),
					term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a")}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(
				term([]k8sv1.NodeSelectorRequirement{req("zone", k8sv1.NodeSelectorOpIn, "a", "b")}, []k8sv1.NodeSelectorRequirement{}),
			))
		})

		It("prunes a tight Gt+Lt disjunct subsumed by an In superset", func() {
			// (Gt:4, Lt:8) admits {5,6,7} which is a strict subset of In:[3,5,6,7,9],
			// so the Gt+Lt term is redundant. Detection requires impliesIn to recognise
			// the bounded range as an enumerable In set.
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpIn, "3", "5", "6", "7", "9"),
					}, nil),
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpGt, "4"),
						req("rank", k8sv1.NodeSelectorOpLt, "8"),
					}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(
				term([]k8sv1.NodeSelectorRequirement{
					req("rank", k8sv1.NodeSelectorOpIn, "3", "5", "6", "7", "9"),
				}, []k8sv1.NodeSelectorRequirement{}),
			))
		})

		It("respects NotIn when checking Gt+Lt subsumption against an In superset", func() {
			// (Gt:4, Lt:8) ∧ NotIn:[6] admits {5,7}, which is a subset of In:[5,7,9].
			// NotIn must be subtracted from the range count for the implication to
			// be detected; otherwise we'd mistakenly require 6 to be in the In set.
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpIn, "5", "7", "9"),
					}, nil),
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpGt, "4"),
						req("rank", k8sv1.NodeSelectorOpLt, "8"),
						req("rank", k8sv1.NodeSelectorOpNotIn, "6"),
					}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(
				term([]k8sv1.NodeSelectorRequirement{
					req("rank", k8sv1.NodeSelectorOpIn, "5", "7", "9"),
				}, []k8sv1.NodeSelectorRequirement{}),
			))
		})

		It("does not prune Gt+Lt and In disjuncts when neither subsumes the other", func() {
			// (Gt:0, Lt:10) admits {1..9}; In:[3,5,7,100] admits {3,5,7,100}.
			// Neither set is a subset of the other (1 ∉ In, 100 ∉ range), so both
			// disjuncts must survive — the Gt+Lt → In implication must NOT fire.
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpIn, "3", "5", "7", "100"),
					}, nil),
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpGt, "0"),
						req("rank", k8sv1.NodeSelectorOpLt, "10"),
					}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(HaveLen(2))
		})

		It("prunes a stricter Lt disjunct subsumed by a looser Lt", func() {
			// Lt:5 admits {..,3,4}; Lt:10 admits {..,9}. The smaller bound is
			// stricter, so Lt:5 ⊆ Lt:10 and the stricter disjunct is redundant.
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{
					term([]k8sv1.NodeSelectorRequirement{req("rank", k8sv1.NodeSelectorOpLt, "10")}, nil),
					term([]k8sv1.NodeSelectorRequirement{req("rank", k8sv1.NodeSelectorOpLt, "5")}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(term(
				[]k8sv1.NodeSelectorRequirement{req("rank", k8sv1.NodeSelectorOpLt, "10")},
				[]k8sv1.NodeSelectorRequirement{},
			)))
		})

		It("prunes a stricter Gt disjunct subsumed by a looser Gt", func() {
			// Gt:10 admits {11,..}; Gt:5 admits {6,..}. The larger Gt bound is
			// stricter, so Gt:10 ⊆ Gt:5 and the stricter disjunct is redundant.
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{
					term([]k8sv1.NodeSelectorRequirement{req("rank", k8sv1.NodeSelectorOpGt, "5")}, nil),
					term([]k8sv1.NodeSelectorRequirement{req("rank", k8sv1.NodeSelectorOpGt, "10")}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(term(
				[]k8sv1.NodeSelectorRequirement{req("rank", k8sv1.NodeSelectorOpGt, "5")},
				[]k8sv1.NodeSelectorRequirement{},
			)))
		})

		It("prunes a Gt+Lt range strictly contained in another Gt+Lt range", func() {
			// (Gt:5, Lt:10) admits {6,7,8,9}; (Gt:0, Lt:15) admits {1..14}.
			// The inner range is a subset, so the inner disjunct is redundant.
			// Detection requires impliesGt and impliesLt to both fire on direct
			// bound comparison.
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpGt, "0"),
						req("rank", k8sv1.NodeSelectorOpLt, "15"),
					}, nil),
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpGt, "5"),
						req("rank", k8sv1.NodeSelectorOpLt, "10"),
					}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(term(
				[]k8sv1.NodeSelectorRequirement{
					req("rank", k8sv1.NodeSelectorOpGt, "0"),
					req("rank", k8sv1.NodeSelectorOpLt, "15"),
				},
				[]k8sv1.NodeSelectorRequirement{},
			)))
		})

		It("does not prune Gt+Lt ranges that overlap without containment", func() {
			// (Gt:0, Lt:10) admits {1..9}; (Gt:5, Lt:15) admits {6..14}. They
			// overlap on {6,7,8,9} but neither is a subset, so both must survive.
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpGt, "0"),
						req("rank", k8sv1.NodeSelectorOpLt, "10"),
					}, nil),
					term([]k8sv1.NodeSelectorRequirement{
						req("rank", k8sv1.NodeSelectorOpGt, "5"),
						req("rank", k8sv1.NodeSelectorOpLt, "15"),
					}, nil),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(HaveLen(2))
		})

		It("fails fast when the cross-product would exceed maxNodeSelectorCrossProduct", func() {
			// One volume with > maxNodeSelectorCrossProduct disjuncts forces the
			// guard at the first merge (1 × N > limit).
			oversized := make([]k8sv1.NodeSelectorTerm, maxNodeSelectorCrossProduct+1)
			for i := range oversized {
				oversized[i] = term([]k8sv1.NodeSelectorRequirement{
					req("zone", k8sv1.NodeSelectorOpIn, "z"+strconv.Itoa(i)),
				}, nil)
			}
			_, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{oversized})
			Expect(err).To(MatchError(ContainSubstring("cross-product")))
		})

		It("fails fast when the simplified disjunction exceeds maxNodeSelectorFinalTerms", func() {
			// Each disjunct is on a distinct key, so none can subsume any other
			// in the prune step. Cross-product stays at len = N (under the
			// cross-product limit), but the final term count exceeds the cap.
			disjuncts := make([]k8sv1.NodeSelectorTerm, maxNodeSelectorFinalTerms+1)
			for i := range disjuncts {
				disjuncts[i] = term([]k8sv1.NodeSelectorRequirement{
					req("k"+strconv.Itoa(i), k8sv1.NodeSelectorOpExists),
				}, nil)
			}
			_, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{disjuncts})
			Expect(err).To(MatchError(ContainSubstring("limit")))
		})

		It("preserves NotIn / Gt / Lt operators end-to-end", func() {
			out, err := mergeNestedNodeSelectorTerms([][]k8sv1.NodeSelectorTerm{
				{term([]k8sv1.NodeSelectorRequirement{
					req("zone", k8sv1.NodeSelectorOpNotIn, "blocked"),
					req("rank", k8sv1.NodeSelectorOpGt, "5"),
					req("rank", k8sv1.NodeSelectorOpLt, "100"),
				}, nil)},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(term(
				[]k8sv1.NodeSelectorRequirement{
					req("rank", k8sv1.NodeSelectorOpGt, "5"),
					req("rank", k8sv1.NodeSelectorOpLt, "100"),
					req("zone", k8sv1.NodeSelectorOpNotIn, "blocked"),
				},
				[]k8sv1.NodeSelectorRequirement{},
			)))
		})
	})
})
