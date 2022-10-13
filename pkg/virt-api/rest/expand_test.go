package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	kubevirtcore "kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/testutils"
)

type prepareFn func() (string, string, string)

var _ = Describe("Instancetype expansion subresources", func() {
	const (
		vmName                    = "test-vm"
		vmNamespace               = "test-vm-namespace"
		vmInstancetypeName        = "test-instancetype"
		vmPreferenceName          = "test-preference"
		vmClusterInstancetypeName = "test-cluster-instancetype"
		vmClusterPreferenceName   = "test-cluster-preference"
	)

	var (
		vmClient            *kubecli.MockVirtualMachineInterface
		virtClient          *kubecli.MockKubevirtClient
		kubeClient          *k8sfake.Clientset
		sarHandler          func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error)
		instancetypeMethods *testutils.MockInstancetypeMethods
		app                 *SubresourceAPIApp

		request  *restful.Request
		recorder *httptest.ResponseRecorder
		response *restful.Response

		vm *v1.VirtualMachine
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().GeneratedKubeVirtClient().Return(fake.NewSimpleClientset()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(vmNamespace).Return(vmClient).AnyTimes()
		virtClient.EXPECT().AuthorizationV1().Return(kubeClient.AuthorizationV1()).AnyTimes()

		sarHandler = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
			panic("unexpected call to sarHandler")
		}

		kubeClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			sar, ok := create.GetObject().(*authv1.SubjectAccessReview)
			Expect(ok).To(BeTrue())

			sarOut, err := sarHandler(sar)
			return true, sarOut, err
		})

		instancetypeMethods = testutils.NewMockInstancetypeMethods()

		app = NewSubresourceAPIApp(virtClient, 0, nil, nil)
		app.instancetypeMethods = instancetypeMethods

		request = restful.NewRequest(&http.Request{})
		request.Request.URL = &url.URL{}
		request.Request.Header = make(map[string][]string)
		request.Request.Header[userHeader] = []string{"user"}
		request.Request.Header[groupHeader] = []string{"userGroup"}

		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)
		response.SetRequestAccepts(restful.MIME_JSON)

		vm = &v1.VirtualMachine{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmName,
				Namespace: vmNamespace,
			},
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
					},
				},
			},
		}
	})

	sarHandlerFn := func(kind, namespace, name string, allowed bool) func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
		return func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
			Expect(sar.Spec.NonResourceAttributes).To(BeNil())
			Expect(sar.Spec.ResourceAttributes).ToNot(BeNil())
			Expect(sar.Spec.ResourceAttributes.Verb).To(Equal("get"))
			Expect(sar.Spec.ResourceAttributes.Resource).To(Equal(strings.ToLower(kind)))
			Expect(sar.Spec.ResourceAttributes.Name).To(Equal(name))
			if namespace != "" {
				Expect(sar.Spec.ResourceAttributes.Namespace).To(Equal(namespace))
			}
			sar.Status.Allowed = allowed
			sar.Status.Reason = "because I said so"
			return sar, nil
		}
	}

	findEmptyInstancetypeSpecFn := func(_ *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineInstancetypeSpec, error) {
		return &instancetypev1alpha2.VirtualMachineInstancetypeSpec{}, nil
	}
	findEmptyPreferenceSpecFn := func(_ *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachinePreferenceSpec, error) {
		return &instancetypev1alpha2.VirtualMachinePreferenceSpec{}, nil
	}

	testCommonFunctionality := func(callExpandSpecApi func(vm *v1.VirtualMachine) *httptest.ResponseRecorder, expectedStatusError int) {
		It("should return unchanged VM, if no instancetype and preference is assigned", func() {
			vm.Spec.Instancetype = nil
			vm.Spec.Preference = nil

			recorder := callExpandSpecApi(vm)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			responseVm := &v1.VirtualMachine{}
			Expect(json.NewDecoder(recorder.Body).Decode(responseVm)).To(Succeed())
			Expect(responseVm).To(Equal(vm))
		})

		It("should fail if VM points to nonexistent instancetype", func() {
			const name = "nonexistent-instancetype"
			sarHandler = sarHandlerFn(apiinstancetype.ClusterSingularResourceName, "", name, true)

			instancetypeMethods.FindInstancetypeSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineInstancetypeSpec, error) {
				return nil, fmt.Errorf("instancetype does not exist")
			}

			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: name,
			}

			recorder := callExpandSpecApi(vm)
			_ = ExpectStatusErrorWithCode(recorder, expectedStatusError)
		})

		It("should fail if VM points to nonexistent preference", func() {
			const name = "nonexistent-preference"
			sarHandler = sarHandlerFn(apiinstancetype.ClusterSingularPreferenceResourceName, "", name, true)

			instancetypeMethods.FindPreferenceSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachinePreferenceSpec, error) {
				return nil, fmt.Errorf("preference does not exist")
			}

			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: name,
			}

			recorder := callExpandSpecApi(vm)
			_ = ExpectStatusErrorWithCode(recorder, expectedStatusError)
		})

		DescribeTable("should apply instancetype to VM", func(prepare prepareFn) {
			instancetypeMethods.FindInstancetypeSpecFunc = findEmptyInstancetypeSpecFn

			// Allow access to instancetype
			kind, namespace, name := prepare()
			sarHandler = sarHandlerFn(kind, namespace, name, true)

			cpu := &v1.CPU{Cores: 2}
			instancetypeMethods.ApplyToVmiFunc = func(field *k8sfield.Path, instancetypespec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) instancetype.Conflicts {
				vmiSpec.Domain.CPU = cpu
				return nil
			}

			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: name,
				Kind: kind,
			}

			expectedVm := vm.DeepCopy()
			expectedVm.Spec.Template.Spec.Domain.CPU = cpu

			recorder := callExpandSpecApi(vm)
			responseVm := &v1.VirtualMachine{}
			Expect(json.NewDecoder(recorder.Body).Decode(responseVm)).To(Succeed())

			Expect(responseVm.Spec.Template.Spec).To(Equal(expectedVm.Spec.Template.Spec))
		},
			Entry("namespaced", func() (string, string, string) {
				return apiinstancetype.SingularResourceName, vmNamespace, vmInstancetypeName
			}),
			Entry("cluster-wide", func() (string, string, string) {
				return apiinstancetype.ClusterSingularResourceName, "", vmClusterInstancetypeName
			}),
		)

		DescribeTable("should apply preference to VM", func(prepare prepareFn) {
			instancetypeMethods.FindPreferenceSpecFunc = findEmptyPreferenceSpecFn

			// Allow access to preference
			kind, namespace, name := prepare()
			sarHandler = sarHandlerFn(kind, namespace, name, true)

			machineType := "test-machine"
			instancetypeMethods.ApplyToVmiFunc = func(field *k8sfield.Path, instancetypespec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) instancetype.Conflicts {
				vmiSpec.Domain.Machine = &v1.Machine{Type: machineType}
				return nil
			}

			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: name,
				Kind: kind,
			}

			expectedVm := vm.DeepCopy()
			expectedVm.Spec.Template.Spec.Domain.Machine = &v1.Machine{Type: machineType}

			recorder := callExpandSpecApi(vm)
			responseVm := &v1.VirtualMachine{}
			Expect(json.NewDecoder(recorder.Body).Decode(responseVm)).To(Succeed())

			Expect(responseVm.Spec.Template.Spec).To(Equal(expectedVm.Spec.Template.Spec))
		},
			Entry("namespaced", func() (string, string, string) {
				return apiinstancetype.SingularPreferenceResourceName, vmNamespace, vmPreferenceName
			}),
			Entry("cluster-wide", func() (string, string, string) {
				return apiinstancetype.ClusterSingularPreferenceResourceName, "", vmClusterPreferenceName
			}),
		)

		It("should fail, if there is a conflict when applying instancetype", func() {
			instancetypeMethods.FindInstancetypeSpecFunc = findEmptyInstancetypeSpecFn

			sarHandler = sarHandlerFn(apiinstancetype.ClusterSingularResourceName, "", vmClusterInstancetypeName, true)

			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: vmClusterInstancetypeName,
			}

			instancetypeMethods.ApplyToVmiFunc = func(field *k8sfield.Path, instancetypespec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) instancetype.Conflicts {
				return instancetype.Conflicts{k8sfield.NewPath("spec", "template", "spec", "example", "path")}
			}

			recorder := callExpandSpecApi(vm)

			_ = ExpectStatusErrorWithCode(recorder, expectedStatusError)
		})

		DescribeTable("should fail if access is denied to", func(prepare prepareFn) {
			kind, namespace, name := prepare()
			sarHandler = sarHandlerFn(kind, namespace, name, false)

			recorder := callExpandSpecApi(vm)

			_ = ExpectStatusErrorWithCode(recorder, http.StatusForbidden)
		},
			Entry("namespaced instancetype", func() (string, string, string) {
				instancetypeMethods.FindInstancetypeSpecFunc = findEmptyInstancetypeSpecFn
				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: vmInstancetypeName,
					Kind: apiinstancetype.SingularResourceName,
				}
				return apiinstancetype.SingularResourceName, vmNamespace, vmInstancetypeName
			}),
			Entry("cluster-wide instancetype", func() (string, string, string) {
				instancetypeMethods.FindInstancetypeSpecFunc = findEmptyInstancetypeSpecFn
				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: vmClusterInstancetypeName,
					Kind: apiinstancetype.ClusterSingularResourceName,
				}
				return apiinstancetype.ClusterSingularResourceName, "", vmClusterInstancetypeName
			}),
			Entry("namespaced preference", func() (string, string, string) {
				instancetypeMethods.FindPreferenceSpecFunc = findEmptyPreferenceSpecFn
				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: vmPreferenceName,
					Kind: apiinstancetype.SingularPreferenceResourceName,
				}
				return apiinstancetype.SingularPreferenceResourceName, vmNamespace, vmPreferenceName
			}),
			Entry("cluster-wide preference", func() (string, string, string) {
				instancetypeMethods.FindPreferenceSpecFunc = findEmptyPreferenceSpecFn
				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: vmClusterPreferenceName,
					Kind: apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				return apiinstancetype.ClusterSingularPreferenceResourceName, "", vmClusterPreferenceName
			}),
		)
	}

	Context("VirtualMachine expand-spec endpoint", func() {
		callExpandSpecApi := func(vm *v1.VirtualMachine) *httptest.ResponseRecorder {
			request.PathParameters()["name"] = vmName
			request.PathParameters()["namespace"] = vmNamespace

			vmClient.EXPECT().Get(vmName, gomock.Any()).Return(vm, nil).AnyTimes()

			app.ExpandSpecVMRequestHandler(request, response)
			return recorder
		}

		testCommonFunctionality(callExpandSpecApi, http.StatusInternalServerError)

		It("should fail if VM does not exist", func() {
			request.PathParameters()["name"] = "nonexistent-vm"
			request.PathParameters()["namespace"] = vmNamespace

			vmClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, errors.NewNotFound(
				schema.GroupResource{
					Group:    kubevirtcore.GroupName,
					Resource: "VirtualMachine",
				},
				"",
			)).AnyTimes()

			app.ExpandSpecVMRequestHandler(request, response)
			_ = ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
		})
	})

	Context("expand-spec endpoint", func() {
		callExpandSpecApi := func(vm *v1.VirtualMachine) *httptest.ResponseRecorder {
			vmJson, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = ioutil.NopCloser(bytes.NewBuffer(vmJson))

			app.ExpandSpecRequestHandler(request, response)
			return recorder
		}

		testCommonFunctionality(callExpandSpecApi, http.StatusBadRequest)

		It("should fail if received invalid JSON", func() {
			invalidJson := "this is invalid JSON {{{{"
			request.Request.Body = ioutil.NopCloser(strings.NewReader(invalidJson))

			app.ExpandSpecRequestHandler(request, response)
			_ = ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
		})

		It("should fail if received object is not a VirtualMachine", func() {
			notVm := struct {
				StringField string `json:"stringField"`
				IntField    int    `json:"intField"`
			}{
				StringField: "test",
				IntField:    10,
			}

			jsonBytes, err := json.Marshal(notVm)
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = ioutil.NopCloser(bytes.NewBuffer(jsonBytes))

			app.ExpandSpecRequestHandler(request, response)
			_ = ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
		})
	})
})
