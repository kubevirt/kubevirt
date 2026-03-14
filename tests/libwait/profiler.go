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

package libwait

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var profilingEnabled = os.Getenv("KUBEVIRT_E2E_PROFILE_VMI_STARTUP") != ""

var (
	profileMu      sync.Mutex
	profileRecords []VMIStartupProfile
)

type PhaseTransition struct {
	Phase     string    `json:"phase"`
	Timestamp time.Time `json:"timestamp"`
}

type ContainerTiming struct {
	Name     string     `json:"name"`
	Started  *time.Time `json:"started,omitempty"`
	Finished *time.Time `json:"finished,omitempty"`
}

type PodTimings struct {
	Created        time.Time         `json:"created"`
	Scheduled      *time.Time        `json:"scheduled,omitempty"`
	InitContainers []ContainerTiming `json:"initContainers,omitempty"`
	Containers     []ContainerTiming `json:"containers,omitempty"`
}

type VMIStartupProfile struct {
	TestName         string            `json:"testName"`
	VMIName          string            `json:"vmiName"`
	Namespace        string            `json:"namespace"`
	TimeoutSeconds   int               `json:"timeoutSeconds"`
	Succeeded        bool              `json:"succeeded"`
	TargetPhases     []string          `json:"targetPhases"`
	FinalPhase       string            `json:"finalPhase"`
	CreationTimestamp time.Time         `json:"creationTimestamp"`
	PhaseTransitions []PhaseTransition `json:"phaseTransitions"`
	PodTimings       *PodTimings       `json:"podTimings,omitempty"`
}

func recordStartupProfile(originalVMI *v1.VirtualMachineInstance, waiting *Waiting) {
	if !profilingEnabled {
		return
	}

	profile := VMIStartupProfile{
		TestName:       ginkgo.CurrentSpecReport().FullText(),
		VMIName:        originalVMI.Name,
		Namespace:      originalVMI.Namespace,
		TimeoutSeconds: waiting.timeout,
		TargetPhases:   phaseStrings(waiting.phases),
	}

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		appendProfile(profile)
		return
	}

	vmi, err := virtClient.VirtualMachineInstance(originalVMI.Namespace).Get(
		context.Background(), originalVMI.Name, metav1.GetOptions{})
	if err != nil {
		appendProfile(profile)
		return
	}

	profile.FinalPhase = string(vmi.Status.Phase)
	profile.CreationTimestamp = vmi.CreationTimestamp.Time
	profile.Succeeded = isTargetPhase(vmi.Status.Phase, waiting.phases)

	for _, pt := range vmi.Status.PhaseTransitionTimestamps {
		profile.PhaseTransitions = append(profile.PhaseTransitions, PhaseTransition{
			Phase:     string(pt.Phase),
			Timestamp: pt.PhaseTransitionTimestamp.Time,
		})
	}

	if vmi.UID != "" {
		profile.PodTimings = collectPodTimings(virtClient, vmi)
	}

	appendProfile(profile)
}

func collectPodTimings(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) *PodTimings {
	labelSelector := fmt.Sprintf("%s=%s", v1.CreatedByLabel, string(vmi.GetUID()))
	pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(
		context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector},
	)
	if err != nil || len(pods.Items) == 0 {
		return nil
	}

	pod := &pods.Items[0]
	timings := &PodTimings{
		Created: pod.CreationTimestamp.Time,
	}

	for _, cond := range pod.Status.Conditions {
		if cond.Type == k8sv1.PodScheduled && cond.Status == k8sv1.ConditionTrue {
			t := cond.LastTransitionTime.Time
			timings.Scheduled = &t
		}
	}

	for _, cs := range pod.Status.InitContainerStatuses {
		ct := ContainerTiming{Name: cs.Name}
		if cs.State.Terminated != nil {
			t := cs.State.Terminated.StartedAt.Time
			ct.Started = &t
			f := cs.State.Terminated.FinishedAt.Time
			ct.Finished = &f
		}
		timings.InitContainers = append(timings.InitContainers, ct)
	}

	for _, cs := range pod.Status.ContainerStatuses {
		ct := ContainerTiming{Name: cs.Name}
		if cs.State.Running != nil {
			t := cs.State.Running.StartedAt.Time
			ct.Started = &t
		} else if cs.State.Terminated != nil {
			t := cs.State.Terminated.StartedAt.Time
			ct.Started = &t
			f := cs.State.Terminated.FinishedAt.Time
			ct.Finished = &f
		}
		timings.Containers = append(timings.Containers, ct)
	}

	return timings
}

func appendProfile(profile VMIStartupProfile) {
	profileMu.Lock()
	defer profileMu.Unlock()
	profileRecords = append(profileRecords, profile)
}

// FlushProfiles writes all collected VMI startup profiles to a JSON file
// in the artifacts directory. It is a no-op when profiling is disabled.
func FlushProfiles() {
	if !profilingEnabled {
		return
	}

	profileMu.Lock()
	defer profileMu.Unlock()

	if len(profileRecords) == 0 {
		return
	}

	artifactsDir := os.Getenv("ARTIFACTS")
	if artifactsDir == "" {
		fmt.Fprintf(ginkgo.GinkgoWriter, "KUBEVIRT_E2E_PROFILE_VMI_STARTUP enabled but ARTIFACTS not set, skipping profile flush\n")
		return
	}

	filename := fmt.Sprintf("vmi-startup-profile-%d.json", ginkgo.GinkgoParallelProcess())
	outputPath := filepath.Join(artifactsDir, filename)

	data, err := json.MarshalIndent(profileRecords, "", "  ")
	if err != nil {
		fmt.Fprintf(ginkgo.GinkgoWriter, "Failed to marshal VMI startup profiles: %v\n", err)
		return
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		fmt.Fprintf(ginkgo.GinkgoWriter, "Failed to write VMI startup profiles to %s: %v\n", outputPath, err)
		return
	}

	fmt.Fprintf(ginkgo.GinkgoWriter, "Wrote %d VMI startup profiles to %s\n", len(profileRecords), outputPath)
}

func phaseStrings(phases []v1.VirtualMachineInstancePhase) []string {
	s := make([]string, len(phases))
	for i, p := range phases {
		s[i] = string(p)
	}
	return s
}

func isTargetPhase(phase v1.VirtualMachineInstancePhase, targets []v1.VirtualMachineInstancePhase) bool {
	for _, t := range targets {
		if phase == t {
			return true
		}
	}
	return false
}
