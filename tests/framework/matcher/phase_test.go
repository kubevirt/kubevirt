package matcher

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v13 "kubevirt.io/api/core/v1"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/tests/framework/matcher/helper"
)

var _ = Describe("Matcher", func() {

	var toNilPointer *v1.Pod = nil
	var toNilSlicePointer []*v1.Pod = nil

	var runningPod = &v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}

	var stoppedPod = &v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodSucceeded,
		},
	}

	var nameAndKindPod = &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: "testpod",
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Pod",
		},
	}

	var nameAndKindDV = &v1beta1.DataVolume{
		ObjectMeta: v12.ObjectMeta{
			Name: "testdv",
		},
		TypeMeta: v12.TypeMeta{
			Kind: "DataVolume",
		},
	}

	var nameAndKindVMI = &v13.VirtualMachineInstance{
		ObjectMeta: v12.ObjectMeta{
			Name: "testvmi",
		},
		TypeMeta: v12.TypeMeta{
			Kind: "VirtualMachineInstance",
		},
	}

	var onlyKindPod = &v1.Pod{
		TypeMeta: v12.TypeMeta{
			Kind: "Pod",
		},
	}

	var onlyKindDV = &v1beta1.DataVolume{
		TypeMeta: v12.TypeMeta{
			Kind: "DataVolume",
		},
	}

	var onlyKindVMI = &v13.VirtualMachineInstance{
		TypeMeta: v12.TypeMeta{
			Kind: "VirtualMachineInstance",
		},
	}

	var onlyNamePod = &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: "testpod",
		},
	}

	var onlyNameDV = &v1beta1.DataVolume{
		ObjectMeta: v12.ObjectMeta{
			Name: "testdv",
		},
	}

	var onlyNameVMI = &v13.VirtualMachineInstance{
		ObjectMeta: v12.ObjectMeta{
			Name: "testvmi",
		},
	}

	table.DescribeTable("should work on a pod", func(exptectedPhase interface{}, pod interface{}, match bool) {
		success, err := BeInPhase(exptectedPhase).Match(pod)
		Expect(err).ToNot(HaveOccurred())
		Expect(success).To(Equal(match))
		Expect(BeInPhase(exptectedPhase).FailureMessage(pod)).ToNot(BeEmpty())
		Expect(BeInPhase(exptectedPhase).NegatedFailureMessage(pod)).ToNot(BeEmpty())
	},
		table.Entry("with expected phase as PodPhase match the pod phase", v1.PodRunning, runningPod, true),
		table.Entry("with expected phase as string match the pod phase", "Running", runningPod, true),
		table.Entry("cope with a nil pod", v1.PodRunning, nil, false),
		table.Entry("cope with an object pointing to nil", v1.PodRunning, toNilPointer, false),
		table.Entry("cope with an object which has no phase", v1.PodRunning, &v1.Service{}, false),
		table.Entry("cope with a non-stringable object as expected phase", nil, runningPod, false),
		table.Entry("with expected phase not match the pod phase", "Succeeded", runningPod, false),
	)

	table.DescribeTable("should work on a pod array", func(exptectedPhase interface{}, array interface{}, match bool) {
		success, err := BeInPhase(exptectedPhase).Match(array)
		Expect(err).ToNot(HaveOccurred())
		Expect(success).To(Equal(match))
		Expect(BeInPhase(exptectedPhase).FailureMessage(array)).ToNot(BeEmpty())
		Expect(BeInPhase(exptectedPhase).NegatedFailureMessage(array)).ToNot(BeEmpty())
	},
		table.Entry("with expected phase as PodPhase match the pod phase", v1.PodRunning, []*v1.Pod{runningPod}, true),
		table.Entry("with expected phase as PodPhase match the pod phase when not a pointer", v1.PodRunning, []v1.Pod{*runningPod}, true),
		table.Entry("with expected phase as string match the pod phase", "Running", []*v1.Pod{runningPod, runningPod}, true),
		table.Entry("with not all pods matching the expected phase", "Running", []*v1.Pod{runningPod, stoppedPod, runningPod}, false),
		table.Entry("cope with a nil array", v1.PodRunning, nil, false),
		table.Entry("cope with a nil array pointer", v1.PodRunning, toNilSlicePointer, false),
		table.Entry("cope with a nil entry", v1.PodRunning, []*v1.Pod{nil}, false),
		table.Entry("cope with an empty array", v1.PodRunning, []*v1.Pod{}, false),
		table.Entry("cope with an object which has no phase", v1.PodRunning, []*v1.Service{{}}, false),
		table.Entry("cope with a non-stringable object as expected phase", nil, []*v1.Pod{runningPod}, false),
		table.Entry("with expected phase not match the pod phase", "Succeeded", []*v1.Pod{runningPod}, false),
	)

	table.DescribeTable("should print kind and name of the object depending on fields", func(object interface{}, kind string, name string) {
		unstructured, err := helper.ToUnstructured(object)
		Expect(err).ToNot(HaveOccurred())
		Expect(unstructured.GetKind()).To(Equal(kind))
		Expect(unstructured.GetName()).To(Equal(name))
		if kind != "" && name != "" {
			Expect(BeInPhase("testPhase").FailureMessage(object)).Should(HavePrefix(fmt.Sprintf("%s/%s", unstructured.GetKind(), unstructured.GetName())))
			Expect(BeInPhase("testPhase").NegatedFailureMessage(object)).Should(HavePrefix(fmt.Sprintf("%s/%s", unstructured.GetKind(), unstructured.GetName())))
		} else if kind != "" {
			Expect(BeInPhase("testPhase").FailureMessage(object)).Should(HavePrefix(fmt.Sprintf("%s/", unstructured.GetKind())))
			Expect(BeInPhase("testPhase").NegatedFailureMessage(object)).Should(HavePrefix(fmt.Sprintf("%s/", unstructured.GetKind())))
		} else if name != "" {
			Expect(BeInPhase("testPhase").FailureMessage(object)).Should(HavePrefix(fmt.Sprintf("%s", unstructured.GetName())))
			Expect(BeInPhase("testPhase").NegatedFailureMessage(object)).Should(HavePrefix(fmt.Sprintf("%s", unstructured.GetName())))
		} else {
			Expect(BeInPhase("testPhase").FailureMessage(object)).ShouldNot(HavePrefix(fmt.Sprintf("%s/", unstructured.GetKind())))
			Expect(BeInPhase("testPhase").NegatedFailureMessage(object)).ShouldNot(HavePrefix(fmt.Sprintf("%s/", unstructured.GetKind())))
			Expect(BeInPhase("testPhase").FailureMessage(object)).Should(HavePrefix(" expected"))
			Expect(BeInPhase("testPhase").NegatedFailureMessage(object)).Should(HavePrefix(" expected"))
		}

	},
		table.Entry("with a Pod having name and kind", nameAndKindPod, nameAndKindPod.Kind, nameAndKindPod.Name),
		table.Entry("with a DataVolume having name and kind", nameAndKindDV, nameAndKindDV.Kind, nameAndKindDV.Name),
		table.Entry("with a VirtualMachineInstance having name and kind", nameAndKindVMI, nameAndKindVMI.Kind, nameAndKindVMI.Name),
		table.Entry("with a Pod having only kind", onlyKindPod, onlyKindPod.Kind, onlyKindPod.Name),
		table.Entry("with a DataVolume having only kind", onlyKindDV, onlyKindDV.Kind, onlyKindDV.Name),
		table.Entry("with a VirtualMachineInstance having only kind", onlyKindVMI, onlyKindVMI.Kind, onlyKindVMI.Name),
		table.Entry("with a Pod having only name", onlyNamePod, onlyNamePod.Kind, onlyNamePod.Name),
		table.Entry("with a DataVolume having only name", onlyNameDV, onlyNameDV.Kind, onlyNameDV.Name),
		table.Entry("with a VirtualMachineInstance having only name", onlyNameVMI, onlyNameVMI.Kind, onlyNameVMI.Name),
		table.Entry("with a Pod having no kind and name", &v1.Pod{}, "", ""),
		table.Entry("with a DataVolume having no kind and name", &v1beta1.DataVolume{}, "", ""),
		table.Entry("with a VirtualMachineInstance having no kind and name", &v13.VirtualMachineInstance{}, "", ""),
	)
})
