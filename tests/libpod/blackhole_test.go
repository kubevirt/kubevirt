/*
 * This file is part of the kubevirt project
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

package libpod

import (
	"context"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func handlerPod(name string) *v1.Pod {
	controller := true
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "kubevirt", UID: types.UID(name),
			Labels:          map[string]string{"kubevirt.io": "virt-handler"},
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "DaemonSet", Name: "virt-handler", UID: "daemonset", Controller: &controller}}},
		Spec:   v1.PodSpec{NodeName: "node01", Containers: []v1.Container{{Name: "virt-handler"}}},
		Status: v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.PodReady, Status: v1.ConditionTrue}}},
	}
}

func TestBlackholeRemovalRetriesMissingContainer(t *testing.T) {
	pod := handlerPod("handler")
	client := fake.NewSimpleClientset(pod, &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "kubernetes", Namespace: "default"}, Spec: v1.ServiceSpec{ClusterIP: "10.96.0.1"}})
	route, missing := false, false
	b := newKubernetesAPIBlackhole(client.CoreV1(), &v1.PodList{Items: []v1.Pod{*pod}}, "virt-handler",
		func(_ context.Context, _ *v1.Pod, _ string, command []string) (string, error) {
			if missing {
				missing = false
				return "", errors.New(`unable to upgrade connection: container not found ("virt-handler")`)
			}
			if command[0] == "sh" {
				return "", nil
			}
			switch command[3] {
			case "show":
				if route {
					return "blackhole 10.96.0.1", nil
				}
			case "add":
				route = true
			case "del":
				route = false
			default:
				t.Fatalf("unexpected command: %s", strings.Join(command, " "))
			}
			return "", nil
		})
	b.pollInterval = time.Millisecond
	if err := b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	missing = true
	if err := b.Remove(context.Background()); err != nil {
		t.Fatal(err)
	}
	if route {
		t.Fatal("API blackhole survived cleanup")
	}
}

// The fake API and executor model the two external boundaries: Kubernetes pod
// lifecycle and the route table owned by each pod UID.
type blackholeFixture struct {
	b      *KubernetesAPIBlackhole
	client *fake.Clientset
	routes map[types.UID]bool
	fault  func(context.Context, *v1.Pod, []string) error
}

func newBlackholeFixture(t *testing.T, pods ...*v1.Pod) *blackholeFixture {
	t.Helper()
	f := &blackholeFixture{client: fake.NewSimpleClientset(), routes: map[types.UID]bool{}}
	_, err := f.client.CoreV1().Services("default").Create(context.Background(), &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "kubernetes"}, Spec: v1.ServiceSpec{ClusterIP: "10.96.0.1"}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	list := &v1.PodList{}
	for _, pod := range pods {
		if _, err := f.client.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
		list.Items = append(list.Items, *pod)
	}
	f.b = newKubernetesAPIBlackhole(f.client.CoreV1(), list, "virt-handler", func(ctx context.Context, pod *v1.Pod, _ string, command []string) (string, error) {
		if f.fault != nil {
			if err := f.fault(ctx, pod, command); err != nil {
				return "", err
			}
		}
		if command[0] == "sh" {
			return "", nil
		}
		switch command[3] {
		case "show":
			if f.routes[pod.UID] {
				return "blackhole 10.96.0.1", nil
			}
		case "add":
			f.routes[pod.UID] = true
		case "del":
			delete(f.routes, pod.UID)
		default:
			t.Fatalf("unexpected command: %v", command)
		}
		return "", nil
	})
	f.b.pollInterval = time.Millisecond
	f.b.execTimeout = 20 * time.Millisecond
	f.b.removalTimeout = 30 * time.Millisecond
	f.b.replacementTimeout = 60 * time.Millisecond
	f.b.readinessTimeout = 30 * time.Millisecond
	return f
}

func TestBlackholeRemovalIsIdempotent(t *testing.T) {
	f := newBlackholeFixture(t, handlerPod("handler"))
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := f.b.Remove(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.b.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBlackholeDoesNotOwnExistingRoute(t *testing.T) {
	f := newBlackholeFixture(t, handlerPod("handler"))
	f.routes["handler"] = true
	if err := f.b.Add(context.Background()); err == nil {
		t.Fatal("accepted an existing route")
	}
	if err := f.b.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !f.routes["handler"] {
		t.Fatal("removed a route this spec did not inject")
	}
}

func TestBlackholeRecoversPartialInjectionWithLostResponse(t *testing.T) {
	f := newBlackholeFixture(t, handlerPod("first"), handlerPod("second"))
	f.fault = func(_ context.Context, pod *v1.Pod, cmd []string) error {
		if pod.Name == "second" && cmd[3] == "add" {
			f.routes[pod.UID] = true // The kernel applied the mutation before exec disconnected.
			return errors.New("exec response lost")
		}
		return nil
	}
	if err := f.b.Add(context.Background()); err == nil {
		t.Fatal("expected lost response")
	}
	f.fault = nil
	if err := f.b.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.routes) != 0 {
		t.Fatalf("injected routes survived: %v", f.routes)
	}
}

func TestBlackholeCleanupBeforeInjection(t *testing.T) {
	for _, b := range []*KubernetesAPIBlackhole{nil, newBlackholeFixture(t, handlerPod("handler")).b} {
		if err := b.Remove(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := b.Recover(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBlackholeExecIsCancelled(t *testing.T) {
	f := newBlackholeFixture(t, handlerPod("handler"))
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.fault = func(ctx context.Context, _ *v1.Pod, _ []string) error {
		<-ctx.Done()
		return ctx.Err()
	}
	start := time.Now()
	if err := f.b.Remove(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected bounded cleanup, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("cleanup exceeded its deadline")
	}
	if !f.routes["handler"] {
		t.Fatal("reported cancellation but silently changed route state")
	}
}

func TestBlackholeRemovalAttemptsEveryTarget(t *testing.T) {
	f := newBlackholeFixture(t, handlerPod("first"), handlerPod("second"))
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.fault = func(_ context.Context, pod *v1.Pod, _ []string) error {
		if pod.Name == "first" {
			return errors.New("container missing")
		}
		return nil
	}
	if err := f.b.Remove(context.Background()); err == nil {
		t.Fatal("expected first target to fail")
	}
	if f.routes["second"] {
		t.Fatal("a failed target prevented another target's cleanup")
	}
	for _, action := range f.client.Actions() {
		if action.GetVerb() == "delete" {
			t.Fatal("normal removal replaced a pod")
		}
	}
}

func TestBlackholeRejectsUnsupportedTargets(t *testing.T) {
	for _, change := range []struct {
		name  string
		apply func(*v1.Pod)
	}{
		{"host network", func(p *v1.Pod) { p.Spec.HostNetwork = true }},
		{"no controller", func(p *v1.Pod) { p.OwnerReferences = nil }},
		{"another controller", func(p *v1.Pod) { p.OwnerReferences[0].Kind = "Deployment" }},
		{"no UID", func(p *v1.Pod) { p.UID = "" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			p := handlerPod("handler")
			change.apply(p)
			f := newBlackholeFixture(t, p)
			if err := f.b.Add(context.Background()); err == nil {
				t.Fatal("unsupported injection accepted")
			}
			if len(f.routes) != 0 {
				t.Fatal("route injected before validation")
			}
		})
	}
}

func TestBlackholeRecoveryReplacesOnlyCapturedPod(t *testing.T) {
	original := handlerPod("old")
	replacement := handlerPod("new")
	f := newBlackholeFixture(t, original)
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.fault = func(_ context.Context, p *v1.Pod, _ []string) error {
		if p.UID == original.UID {
			return errors.New("container not found")
		}
		return nil
	}
	f.client.PrependReactor("delete", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		deletion := action.(ktesting.DeleteAction)
		opts := deletion.GetDeleteOptions()
		if deletion.GetName() != original.Name || opts.Preconditions == nil || opts.Preconditions.UID == nil || *opts.Preconditions.UID != original.UID {
			t.Fatal("replacement was not guarded by the original UID")
		}
		if opts.GracePeriodSeconds != nil {
			t.Fatal("recovery bypassed normal graceful termination")
		}
		if err := f.client.Tracker().Delete(v1.SchemeGroupVersion.WithResource("pods"), original.Namespace, original.Name); err != nil {
			t.Fatal(err)
		}
		if err := f.client.Tracker().Add(replacement); err != nil {
			t.Fatal(err)
		}
		return true, nil, nil
	})
	if err := f.b.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.client.CoreV1().Pods(original.Namespace).Get(context.Background(), replacement.Name, metav1.GetOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestBlackholeRecoveryDoesNotDeleteReplacementAfterUIDConflict(t *testing.T) {
	original := handlerPod("handler")
	replacement := original.DeepCopy()
	replacement.UID = "replacement"
	f := newBlackholeFixture(t, original)
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.fault = func(_ context.Context, p *v1.Pod, _ []string) error {
		if p.UID == original.UID {
			return errors.New("container missing")
		}
		return nil
	}
	deleted := false
	f.client.PrependReactor("delete", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		if deleted {
			t.Fatal("attempted to delete the replacement")
		}
		deleted = true
		if err := f.client.Tracker().Update(v1.SchemeGroupVersion.WithResource("pods"), replacement, original.Namespace); err != nil {
			t.Fatal(err)
		}
		return true, nil, apierrors.NewConflict(v1.Resource("pods"), original.Name, errors.New("UID precondition failed"))
	})
	if err := f.b.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBlackholeRecoveryVerifiesAlreadyReplacedPod(t *testing.T) {
	original := handlerPod("old")
	f := newBlackholeFixture(t, original)
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.client.Tracker().Delete(v1.SchemeGroupVersion.WithResource("pods"), original.Namespace, original.Name); err != nil {
		t.Fatal(err)
	}
	replacement := handlerPod("new")
	if err := f.client.Tracker().Add(replacement); err != nil {
		t.Fatal(err)
	}
	f.routes[replacement.UID] = true
	if err := f.b.Recover(context.Background()); err == nil {
		t.Fatal("accepted a replacement with an API blackhole")
	}
	for _, action := range f.client.Actions() {
		if action.GetVerb() == "delete" {
			t.Fatal("deleted an already-replaced pod")
		}
	}
}

func TestBlackholeReadinessFailureDoesNotReplacePod(t *testing.T) {
	p := handlerPod("handler")
	p.Status.Conditions = nil
	f := newBlackholeFixture(t, p)
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.b.Recover(context.Background()); err == nil || !strings.Contains(err.Error(), "not ready") {
		t.Fatalf("expected readiness failure, got %v", err)
	}
	if f.routes[p.UID] {
		t.Fatal("readiness failure left route installed")
	}
	for _, action := range f.client.Actions() {
		if action.GetVerb() == "delete" {
			t.Fatal("readiness-only failure caused replacement")
		}
	}
}

func TestBlackholeRecoveryAttemptsEveryTargetAfterReplacementFailure(t *testing.T) {
	f := newBlackholeFixture(t, handlerPod("first"), handlerPod("second"))
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.fault = func(_ context.Context, p *v1.Pod, _ []string) error {
		if p.Name == "first" {
			return errors.New("container missing")
		}
		return nil
	}
	f.client.PrependReactor("delete", "pods", func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, errors.New("API unavailable") })
	if err := f.b.Recover(context.Background()); err == nil {
		t.Fatal("expected unrecoverable first target")
	}
	if f.routes["second"] {
		t.Fatal("replacement failure prevented cleanup of the second target")
	}
}

func TestBlackholeUsesCapturedIPv6Destination(t *testing.T) {
	f := newBlackholeFixture(t, handlerPod("handler"))
	service, err := f.client.CoreV1().Services("default").Get(context.Background(), "kubernetes", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	service.Spec.ClusterIP = "fd00::1"
	if _, err := f.client.CoreV1().Services("default").Update(context.Background(), service, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	f.fault = func(_ context.Context, _ *v1.Pod, command []string) error {
		if command[1] != "-6" || command[len(command)-1] != "fd00::1" {
			t.Fatalf("unexpected family/destination: %v", command)
		}
		return nil
	}
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	service.Spec.ClusterIP = "fd00::2"
	if _, err := f.client.CoreV1().Services("default").Update(context.Background(), service, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := f.b.Remove(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBlackholeNormalRemovalDoesNotAcceptPodReplacement(t *testing.T) {
	original := handlerPod("old")
	f := newBlackholeFixture(t, original)
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.client.Tracker().Delete(v1.SchemeGroupVersion.WithResource("pods"), original.Namespace, original.Name); err != nil {
		t.Fatal(err)
	}
	if err := f.client.Tracker().Add(handlerPod("new")); err != nil {
		t.Fatal(err)
	}
	if err := f.b.Remove(context.Background()); err == nil {
		t.Fatal("replacement masked failure of in-place recovery")
	}
	if err := f.b.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBlackholeRecoveryWaitsThroughReplacementTransitions(t *testing.T) {
	original := handlerPod("old")
	f := newBlackholeFixture(t, original)
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.client.Tracker().Delete(v1.SchemeGroupVersion.WithResource("pods"), original.Namespace, original.Name); err != nil {
		t.Fatal(err)
	}
	replacement := handlerPod("new")
	if err := f.client.Tracker().Add(replacement); err != nil {
		t.Fatal(err)
	}
	lists := 0
	f.client.PrependReactor("list", "pods", func(ktesting.Action) (bool, runtime.Object, error) {
		lists++
		switch lists {
		case 1:
			return true, &v1.PodList{}, nil
		case 2:
			return true, &v1.PodList{Items: []v1.Pod{*replacement, *handlerPod("extra")}}, nil
		default:
			return false, nil, nil
		}
	})
	if err := f.b.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBlackholeRecoveryRequiresAPIAccess(t *testing.T) {
	f := newBlackholeFixture(t, handlerPod("handler"))
	if err := f.b.Add(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.fault = func(_ context.Context, _ *v1.Pod, command []string) error {
		if command[0] == "sh" {
			return errors.New("API request failed")
		}
		return nil
	}
	if err := f.b.Recover(context.Background()); err == nil || !strings.Contains(err.Error(), "API request failed") {
		t.Fatalf("accepted ready pod without API connectivity: %v", err)
	}
	for _, action := range f.client.Actions() {
		if action.GetVerb() == "delete" {
			t.Fatal("API failure after verified route removal replaced pod")
		}
	}
}

func TestBlackholeCleanupOrdering(t *testing.T) {
	// Run intentionally failing specs in a child process to exercise real Ginkgo
	// cleanup ordering, failure preservation and abort behavior.
	mode := os.Getenv("KUBEVIRT_BLACKHOLE_CLEANUP_CHILD")
	if mode == "" {
		for _, scenario := range []string{"restored", "blocked"} {
			t.Run(scenario, func(t *testing.T) {
				command := osexec.Command(os.Args[0], "-test.run=^TestBlackholeCleanupOrdering$")
				command.Env = append(os.Environ(), "KUBEVIRT_BLACKHOLE_CLEANUP_CHILD="+scenario)
				output, err := command.CombinedOutput()
				if err == nil {
					t.Fatalf("expected intentional spec failure: %s", output)
				}
				if scenario == "restored" {
					if !strings.Contains(string(output), "global cleanup saw restored routes") || !strings.Contains(string(output), "original failure preserved") {
						t.Fatalf("unexpected cleanup lifecycle: %s", output)
					}
				} else if !strings.Contains(string(output), "suite aborted") || strings.Contains(string(output), "following spec executed") {
					t.Fatalf("contaminated suite continued: %s", output)
				}
			})
		}
		return
	}
	f := newBlackholeFixture(t, handlerPod("handler"))
	ginkgo.Describe("suite cleanup", func() {
		ginkgo.AfterEach(func() {
			if mode == "restored" {
				if len(f.routes) != 0 {
					ginkgo.Fail("global cleanup ran before route restoration")
				}
				fmt.Println("global cleanup saw restored routes")
			}
		})
		ginkgo.Context("disruptive test", func() {
			ginkgo.AfterEach(func() {
				if err := f.b.Recover(context.Background()); err != nil {
					ginkgo.AbortSuite(err.Error())
				}
			})
			ginkgo.It("fails after injection", func() {
				if err := f.b.Add(context.Background()); err != nil {
					ginkgo.Fail(err.Error())
				}
				if mode == "blocked" {
					f.fault = func(context.Context, *v1.Pod, []string) error { return errors.New("container missing") }
					f.client.PrependReactor("delete", "pods", func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, errors.New("API unavailable") })
				}
				ginkgo.Fail("original behavioral failure")
			})
			if mode == "blocked" {
				ginkgo.It("must not run after failed recovery", func() { fmt.Println("following spec executed") })
			}
		})
	})
	ginkgo.ReportAfterEach(func(report ginkgo.SpecReport) {
		if report.State.String() == "aborted" {
			fmt.Println("suite aborted")
		}
		if report.Failed() && report.Failure.Message == "original behavioral failure" {
			fmt.Println("original failure preserved")
		}
	})
	ginkgo.RunSpecs(t, "Blackhole cleanup ordering")
}
