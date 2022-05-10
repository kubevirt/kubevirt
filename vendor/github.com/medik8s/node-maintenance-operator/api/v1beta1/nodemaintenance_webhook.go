/*
Copyright 2021.

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
	"os"
	"path/filepath"

	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	ErrorNodeNotExists           = "invalid nodeName, no node with name %s found"
	ErrorNodeMaintenanceExists   = "invalid nodeName, a NodeMaintenance for node %s already exists"
	ErrorNodeNameUpdateForbidden = "updating spec.NodeName isn't allowed"
	ErrorMasterQuorumViolation   = "can not put master node into maintenance at this moment, it would violate the master quorum"
)

const (
	EtcdQuorumPDBNewName   = "etcd-guard-pdb"    // The new name of the PDB - From OCP 4.11
	EtcdQuorumPDBOldName   = "etcd-quorum-guard" // The old name of the PDB - Up to OCP 4.10
	EtcdQuorumPDBNamespace = "openshift-etcd"
	LabelNameRoleMaster    = "node-role.kubernetes.io/master"
)

const (
	WebhookCertDir  = "/apiserver.local.config/certificates"
	WebhookCertName = "apiserver.crt"
	WebhookKeyName  = "apiserver.key"
)

// log is for logging in this package.
var nodemaintenancelog = logf.Log.WithName("nodemaintenance-resource")

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// NodeMaintenanceValidator validates NodeMaintenance resources. Needed because we need a client for validation
// +k8s:deepcopy-gen=false
type NodeMaintenanceValidator struct {
	client client.Client
}

var validator *NodeMaintenanceValidator

func (r *NodeMaintenance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// init the validator!
	validator = &NodeMaintenanceValidator{
		client: mgr.GetClient(),
	}

	// check if OLM injected certs
	certs := []string{filepath.Join(WebhookCertDir, WebhookCertName), filepath.Join(WebhookCertDir, WebhookKeyName)}
	certsInjected := true
	for _, fname := range certs {
		if _, err := os.Stat(fname); err != nil {
			certsInjected = false
			break
		}
	}
	if certsInjected {
		server := mgr.GetWebhookServer()
		server.CertDir = WebhookCertDir
		server.CertName = WebhookCertName
		server.KeyName = WebhookKeyName
	} else {
		nodemaintenancelog.Info("OLM injected certs for webhooks not found")
	}

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-nodemaintenance-medik8s-io-v1beta1-nodemaintenance,mutating=false,failurePolicy=fail,sideEffects=None,groups=nodemaintenance.medik8s.io,resources=nodemaintenances,verbs=create;update,versions=v1beta1,name=vnodemaintenance.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &NodeMaintenance{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeMaintenance) ValidateCreate() error {
	nodemaintenancelog.Info("validate create", "name", r.Name)

	if validator == nil {
		return fmt.Errorf("nodemaintenance validator isn't initialized yet")
	}
	return validator.ValidateCreate(r)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeMaintenance) ValidateUpdate(old runtime.Object) error {
	nodemaintenancelog.Info("validate update", "name", r.Name)

	if validator == nil {
		return fmt.Errorf("nodemaintenance validator isn't initialized yet")
	}
	return validator.ValidateUpdate(r, old.(*NodeMaintenance))
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NodeMaintenance) ValidateDelete() error {
	nodemaintenancelog.Info("validate delete", "name", r.Name)

	if validator == nil {
		return fmt.Errorf("nodemaintenance validator isn't initialized yet")
	}
	return nil
}

func (v *NodeMaintenanceValidator) ValidateCreate(nm *NodeMaintenance) error {
	// Validate that node with given name exists
	if err := v.validateNodeExists(nm.Spec.NodeName); err != nil {
		nodemaintenancelog.Info("validation failed", "error", err)
		return err
	}

	// Validate that no NodeMaintenance for given node exists yet
	if err := v.validateNoNodeMaintenanceExists(nm.Spec.NodeName); err != nil {
		nodemaintenancelog.Info("validation failed", "error", err)
		return err
	}

	// Validate that NodeMaintenance for master nodes don't violate quorum
	if err := v.validateMasterQuorum(nm.Spec.NodeName); err != nil {
		nodemaintenancelog.Info("validation failed", "error", err)
		return err
	}

	return nil
}

func (v *NodeMaintenanceValidator) ValidateUpdate(new, old *NodeMaintenance) error {
	// Validate that node name didn't change
	if new.Spec.NodeName != old.Spec.NodeName {
		nodemaintenancelog.Info("validation failed", "error", ErrorNodeNameUpdateForbidden)
		return fmt.Errorf(ErrorNodeNameUpdateForbidden)
	}
	return nil
}

func (v *NodeMaintenanceValidator) validateNodeExists(nodeName string) error {
	if node, err := getNode(nodeName, v.client); err != nil {
		return fmt.Errorf("could not get node for validating spec.NodeName, please try again: %v", err)
	} else if node == nil {
		return fmt.Errorf(ErrorNodeNotExists, nodeName)
	}
	return nil
}

func (v *NodeMaintenanceValidator) validateNoNodeMaintenanceExists(nodeName string) error {
	var nodeMaintenances NodeMaintenanceList
	if err := v.client.List(context.TODO(), &nodeMaintenances, &client.ListOptions{}); err != nil {
		return fmt.Errorf("could not list NodeMaintenances for validating spec.NodeName, please try again: %v", err)
	}

	for _, nm := range nodeMaintenances.Items {
		if nm.Spec.NodeName == nodeName {
			return fmt.Errorf(ErrorNodeMaintenanceExists, nodeName)
		}
	}

	return nil
}

func (v *NodeMaintenanceValidator) validateMasterQuorum(nodeName string) error {
	// check if the node is a master node
	if node, err := getNode(nodeName, v.client); err != nil {
		return fmt.Errorf("could not get node for master quorum validation, please try again: %v", err)
	} else if node == nil {
		// this should have been catched already, but just in case
		return fmt.Errorf(ErrorNodeNotExists, nodeName)
	} else if !isMasterNode(node) {
		// not a master node, nothing to do
		return nil
	}

	// check the etcd-quorum-guard PodDisruptionBudget if we can drain a master node
	disruptionsAllowed := int32(-1)
	for _, pdbName := range []string{EtcdQuorumPDBNewName, EtcdQuorumPDBOldName} {
		var pdb policyv1.PodDisruptionBudget
		key := types.NamespacedName{
			Namespace: EtcdQuorumPDBNamespace,
			Name:      pdbName,
		}
		if err := v.client.Get(context.TODO(), key, &pdb); err != nil {
			if apierrors.IsNotFound(err) {
				// try next one
				continue
			}
			return fmt.Errorf("could not get the etcd quorum guard PDB for master quorum validation, please try again: %v", err)
		}
		disruptionsAllowed = pdb.Status.DisruptionsAllowed
		break
	}
	if disruptionsAllowed == -1 {
		// TODO do we need a fallback for k8s clusters?
		nodemaintenancelog.Info("etcd quorum guard PDB hasn't been found. Skipping master quorum validation.")
		return nil
	}
	if disruptionsAllowed == 0 {
		return fmt.Errorf(ErrorMasterQuorumViolation)
	}
	return nil
}

// if the returned node is nil, it wasn't found
func getNode(nodeName string, client client.Client) (*v1.Node, error) {
	var node v1.Node
	key := types.NamespacedName{
		Name: nodeName,
	}
	if err := client.Get(context.TODO(), key, &node); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get node: %v", err)
	}
	return &node, nil
}

func isMasterNode(node *v1.Node) bool {
	if _, ok := node.Labels[LabelNameRoleMaster]; ok {
		return true
	}
	return false
}
