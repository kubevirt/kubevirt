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
 */

package libnode

import (
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func TestGetSchedulableNodeWithCPUManager(t *testing.T) {
	eligible := func(name string) *corev1.Node {
		return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{v1.NodeSchedulable: "true", v1.CPUManager: "true"}}}
	}
	cordoned := eligible("cordoned")
	cordoned.Spec.Unschedulable = true
	noCPU := eligible("a-no-cpu")
	delete(noCPU.Labels, v1.CPUManager)
	noSched := eligible("a-no-sched")
	delete(noSched.Labels, v1.NodeSchedulable)
	apiErr := errors.New("node list unavailable")
	cases := []struct {
		name   string
		nodes  []runtime.Object
		apiErr error
		want   string
	}{
		{name: "empty"},
		{name: "one eligible", nodes: []runtime.Object{eligible("only")}, want: "only"},
		{name: "multiple eligible", nodes: []runtime.Object{eligible("a"), eligible("b")}, want: "a"},
		{name: "cpu manager missing", nodes: []runtime.Object{noCPU, eligible("z")}, want: "z"},
		{name: "schedulable missing", nodes: []runtime.Object{noSched, eligible("z")}, want: "z"},
		{name: "only non-schedulable cpu manager", nodes: []runtime.Object{noSched}},
		{name: "cordoned", nodes: []runtime.Object{cordoned}},
		{name: "cordoned then eligible", nodes: []runtime.Object{cordoned, eligible("z")}, want: "z"},
		{name: "list error", apiErr: apiErr},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := kubecli.NewMockKubevirtClient(ctrl)
			nodes := fake.NewSimpleClientset(tc.nodes...)
			if tc.apiErr != nil {
				nodes.PrependReactor("list", "nodes", func(clienttesting.Action) (bool, runtime.Object, error) { return true, nil, tc.apiErr })
			}
			client.EXPECT().CoreV1().Return(nodes.CoreV1()).AnyTimes()
			node, err := GetSchedulableNodeWithCPUManager(client)
			if tc.apiErr != nil {
				if !errors.Is(err, tc.apiErr) || node != nil {
					t.Fatalf("got node %v, error %v", node, err)
				}
				return
			}
			if tc.want == "" {
				if err == nil || !strings.Contains(err.Error(), "no schedulable node with CPU manager enabled") || node != nil {
					t.Fatalf("got node %v, error %v", node, err)
				}
				return
			}
			if err != nil || node == nil || node.Name != tc.want {
				t.Fatalf("got node %v, error %v; want %s", node, err, tc.want)
			}
		})
	}
}
