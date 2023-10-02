package testutils

import (
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

func expectPatch(kubeClient *fake.Clientset, expect bool, expectedPatches ...string) {
	kubeClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		patch, ok := action.(testing.PatchAction)
		Expect(ok).To(BeTrue())
		for _, expectedPatch := range expectedPatches {
			if expect {
				Expect(string(patch.GetPatch())).To(ContainSubstring(expectedPatch))
			} else {
				Expect(string(patch.GetPatch())).ToNot(ContainSubstring(expectedPatch))
			}
		}
		return true, nil, nil
	})
}

func ExpectNodePatch(kubeClient *fake.Clientset, expectedPatches ...string) {
	expectPatch(kubeClient, true, expectedPatches...)
}

func DoNotExpectNodePatch(kubeClient *fake.Clientset, expectedPatches ...string) {
	expectPatch(kubeClient, false, expectedPatches...)
}
