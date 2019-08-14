package annotator

import (
	"encoding/json"
	"fmt"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// Annotator talks to kubernetes and adds annotations to objects.
type Annotator struct {
	OpClient    operatorclient.ClientInterface
	Annotations map[string]string
}

func NewAnnotator(opClient operatorclient.ClientInterface, annotations map[string]string) *Annotator {
	return &Annotator{
		OpClient:    opClient,
		Annotations: annotations,
	}
}

// AnnotateNamespaces takes a list of namespace names and adds annotations to them
func (a *Annotator) AnnotateNamespaces(namespaceNames []string) error {
	if a.Annotations == nil {
		return nil
	}

	namespaces, err := a.getNamespaces(namespaceNames)
	if err != nil {
		return err
	}

	for _, n := range namespaces {
		if err := a.AnnotateNamespace(&n); err != nil {
			return err
		}
	}

	return nil
}

// CleanNamespaceAnnotations takes a list of namespace names and removes annotations from them
func (a *Annotator) CleanNamespaceAnnotations(namespaceNames []string) error {
	if a.Annotations == nil {
		return nil
	}

	namespaces, err := a.getNamespaces(namespaceNames)
	if err != nil {
		return err
	}

	for _, n := range namespaces {
		if err := a.CleanNamespaceAnnotation(&n); err != nil {
			return err
		}
	}

	return nil
}

// getNamespaces gets the set of Namespace API objects given a list of names
// if NamespaceAll is passed (""), all namespaces will be returned
func (a *Annotator) getNamespaces(namespaceNames []string) (namespaces []corev1.Namespace, err error) {
	if len(namespaceNames) == 1 && namespaceNames[0] == corev1.NamespaceAll {
		namespaceList, err := a.OpClient.KubernetesInterface().CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		namespaces = namespaceList.Items
		return namespaces, nil
	}
	for _, n := range namespaceNames {
		namespace, err := a.OpClient.KubernetesInterface().CoreV1().Namespaces().Get(n, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		namespaces = append(namespaces, *namespace)
	}
	return namespaces, nil
}

func (a *Annotator) AnnotateNamespace(namespace *corev1.Namespace) error {
	originalName := namespace.GetName()
	originalData, err := json.Marshal(namespace)
	if err != nil {
		return err
	}

	if namespace.Annotations == nil {
		namespace.Annotations = map[string]string{}
	}

	for key, value := range a.Annotations {
		if existing, ok := namespace.Annotations[key]; ok && existing != value {
			return fmt.Errorf("attempted to annotate namespace %s with %s:%s, but already annotated by %s:%s", namespace.Name, key, value, key, existing)
		}
		namespace.Annotations[key] = value
	}

	modifiedData, err := json.Marshal(namespace)
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalData, modifiedData, corev1.Namespace{})
	if err != nil {
		return fmt.Errorf("error creating patch for Namespace: %v", err)
	}
	_, err = a.OpClient.KubernetesInterface().CoreV1().Namespaces().Patch(originalName, types.StrategicMergePatchType, patchBytes)
	if err != nil {
		return err
	}
	return nil
}

func (a *Annotator) CleanNamespaceAnnotation(namespace *corev1.Namespace) error {
	originalName := namespace.GetName()
	originalData, err := json.Marshal(namespace)
	if err != nil {
		return err
	}

	if namespace.Annotations == nil {
		namespace.Annotations = map[string]string{}
	}

	annotations := map[string]string{}
	for k, v := range namespace.Annotations {
		annotations[k] = v
	}

	for key, value := range a.Annotations {
		if existing, ok := namespace.Annotations[key]; ok && existing != value {
			return fmt.Errorf("attempted to clean annotation %s:%s from namespace %s, but found unexpected annotation %s:%s", key, value, namespace.Name, key, existing)
		} else if !ok {
			// no namespace key to remove
			return nil
		}
		delete(annotations, key)
	}
	namespace.SetAnnotations(annotations)

	modifiedData, err := json.Marshal(namespace)
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalData, modifiedData, corev1.Namespace{})
	if err != nil {
		return fmt.Errorf("error creating patch for Namespace: %v", err)
	}
	fmt.Println(string(patchBytes))
	_, err = a.OpClient.KubernetesInterface().CoreV1().Namespaces().Patch(originalName, types.StrategicMergePatchType, patchBytes)
	if err != nil {
		return err
	}
	return nil
}
