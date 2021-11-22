package libnet

import (
	"context"
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"kubevirt.io/client-go/kubecli"
)

func AddLabelToNamespace(client kubecli.KubevirtClient, namespace, key, value string) error {
	return patchNamespace(client, namespace, func(ns *v1.Namespace) {
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[key] = value
	})
}

func RemoveLabelFromNamespace(client kubecli.KubevirtClient, namespace, key string) error {
	return patchNamespace(client, namespace, func(ns *v1.Namespace) {
		if ns.Labels == nil {
			return
		}
		delete(ns.Labels, key)
	})
}

func RemoveAllLabelsFromNamespace(client kubecli.KubevirtClient, namespace string) error {
	return patchNamespace(client, namespace, func(ns *v1.Namespace) {
		if ns.Labels == nil {
			return
		}
		ns.Labels = map[string]string{}
	})
}

func patchNamespace(client kubecli.KubevirtClient, namespace string, patchFunc func(*v1.Namespace)) error {
	ns, err := client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	old, err := json.Marshal(ns)
	if err != nil {
		return err
	}

	new := ns.DeepCopy()
	patchFunc(new)

	newJson, err := json.Marshal(new)
	if err != nil {
		return err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, ns)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().Namespaces().Patch(context.Background(), ns.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}
