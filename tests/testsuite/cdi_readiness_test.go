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

package testsuite

import (
	"context"
	"errors"
	"strings"
	"testing"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
	cdiclientfake "kubevirt.io/client-go/containerizeddataimporter/fake"
	cditypes "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"
)

func TestCDIReadinessOptionalAndRequired(t *testing.T) {
	client := cdiclientfake.NewSimpleClientset()
	if err := cdiReadiness(context.Background(), client, false); err != nil {
		t.Fatalf("optional missing CDI blocked startup: %v", err)
	}
	if err := cdiReadiness(context.Background(), client, true); err == nil {
		t.Fatal("required missing CDI was accepted")
	}
}

func TestCDIReadinessConditions(t *testing.T) {
	cdi := &cditypes.CDI{
		ObjectMeta: metav1.ObjectMeta{Name: "custom-name"},
		Status: cditypes.CDIStatus{Status: sdkapi.Status{
			Phase: sdkapi.PhaseDeployed,
			Conditions: []conditionsv1.Condition{
				{Type: conditionsv1.ConditionAvailable, Status: k8sv1.ConditionTrue},
				{Type: conditionsv1.ConditionProgressing, Status: k8sv1.ConditionFalse},
				{Type: conditionsv1.ConditionDegraded, Status: k8sv1.ConditionFalse},
			},
		}},
	}
	client := cdiclientfake.NewSimpleClientset(cdi)
	if err := cdiReadiness(context.Background(), client, false); err != nil {
		t.Fatalf("ready CDI rejected: %v", err)
	}

	cdi.Status.Conditions[0].Status = k8sv1.ConditionFalse
	if _, err := client.CdiV1beta1().CDIs().Update(context.Background(), cdi, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := cdiReadiness(context.Background(), client, false); err == nil || !strings.Contains(err.Error(), "Available") {
		t.Fatalf("unavailable CDI did not identify its condition: %v", err)
	}

	cdi.Status.Conditions = cdi.Status.Conditions[:1]
	cdi.Status.Conditions[0].Status = k8sv1.ConditionTrue
	if _, err := client.CdiV1beta1().CDIs().Update(context.Background(), cdi, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := cdiReadiness(context.Background(), client, false); err == nil || !strings.Contains(err.Error(), "no Progressing condition") {
		t.Fatalf("missing CDI condition was accepted: %v", err)
	}
}

func TestCDIReadinessDoesNotHideAPIErrors(t *testing.T) {
	client := cdiclientfake.NewSimpleClientset()
	client.PrependReactor("list", "cdis", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("API unavailable")
	})
	if err := cdiReadiness(context.Background(), client, false); err == nil || !strings.Contains(err.Error(), "API unavailable") {
		t.Fatalf("CDI API error was treated as optional absence: %v", err)
	}
}

func TestCDIReadinessHandlesAbsentResourceAPI(t *testing.T) {
	client := cdiclientfake.NewSimpleClientset()
	client.PrependReactor("list", "cdis", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "cdi.kubevirt.io", Resource: "cdis"}, "")
	})
	if err := cdiReadiness(context.Background(), client, false); err != nil {
		t.Fatalf("optional missing CDI API blocked startup: %v", err)
	}
	if err := cdiReadiness(context.Background(), client, true); err == nil {
		t.Fatal("required missing CDI API was accepted")
	}
}
