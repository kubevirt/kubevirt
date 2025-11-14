package libnamespace

import (
	"context"
	"encoding/json"
	"reflect"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
)

func AddLabelToNamespace(client kubernetes.Interface, namespace, key, value string) error {
	return PatchNamespace(client, namespace, func(ns *v1.Namespace) {
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[key] = value
	})
}

func RemoveLabelFromNamespace(client kubernetes.Interface, namespace, key string) error {
	return PatchNamespace(client, namespace, func(ns *v1.Namespace) {
		if ns.Labels == nil {
			return
		}
		delete(ns.Labels, key)
	})
}

func PatchNamespace(client kubernetes.Interface, namespace string, patchFunc func(*v1.Namespace)) error {
	ns, err := client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	newNS := ns.DeepCopy()
	patchFunc(newNS)
	if reflect.DeepEqual(ns, newNS) {
		return nil
	}

	oldJSON, err := json.Marshal(ns)
	if err != nil {
		return err
	}

	newJSON, err := json.Marshal(newNS)
	if err != nil {
		return err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(oldJSON, newJSON, ns)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().Namespaces().Patch(context.Background(), ns.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}
