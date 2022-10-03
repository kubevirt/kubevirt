package utils

import (
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/vmexport"
)

type AtomicBool struct {
	Lock  *sync.Mutex
	value bool
}

func (b *AtomicBool) IsTrue() bool {
	b.Lock.Lock()
	defer b.Lock.Unlock()
	return b.value
}

func (b *AtomicBool) True() {
	b.Lock.Lock()
	defer b.Lock.Unlock()
	b.value = true
}

func (b *AtomicBool) False() {
	b.Lock.Lock()
	defer b.Lock.Unlock()
	b.value = false
}

func VMExportSpec(name, namespace, kind, resourceName, secretName string) *exportv1.VirtualMachineExport {
	tokenSecretRef := secretName
	vmexport := &exportv1.VirtualMachineExport{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: exportv1.VirtualMachineExportSpec{
			TokenSecretRef: &tokenSecretRef,
			Source: v1.TypedLocalObjectReference{
				APIGroup: &v1.SchemeGroupVersion.Group,
				Kind:     kind,
				Name:     resourceName,
			},
		},
	}

	return vmexport
}

func HandleVMExportGet(client *kubevirtfake.Clientset, vme *exportv1.VirtualMachineExport, vmexportName string) {
	client.Fake.PrependReactor("get", "virtualmachineexports", func(action testing.Action) (bool, runtime.Object, error) {
		get, ok := action.(testing.GetAction)
		Expect(ok).To(BeTrue())
		Expect(get.GetNamespace()).To(Equal(k8smetav1.NamespaceDefault))
		Expect(get.GetName()).To(Equal(vmexportName))
		if vme == nil {
			return true, nil, errors.NewNotFound(v1.Resource("virtualmachineexport"), vmexportName)
		}
		return true, vme, nil
	})
}

func HandleVMExportCreate(client *kubevirtfake.Clientset, vme *exportv1.VirtualMachineExport) {
	client.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		if vme == nil {
			vme, ok = create.GetObject().(*exportv1.VirtualMachineExport)
		} else {
			_, ok = create.GetObject().(*exportv1.VirtualMachineExport)
		}

		Expect(ok).To(BeTrue())
		HandleVMExportGet(client, vme, vme.Name)
		return true, vme, nil
	})
}

func HandleSecretGet(k8sClient *fakek8sclient.Clientset, secretName string) {
	secret := &v1.Secret{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      secretName,
			Namespace: k8smetav1.NamespaceDefault,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			"token": []byte("test"),
		},
	}

	k8sClient.Fake.PrependReactor("get", "secrets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		get, ok := action.(testing.GetAction)
		Expect(ok).To(BeTrue())
		Expect(get.GetNamespace()).To(Equal(k8smetav1.NamespaceDefault))
		Expect(get.GetName()).To(Equal(secretName))
		if secret == nil {
			return true, nil, errors.NewNotFound(v1.Resource("Secret"), secretName)
		}
		return true, secret, nil
	})
}

func GetExportVolumeFormat(url string, format exportv1.ExportVolumeFormat) []exportv1.VirtualMachineExportVolumeFormat {
	return []exportv1.VirtualMachineExportVolumeFormat{
		{
			Format: format,
			Url:    url,
		},
	}
}

func GetVMEStatus(volumes []exportv1.VirtualMachineExportVolume, secretName string) *exportv1.VirtualMachineExportStatus {
	tokenSecretRef := secretName
	// Mock the expected vme status
	return &exportv1.VirtualMachineExportStatus{
		Phase: exportv1.Ready,
		Links: &exportv1.VirtualMachineExportLinks{
			External: &exportv1.VirtualMachineExportLink{
				Volumes: volumes,
			},
		},
		TokenSecretRef: &tokenSecretRef,
	}
}

func WaitExportCompleteDefault(kubecli.KubevirtClient, *vmexport.VMExportInfo, time.Duration, time.Duration) error {
	return nil
}

func WaitExportCompleteError(kubecli.KubevirtClient, *vmexport.VMExportInfo, time.Duration, time.Duration) error {
	return fmt.Errorf("processing failed: Test error")
}
