package services

import v1 "k8s.io/api/core/v1"

// AppendNonDaemonsetApplicableAffinity merges all affinity rules, except for required node affinities and node selectors.
// This is useful to add affinities which can not be picked up for the virt-handler daemonset to each VMI pod.
func AppendNonDaemonsetApplicableAffinity(given *v1.Affinity, additional *v1.Affinity) *v1.Affinity {
	result := given.DeepCopy()
	if additional.PodAffinity != nil {
		if result.PodAffinity == nil {
			result.PodAffinity = &v1.PodAffinity{}
		}
		result.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(result.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution, additional.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution...)
		result.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(result.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, additional.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution...)
	}

	if additional.PodAntiAffinity != nil {
		if result.PodAntiAffinity == nil {
			result.PodAntiAffinity = &v1.PodAntiAffinity{}
		}
		result.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(result.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, additional.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution...)
		result.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(result.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, additional.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution...)
	}

	if additional.NodeAffinity != nil {
		if result.NodeAffinity == nil {
			result.NodeAffinity = &v1.NodeAffinity{}
		}
		result.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(result.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution, additional.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution...)
	}
	return result
}
