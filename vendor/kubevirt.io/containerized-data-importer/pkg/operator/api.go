/*
Copyright 2018 The CDI Authors.

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

package operator

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/containerized-data-importer/pkg/operator/controller"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

// SetOwner makes the current "active" CDI CR the owner of the object
func SetOwner(client kubernetes.Interface, object metav1.Object) error {
	namespace := util.GetNamespace()
	configMap, err := client.CoreV1().ConfigMaps(namespace).Get(controller.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	configMapOwner := getController(configMap.GetOwnerReferences())

	if configMapOwner == nil {
		return fmt.Errorf("Configmap has no owner")
	}

	for _, o := range object.GetOwnerReferences() {
		if o.Controller != nil && *o.Controller {
			if o.UID == configMapOwner.UID {
				// already set to current obj
				return nil
			}

			return fmt.Errorf("Object %+v already owned by %+v", object, o)
		}
	}

	object.SetOwnerReferences(append(object.GetOwnerReferences(), *configMapOwner))

	return nil
}

func getController(owners []metav1.OwnerReference) *metav1.OwnerReference {
	for _, owner := range owners {
		if owner.Controller != nil && *owner.Controller {
			return &owner
		}
	}
	return nil
}
