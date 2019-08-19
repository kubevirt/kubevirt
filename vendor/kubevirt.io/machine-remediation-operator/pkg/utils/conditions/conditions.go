package conditions

import (
	"context"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetNodeCondition returns node condition by type
func GetNodeCondition(node *corev1.Node, conditionType corev1.NodeConditionType) *corev1.NodeCondition {
	for _, cond := range node.Status.Conditions {
		if cond.Type == conditionType {
			return &cond
		}
	}
	return nil
}

// NodeHasCondition returns true when the node has condition of the specific type and status
func NodeHasCondition(node *corev1.Node, conditionType corev1.NodeConditionType, contidionStatus corev1.ConditionStatus) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == conditionType && cond.Status == contidionStatus {
			return true
		}
	}
	return false
}

// UnhealthyConditions contains a list of UnhealthyCondition
type UnhealthyConditions struct {
	Items []UnhealthyCondition `json:"items"`
}

// UnhealthyCondition is the representation of unhealthy conditions under the config map
type UnhealthyCondition struct {
	Name    corev1.NodeConditionType `json:"name"`
	Status  corev1.ConditionStatus   `json:"status"`
	Timeout string                   `json:"timeout"`
}

// CreateDummyUnhealthyConditionsConfigMap creates dummy config map with default unhealthy conditions
func createDummyUnhealthyConditionsConfigMap() (*corev1.ConfigMap, error) {
	unhealthyConditions := &UnhealthyConditions{
		Items: []UnhealthyCondition{
			{
				Name:    "Ready",
				Status:  "Unknown",
				Timeout: "300s",
			},
			{
				Name:    "Ready",
				Status:  "False",
				Timeout: "300s",
			},
		},
	}
	conditionsData, err := yaml.Marshal(unhealthyConditions)
	if err != nil {
		return nil, err
	}
	return &corev1.ConfigMap{Data: map[string]string{"conditions": string(conditionsData)}}, nil
}

// GetNodeUnhealthyConditions returns node unhealthy conditions
func GetNodeUnhealthyConditions(node *corev1.Node, cmUnealthyConditions *corev1.ConfigMap) ([]UnhealthyCondition, error) {
	data, ok := cmUnealthyConditions.Data["conditions"]
	if !ok {
		return nil, fmt.Errorf("can not find \"conditions\" under the configmap")
	}

	var unealthyConditions UnhealthyConditions
	err := yaml.Unmarshal([]byte(data), &unealthyConditions)
	if err != nil {
		glog.Errorf("failed to umarshal: %v", err)
		return nil, err
	}

	conditions := []UnhealthyCondition{}
	for _, c := range unealthyConditions.Items {
		cond := GetNodeCondition(node, c.Name)
		if cond != nil && cond.Status == c.Status {
			conditions = append(conditions, c)
		}
	}
	return conditions, nil
}

// GetUnhealthyConditionsConfigMap get config map with unhealthy node conditions, when the config map
// does not exist, returns dummy config map with default unhealthy conditions
func GetUnhealthyConditionsConfigMap(c client.Client, namespace string) (*corev1.ConfigMap, error) {
	cmUnhealtyConditions := &corev1.ConfigMap{}
	cmKey := types.NamespacedName{
		Name:      mrv1.ConfigMapNodeUnhealthyConditions,
		Namespace: namespace,
	}
	err := c.Get(context.TODO(), cmKey, cmUnhealtyConditions)
	if err != nil {
		// Error reading the object - requeue the request
		if !errors.IsNotFound(err) {
			return nil, err
		}

		// creates dummy config map with default values if it does not exist
		cmUnhealtyConditions, err = createDummyUnhealthyConditionsConfigMap()
		if err != nil {
			return nil, err
		}
		glog.Infof(
			"ConfigMap %s not found under the namespace %s, fallback to default values: %s",
			mrv1.ConfigMapNodeUnhealthyConditions,
			namespace,
			cmUnhealtyConditions.Data["conditions"],
		)
	}
	return cmUnhealtyConditions, nil
}
