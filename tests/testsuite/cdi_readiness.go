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
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cdiclient "kubevirt.io/client-go/containerizeddataimporter"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

// EnsureCDIReady checks a CDI installation when one exists, or requires one
// when the test job declares CDI as a dependency.
func EnsureCDIReady(timeout time.Duration) {
	client := kubevirt.Client().CdiClient()
	Eventually(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return cdiReadiness(ctx, client, flags.RequireCDI)
	}, timeout, 2*time.Second).Should(Succeed(), "CDI is not ready for suite startup")
}

func cdiReadiness(ctx context.Context, client cdiclient.Interface, required bool) error {
	cdis, err := client.CdiV1beta1().CDIs().List(ctx, metav1.ListOptions{})
	if apierrors.IsNotFound(err) {
		if required {
			return fmt.Errorf("CDI is required but its resource API is absent: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("list CDI resources: %w", err)
	}
	if len(cdis.Items) == 0 {
		if required {
			return fmt.Errorf("CDI is required but no CDI resource exists")
		}
		return nil
	}
	if len(cdis.Items) != 1 {
		return fmt.Errorf("expected one CDI resource, found %d", len(cdis.Items))
	}

	cdi := &cdis.Items[0]
	if cdi.Status.Phase != sdkapi.PhaseDeployed {
		return fmt.Errorf("CDI %q phase is %q, expected %q", cdi.Name, cdi.Status.Phase, sdkapi.PhaseDeployed)
	}
	for _, expected := range []struct {
		condition conditionsv1.ConditionType
		status    k8sv1.ConditionStatus
	}{
		{conditionsv1.ConditionAvailable, k8sv1.ConditionTrue},
		{conditionsv1.ConditionProgressing, k8sv1.ConditionFalse},
		{conditionsv1.ConditionDegraded, k8sv1.ConditionFalse},
	} {
		condition := conditionsv1.FindStatusCondition(cdi.Status.Conditions, expected.condition)
		if condition == nil {
			return fmt.Errorf("CDI %q has no %s condition", cdi.Name, expected.condition)
		}
		if condition.Status != expected.status {
			return fmt.Errorf("CDI %q condition %s is %s, expected %s: %s", cdi.Name, expected.condition, condition.Status, expected.status, condition.Message)
		}
	}
	return nil
}
