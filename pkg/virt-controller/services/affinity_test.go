package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Affinity", func() {
	DescribeTable("should be merged", func(given, additional, desired *v1.Affinity) {
		Expect(AppendNonDaemonsetApplicableAffinity(given, additional)).To(Equal(desired))
	},
		Entry("with empty affinities",
			&v1.Affinity{},
			&v1.Affinity{},
			&v1.Affinity{},
		),
		Entry("with minimal given affinity, and fully populated additional affinity, it should ignore required node constraints",
			&v1.Affinity{},
			getFullyPopulatedAffinity(),
			getFullyPopulatedAffinityWithoutNodeRequires(),
		),
		Entry("with fully populated given affinity, it should stay unmodified if the additional affinity is empty",
			getFullyPopulatedAffinity(),
			&v1.Affinity{},
			getFullyPopulatedAffinity(),
		),
		Entry("with fully populated given affinity, and fully populated additional affinity, it should merge everything except required node constraints",
			getFullyPopulatedAffinity(),
			getFullyPopulatedAffinity(),
			getDoublePopulatedAffinityWithoutDuplicatedRequiredNodeConstraints(),
		),
	)
})

func getFullyPopulatedAffinity() *v1.Affinity {
	return &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					{
						MatchExpressions: []v1.NodeSelectorRequirement{
							{
								Key:      "KeyA",
								Operator: "OperatorA",
								Values:   []string{"ValueA"},
							},
						},
						MatchFields: []v1.NodeSelectorRequirement{
							{
								Key:      "KeyB",
								Operator: "OperatorB",
								Values:   []string{"ValueB"},
							},
						},
					},
				},
			},
			PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
				{
					Weight: 0,
					Preference: v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{
							{
								Key:      "KeyC",
								Operator: "OperatorC",
								Values:   []string{"ValueC"},
							},
						},
						MatchFields: []v1.NodeSelectorRequirement{
							{
								Key:      "KeyD",
								Operator: "OperatorD",
								Values:   []string{"ValueD"},
							},
						},
					},
				},
			},
		},
		PodAffinity: &v1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
				{
					LabelSelector: &v12.LabelSelector{
						MatchLabels: map[string]string{},
						MatchExpressions: []v12.LabelSelectorRequirement{
							{
								Key:      "KeyE",
								Operator: "OperatorE",
								Values:   []string{"ValueE"},
							},
						},
					},
					Namespaces:  []string{},
					TopologyKey: "",
					NamespaceSelector: &v12.LabelSelector{
						MatchLabels: map[string]string{},
						MatchExpressions: []v12.LabelSelectorRequirement{
							{
								Key:      "KeyF",
								Operator: "OperatorF",
								Values:   []string{"ValueF"},
							},
						},
					},
				},
			},
			PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
				{
					Weight: 0,
					PodAffinityTerm: v1.PodAffinityTerm{
						LabelSelector: &v12.LabelSelector{
							MatchLabels: map[string]string{},
							MatchExpressions: []v12.LabelSelectorRequirement{
								{
									Key:      "KeyG",
									Operator: "OperatorG",
									Values:   []string{"ValueG"},
								},
							},
						},
						Namespaces:  []string{},
						TopologyKey: "",
						NamespaceSelector: &v12.LabelSelector{
							MatchLabels: map[string]string{},
							MatchExpressions: []v12.LabelSelectorRequirement{
								{
									Key:      "KeyH",
									Operator: "OperatorH",
									Values:   []string{"ValueH"},
								},
							},
						},
					},
				},
			},
		},
		PodAntiAffinity: &v1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
				{
					LabelSelector: &v12.LabelSelector{
						MatchLabels: map[string]string{},
						MatchExpressions: []v12.LabelSelectorRequirement{
							{
								Key:      "KeyJ",
								Operator: "OperatorJ",
								Values:   []string{"ValueJ"},
							},
						},
					},
					Namespaces:  []string{},
					TopologyKey: "",
					NamespaceSelector: &v12.LabelSelector{
						MatchLabels: map[string]string{},
						MatchExpressions: []v12.LabelSelectorRequirement{
							{
								Key:      "KeyK",
								Operator: "OperatorK",
								Values:   []string{"ValueK"},
							},
						},
					},
				},
			},
			PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
				{
					Weight: 0,
					PodAffinityTerm: v1.PodAffinityTerm{
						LabelSelector: &v12.LabelSelector{
							MatchLabels: map[string]string{},
							MatchExpressions: []v12.LabelSelectorRequirement{
								{
									Key:      "KeyL",
									Operator: "OeratorL",
									Values:   []string{"ValueL"},
								},
							},
						},
						Namespaces:  []string{},
						TopologyKey: "something",
						NamespaceSelector: &v12.LabelSelector{
							MatchLabels: map[string]string{},
							MatchExpressions: []v12.LabelSelectorRequirement{
								{
									Key:      "KeyM",
									Operator: "OperatorM",
									Values:   []string{"ValueL"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func getDoublePopulatedAffinityWithoutDuplicatedRequiredNodeConstraints() *v1.Affinity {
	base := getFullyPopulatedAffinity()
	base.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(base.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, base.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution...)
	base.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(base.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution, base.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution...)
	base.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(base.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, base.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution...)
	base.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(base.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, base.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution...)
	base.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(base.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution, base.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution...)
	return base
}

func getFullyPopulatedAffinityWithoutNodeRequires() *v1.Affinity {
	affinity := getFullyPopulatedAffinity()
	affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = nil
	return affinity
}
