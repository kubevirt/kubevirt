/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var ssplog = logf.Log.WithName("ssp-resource")
var clt client.Client

func (r *SSP) SetupWebhookWithManager(mgr ctrl.Manager) error {
	clt = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-ssp-kubevirt-io-v1beta1-ssp,mutating=false,failurePolicy=fail,groups=ssp.kubevirt.io,resources=ssps,versions=v1beta1,name=validation.ssp.kubevirt.io,admissionReviewVersions={v1,v1beta1},sideEffects=None

var _ webhook.Validator = &SSP{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *SSP) ValidateCreate() error {
	var ssps SSPList

	// Check if no other SSP resources are present in the cluster
	ssplog.Info("validate create", "name", r.Name)
	err := clt.List(context.TODO(), &ssps, &client.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list SSPs for validation, please try again: %v", err)
	}
	if len(ssps.Items) > 0 {
		return fmt.Errorf("creation failed, an SSP CR already exists in namespace %v: %v", ssps.Items[0].ObjectMeta.Namespace, ssps.Items[0].ObjectMeta.Name)
	}

	// Check if the common templates namespace exists
	namespaceName := r.Spec.CommonTemplates.Namespace
	var namespace v1.Namespace
	err = clt.Get(context.TODO(), client.ObjectKey{Name: namespaceName}, &namespace)
	if err != nil {
		return fmt.Errorf("creation failed, the configured namespace for common templates does not exist: %v", namespaceName)
	}

	if err = validatePlacement(r); err != nil {
		return errors.Wrap(err, "placement api validation error")
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *SSP) ValidateUpdate(old runtime.Object) error {
	ssplog.Info("validate update", "name", r.Name)

	oldSsp := old.(*SSP)
	if r.Spec.CommonTemplates.Namespace != oldSsp.Spec.CommonTemplates.Namespace {
		return fmt.Errorf("commonTemplates.namespace cannot be changed. Attempting to change from: %v to %v",
			oldSsp.Spec.CommonTemplates.Namespace,
			r.Spec.CommonTemplates.Namespace)
	}

	if err := validatePlacement(r); err != nil {
		return errors.Wrap(err, "placement api validation error")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *SSP) ValidateDelete() error {
	return nil
}

// Forces the value of clt, to be used in unit tests
func setClientForWebhook(c client.Client) {
	clt = c
}

func validatePlacement(ssp *SSP) error {
	return validateOperandPlacement(ssp.Namespace, ssp.Spec.TemplateValidator.Placement)
}

func validateOperandPlacement(namespace string, placement *api.NodePlacement) error {
	if placement == nil {
		return nil
	}

	const (
		dplName          = "ssp-webhook-placement-verification-deployment"
		webhookTestLabel = "webhook.ssp.kubevirt.io/placement-verification-pod"
		podName          = "ssp-webhook-placement-verification-pod"
		naImage          = "ssp.kubevirt.io/not-available"
	)

	// Does a dry-run on a deployment creation to verify that placement fields are correct
	deployment := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dplName,
			Namespace: namespace,
		},
		Spec: apps.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					webhookTestLabel: "",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: podName,
					Labels: map[string]string{
						webhookTestLabel: "",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  podName,
							Image: naImage,
						},
					},
					// Inject placement fields here
					NodeSelector: placement.NodeSelector,
					Affinity:     placement.Affinity,
					Tolerations:  placement.Tolerations,
				},
			},
		},
	}

	return clt.Create(context.TODO(), deployment, &client.CreateOptions{DryRun: []string{metav1.DryRunAll}})
}
