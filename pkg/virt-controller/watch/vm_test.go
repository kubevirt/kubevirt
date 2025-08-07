package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/api"
	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	instancetypeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubeclifake "kubevirt.io/client-go/kubecli/fake"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"

	gomegatypes "github.com/onsi/gomega/types"
)

var (
	vmUID      types.UID = "vm-uid"
	t                    = true
	stage                = v1.VMRolloutStrategyStage
	liveUpdate           = v1.VMRolloutStrategyLiveUpdate
)

var _ = Describe("VirtualMachine", func() {

	Context("One valid VirtualMachine controller given", func() {

		var controller *VMController
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		var cdiClient *cdifake.Clientset
		var k8sClient *k8sfake.Clientset
		var virtClient *kubecli.MockKubevirtClient
		var config *virtconfig.ClusterConfig
		var kvInformer cache.SharedIndexInformer
		var virtFakeClient *fake.Clientset

		BeforeEach(func() {
			virtClient = kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
			virtFakeClient = fake.NewSimpleClientset()
			// enable /status, this assumes that no other reactor will be prepend.
			// if you need to prepend reactor it need to not handle the object or use the
			// modify function
			virtFakeClient.PrependReactor("update", "virtualmachines",
				UpdateReactor(SubresourceHandle, virtFakeClient.Tracker(), ModifyStatusOnlyVM))
			virtFakeClient.PrependReactor("update", "virtualmachines",
				UpdateReactor(Handle, virtFakeClient.Tracker(), ModifyVM))

			virtFakeClient.PrependReactor("patch", "virtualmachines",
				PatchReactor(SubresourceHandle, virtFakeClient.Tracker(), ModifyStatusOnlyVM))
			virtFakeClient.PrependReactor("patch", "virtualmachines",
				PatchReactor(Handle, virtFakeClient.Tracker(), ModifyVM))

			dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
			dataSourceInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
			vmiInformer, _ := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachineInstance{}, virtcontroller.GetVMIInformerIndexers())
			vmInformer, _ := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachine{}, virtcontroller.GetVirtualMachineInformerIndexers())
			pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
			namespaceInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})
			ns1 := &k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
				},
			}
			ns2 := &k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			}
			Expect(namespaceInformer.GetStore().Add(ns1)).To(Succeed())
			Expect(namespaceInformer.GetStore().Add(ns2)).To(Succeed())

			crInformer, _ := testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, cache.Indexers{
				"vm": func(obj interface{}) ([]string, error) {
					cr := obj.(*appsv1.ControllerRevision)
					for _, ref := range cr.OwnerReferences {
						if ref.Kind == "VirtualMachine" {
							return []string{string(ref.UID)}, nil
						}
					}
					return nil, nil
				},
			})
			podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})

			instancetypeMethods := testutils.NewMockInstancetypeMethods()

			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true

			config, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

			controller, _ = NewVMController(vmiInformer,
				vmInformer,
				dataVolumeInformer,
				dataSourceInformer,
				namespaceInformer.GetStore(),
				pvcInformer,
				crInformer,
				podInformer,
				instancetypeMethods,
				recorder,
				virtClient,
				config)

			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue

			// Set up mock client
			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
				virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault),
			).AnyTimes()

			virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
				virtFakeClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault),
			).AnyTimes()
			// TODO Remove GeneratedKubeVirtClient
			virtClient.EXPECT().GeneratedKubeVirtClient().Return(virtFakeClient).AnyTimes()

			cdiClient = cdifake.NewSimpleClientset()
			virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
			cdiClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
			virtClient.EXPECT().AuthorizationV1().Return(k8sClient.AuthorizationV1()).AnyTimes()
		})

		// TODO: We need to make sure the action was triggered
		shouldExpectVMIFinalizerRemoval := func() {
			virtFakeClient.PrependReactor("update", "virtualmachineinstance", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				switch action := action.(type) {
				case testing.PatchActionImpl:
					patch := fmt.Sprintf(`[{ "op": "test", "path": "/metadata/finalizers", "value": ["%s"] }, { "op": "replace", "path": "/metadata/finalizers", "value": [] }]`, v1.VirtualMachineControllerFinalizer)
					Expect(action.Patch).To(Equal([]byte(patch)))
				default:
					Fail("Expected to see patch")
				}
				return false, nil, nil
			})

		}

		shouldExpectDataVolumeCreationPriorityClass := func(uid types.UID, labels map[string]string, annotations map[string]string, priorityClassName string, idx *int) {
			allAnnotations := map[string]string{
				"cdi.kubevirt.io/allowClaimAdoption": "true",
			}
			for k, v := range annotations {
				allAnnotations[k] = v
			}
			cdiClient.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				*idx++
				dataVolume := update.GetObject().(*cdiv1.DataVolume)
				Expect(dataVolume.ObjectMeta.OwnerReferences[0].UID).To(Equal(uid))
				Expect(dataVolume.ObjectMeta.Labels).To(Equal(labels))
				Expect(dataVolume.ObjectMeta.Annotations).To(Equal(allAnnotations))
				Expect(dataVolume.Spec.PriorityClassName).To(Equal(priorityClassName))
				return true, update.GetObject(), nil
			})
		}

		shouldExpectDataVolumeCreation := func(uid types.UID, labels map[string]string, annotations map[string]string, idx *int) {
			shouldExpectDataVolumeCreationPriorityClass(uid, labels, annotations, "", idx)
		}

		shouldFailDataVolumeCreationNoResourceFound := func() {
			cdiClient.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("the server could not find the requested resource (post datavolumes.cdi.kubevirt.io)")
			})
		}

		shouldFailDataVolumeCreationClaimAlreadyExists := func() {
			cdiClient.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("PVC already exists")
			})
		}

		shouldExpectDataVolumeDeletion := func(idx *int) {
			cdiClient.Fake.PrependReactor("delete", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())

				*idx++
				return true, nil, nil
			})
		}

		expectControllerRevisionList := func(vmRevision *appsv1.ControllerRevision) {
			k8sClient.Fake.PrependReactor("list", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				list, _ := action.(testing.ListAction)
				if strings.Contains(list.GetListRestrictions().Labels.String(), string(vmUID)) {
					return true, &appsv1.ControllerRevisionList{Items: []appsv1.ControllerRevision{*vmRevision}}, nil
				}
				return true, &appsv1.ControllerRevisionList{}, nil
			})
		}

		expectControllerRevisionDelete := func(vmRevision *appsv1.ControllerRevision) {
			k8sClient.Fake.PrependReactor("delete", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				deleted, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())
				Expect(deleted.GetNamespace()).To(Equal(vmRevision.Namespace))
				Expect(deleted.GetName()).To(Equal(vmRevision.Name))
				return true, nil, nil
			})
		}

		expectControllerRevisionCreation := func(vmRevision *appsv1.ControllerRevision) {
			k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				created, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())

				createObj := created.GetObject().(*appsv1.ControllerRevision)

				return createObj == vmRevision, created.GetObject(), nil
			})
		}

		patchVMRevision := func(vm *v1.VirtualMachine) runtime.RawExtension {
			vmBytes, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			var raw map[string]interface{}
			err = json.Unmarshal(vmBytes, &raw)
			Expect(err).ToNot(HaveOccurred())

			objCopy := make(map[string]interface{})
			spec := raw["spec"].(map[string]interface{})
			objCopy["spec"] = spec
			patch, err := json.Marshal(objCopy)
			Expect(err).ToNot(HaveOccurred())
			return runtime.RawExtension{Raw: patch}
		}

		createVMRevision := func(vm *v1.VirtualMachine) *appsv1.ControllerRevision {
			return &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      getVMRevisionName(vm.UID, vm.Generation),
					Namespace: vm.Namespace,
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion:         v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
						Kind:               v1.VirtualMachineGroupVersionKind.Kind,
						Name:               vm.ObjectMeta.Name,
						UID:                vm.ObjectMeta.UID,
						Controller:         &t,
						BlockOwnerDeletion: &t,
					}},
				},
				Data:     patchVMRevision(vm),
				Revision: vm.Generation,
			}
		}

		addVirtualMachine := func(vm *v1.VirtualMachine) {
			key, err := virtcontroller.KeyFunc(vm)
			Expect(err).To(Not(HaveOccurred()))
			controller.Queue.Add(key)
			Expect(controller.vmIndexer.Add(vm)).To(Succeed())
		}

		sanityExecute := func(vm *v1.VirtualMachine) {
			added := vm.DeepCopy()
			controller.Execute()
			Expect(equality.Semantic.DeepEqual(vm, added)).To(BeTrue(), "A cached VM was modified")
		}

		It("should update conditions when failed creating DataVolume for virtualMachineInstance", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			shouldFailDataVolumeCreationNoResourceFound()

			sanityExecute(vm)
			testutils.ExpectEvent(recorder, FailedDataVolumeCreateReason)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())
			cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
			Expect(cond).To(Not(BeNil()))
			Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":   Equal(v1.VirtualMachineFailure),
				"Reason": Equal("FailedCreate"),
				"Message": And(
					ContainSubstring("Error encountered while creating DataVolumes: failed to create DataVolume"),
					ContainSubstring("the server could not find the requested resource (post datavolumes.cdi.kubevirt.io)"),
				),
			}))
		})

		It("should create missing DataVolume for VirtualMachineInstance", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"my": "label"},
					Annotations: map[string]string{"my": "annotation"},
					Name:        "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})

			vm.Status.PrintableStatus = v1.VirtualMachineStatusStopped
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			existingDataVolume, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume.Namespace = "default"
			controller.dataVolumeStore.Add(existingDataVolume)

			createCount := 0
			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID), "my": "label"}, map[string]string{"my": "annotation"}, &createCount)

			sanityExecute(vm)
			Expect(createCount).To(Equal(1))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())
			Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusProvisioning))

		})
		Context("Disk un/hotplug", func() {
			addVolumeReactor := func(virtFakeClient *fake.Clientset) {
				virtFakeClient.PrependReactor("put", "virtualmachineinstances/addvolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
					switch action := action.(type) {
					case kubeclifake.PutAction[*v1.AddVolumeOptions]:
						obj, err := virtFakeClient.Tracker().Get(action.GetResource(), action.GetNamespace(), action.GetName())
						Expect(err).NotTo(HaveOccurred())
						vmi := obj.(*v1.VirtualMachineInstance)
						option := action.GetOptions()
						spec := virtcontroller.ApplyVolumeRequestOnVMISpec(&vmi.Spec, &v1.VirtualMachineVolumeRequest{
							AddVolumeOptions: option,
						})

						vmi.Spec = *spec
						return true, nil, virtFakeClient.Tracker().Update(action.GetResource(), vmi, action.GetNamespace())
					default:
						Fail("unexpected action type on addvolume")
						return false, nil, nil
					}

				})
			}

			removeVolumeReactor := func(virtFakeClient *fake.Clientset) {
				virtFakeClient.PrependReactor("put", "virtualmachineinstances/removevolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
					switch action := action.(type) {
					case kubeclifake.PutAction[*v1.RemoveVolumeOptions]:
						obj, err := virtFakeClient.Tracker().Get(action.GetResource(), action.GetNamespace(), action.GetName())
						Expect(err).NotTo(HaveOccurred())
						vmi := obj.(*v1.VirtualMachineInstance)
						option := action.GetOptions()
						spec := virtcontroller.ApplyVolumeRequestOnVMISpec(&vmi.Spec, &v1.VirtualMachineVolumeRequest{
							RemoveVolumeOptions: option,
						})

						vmi.Spec = *spec
						return true, nil, virtFakeClient.Tracker().Update(action.GetResource(), vmi, action.GetNamespace())
					default:
						Fail("unexpected action type on removevolume")
						return false, nil, nil
					}

				})
			}

			addVolumeFailureReactor := func(virtFakeClient *fake.Clientset) {
				virtFakeClient.PrependReactor("put", "virtualmachineinstances/addvolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
					switch action := action.(type) {
					case kubeclifake.PutAction[*v1.AddVolumeOptions]:
						_, err := virtFakeClient.Tracker().Get(action.GetResource(), action.GetNamespace(), action.GetName())
						Expect(err).NotTo(HaveOccurred())
						return true, nil, fmt.Errorf("conflict")
					default:
						Fail("unexpected action type on addvolume")
						return false, nil, nil
					}
				})
			}

			removeVolumeFailureReactor := func(virtFakeClient *fake.Clientset) {
				virtFakeClient.PrependReactor("put", "virtualmachineinstances/removevolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
					switch action := action.(type) {
					case kubeclifake.PutAction[*v1.RemoveVolumeOptions]:
						_, err := virtFakeClient.Tracker().Get(action.GetResource(), action.GetNamespace(), action.GetName())
						Expect(err).NotTo(HaveOccurred())
						return true, nil, fmt.Errorf("conflict")
					default:
						Fail("unexpected action type on removevolume")
						return false, nil, nil
					}

				})
			}

			DescribeTable("should hotplug a vm", func(isRunning bool) {

				vm, vmi := DefaultVirtualMachine(isRunning)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
					{
						AddVolumeOptions: &v1.AddVolumeOptions{
							Name:         "vol1",
							Disk:         &v1.Disk{},
							VolumeSource: &v1.HotplugVolumeSource{},
						},
					},
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				if isRunning {
					markAsReady(vmi)
					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					addVolumeReactor(virtFakeClient)
				}

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal("vol1"))
				Expect(vm.Status.VolumeRequests).To(BeEmpty())

				if isRunning {
					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Volumes[0].Name).To(Equal("vol1"))
				}
			},

				Entry("that is running", true),
				Entry("that is not running", false),
			)

			Context("add volume request", func() {
				It("should not be cleared when hotplug fails on Running VM", func() {
					By("Running VM")
					vm, vmi := DefaultVirtualMachine(true)
					vm.Status.Created = true
					vm.Status.Ready = true
					vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
						{
							AddVolumeOptions: &v1.AddVolumeOptions{
								Name:         "vol1",
								Disk:         &v1.Disk{},
								VolumeSource: &v1.HotplugVolumeSource{},
							},
						},
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					markAsReady(vmi)
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					By("Simulate a failure in hotplug")
					addVolumeFailureReactor(virtFakeClient)

					By("Not expecting to see a status update")
					sanityExecute(vm)

					By("Expecting to not clear volume requests")
					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.VolumeRequests).To(HaveLen(1), "Volume request should stay, otherwise hotplug will not retry")
				})

				It("should not be cleared when VM update fails on Running VM", func() {
					By("Running VM")
					vm, vmi := DefaultVirtualMachine(true)
					vm.Status.Created = true
					vm.Status.Ready = true
					vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
						{
							AddVolumeOptions: &v1.AddVolumeOptions{
								Name:         "vol1",
								Disk:         &v1.Disk{},
								VolumeSource: &v1.HotplugVolumeSource{},
							},
						},
					}
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					markAsReady(vmi)
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					By("Simulate hotplug")
					addVolumeReactor(virtFakeClient)

					By("Simulate update failure")
					failVMSpecUpdate(virtFakeClient)

					sanityExecute(vm)

					By("Expecting Volume request not to be cleared")
					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.VolumeRequests).To(HaveLen(1), "Volume request should stay, otherwise hotplug will not retry")

					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Volumes[0].Name).To(Equal("vol1"))
				})

				It("should be cleared when hotplug on Stopped VM", func() {
					By("Stopped VM")
					vm, _ := DefaultVirtualMachine(false)
					vm.Status.Created = false
					vm.Status.Ready = false
					vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
						{
							AddVolumeOptions: &v1.AddVolumeOptions{
								Name:         "vol1",
								Disk:         &v1.Disk{},
								VolumeSource: &v1.HotplugVolumeSource{},
							},
						},
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					By("Expecting Volume to be added to VM")
					Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal("vol1"))
					By("Expecting Volume request to be cleared")
					Expect(vm.Status.VolumeRequests).To(BeEmpty(),
						"Volume request should stay, otherwise hotplug will not retry",
					)
				})

				It("should not be cleared when hotplug fails on Stopped VM", func() {
					By("Stopped VM")
					vm, _ := DefaultVirtualMachine(false)
					vm.Status.Created = false
					vm.Status.Ready = false
					vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
						{
							AddVolumeOptions: &v1.AddVolumeOptions{
								Name:         "vol1",
								Disk:         &v1.Disk{},
								VolumeSource: &v1.HotplugVolumeSource{},
							},
						},
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					By("Simulate a conflict on Update")
					failVMSpecUpdate(virtFakeClient)

					sanityExecute(vm)
					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
					By("Expecting Volume request to not be cleared")
					Expect(vm.Status.VolumeRequests).To(HaveLen(1),
						"Volume request should stay, otherwise hotplug will not retry",
					)
				})
			})

			Context("remove volume request", func() {
				It("should not be cleared when hotplug fails on Running VM", func() {
					By("Running VM with a disk and matching volume")
					vm, vmi := DefaultVirtualMachine(true)
					vm.Status.Created = true
					vm.Status.Ready = true
					volume := v1.Volume{
						Name: "vol1",
						VolumeSource: v1.VolumeSource{
							ContainerDisk: &v1.ContainerDiskSource{
								Image: "random",
							},
						},
					}
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, volume)
					vmi.Spec.Volumes = append(vmi.Spec.Volumes, volume)

					disk := v1.Disk{
						Name: "vol1",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{},
						},
					}
					vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, disk)
					vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)

					vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
						{
							RemoveVolumeOptions: &v1.RemoveVolumeOptions{
								Name: "vol1",
							},
						},
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					markAsReady(vmi)
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					By("Simulate a un-hotplug failure")
					removeVolumeFailureReactor(virtFakeClient)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					By("Expecting Volume request to not be cleared")
					Expect(vm.Status.VolumeRequests).To(HaveLen(1),
						"Volume request should stay, otherwise hotplug will not retry",
					)
				})

				It("should be cleared when hotplug on Stopped VM", func() {
					By("Stopped VM with a disk and matching volume")
					vm, _ := DefaultVirtualMachine(false)
					vm.Status.Created = false
					vm.Status.Ready = false

					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes,
						v1.Volume{
							Name: "vol1",
							VolumeSource: v1.VolumeSource{
								ContainerDisk: &v1.ContainerDiskSource{
									Image: "random",
								},
							},
						},
					)
					vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks,
						v1.Disk{
							Name: "vol1",
							DiskDevice: v1.DiskDevice{
								Disk: &v1.DiskTarget{},
							},
						},
					)
					vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
						{
							RemoveVolumeOptions: &v1.RemoveVolumeOptions{
								Name: "vol1",
							},
						},
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					By("Expect the volume to be cleared")
					Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
					By("Expecting Volume request to not be cleared")
					Expect(vm.Status.VolumeRequests).To(BeEmpty())
				})

				It("should not be cleared when hotplug fails on Stopped VM", func() {
					By("Stopped VM with a disk and matching volume")
					vm, _ := DefaultVirtualMachine(false)
					vm.Status.Created = false
					vm.Status.Ready = false
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes,
						v1.Volume{
							Name: "vol1",
							VolumeSource: v1.VolumeSource{
								ContainerDisk: &v1.ContainerDiskSource{
									Image: "random",
								},
							},
						},
					)
					vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks,
						v1.Disk{
							Name: "vol1",
							DiskDevice: v1.DiskDevice{
								Disk: &v1.DiskTarget{},
							},
						},
					)
					vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
						{
							RemoveVolumeOptions: &v1.RemoveVolumeOptions{
								Name: "vol1",
							},
						},
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					By("Simulate a conflict on VM update")
					failVMSpecUpdate(virtFakeClient)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					By("Expecting a Volume request to not be cleared")
					Expect(vm.Status.VolumeRequests).To(HaveLen(1),
						"Volume request should stay, otherwise hotplug will not retry",
					)
				})
			})

			DescribeTable("should unhotplug a vm", func(isRunning bool) {
				vm, vmi := DefaultVirtualMachine(isRunning)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
					{
						RemoveVolumeOptions: &v1.RemoveVolumeOptions{
							Name: "vol1",
						},
					},
				}
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "vol1",
				})
				vm.Spec.Template.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "vol1",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim",
						}},
					},
				})

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				if isRunning {
					vmi.Spec.Volumes = vm.Spec.Template.Spec.Volumes
					vmi.Spec.Domain.Devices.Disks = vm.Spec.Template.Spec.Domain.Devices.Disks
					markAsReady(vmi)
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					removeVolumeReactor(virtFakeClient)
				}

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.VolumeRequests).To(BeEmpty())
				Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
			},

				Entry("that is running", true),
				Entry("that is not running", false),
			)

			DescribeTable("should clear VolumeRequests for added volumes that are satisfied", func(isRunning bool) {
				vm, vmi := DefaultVirtualMachine(isRunning)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
					{
						AddVolumeOptions: &v1.AddVolumeOptions{
							Name:         "vol1",
							Disk:         &v1.Disk{},
							VolumeSource: &v1.HotplugVolumeSource{},
						},
					},
				}
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "vol1",
				})
				vm.Spec.Template.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "vol1",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim",
						}},
					},
				})

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				if isRunning {
					vmi.Spec.Volumes = vm.Spec.Template.Spec.Volumes
					vmi.Spec.Domain.Devices.Disks = vm.Spec.Template.Spec.Domain.Devices.Disks
					markAsReady(vmi)
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					virtFakeClient.PrependReactor("put", "virtualmachineinstances/addvolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
						Fail("add volume should not be called")
						return false, nil, nil
					})
				}

				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.VolumeRequests).To(BeEmpty())
			},

				Entry("that is running", true),
				Entry("that is not running", false),
			)

			DescribeTable("should clear VolumeRequests for removed volumes that are satisfied", func(isRunning bool) {
				vm, vmi := DefaultVirtualMachine(isRunning)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
					{
						RemoveVolumeOptions: &v1.RemoveVolumeOptions{
							Name: "vol1",
						},
					},
				}
				vm.Spec.Template.Spec.Volumes = []v1.Volume{}
				vm.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				if isRunning {
					markAsReady(vmi)
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					virtFakeClient.PrependReactor("put", "virtualmachineinstances/removevolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
						Fail("remove volume should not be called")
						return false, nil, nil
					})
				}

				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.VolumeRequests).To(BeEmpty())
			},

				Entry("that is running", true),
				Entry("that is not running", false),
			)
		})

		It("should not delete failed DataVolume for VirtualMachineInstance", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			existingDataVolume1, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed

			existingDataVolume2, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded

			controller.dataVolumeStore.Add(existingDataVolume1)
			controller.dataVolumeStore.Add(existingDataVolume2)

			deletionCount := 0

			shouldExpectDataVolumeDeletion(&deletionCount)

			sanityExecute(vm)

			Expect(deletionCount).To(Equal(0))
			testutils.ExpectEvent(recorder, virtcontroller.FailedDataVolumeImportReason)
		})

		It("should not delete failed DataVolume for VirtualMachineInstance unless deletion timestamp expires ", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			existingDataVolume1, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed

			existingDataVolume2, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded

			controller.dataVolumeStore.Add(existingDataVolume1)
			controller.dataVolumeStore.Add(existingDataVolume2)

			sanityExecute(vm)

			testutils.ExpectEvent(recorder, virtcontroller.FailedDataVolumeImportReason)
		})

		It("should handle failed DataVolume without Annotations", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			existingDataVolume1, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed
			// explicitly delete the annotations field
			existingDataVolume1.Annotations = nil

			existingDataVolume2, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded
			existingDataVolume2.Annotations = nil

			controller.dataVolumeStore.Add(existingDataVolume1)
			controller.dataVolumeStore.Add(existingDataVolume2)

			sanityExecute(vm)

			testutils.ExpectEvent(recorder, virtcontroller.FailedDataVolumeImportReason)
		})

		DescribeTable("should start VMI once DataVolumes are complete", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.Running = nil
			vm.Spec.RunStrategy = &runStrategy

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())

			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID)}, nil, pointer.P(0))

			addVirtualMachine(vm)
			sanityExecute(vm)
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))

			dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
			dv.Namespace = "default"
			dv.Status.Phase = cdiv1.Succeeded

			controller.dataVolumeStore.Add(dv)
			controller.addDataVolume(dv)
			sanityExecute(vm)

			Expect(virtFakeClient.Actions()).To(WithTransform(func(actions []testing.Action) []testing.Action {
				var patchActions []testing.Action
				for _, action := range actions {
					if action.GetVerb() == "update" &&
						action.GetResource().Resource == "virtualmachines" &&
						action.GetSubresource() == "status" {
						patchActions = append(patchActions, action)
					}
				}
				return patchActions
			}, Not(BeEmpty())))

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Status.Created).To(BeFalse())
			Expect(vm.Status.Ready).To(BeFalse())

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		},
			Entry("with runStrategy set to Always", v1.RunStrategyAlways),
			Entry("with runStrategy set to Once", v1.RunStrategyOnce),
			Entry("with runStrategy set to RerunOnFailure", v1.RunStrategyRerunOnFailure),
		)

		It("should start VMI once DataVolumes (not templates) are complete", func() {

			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			dvt := v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			}

			existingDataVolume, _ := watchutil.CreateDataVolumeManifest(virtClient, dvt, vm)

			existingDataVolume.OwnerReferences = nil
			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.Succeeded

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			controller.dataVolumeStore.Add(existingDataVolume)

			sanityExecute(vm)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())
			// TODO expect update status is called
			Expect(vm.Status.Created).To(BeFalse())
			Expect(vm.Status.Ready).To(BeFalse())

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should start VMI once DataVolumes are complete or WaitForFirstConsumer", func() {
			// WaitForFirstConsumer state can only be handled by VMI

			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})

			existingDataVolume, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.WaitForFirstConsumer

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			controller.dataVolumeStore.Add(existingDataVolume)

			sanityExecute(vm)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())
			// TODO // expect update status is called
			Expect(vm.Status.Created).To(BeFalse())
			Expect(vm.Status.Ready).To(BeFalse())

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should Not delete Datavolumes when VMI is stopped", func() {
			vm, vmi := DefaultVirtualMachine(false)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})

			existingDataVolume, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.Succeeded

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())

			addVirtualMachine(vm)

			controller.dataVolumeStore.Add(existingDataVolume)

			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)
			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		DescribeTable("should create multiple DataVolumes for VirtualMachineInstance when running", func(running bool, anno string, shouldCreate bool) {
			vm, _ := DefaultVirtualMachine(running)
			if anno != "" {
				if vm.Annotations == nil {
					vm.Annotations = make(map[string]string)
				}
				vm.Annotations[v1.ImmediateDataVolumeCreation] = anno
			}
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})

			vm.Status.PrintableStatus = v1.VirtualMachineStatusStopped
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			createCount := 0
			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID)}, map[string]string{}, &createCount)

			sanityExecute(vm)
			if shouldCreate {
				Expect(createCount).To(Equal(2))
				testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
			} else {
				Expect(createCount).To(Equal(0))
			}
		},
			Entry("when VM is running no annotation", true, "", true),
			Entry("when VM stopped no annotation", false, "", true),
			Entry("when VM is running annotation true", true, "true", true),
			Entry("when VM stopped annotation true", false, "true", true),
			Entry("when VM is running annotation false", true, "false", true),
			Entry("when VM stopped annotation false", false, "false", false),
		)

		DescribeTable("should properly handle PVC existing before DV created", func(annotations map[string]string, expectedCreations int, initFunc func()) {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})

			vm.Status.PrintableStatus = v1.VirtualMachineStatusStopped
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			pvc := k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "dv1",
					Namespace:   vm.Namespace,
					Annotations: annotations,
				},
				Status: k8sv1.PersistentVolumeClaimStatus{
					Phase: k8sv1.ClaimBound,
				},
			}
			Expect(controller.pvcStore.Add(&pvc)).To(Succeed())

			createCount := 0
			if expectedCreations > 0 {
				shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID)}, map[string]string{}, &createCount)
			}

			if initFunc != nil {
				initFunc()
			}

			sanityExecute(vm)
			Expect(createCount).To(Equal(expectedCreations))
			if expectedCreations > 0 {
				testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
			}
		},
			Entry("when PVC has no annotation", map[string]string{}, 1, nil),
			Entry("when PVC was garbage collected", map[string]string{"cdi.kubevirt.io/garbageCollected": "true"}, 0, nil),
			Entry("when PVC has owned by annotation and CDI does not support adoption", map[string]string{}, 0, shouldFailDataVolumeCreationClaimAlreadyExists),
		)

		DescribeTable("should properly set priority class", func(dvPriorityClass, vmPriorityClass, expectedPriorityClass string) {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PriorityClassName: dvPriorityClass,
				},
			})
			vm.Spec.Template.Spec.PriorityClassName = vmPriorityClass
			vm.Status.PrintableStatus = v1.VirtualMachineStatusStopped

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			createCount := 0
			shouldExpectDataVolumeCreationPriorityClass(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID)}, map[string]string{}, expectedPriorityClass, &createCount)

			sanityExecute(vm)
			Expect(createCount).To(Equal(1))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		},
			Entry("when dv priorityclass is not defined and VM priorityclass is defined", "", "vmpriority", "vmpriority"),
			Entry("when dv priorityclass is defined and VM priorityclass is defined", "dvpriority", "vmpriority", "dvpriority"),
			Entry("when dv priorityclass is defined and VM priorityclass is not defined", "dvpriority", "", "dvpriority"),
			Entry("when dv priorityclass is not defined and VM priorityclass is not defined", "", "", ""),
		)

		Context("crashloop backoff tests", func() {

			It("should track start failures when VMIs fail without hitting running state", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.UID = "123"
				vmi.Status.Phase = v1.Failed

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				shouldExpectVMIFinalizerRemoval()

				sanityExecute(vm)

				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())

				Expect(vm.Status.StartFailure).ToNot(BeNil())
				Expect(vm.Status.StartFailure.RetryAfterTimestamp).ToNot(BeNil())
				Expect(vm.Status.StartFailure.LastFailedVMIUID).To(Equal(vmi.UID))
				Expect(vm.Status.StartFailure.ConsecutiveFailCount).To(Equal(1))
			})

			It("should track a new start failures when a new VMI fails without hitting running state", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.UID = "456"
				vmi.Status.Phase = v1.Failed

				oldRetry := time.Now().Add(-300 * time.Second)
				vm.Status.StartFailure = &v1.VirtualMachineStartFailure{
					LastFailedVMIUID:     "123",
					ConsecutiveFailCount: 1,
					RetryAfterTimestamp: &metav1.Time{
						Time: oldRetry,
					},
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				shouldExpectVMIFinalizerRemoval()

				sanityExecute(vm)

				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())

				Expect(vm.Status.StartFailure).ToNot(BeNil())
				Expect(vm.Status.StartFailure.RetryAfterTimestamp).ToNot(BeNil())
				Expect(vm.Status.StartFailure.RetryAfterTimestamp.Time).ToNot(Equal(oldRetry))
				Expect(vm.Status.StartFailure.LastFailedVMIUID).To(Equal(vmi.UID))
				Expect(vm.Status.StartFailure.ConsecutiveFailCount).To(Equal(2))
			})

			It("should clear start failures when VMI hits running state", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.UID = "456"
				vmi.Status.Phase = v1.Running
				vmi.Status.PhaseTransitionTimestamps = []v1.VirtualMachineInstancePhaseTransitionTimestamp{
					{
						Phase:                    v1.Running,
						PhaseTransitionTimestamp: metav1.Now(),
					},
				}

				oldRetry := time.Now().Add(-300 * time.Second)
				vm.Status.StartFailure = &v1.VirtualMachineStartFailure{
					LastFailedVMIUID:     "123",
					ConsecutiveFailCount: 1,
					RetryAfterTimestamp: &metav1.Time{
						Time: oldRetry,
					},
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.StartFailure).To(BeNil())

				_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			DescribeTable("should clear existing start failures when runStrategy is halted or manual", func(runStrategy v1.VirtualMachineRunStrategy) {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.UID = "456"
				vmi.Status.Phase = v1.Failed
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = &runStrategy

				oldRetry := time.Now().Add(300 * time.Second)
				vm.Status.StartFailure = &v1.VirtualMachineStartFailure{
					LastFailedVMIUID:     "123",
					ConsecutiveFailCount: 1,
					RetryAfterTimestamp: &metav1.Time{
						Time: oldRetry,
					},
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				shouldExpectVMIFinalizerRemoval()

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())

				if runStrategy == v1.RunStrategyHalted || runStrategy == v1.RunStrategyManual {
					Expect(vm.Status.StartFailure).To(BeNil())
				} else {
					Expect(vm.Status.StartFailure).ToNot(BeNil())
				}

				if runStrategy == v1.RunStrategyRerunOnFailure {
					Expect(vm.Status.StateChangeRequests).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Action": Equal(v1.StateChangeRequestAction("Start")),
					})))
				}

				if runStrategy != v1.RunStrategyManual && runStrategy != v1.RunStrategyOnce {
					testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
				}
			},

				Entry("runStrategyHalted", v1.RunStrategyHalted),
				Entry("always", v1.RunStrategyAlways),
				Entry("manual", v1.RunStrategyManual),
				Entry("rerunOnFailure", v1.RunStrategyRerunOnFailure),
				Entry("once", v1.RunStrategyOnce),
			)

			DescribeTable("should calculated expected backoff delay", func(failCount, minExpectedDelay int, maxExpectedDelay int) {

				for i := 0; i < 1000; i++ {
					delay := calculateStartBackoffTime(failCount, defaultMaxCrashLoopBackoffDelaySeconds)

					// check that minExpectedDelay <= delay <= maxExpectedDelay
					Expect(delay).To(And(BeNumerically(">=", minExpectedDelay), BeNumerically("<=", maxExpectedDelay)))
				}
			},

				Entry("failCount 0", 0, 10, 15),
				Entry("failCount 1", 1, 10, 15),
				Entry("failCount 2", 2, 40, 60),
				Entry("failCount 3", 3, 90, 135),
				Entry("failCount 4", 4, 160, 240),
				Entry("failCount 5", 5, 250, 300),
				Entry("failCount 6", 6, 300, 300),
			)

			DescribeTable("has start failure backoff expired", func(vmFunc func() *v1.VirtualMachine, expected int64) {
				vm := vmFunc()
				seconds := startFailureBackoffTimeLeft(vm)

				// since the tests all run in parallel, it's difficult to
				// do precise timing. We use a tolerance of 2 seconds to account
				// for some delays in test execution and make sure the calculation
				// falls within the ballpark of what we expect.
				const tolerance = 2
				Expect(seconds).To(BeNumerically("~", expected, tolerance))
				Expect(seconds).To(BeNumerically(">=", 0))
			},

				Entry("no vm start failures",
					func() *v1.VirtualMachine {
						return &v1.VirtualMachine{}
					},
					int64(0)),
				Entry("vm failure waiting 300 seconds",
					func() *v1.VirtualMachine {
						return &v1.VirtualMachine{
							Status: v1.VirtualMachineStatus{
								StartFailure: &v1.VirtualMachineStartFailure{
									RetryAfterTimestamp: &metav1.Time{
										Time: time.Now().Add(300 * time.Second),
									},
								},
							},
						}
					},
					int64(300)),
				Entry("vm failure 300 seconds past retry time",
					func() *v1.VirtualMachine {
						return &v1.VirtualMachine{
							Status: v1.VirtualMachineStatus{
								StartFailure: &v1.VirtualMachineStartFailure{
									RetryAfterTimestamp: &metav1.Time{
										Time: time.Now().Add(-300 * time.Second),
									},
								},
							},
						}
					},
					int64(0)),
			)
		})

		Context("clone authorization tests", func() {
			dv1 := &v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Namespace: "ns1",
							Name:      "source-pvc",
						},
					},
				},
			}

			dv2 := &v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Name: "source-pvc",
						},
					},
				},
			}

			ds := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns2",
					Name:      "source-ref",
				},
				Spec: cdiv1.DataSourceSpec{
					Source: cdiv1.DataSourceSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Namespace: "ns1",
							Name:      "source-pvc",
						},
					},
				},
			}

			dv3 := &v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv3",
				},
				Spec: cdiv1.DataVolumeSpec{
					SourceRef: &cdiv1.DataVolumeSourceRef{
						Kind:      "DataSource",
						Namespace: &ds.Namespace,
						Name:      ds.Name,
					},
				},
			}

			serviceAccountVol := &v1.Volume{
				Name: "sa",
				VolumeSource: v1.VolumeSource{
					ServiceAccount: &v1.ServiceAccountVolumeSource{
						ServiceAccountName: "sa",
					},
				},
			}

			DescribeTable("create clone DataVolume for VirtualMachineInstance", func(dv *v1.DataVolumeTemplateSpec, saVol *v1.Volume, ds *cdiv1.DataSource, fail, sourcePVC bool) {
				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes,
					v1.Volume{
						Name: "test1",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: dv.Name,
							},
						},
					},
				)

				if saVol != nil {
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, *saVol)
				}

				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dv)

				vm.Status.PrintableStatus = v1.VirtualMachineStatusStopped
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				createCount := 0
				shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID)}, map[string]string{}, &createCount)

				if ds != nil {
					cdiClient.PrependReactor("get", "datasources", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
						ga := action.(testing.GetAction)
						Expect(ga.GetNamespace()).To(Equal(ds.Namespace))
						Expect(ga.GetName()).To(Equal(ds.Name))
						return true, ds, nil
					})

					cdiClient.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
						ca := action.(testing.CreateAction)
						dv := ca.GetObject().(*cdiv1.DataVolume)
						Expect(dv.Spec.SourceRef).To(BeNil())
						Expect(dv.Spec.Source).ToNot(BeNil())
						Expect(dv.Spec.Source.PVC).ToNot(BeNil())
						Expect(dv.Spec.Source.PVC.Namespace).To(Equal(ds.Spec.Source.PVC.Namespace))
						Expect(dv.Spec.Source.PVC.Name).To(Equal(ds.Spec.Source.PVC.Name))
						return false, ds, nil
					})
				}

				// Add source PVC to cache
				if sourcePVC {
					pvc := k8sv1.PersistentVolumeClaim{}
					if dv.Spec.Source != nil {
						pvc.Name = dv.Spec.Source.PVC.Name
						pvc.Namespace = dv.Spec.Source.PVC.Namespace
					} else {
						pvc.Name = ds.Spec.Source.PVC.Name
						pvc.Namespace = ds.Spec.Source.PVC.Namespace
					}
					if pvc.Namespace == "" {
						pvc.Namespace = vm.Namespace
					}
					Expect(controller.pvcStore.Add(&pvc)).To(Succeed())
				}

				controller.cloneAuthFunc = func(dv *cdiv1.DataVolume, requestNamespace, requestName string, proxy cdiv1.AuthorizationHelperProxy, saNamespace, saName string) (bool, string, error) {
					k8sClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
						return true, &authorizationv1.SubjectAccessReview{
							Status: authorizationv1.SubjectAccessReviewStatus{
								Allowed: true,
							},
						}, nil
					})
					response, err := dv.AuthorizeSA(requestNamespace, requestName, proxy, saNamespace, saName)
					Expect(err).ToNot(HaveOccurred())
					if dv.Spec.Source != nil {
						if dv.Spec.Source.PVC.Namespace != "" {
							Expect(response.Handler.SourceNamespace).Should(Equal(dv.Spec.Source.PVC.Namespace))
						} else {
							Expect(response.Handler.SourceNamespace).Should(Equal(vm.Namespace))
						}

						Expect(response.Handler.SourceName).Should(Equal(dv.Spec.Source.PVC.Name))
					} else {
						Expect(response.Handler.SourceNamespace).Should(Equal(ds.Spec.Source.PVC.Namespace))
						Expect(response.Handler.SourceName).Should(Equal(ds.Spec.Source.PVC.Name))
					}

					Expect(saNamespace).Should(Equal(vm.Namespace))

					if saVol != nil {
						Expect(saName).Should(Equal("sa"))
					} else {
						Expect(saName).Should(Equal("default"))
					}

					if fail {
						return false, "Authorization failed", nil
					}

					return true, "", nil
				}

				sanityExecute(vm)
				if fail {
					Expect(createCount).To(Equal(0))
					testutils.ExpectEvent(recorder, UnauthorizedDataVolumeCreateReason)
				} else {
					if !sourcePVC {
						testutils.ExpectEvent(recorder, SourcePVCNotAvailabe)
					}
					Expect(createCount).To(Equal(1))
					testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
				}
			},
				Entry("with auth, source PVC and source namespace defined", dv1, serviceAccountVol, nil, false, true),
				Entry("with auth and no source namespace defined", dv2, serviceAccountVol, nil, false, true),
				Entry("with auth and source namespace no serviceaccount defined", dv1, nil, nil, false, true),
				Entry("with no auth and source namespace defined", dv1, serviceAccountVol, nil, true, true),
				Entry("with auth, source PVC, datasource and source namespace defined", dv3, serviceAccountVol, ds, false, true),
				Entry("with auth, datasource and source namespace but no source PVC", dv3, serviceAccountVol, ds, false, false),
			)
		})

		It("should create VMI with vmRevision", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Generation = 1

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			vmRevision := createVMRevision(vm)
			expectControllerRevisionCreation(vmRevision)

			sanityExecute(vm)

			//TODO expect update status is called
			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())
			Expect(vm.Status.Created).To(BeFalse())
			Expect(vm.Status.Ready).To(BeFalse())

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(vmRevision.Name))
		})

		It("should delete older vmRevision and create VMI with new one", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Generation = 1
			oldVMRevision := createVMRevision(vm)

			vm.Generation = 2
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			vmRevision := createVMRevision(vm)

			expectControllerRevisionList(oldVMRevision)
			expectControllerRevisionDelete(oldVMRevision)
			expectControllerRevisionCreation(vmRevision)

			sanityExecute(vm)

			//TODO expect update status is called
			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())
			Expect(vm.Status.Created).To(BeFalse())
			Expect(vm.Status.Ready).To(BeFalse())

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(vmRevision.Name))
		})

		Context("VM generation tests", func() {

			DescribeTable("should add the generation annotation onto the VMI", func(startingAnnotations map[string]string, endAnnotations map[string]string) {
				_, vmi := DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = startingAnnotations

				annotations := endAnnotations
				setGenerationAnnotationOnVmi(6, vmi)
				Expect(vmi.ObjectMeta.Annotations).To(Equal(annotations))
			},
				Entry("with previous annotations", map[string]string{"test": "test"}, map[string]string{"test": "test", v1.VirtualMachineGenerationAnnotation: "6"}),
				Entry("without previous annotations", map[string]string{}, map[string]string{v1.VirtualMachineGenerationAnnotation: "6"}),
			)

			DescribeTable("should add generation annotation during VMI creation", func(runStrategy v1.VirtualMachineRunStrategy) {
				vm, _ := DefaultVirtualMachine(true)

				vm.Spec.Running = nil
				vm.Spec.RunStrategy = &runStrategy
				vm.Generation = 3

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())

				addVirtualMachine(vm)

				sanityExecute(vm)

				//TODO expect update status is called
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.Created).To(BeFalse())
				Expect(vm.Status.Ready).To(BeFalse())

				if runStrategy == v1.RunStrategyRerunOnFailure || runStrategy == v1.RunStrategyAlways {
					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vmi).ToNot(BeNil())
				}

				testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				annotations := map[string]string{v1.VirtualMachineGenerationAnnotation: "3"}
				Expect(vmi.ObjectMeta.Annotations).To(Equal(annotations))
			},

				Entry("with run strategy Always", v1.RunStrategyAlways),
				Entry("with run strategy Once", v1.RunStrategyOnce),
				Entry("with run strategy RerunOnFailure", v1.RunStrategyRerunOnFailure),
			)

			It("should patch the generation annotation onto the vmi", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = nil
				addVirtualMachine(vm)

				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				err = controller.patchVmGenerationAnnotationOnVmi(4, vmi)
				Expect(err).ToNot(HaveOccurred())

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Annotations).To(HaveKeyWithValue("kubevirt.io/vm-generation", "4"))
			})

			DescribeTable("should get the generation annotation from the vmi", func(annotations map[string]string, desiredGeneration *string, desiredErr error) {
				_, vmi := DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = annotations

				gen, err := getGenerationAnnotation(vmi)
				if desiredGeneration == nil {
					Expect(gen).To(BeNil())
				} else {
					Expect(gen).To(Equal(desiredGeneration))
				}
				if desiredErr == nil {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(Equal(desiredErr))
				}
			},
				Entry("with only one entry in the annotations", map[string]string{v1.VirtualMachineGenerationAnnotation: "6"}, pointer.P("6"), nil),
				Entry("with multiple entries in the annotations", map[string]string{"test": "test", v1.VirtualMachineGenerationAnnotation: "5"}, pointer.P("5"), nil),
				Entry("with no generation annotation existing", map[string]string{"test": "testing"}, nil, nil),
				Entry("with empty annotations map", map[string]string{}, nil, nil),
			)

			DescribeTable("should parse generation from vm controller revision name", func(name string, desiredGeneration *int64) {

				gen := parseGeneration(name, log.DefaultLogger())
				if desiredGeneration == nil {
					Expect(gen).To(BeNil())
				} else {
					Expect(gen).To(Equal(desiredGeneration))
				}
			},
				Entry("with standard name", getVMRevisionName("9160e5de-2540-476a-86d9-af0081aee68a", 3), pointer.P(int64(3))),
				Entry("with one dash in name", getVMRevisionName("abcdef", 5), pointer.P(int64(5))),
				Entry("with no dash in name", "12345", nil),
				Entry("with ill formatted generation", "123-456-2b3b", nil),
			)

			Context("conditionally bump generation tests", func() {
				// Needed for the default values in each Entry(..)
				vm, vmi := DefaultVirtualMachine(true)

				BeforeEach(func() {
					// Reset every time
					vm, vmi = DefaultVirtualMachine(true)
				})

				DescribeTable("should conditionally bump the generation annotation on the vmi", func(initialAnnotations map[string]string, desiredAnnotations map[string]string, revisionVmSpec v1.VirtualMachineSpec, newVMSpec v1.VirtualMachineSpec, vmGeneration int64, desiredErr error, expectPatch bool) {
					// Spec and generation for the vmRevision and 'old' objects
					vmi.ObjectMeta.Annotations = initialAnnotations
					vm.Generation = 1
					vm.Spec = revisionVmSpec

					crName, err := controller.createVMRevision(vm)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), crName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vmi.Status.VirtualMachineRevisionName = crName

					addVirtualMachine(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					// This is the 'updated' details on the vm
					vm.Generation = vmGeneration
					vm.Spec = newVMSpec

					err = controller.conditionallyBumpGenerationAnnotationOnVmi(vm, vmi)
					if desiredErr == nil {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(Equal(desiredErr))
					}

					// TODO patch should not be called if expectPatch == false
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmi.Annotations).To(Equal(desiredAnnotations))

				},
					Entry(
						"with generation and template staying the same",
						map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vm.ObjectMeta.Name,
									Labels: vm.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vm.ObjectMeta.Name,
									Labels: vm.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						int64(2),
						nil,
						false, // Expect no patch
					),
					Entry(
						"with generation increasing and a change in template",
						map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 3,
										},
									},
								},
							},
						},
						int64(3),
						nil,
						false, // No patch because template has changed
					),
					Entry(
						"with generation increasing and no change in template",
						map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{v1.VirtualMachineGenerationAnnotation: "3"},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						int64(3),
						nil,
						true, // Patch since there is no change and we can bump
					),
					Entry(
						"with generation increasing, no change in template, and run strategy changing",
						map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{v1.VirtualMachineGenerationAnnotation: "7"},
						v1.VirtualMachineSpec{
							RunStrategy: func(rs v1.VirtualMachineRunStrategy) *v1.VirtualMachineRunStrategy { return &rs }(v1.RunStrategyAlways),
							Running:     func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						v1.VirtualMachineSpec{
							RunStrategy: func(rs v1.VirtualMachineRunStrategy) *v1.VirtualMachineRunStrategy { return &rs }(v1.RunStrategyRerunOnFailure),
							Running:     func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						int64(7),
						nil,
						true, // Patch since only template matters, not run strategy
					),
				)
			})

			DescribeTable("should sync the generation info", func(initialAnnotations map[string]string, desiredAnnotations map[string]string, revisionVmGeneration int64, vmGeneration int64, desiredErr error, expectPatch bool, desiredObservedGeneration int64, desiredDesiredGeneration int64) {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = initialAnnotations
				vm.Generation = revisionVmGeneration

				crName, err := controller.createVMRevision(vm)
				Expect(err).ToNot(HaveOccurred())

				vmi.Status.VirtualMachineRevisionName = crName

				addVirtualMachine(vm)

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				vm.Generation = vmGeneration

				err = controller.syncGenerationInfo(vm, vmi, log.DefaultLogger())
				if desiredErr == nil {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(Equal(desiredErr))
				}

				Expect(vm.Status.ObservedGeneration).To(Equal(desiredObservedGeneration))
				Expect(vm.Status.DesiredGeneration).To(Equal(desiredDesiredGeneration))

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Annotations).To(Equal(desiredAnnotations))
			},
				Entry(
					"with annotation existing - generation updates",
					map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
					map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
					int64(2),
					int64(3),
					nil,
					false,
					int64(2),
					int64(3),
				),
				Entry(
					"with annotation existing - generation does not change",
					map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
					map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
					int64(2),
					int64(2),
					nil,
					false,
					int64(2),
					int64(2),
				),
				Entry(
					// In this case the annotation should be back filled from the revision
					"with annotation existing - ill formatted generation annotation",
					map[string]string{v1.VirtualMachineGenerationAnnotation: "2b3c"},
					map[string]string{v1.VirtualMachineGenerationAnnotation: "3"},
					int64(3),
					int64(3),
					nil,
					true,
					int64(3),
					int64(3),
				),
				Entry(
					"with annotation not existing - generation updates and patches vmi",
					nil,
					map[string]string{v1.VirtualMachineGenerationAnnotation: "3"},
					int64(3),
					int64(4),
					nil,
					true,
					int64(3),
					int64(4),
				),
				Entry(
					"with annotation not existing - generation does not update and patches vmi",
					nil,
					map[string]string{v1.VirtualMachineGenerationAnnotation: "7"},
					int64(7),
					int64(7),
					nil,
					true,
					int64(7),
					int64(7),
				),
			)

			Context("generation tests with Execute()", func() {
				// Needed for the default values in each Entry(..)
				vm, vmi := DefaultVirtualMachine(true)

				BeforeEach(func() {
					// Reset every time
					vm, vmi = DefaultVirtualMachine(true)
				})

				DescribeTable("should update annotations and sync during Execute()", func(initialAnnotations map[string]string, desiredAnnotations map[string]string, revisionVmSpec v1.VirtualMachineSpec, newVMSpec v1.VirtualMachineSpec, revisionVmGeneration int64, vmGeneration int64, desiredErr error, expectPatch bool, desiredObservedGeneration int64, desiredDesiredGeneration int64) {
					vmi.ObjectMeta.Annotations = initialAnnotations
					vm.Generation = revisionVmGeneration
					vm.Spec = revisionVmSpec

					crName, err := controller.createVMRevision(vm)
					Expect(err).ToNot(HaveOccurred())

					vmi.Status.VirtualMachineRevisionName = crName

					vm.Generation = vmGeneration
					vm.Spec = newVMSpec

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())

					Expect(vm.Status.ObservedGeneration).To(Equal(desiredObservedGeneration))
					Expect(vm.Status.DesiredGeneration).To(Equal(desiredDesiredGeneration))

					// TODO should not patch if expectPatch == false
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmi.Annotations).To(Equal(desiredAnnotations))
				},
					Entry(
						// Expect no patch on vmi annotations, and vm status to be correct
						"with annotation existing, new changes in VM spec",
						map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 2,
										},
									},
								},
							},
						},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 4, // changed
										},
									},
								},
							},
						},
						int64(2),
						int64(3),
						nil,
						false,
						int64(2),
						int64(3),
					),
					Entry(
						// Expect a patch on vmi annotations, and vm status to be correct
						"with annotation existing, no new changes in VM spec",
						map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{v1.VirtualMachineGenerationAnnotation: "3"},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 2,
										},
									},
								},
							},
						},
						v1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: v1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &v1.CPU{
											Cores: 2,
										},
									},
								},
							},
						},
						int64(2),
						int64(3),
						nil,
						true,
						int64(3),
						int64(3),
					),
				)
			})
		})

		DescribeTable("should create missing VirtualMachineInstance", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm, _ := DefaultVirtualMachine(true)

			vm.Spec.Running = nil
			vm.Spec.RunStrategy = &runStrategy

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			//TODO expect update status is called
			Expect(vm.Status.Created).To(BeFalse())
			Expect(vm.Status.Ready).To(BeFalse())
			if runStrategy == v1.RunStrategyRerunOnFailure || runStrategy == v1.RunStrategyAlways {
				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vmi).ToNot(BeNil())
			}

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		},

			Entry("with run strategy Always", v1.RunStrategyAlways),
			Entry("with run strategy Once", v1.RunStrategyOnce),
			Entry("with run strategy RerunOnFailure", v1.RunStrategyRerunOnFailure),
		)

		It("should ignore the name of a VirtualMachineInstance templates", func() {
			vm, _ := DefaultVirtualMachineWithNames(true, "vmname", "vminame")

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			//TODO expect update status is called
			Expect(vm.Status.Created).To(BeFalse())
			Expect(vm.Status.Ready).To(BeFalse())

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should update status to created if the vmi exists", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vmi.Status.Phase = v1.Scheduled

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			//TODO expect update status is called
			Expect(vm.Status.Created).To(BeTrue())
			Expect(vm.Status.Ready).To(BeFalse())
		})

		It("should update status to created and ready when vmi is running and running", func() {
			vm, vmi := DefaultVirtualMachine(true)
			markAsReady(vmi)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			//TODO expect update status is called
			Expect(vm.Status.Created).To(BeTrue())
			Expect(vm.Status.Ready).To(BeTrue())
		})

		It("should have stable firmware UUIDs", func() {
			vm1, _ := DefaultVirtualMachineWithNames(true, "testvm1", "testvmi1")
			vmi1 := controller.setupVMIFromVM(vm1)

			// intentionally use the same names
			vm2, _ := DefaultVirtualMachineWithNames(true, "testvm1", "testvmi1")
			vmi2 := controller.setupVMIFromVM(vm2)
			Expect(vmi1.Spec.Domain.Firmware.UUID).To(Equal(vmi2.Spec.Domain.Firmware.UUID))

			// now we want different names
			vm3, _ := DefaultVirtualMachineWithNames(true, "testvm3", "testvmi3")
			vmi3 := controller.setupVMIFromVM(vm3)
			Expect(vmi1.Spec.Domain.Firmware.UUID).NotTo(Equal(vmi3.Spec.Domain.Firmware.UUID))
		})

		It("should honour any firmware UUID present in the template", func() {
			uid := uuid.NewString()
			vm1, _ := DefaultVirtualMachineWithNames(true, "testvm1", "testvmi1")
			vm1.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{UUID: types.UID(uid)}

			vmi1 := controller.setupVMIFromVM(vm1)
			Expect(string(vmi1.Spec.Domain.Firmware.UUID)).To(Equal(uid))
		})

		It("should delete VirtualMachineInstance when stopped", func() {
			vm, vmi := DefaultVirtualMachine(false)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		It("should add controller finalizer if VirtualMachine does not have it", func() {
			vm, _ := DefaultVirtualMachine(false)
			vm.Finalizers = nil

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())

			addVirtualMachine(vm)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())
			Expect(vm.Finalizers).To(HaveExactElements(v1.VirtualMachineControllerFinalizer))
		})

		It("should add controller finalizer only once", func() {
			//DefaultVirtualMachine already set finalizer
			vm, _ := DefaultVirtualMachine(false)
			Expect(vm.Finalizers).To(HaveLen(1))
			Expect(vm.Finalizers[0]).To(BeEquivalentTo(v1.VirtualMachineControllerFinalizer))

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			//TODO Expect only update status, not Patch on vmInterface
			sanityExecute(vm)
		})

		It("should delete VirtualMachineInstance when VirtualMachine marked for deletion", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.DeletionTimestamp = pointer.P(metav1.Now())

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		It("should remove controller finalizer once VirtualMachineInstance is gone", func() {
			//DefaultVirtualMachine already set finalizer
			vm, _ := DefaultVirtualMachine(true)
			vm.DeletionTimestamp = pointer.P(metav1.Now())

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())
			Expect(vm.Finalizers).To(BeEmpty())
		})

		DescribeTable("should not delete VirtualMachineInstance when vmi failed", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm, vmi := DefaultVirtualMachine(true)

			vm.Spec.Running = nil
			vm.Spec.RunStrategy = &runStrategy

			vmi.Status.Phase = v1.Failed

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			controller.vmiIndexer.Add(vmi)

			shouldExpectVMIFinalizerRemoval()

			sanityExecute(vm)

		},

			Entry("with run strategy Once", v1.RunStrategyOnce),
			Entry("with run strategy Manual", v1.RunStrategyManual),
		)

		It("should not delete the VirtualMachineInstance again if it is already marked for deletion", func() {
			vm, vmi := DefaultVirtualMachine(false)
			vmi.DeletionTimestamp = pointer.P(metav1.Now())

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)
		})

		It("should ignore non-matching VMIs", func() {
			vm, _ := DefaultVirtualMachine(true)

			nonMatchingVMI := api.NewMinimalVMI("testvmi1")
			nonMatchingVMI.ObjectMeta.Labels = map[string]string{"test": "test1"}

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			// We still expect three calls to create VMIs, since VirtualMachineInstance does not meet the requirements
			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), nonMatchingVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(nonMatchingVMI)

			sanityExecute(vm)

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should detect that a VirtualMachineInstance already exists and adopt it", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vmi.OwnerReferences = []metav1.OwnerReference{}

			addVirtualMachine(vm)
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.OwnerReferences).NotTo(BeEmpty())
		})

		It("should detect that a DataVolume already exists and adopt it", func() {
			vm, _ := DefaultVirtualMachine(false)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dv1",
					Namespace: vm.Namespace,
				},
			})

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
			dv.Status.Phase = cdiv1.Succeeded

			orphanDV := dv.DeepCopy()
			orphanDV.ObjectMeta.OwnerReferences = nil
			Expect(controller.dataVolumeStore.Add(orphanDV)).To(Succeed())

			cdiClient.Fake.PrependReactor("patch", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				patch, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				Expect(patch.GetName()).To(Equal(dv.Name))
				Expect(patch.GetNamespace()).To(Equal(dv.Namespace))
				Expect(string(patch.GetPatch())).To(ContainSubstring(string(vm.UID)))
				Expect(string(patch.GetPatch())).To(ContainSubstring("ownerReferences"))
				return true, dv, nil
			})

			sanityExecute(vm)
		})

		It("should detect that it has nothing to do beside updating the status", func() {
			vm, vmi := DefaultVirtualMachine(true)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)
		})

		It("should add a fail condition if start up fails", func() {
			vm, _ := DefaultVirtualMachine(true)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			virtFakeClient.PrependReactor("create", "virtualmachineinstances", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				return true, &v1.VirtualMachineInstance{}, fmt.Errorf("some random failure")
			})
			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
			Expect(cond).To(Not(BeNil()))
			Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":    Equal(v1.VirtualMachineFailure),
				"Reason":  Equal("FailedCreate"),
				"Message": ContainSubstring("some random failure"),
				"Status":  Equal(k8sv1.ConditionTrue),
			}))

			testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		It("should add a fail condition if deletion fails", func() {
			vm, vmi := DefaultVirtualMachine(false)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			controller.vmiIndexer.Add(vmi)

			virtFakeClient.PrependReactor("delete", "virtualmachineinstances", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, fmt.Errorf("some random failure")
			})

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
			Expect(cond).To(Not(BeNil()))
			Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":    Equal(v1.VirtualMachineFailure),
				"Reason":  Equal("FailedDelete"),
				"Message": ContainSubstring("some random failure"),
				"Status":  Equal(k8sv1.ConditionTrue),
			}))

			testutils.ExpectEvents(recorder, FailedDeleteVirtualMachineReason)
		})

		DescribeTable("should add ready condition when VMI exists", func(setup func(vmi *v1.VirtualMachineInstance), status k8sv1.ConditionStatus) {
			vm, vmi := DefaultVirtualMachine(true)
			virtcontroller.NewVirtualMachineConditionManager().RemoveCondition(vm, v1.VirtualMachineReady)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			setup(vmi)
			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			cond := virtcontroller.NewVirtualMachineConditionManager().
				GetCondition(vm, v1.VirtualMachineReady)
			Expect(cond).To(Not(BeNil()))
			Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Status": Equal(status),
			}))

		},
			Entry("VMI Ready condition is True", markAsReady, k8sv1.ConditionTrue),
			Entry("VMI Ready condition is False", markAsNonReady, k8sv1.ConditionFalse),
			Entry("VMI Ready condition doesn't exist", unmarkReady, k8sv1.ConditionFalse),
		)

		It("should sync VMI conditions", func() {
			vm, vmi := DefaultVirtualMachine(true)
			virtcontroller.NewVirtualMachineConditionManager().RemoveCondition(vm, v1.VirtualMachineReady)

			cm := virtcontroller.NewVirtualMachineInstanceConditionManager()
			cmVM := virtcontroller.NewVirtualMachineConditionManager()

			addCondList := []v1.VirtualMachineInstanceConditionType{
				v1.VirtualMachineInstanceProvisioning,
				v1.VirtualMachineInstanceSynchronized,
				v1.VirtualMachineInstancePaused,
			}

			removeCondList := []v1.VirtualMachineInstanceConditionType{
				v1.VirtualMachineInstanceAgentConnected,
				v1.VirtualMachineInstanceAccessCredentialsSynchronized,
				v1.VirtualMachineInstanceUnsupportedAgent,
			}

			updateCondList := []v1.VirtualMachineInstanceConditionType{
				v1.VirtualMachineInstanceIsMigratable,
			}

			now := metav1.Now()
			for _, condName := range addCondList {
				cm.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
					Type:               condName,
					Status:             k8sv1.ConditionTrue,
					Reason:             "fakereason",
					Message:            "fakemsg",
					LastProbeTime:      now,
					LastTransitionTime: now,
				})
			}

			for _, condName := range updateCondList {
				// Set to true on VMI
				cm.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
					Type:               condName,
					Status:             k8sv1.ConditionTrue,
					Reason:             "fakereason",
					Message:            "fakemsg",
					LastProbeTime:      now,
					LastTransitionTime: now,
				})

				// Set to false on VM, expect sync to update it to true
				cmVM.UpdateCondition(vm, &v1.VirtualMachineCondition{
					Type:               v1.VirtualMachineConditionType(condName),
					Status:             k8sv1.ConditionFalse,
					Reason:             "fakereason",
					Message:            "fakemsg",
					LastProbeTime:      now,
					LastTransitionTime: now,
				})
			}

			for _, condName := range removeCondList {
				cmVM.UpdateCondition(vm, &v1.VirtualMachineCondition{
					Type:               v1.VirtualMachineConditionType(condName),
					Status:             k8sv1.ConditionTrue,
					Reason:             "fakereason",
					Message:            "fakemsg",
					LastProbeTime:      now,
					LastTransitionTime: now,
				})
			}

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())

			addVirtualMachine(vm)

			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			for _, condName := range addCondList {
				cond := cmVM.GetCondition(vm, v1.VirtualMachineConditionType(condName))
				Expect(cond).To(Not(BeNil()))
				Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Status": Equal(k8sv1.ConditionTrue),
				}))
			}
			// these conditions shouldn't exist anymore
			for _, condName := range removeCondList {
				cond := cmVM.GetCondition(vm, v1.VirtualMachineConditionType(condName))
				Expect(cond).To(BeNil())
			}
			// these conditsion should be updated
			for _, condName := range updateCondList {
				cond := cmVM.GetCondition(vm, v1.VirtualMachineConditionType(condName))
				Expect(cond).To(Not(BeNil()))
				Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Status": Equal(k8sv1.ConditionTrue),
				}))
			}
		})

		It("should add ready condition when VMI doesn't exists", func() {
			vm, _ := DefaultVirtualMachine(true)
			virtcontroller.NewVirtualMachineConditionManager().RemoveCondition(vm, v1.VirtualMachineReady)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())

			addVirtualMachine(vm)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			cond := virtcontroller.NewVirtualMachineConditionManager().
				GetCondition(vm, v1.VirtualMachineReady)
			Expect(cond).To(Not(BeNil()))
			Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Status": Equal(k8sv1.ConditionFalse),
			}))

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

		})

		It("should add paused condition", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			markAsReady(vmi)
			vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstancePaused,
				Status: k8sv1.ConditionTrue,
			})
			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			cond := virtcontroller.NewVirtualMachineConditionManager().
				GetCondition(vm, v1.VirtualMachinePaused)
			Expect(cond).To(Not(BeNil()))
			Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Status": Equal(k8sv1.ConditionTrue),
			}))

		})

		It("should remove paused condition", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Status.Conditions = append(vm.Status.Conditions, v1.VirtualMachineCondition{
				Type:   v1.VirtualMachinePaused,
				Status: k8sv1.ConditionTrue,
			})
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			markAsReady(vmi)
			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			cond := virtcontroller.NewVirtualMachineConditionManager().
				GetCondition(vm, v1.VirtualMachinePaused)
			Expect(cond).To(BeNil())
		})

		It("should back off if a sync error occurs", func() {
			vm, vmi := DefaultVirtualMachine(false)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			virtFakeClient.PrependReactor("delete", "virtualmachineinstances", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, fmt.Errorf("some random failure")
			})
			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
			Expect(cond).To(Not(BeNil()))
			Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":    Equal(v1.VirtualMachineFailure),
				"Reason":  Equal("FailedDelete"),
				"Message": ContainSubstring("some random failure"),
				"Status":  Equal(k8sv1.ConditionTrue),
			}))

			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))
			testutils.ExpectEvents(recorder, FailedDeleteVirtualMachineReason)
		})

		It("should copy annotations from spec.template to vmi", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"test": "test"}
			annotations := map[string]string{"test": "test", v1.VirtualMachineGenerationAnnotation: "0"}

			vm.Status.PrintableStatus = v1.VirtualMachineStatusStarting
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Annotations).To(Equal(annotations))
		})

		It("should copy kubevirt ignitiondata annotation from spec.template to vmi", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"kubevirt.io/ignitiondata": "test"}
			annotations := map[string]string{"kubevirt.io/ignitiondata": "test", v1.VirtualMachineGenerationAnnotation: "0"}

			vm.Status.PrintableStatus = v1.VirtualMachineStatusStarting
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Annotations).To(Equal(annotations))
		})

		It("should copy kubernetes annotations from spec.template to vmi", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"}
			annotations := map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true", v1.VirtualMachineGenerationAnnotation: "0"}

			vm.Status.PrintableStatus = v1.VirtualMachineStatusStarting
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Annotations).To(Equal(annotations))
		})

		Context("VM memory dump", func() {
			const (
				testPVCName    = "testPVC"
				targetFileName = "memory.dump"
			)

			applyVMIMemoryDumpVol := func(spec *v1.VirtualMachineInstanceSpec) *v1.VirtualMachineInstanceSpec {
				newVolume := v1.Volume{
					Name: testPVCName,
					VolumeSource: v1.VolumeSource{
						MemoryDump: &v1.MemoryDumpVolumeSource{
							PersistentVolumeClaimVolumeSource: v1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
									ClaimName: testPVCName,
								},
								Hotpluggable: true,
							},
						},
					},
				}

				spec.Volumes = append(spec.Volumes, newVolume)

				return spec
			}

			expectPVCAnnotationUpdate := func(expectedAnnotation string, pvcAnnotationUpdated chan bool) {
				virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
				k8sClient.Fake.PrependReactor("update", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
					update, ok := action.(testing.UpdateAction)
					Expect(ok).To(BeTrue())

					pvc, ok := update.GetObject().(*k8sv1.PersistentVolumeClaim)
					Expect(ok).To(BeTrue())
					Expect(pvc.Name).To(Equal(testPVCName))
					Expect(pvc.Annotations[v1.PVCMemoryDumpAnnotation]).To(Equal(expectedAnnotation))
					pvcAnnotationUpdated <- true

					return true, nil, nil
				})
			}

			It("should add memory dump volume and update vmi volumes", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpAssociating,
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				markAsReady(vmi)
				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(testPVCName))
			})

			It("should update memory dump phase to InProgress when memory dump in vm volumes", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpAssociating,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)
				vmi.Spec = vm.Spec.Template.Spec
				markAsReady(vmi)
				controller.vmiIndexer.Add(vmi)

				// when the memory dump volume is in the vm volume list we should change status to in progress
				updatedMemoryDump := &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpInProgress,
				}

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
			})

			It("should change status to unmounting when memory dump timestamp updated", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpInProgress,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				vmi.Spec = vm.Spec.Template.Spec
				now := metav1.Now()
				vmi.Status.VolumeStatus = []v1.VolumeStatus{
					{
						Name:  testPVCName,
						Phase: v1.MemoryDumpVolumeCompleted,
						MemoryDumpVolume: &v1.DomainMemoryDumpInfo{
							StartTimestamp: &now,
							EndTimestamp:   &now,
							ClaimName:      testPVCName,
							TargetFileName: targetFileName,
						},
					},
				}
				markAsReady(vmi)
				controller.vmiIndexer.Add(vmi)

				updatedMemoryDump := &v1.VirtualMachineMemoryDumpRequest{
					ClaimName:      testPVCName,
					Phase:          v1.MemoryDumpUnmounting,
					EndTimestamp:   &now,
					StartTimestamp: &now,
					FileName:       &vmi.Status.VolumeStatus[0].MemoryDumpVolume.TargetFileName,
				}

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
			})

			It("should update status to failed when memory dump failed", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpInProgress,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				vmi.Spec = vm.Spec.Template.Spec
				now := metav1.Now()
				vmi.Status.VolumeStatus = []v1.VolumeStatus{
					{
						Name:    testPVCName,
						Phase:   v1.MemoryDumpVolumeFailed,
						Message: "Memory dump failed",
						MemoryDumpVolume: &v1.DomainMemoryDumpInfo{
							ClaimName:    testPVCName,
							EndTimestamp: &now,
						},
					},
				}
				markAsReady(vmi)
				controller.vmiIndexer.Add(vmi)

				updatedMemoryDump := &v1.VirtualMachineMemoryDumpRequest{
					ClaimName:    testPVCName,
					Phase:        v1.MemoryDumpFailed,
					Message:      vmi.Status.VolumeStatus[0].Message,
					EndTimestamp: &now,
				}

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
			})

			DescribeTable("should remove memory dump volume from vmi volumes and update pvc annotation", func(phase v1.MemoryDumpPhase, expectedAnnotation string) {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     phase,
				}
				if phase != v1.MemoryDumpFailed {
					fileName := targetFileName
					vm.Status.MemoryDumpRequest.FileName = &fileName
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				vmi.Spec = vm.Spec.Template.Spec
				vmi.Status.VolumeStatus = []v1.VolumeStatus{
					{
						Name: testPVCName,
						MemoryDumpVolume: &v1.DomainMemoryDumpInfo{
							ClaimName: testPVCName,
						},
					},
				}
				markAsReady(vmi)
				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				pvc := k8sv1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testPVCName,
						Namespace: vm.Namespace,
					},
				}
				Expect(controller.pvcStore.Add(&pvc)).To(Succeed())

				pvcAnnotationUpdated := make(chan bool, 1)
				defer close(pvcAnnotationUpdated)
				// TODO: convert this to action check
				expectPVCAnnotationUpdate(expectedAnnotation, pvcAnnotationUpdated)

				sanityExecute(vm)

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(vmi.Spec.Volumes).To(BeEmpty())

				Eventually(func() bool {
					select {
					case updated := <-pvcAnnotationUpdated:
						return updated
					default:
					}
					return false
				}, 10*time.Second, 2).Should(BeTrue(), "failed, pvc annotation wasn't updated")
			},
				Entry("when phase is Unmounting", v1.MemoryDumpUnmounting, targetFileName),
				Entry("when phase is Failed", v1.MemoryDumpFailed, "Memory dump failed"),
			)

			It("should update memory dump to complete once memory dump volume unmounted", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				now := metav1.Now()
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName:    testPVCName,
					Phase:        v1.MemoryDumpUnmounting,
					EndTimestamp: &now,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				markAsReady(vmi)
				controller.vmiIndexer.Add(vmi)

				// in case the volume is not in vmi volume status we should update status to completed
				updatedMemoryDump := &v1.VirtualMachineMemoryDumpRequest{
					ClaimName:    testPVCName,
					Phase:        v1.MemoryDumpCompleted,
					EndTimestamp: &now,
				}

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
			})

			It("should remove memory dump volume from vm volumes list when status is Dissociating", func() {
				// No need to add vmi - can do this action even if vm not running
				vm, _ := DefaultVirtualMachine(false)
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpDissociating,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
			})

			It("should dissociate memory dump request when status is Dissociating and not in vm volumes", func() {
				// No need to add vmi - can do this action even if vm not running
				vm, _ := DefaultVirtualMachine(false)
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpDissociating,
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.MemoryDumpRequest).To(BeNil())
			})

			DescribeTable("should not setup vmi with memory dump if memory dump", func(phase v1.MemoryDumpPhase) {
				vm, _ := DefaultVirtualMachine(true)
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     phase,
				}

				vmi := controller.setupVMIFromVM(vm)
				Expect(vmi.Spec.Volumes).To(BeEmpty())

			},
				Entry("in phase Unmounting", v1.MemoryDumpUnmounting),
				Entry("in phase Completed", v1.MemoryDumpCompleted),
				Entry("in phase Dissociating", v1.MemoryDumpDissociating),
			)

		})

		Context("VM printableStatus", func() {

			It("Should set a Stopped status when running=false and VMI doesn't exist", func() {
				vm, _ := DefaultVirtualMachine(false)

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStopped))
			})

			DescribeTable("should set a Stopped status when VMI exists but stopped", func(phase v1.VirtualMachineInstancePhase, deletionTimestamp *metav1.Time) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = phase
				vmi.Status.PhaseTransitionTimestamps = []v1.VirtualMachineInstancePhaseTransitionTimestamp{
					{
						Phase:                    v1.Running,
						PhaseTransitionTimestamp: metav1.Now(),
					},
				}
				vmi.ObjectMeta.DeletionTimestamp = deletionTimestamp

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				shouldExpectVMIFinalizerRemoval()

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStopped))

				// If the VMI is not already marked to be deleted (deletion timestamp is set), it should be deleted
				if deletionTimestamp == nil {
					_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(MatchError(ContainSubstring("not found")))
				}

			},

				Entry("in Succeeded state", v1.Succeeded, nil),
				Entry("in Succeeded state with a deletionTimestamp", v1.Succeeded, &metav1.Time{Time: time.Now()}),
				Entry("in Failed state", v1.Failed, nil),
				Entry("in Failed state with a deletionTimestamp", v1.Failed, &metav1.Time{Time: time.Now()}),
			)

			It("Should set a Starting status when running=true and VMI doesn't exist", func() {
				vm, _ := DefaultVirtualMachine(true)
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				sanityExecute(vm)

				_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStarting))
			})

			DescribeTable("Should set a Starting status when VMI is in a startup phase", func(phase v1.VirtualMachineInstancePhase) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = phase

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStarting))
			},

				Entry("VMI has no phase set", v1.VmPhaseUnset),
				Entry("VMI is in Pending phase", v1.Pending),
				Entry("VMI is in Scheduling phase", v1.Scheduling),
				Entry("VMI is in Scheduled phase", v1.Scheduled),
			)

			DescribeTable("Should set a CrashLoop status when VMI is deleted and VM is in crash loop backoff", func(status v1.VirtualMachineStatus, runStrategy v1.VirtualMachineRunStrategy, hasVMI bool, expectCrashloop bool) {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = &runStrategy
				vm.Status = status

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)
				if hasVMI {
					vmi.Status.Phase = v1.Running
					controller.vmiIndexer.Add(vmi)
				}

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				if expectCrashloop {
					Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusCrashLoopBackOff))
				} else {
					Expect(vm.Status.PrintableStatus).ToNot(Equal(v1.VirtualMachineStatusCrashLoopBackOff))
				}
			},

				Entry("vm with runStrategy always and crash loop",
					v1.VirtualMachineStatus{
						StartFailure: &v1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					v1.RunStrategyAlways,
					false,
					true),
				Entry("vm with runStrategy rerun on failure and crash loop",
					v1.VirtualMachineStatus{
						StartFailure: &v1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					v1.RunStrategyRerunOnFailure,
					false,
					true),
				Entry("vm with runStrategy halt should not report crash loop",
					v1.VirtualMachineStatus{
						StartFailure: &v1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					v1.RunStrategyHalted,
					false,
					false),
				Entry("vm with runStrategy manual should not report crash loop",
					v1.VirtualMachineStatus{
						StartFailure: &v1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					v1.RunStrategyManual,
					false,
					false),
				Entry("vm with runStrategy once should not report crash loop",
					v1.VirtualMachineStatus{
						StartFailure: &v1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					v1.RunStrategyOnce,
					true,
					false),
				Entry("vm with runStrategy always and VMI still exists should not report crash loop",
					v1.VirtualMachineStatus{
						StartFailure: &v1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					v1.RunStrategyAlways,
					true,
					false),
			)
			Context("VM with DataVolumes", func() {
				var vm *v1.VirtualMachine
				var vmi *v1.VirtualMachineInstance

				BeforeEach(func() {
					vm, vmi = DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "test1",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv1",
							},
						},
					})

					vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "dv1",
							Namespace: vm.Namespace,
						},
					})
				})

				DescribeTable("Should set a appropriate status when DataVolume exists but not bound", func(running bool, phase cdiv1.DataVolumePhase, status v1.VirtualMachinePrintableStatus) {
					vm.Spec.Running = &running

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
					dv.Status.Phase = phase
					controller.dataVolumeStore.Add(dv)

					pvc := k8sv1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      dv.Name,
							Namespace: dv.Namespace,
						},
						Status: k8sv1.PersistentVolumeClaimStatus{
							Phase: k8sv1.ClaimPending,
						},
					}
					Expect(controller.pvcStore.Add(&pvc)).To(Succeed())

					sanityExecute(vm)

					if running {
						_, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
					}

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.PrintableStatus).To(Equal(status))
				},
					Entry("Started VM PendingPopulation", true, cdiv1.PendingPopulation, v1.VirtualMachineStatusWaitingForVolumeBinding),
					Entry("Started VM WFFC", true, cdiv1.WaitForFirstConsumer, v1.VirtualMachineStatusWaitingForVolumeBinding),
					Entry("Stopped VM PendingPopulation", false, cdiv1.PendingPopulation, v1.VirtualMachineStatusStopped),
					Entry("Stopped VM", false, cdiv1.WaitForFirstConsumer, v1.VirtualMachineStatusStopped),
				)

				DescribeTable("Should set a Provisioning status when DataVolume bound but not ready",
					func(dvPhase cdiv1.DataVolumePhase) {
						vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
						Expect(err).To(Succeed())
						addVirtualMachine(vm)

						dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
						dv.Status.Phase = dvPhase
						dv.Status.Conditions = append(dv.Status.Conditions, cdiv1.DataVolumeCondition{
							Type:   cdiv1.DataVolumeBound,
							Status: k8sv1.ConditionTrue,
						})
						controller.dataVolumeStore.Add(dv)

						sanityExecute(vm)

						_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						matcher := MatchError(ContainSubstring("not found"))
						if dvPhase == cdiv1.WaitForFirstConsumer {
							matcher = Succeed()
						}
						Expect(err).To(matcher)

						vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
						Expect(err).To(Succeed())
						Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusProvisioning))
					},

					Entry("DataVolume is in ImportScheduled phase", cdiv1.ImportScheduled),
					Entry("DataVolume is in ImportInProgress phase", cdiv1.ImportInProgress),
					Entry("DataVolume is in WaitForFirstConsumer phase", cdiv1.WaitForFirstConsumer),
				)

				DescribeTable("Should set a DataVolumeError status when DataVolume reports an error", func(dvFunc func(*cdiv1.DataVolume)) {
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
					dvFunc(dv)
					controller.dataVolumeStore.Add(dv)

					sanityExecute(vm)
					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusDataVolumeError))
				},

					Entry(
						"DataVolume is in Failed phase",
						func(dv *cdiv1.DataVolume) {
							dv.Status.Phase = cdiv1.Failed
						},
					),
					Entry(
						"DataVolume Running condition is in error",
						func(dv *cdiv1.DataVolume) {
							dv.Status.Conditions = append(dv.Status.Conditions, cdiv1.DataVolumeCondition{
								Type:   cdiv1.DataVolumeRunning,
								Status: k8sv1.ConditionFalse,
								Reason: "Error",
							})
						},
					),
				)

				It("Should clear a DataVolumeError status when the DataVolume error is gone", func() {
					vm.Status.PrintableStatus = v1.VirtualMachineStatusDataVolumeError
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
					dv.Status.Phase = cdiv1.CloneInProgress
					dv.Status.Conditions = append(dv.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8sv1.ConditionTrue,
					})
					controller.dataVolumeStore.Add(dv)

					sanityExecute(vm)
					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusProvisioning))
				})

				It("Should set a Provisioning status when one DataVolume is ready and another isn't", func() {
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "test2",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv2",
							},
						},
					})

					vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "dv2",
							Namespace: vm.Namespace,
						},
					})

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					dv1, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
					dv1.Status.Phase = cdiv1.Succeeded
					dv1.Status.Conditions = append(dv1.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8sv1.ConditionTrue,
					})
					dv2, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
					dv2.Status.Phase = cdiv1.ImportInProgress
					dv2.Status.Conditions = append(dv2.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8sv1.ConditionTrue,
					})

					controller.dataVolumeStore.Add(dv1)
					controller.dataVolumeStore.Add(dv2)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusProvisioning))
				})
			})

			Context("VM with PersistentVolumeClaims", func() {
				var vm *v1.VirtualMachine

				BeforeEach(func() {
					vm, _ = DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "test1",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "pvc1",
							}},
						},
					})

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)
				})

				DescribeTable("Should set a WaitingForVolumeBinding status when PersistentVolumeClaim exists but unbound", func(pvcPhase k8sv1.PersistentVolumeClaimPhase) {
					pvc := k8sv1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pvc1",
							Namespace: vm.Namespace,
						},
						Status: k8sv1.PersistentVolumeClaimStatus{
							Phase: pvcPhase,
						},
					}
					Expect(controller.pvcStore.Add(&pvc)).To(Succeed())

					sanityExecute(vm)

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusWaitingForVolumeBinding))

				},

					Entry("PersistentVolumeClaim is in Pending phase", k8sv1.ClaimPending),
					Entry("PersistentVolumeClaim is in Lost phase", k8sv1.ClaimLost),
				)

			})

			It("should set a Running status when VMI is running but not paused", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = v1.Running
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusRunning))
			})

			It("should set a Paused status when VMI is running but is paused", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = v1.Running
				vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
					Type:   v1.VirtualMachineInstancePaused,
					Status: k8sv1.ConditionTrue,
				})

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusPaused))
			})

			DescribeTable("should set a Stopping status when VMI has a deletion timestamp set", func(phase v1.VirtualMachineInstancePhase, condType v1.VirtualMachineInstanceConditionType) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				vmi.Status.Phase = phase

				if condType != "" {
					vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
						Type:   condType,
						Status: k8sv1.ConditionTrue,
					})
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStopping))
			},

				Entry("when VMI is pending", v1.Pending, v1.VirtualMachineInstanceConditionType("")),
				Entry("when VMI is provisioning", v1.Pending, v1.VirtualMachineInstanceProvisioning),
				Entry("when VMI is scheduling", v1.Scheduling, v1.VirtualMachineInstanceConditionType("")),
				Entry("when VMI is scheduled", v1.Scheduling, v1.VirtualMachineInstanceConditionType("")),
				Entry("when VMI is running", v1.Running, v1.VirtualMachineInstanceConditionType("")),
				Entry("when VMI is paused", v1.Running, v1.VirtualMachineInstancePaused),
			)

			Context("should set a Terminating status when VM has a deletion timestamp set", func() {
				DescribeTable("when VMI exists", func(phase v1.VirtualMachineInstancePhase, condType v1.VirtualMachineInstanceConditionType) {
					vm, vmi := DefaultVirtualMachine(true)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.Status.Phase = phase

					if condType != "" {
						vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
							Type:   condType,
							Status: k8sv1.ConditionTrue,
						})
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusTerminating))

					_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).To(MatchError(ContainSubstring("not found")))
				},

					Entry("when VMI is pending", v1.Pending, v1.VirtualMachineInstanceConditionType("")),
					Entry("when VMI is provisioning", v1.Pending, v1.VirtualMachineInstanceProvisioning),
					Entry("when VMI is scheduling", v1.Scheduling, v1.VirtualMachineInstanceConditionType("")),
					Entry("when VMI is scheduled", v1.Scheduling, v1.VirtualMachineInstanceConditionType("")),
					Entry("when VMI is running", v1.Running, v1.VirtualMachineInstanceConditionType("")),
					Entry("when VMI is paused", v1.Running, v1.VirtualMachineInstancePaused),
				)

				It("when VMI exists and has a deletion timestamp set", func() {
					vm, vmi := DefaultVirtualMachine(true)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.Status.Phase = v1.Running

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)
					controller.vmiIndexer.Add(vmi)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusTerminating))
				})

				DescribeTable("when VMI does not exist", func(running bool) {
					vm, _ := DefaultVirtualMachine(running)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Finalizers).To(BeEmpty())
					Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusTerminating))
				},

					Entry("with running: true", true),
					Entry("with running: false", false),
				)
			})

			It("should set a Migrating status when VMI is migrating", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = v1.Running
				vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					StartTimestamp: &metav1.Time{Time: time.Now()},
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusMigrating))
			})

			It("should set an Unknown status when VMI is in unknown phase", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = v1.Unknown

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusUnknown))
			})

			DescribeTable("should set a failure status in accordance to VMI condition",
				func(status v1.VirtualMachinePrintableStatus, cond v1.VirtualMachineInstanceCondition) {

					vm, vmi := DefaultVirtualMachine(true)
					vmi.Status.Phase = v1.Scheduling
					vmi.Status.Conditions = append(vmi.Status.Conditions, cond)

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)
					controller.vmiIndexer.Add(vmi)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.PrintableStatus).To(Equal(status))
				},

				Entry("FailedUnschedulable", v1.VirtualMachineStatusUnschedulable,
					v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
						Status: k8sv1.ConditionFalse,
						Reason: k8sv1.PodReasonUnschedulable,
					},
				),
				Entry("FailedPvcNotFound", v1.VirtualMachineStatusPvcNotFound,
					v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceSynchronized,
						Status: k8sv1.ConditionFalse,
						Reason: virtcontroller.FailedPvcNotFoundReason,
					},
				),
			)

			DescribeTable("should set an ImagePullBackOff/ErrPullImage statuses according to VMI Synchronized condition", func(reason string) {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.Status.Phase = v1.Scheduling
				vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
					{
						Type:   v1.VirtualMachineInstanceSynchronized,
						Status: k8sv1.ConditionFalse,
						Reason: reason,
					},
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachinePrintableStatus(reason)))
			},
				Entry("Reason: ErrImagePull", virtcontroller.ErrImagePullReason),
				Entry("Reason: ImagePullBackOff", virtcontroller.ImagePullBackOffReason),
			)
		})

		Context("Instancetype and Preferences", func() {

			const resourceUID types.UID = "9160e5de-2540-476a-86d9-af0081aee68a"
			const resourceGeneration int64 = 1

			var (
				vm *v1.VirtualMachine

				fakeInstancetypeClients       instancetypeclientset.InstancetypeV1beta1Interface
				fakeInstancetypeClient        instancetypeclientset.VirtualMachineInstancetypeInterface
				fakeClusterInstancetypeClient instancetypeclientset.VirtualMachineClusterInstancetypeInterface
				fakePreferenceClient          instancetypeclientset.VirtualMachinePreferenceInterface
				fakeClusterPreferenceClient   instancetypeclientset.VirtualMachineClusterPreferenceInterface

				instancetypeInformerStore        cache.Store
				clusterInstancetypeInformerStore cache.Store
				preferenceInformerStore          cache.Store
				clusterPreferenceInformerStore   cache.Store
				controllerrevisionInformerStore  cache.Store
			)

			BeforeEach(func() {
				vm, _ = DefaultVirtualMachine(true)

				// We need to clear the domainSpec here to ensure the instancetype doesn't conflict
				vm.Spec.Template.Spec.Domain = v1.DomainSpec{}

				fakeInstancetypeClients = fake.NewSimpleClientset().InstancetypeV1beta1()

				fakeInstancetypeClient = fakeInstancetypeClients.VirtualMachineInstancetypes(metav1.NamespaceDefault)
				virtClient.EXPECT().VirtualMachineInstancetype(gomock.Any()).Return(fakeInstancetypeClient).AnyTimes()

				fakeClusterInstancetypeClient = fakeInstancetypeClients.VirtualMachineClusterInstancetypes()
				virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(fakeClusterInstancetypeClient).AnyTimes()

				fakePreferenceClient = fakeInstancetypeClients.VirtualMachinePreferences(metav1.NamespaceDefault)
				virtClient.EXPECT().VirtualMachinePreference(gomock.Any()).Return(fakePreferenceClient).AnyTimes()

				fakeClusterPreferenceClient = fakeInstancetypeClients.VirtualMachineClusterPreferences()
				virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeClusterPreferenceClient).AnyTimes()

				k8sClient = k8sfake.NewSimpleClientset()
				virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()

				instancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineInstancetype{})
				instancetypeInformerStore = instancetypeInformer.GetStore()

				clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
				clusterInstancetypeInformerStore = clusterInstancetypeInformer.GetStore()

				preferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachinePreference{})
				preferenceInformerStore = preferenceInformer.GetStore()

				clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
				clusterPreferenceInformerStore = clusterPreferenceInformer.GetStore()

				controllerrevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
				controllerrevisionInformerStore = controllerrevisionInformer.GetStore()

				controller.instancetypeMethods = &instancetype.InstancetypeMethods{
					InstancetypeStore:        instancetypeInformerStore,
					ClusterInstancetypeStore: clusterInstancetypeInformerStore,
					PreferenceStore:          preferenceInformerStore,
					ClusterPreferenceStore:   clusterPreferenceInformerStore,
					ControllerRevisionStore:  controllerrevisionInformerStore,
					Clientset:                virtClient,
				}
			})

			Context("instancetype", func() {
				var (
					instancetypeObj        *instancetypev1beta1.VirtualMachineInstancetype
					clusterInstancetypeObj *instancetypev1beta1.VirtualMachineClusterInstancetype
				)

				BeforeEach(func() {
					instancetypeSpec := instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: uint32(2),
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128M"),
						},
					}
					instancetypeObj = &instancetypev1beta1.VirtualMachineInstancetype{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineInstancetype",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:       "instancetype",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypeSpec,
					}
					_, err := virtClient.VirtualMachineInstancetype(vm.Namespace).Create(context.Background(), instancetypeObj, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = instancetypeInformerStore.Add(instancetypeObj)
					Expect(err).NotTo(HaveOccurred())

					clusterInstancetypeObj = &instancetypev1beta1.VirtualMachineClusterInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "clusterInstancetype",
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineClusterInstancetype",
						},
						Spec: instancetypeSpec,
					}
					_, err = virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetypeObj, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = clusterInstancetypeInformerStore.Add(clusterInstancetypeObj)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should apply VirtualMachineInstancetype to VirtualMachineInstance", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: instancetypeapi.SingularResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					expectedRevisionName := instancetype.GetRevisionName(vm.Name, instancetypeObj.Name, instancetypeObj.GroupVersionKind().Version, instancetypeObj.UID, instancetypeObj.Generation)
					expectedRevision, err := instancetype.CreateControllerRevision(vm, instancetypeObj)
					Expect(err).ToNot(HaveOccurred())

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeObj.Spec.CPU.Guest))
					Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetypeObj.Spec.Memory.Guest))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.InstancetypeAnnotation, instancetypeObj.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Instancetype.RevisionName).To(Equal(expectedRevision.Name))

					revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevisionName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					revisionInstancetype, ok := revision.Data.Object.(*instancetypev1beta1.VirtualMachineInstancetype)
					Expect(ok).To(BeTrue(), "Expected Instancetype in ControllerRevision")

					Expect(revisionInstancetype.Spec).To(Equal(instancetypeObj.Spec))
				})

				DescribeTable("should apply VirtualMachineInstancetype from ControllerRevision to VirtualMachineInstance", func(getRevisionData func() []byte) {
					instancetypeRevision := &appsv1.ControllerRevision{
						ObjectMeta: metav1.ObjectMeta{
							Name: "crName",
						},
						Data: runtime.RawExtension{
							Raw: getRevisionData(),
						},
					}

					instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name:         instancetypeObj.Name,
						Kind:         instancetypeapi.SingularResourceName,
						RevisionName: instancetypeRevision.Name,
					}
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeObj.Spec.CPU.Guest))
					Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetypeObj.Spec.Memory.Guest))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.InstancetypeAnnotation, instancetypeObj.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
				},
					Entry("using v1alpha1 and VirtualMachineInstancetypeSpecRevision with APIVersion", func() []byte {
						v1alpha1instancetypeSpec := instancetypev1alpha1.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1alpha1.CPUInstancetype{
								Guest: instancetypeObj.Spec.CPU.Guest,
							},
							Memory: instancetypev1alpha1.MemoryInstancetype{
								Guest: instancetypeObj.Spec.Memory.Guest,
							},
						}

						specBytes, err := json.Marshal(&v1alpha1instancetypeSpec)
						Expect(err).ToNot(HaveOccurred())

						specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
							APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
							Spec:       specBytes,
						}
						specRevisionBytes, err := json.Marshal(specRevision)
						Expect(err).ToNot(HaveOccurred())

						return specRevisionBytes
					}),
					Entry("using v1alpha1 and VirtualMachineInstancetypeSpecRevision without APIVersion", func() []byte {
						v1alpha1instancetypeSpec := instancetypev1alpha1.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1alpha1.CPUInstancetype{
								Guest: instancetypeObj.Spec.CPU.Guest,
							},
							Memory: instancetypev1alpha1.MemoryInstancetype{
								Guest: instancetypeObj.Spec.Memory.Guest,
							},
						}

						specBytes, err := json.Marshal(&v1alpha1instancetypeSpec)
						Expect(err).ToNot(HaveOccurred())

						specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
							APIVersion: "",
							Spec:       specBytes,
						}
						specRevisionBytes, err := json.Marshal(specRevision)
						Expect(err).ToNot(HaveOccurred())

						return specRevisionBytes
					}),
					Entry("using v1alpha1", func() []byte {
						v1alpha1instancetype := &instancetypev1alpha1.VirtualMachineInstancetype{
							TypeMeta: metav1.TypeMeta{
								APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
								Kind:       "VirtualMachineInstancetype",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: instancetypeObj.Name,
							},
							Spec: instancetypev1alpha1.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1alpha1.CPUInstancetype{
									Guest: instancetypeObj.Spec.CPU.Guest,
								},
								Memory: instancetypev1alpha1.MemoryInstancetype{
									Guest: instancetypeObj.Spec.Memory.Guest,
								},
							},
						}
						instancetypeBytes, err := json.Marshal(v1alpha1instancetype)
						Expect(err).ToNot(HaveOccurred())

						return instancetypeBytes
					}),
					Entry("using v1alpha2", func() []byte {
						v1alpha2instancetype := &instancetypev1alpha2.VirtualMachineInstancetype{
							TypeMeta: metav1.TypeMeta{
								APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
								Kind:       "VirtualMachineInstancetype",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: instancetypeObj.Name,
							},
							Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1alpha2.CPUInstancetype{
									Guest: instancetypeObj.Spec.CPU.Guest,
								},
								Memory: instancetypev1alpha2.MemoryInstancetype{
									Guest: instancetypeObj.Spec.Memory.Guest,
								},
							},
						}
						instancetypeBytes, err := json.Marshal(v1alpha2instancetype)
						Expect(err).ToNot(HaveOccurred())

						return instancetypeBytes
					}),
					Entry("using v1beta1", func() []byte {
						instancetypeBytes, err := json.Marshal(instancetypeObj)
						Expect(err).ToNot(HaveOccurred())

						return instancetypeBytes
					}),
				)

				It("should apply VirtualMachineInstancetype to VirtualMachineInstance if an existing ControllerRevision is present but not referenced by InstancetypeMatcher", func() {
					instancetypeRevision, err := instancetype.CreateControllerRevision(vm, instancetypeObj)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: instancetypeapi.SingularResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeObj.Spec.CPU.Guest))
					Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetypeObj.Spec.Memory.Guest))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.InstancetypeAnnotation, instancetypeObj.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeRevision.Name))

				})

				It("should apply VirtualMachineClusterInstancetype to VirtualMachineInstance", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: clusterInstancetypeObj.Name,
						Kind: instancetypeapi.ClusterSingularResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					expectedRevisionName := instancetype.GetRevisionName(vm.Name, clusterInstancetypeObj.Name, clusterInstancetypeObj.GroupVersionKind().Version, clusterInstancetypeObj.UID, clusterInstancetypeObj.Generation)
					expectedRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetypeObj)
					Expect(err).ToNot(HaveOccurred())

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(clusterInstancetypeObj.Spec.CPU.Guest))
					Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(clusterInstancetypeObj.Spec.Memory.Guest))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.ClusterInstancetypeAnnotation, clusterInstancetypeObj.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Instancetype.RevisionName).To(Equal(expectedRevision.Name))

					revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevisionName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					revisionClusterInstancetype, ok := revision.Data.Object.(*instancetypev1beta1.VirtualMachineClusterInstancetype)
					Expect(ok).To(BeTrue(), "Expected ClusterInstancetype in ControllerRevision")

					Expect(revisionClusterInstancetype.Spec).To(Equal(clusterInstancetypeObj.Spec))
				})

				It("should apply VirtualMachineClusterInstancetype from ControllerRevision to VirtualMachineInstance", func() {
					instancetypeRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetypeObj)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name:         clusterInstancetypeObj.Name,
						Kind:         instancetypeapi.ClusterSingularResourceName,
						RevisionName: instancetypeRevision.Name,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(clusterInstancetypeObj.Spec.CPU.Guest))
					Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(clusterInstancetypeObj.Spec.Memory.Guest))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.ClusterInstancetypeAnnotation, clusterInstancetypeObj.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))

				})

				It("should apply VirtualMachineClusterInstancetype to VirtualMachineInstance if an existing ControllerRevision is present but not referenced by InstancetypeMatcher", func() {
					instancetypeRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetypeObj)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: clusterInstancetypeObj.Name,
						Kind: instancetypeapi.ClusterSingularResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(clusterInstancetypeObj.Spec.CPU.Guest))
					Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(clusterInstancetypeObj.Spec.Memory.Guest))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.ClusterInstancetypeAnnotation, clusterInstancetypeObj.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeRevision.Name))
				})

				It("should reject request if an invalid InstancetypeMatcher Kind is provided", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: "foobar",
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":    Equal(v1.VirtualMachineFailure),
						"Reason":  Equal("FailedCreate"),
						"Message": ContainSubstring("got unexpected kind in InstancetypeMatcher"),
						"Status":  Equal(k8sv1.ConditionTrue),
					}))

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should reject the request if a VirtualMachineInstancetype cannot be found", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: "foobar",
						Kind: instancetypeapi.SingularResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":   Equal(v1.VirtualMachineFailure),
						"Reason": Equal("FailedCreate"),
					}))

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should reject the request if a VirtualMachineClusterInstancetype cannot be found", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: "foobar",
						Kind: instancetypeapi.ClusterSingularResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":   Equal(v1.VirtualMachineFailure),
						"Reason": Equal("FailedCreate"),
					}))

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should fail if the VirtualMachineInstancetype conflicts with the VirtualMachineInstance", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: instancetypeapi.SingularResourceName,
					}

					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						Sockets: uint32(1),
						Cores:   uint32(4),
						Threads: uint32(1),
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":   Equal(v1.VirtualMachineFailure),
						"Reason": Equal("FailedCreate"),
						"Message": And(
							ContainSubstring("Error encountered while storing Instancetype ControllerRevisions: VM field conflicts with selected Instancetype"),
							ContainSubstring("spec.template.spec.domain.cpu"),
						),
					}))
					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
				})

				It("should reject if an existing ControllerRevision is found with unexpected VirtualMachineInstancetypeSpec data", func() {
					unexpectedInstancetype := instancetypeObj.DeepCopy()
					unexpectedInstancetype.Spec.CPU.Guest = 15

					instancetypeRevision, err := instancetype.CreateControllerRevision(vm, unexpectedInstancetype)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: instancetypeapi.SingularResourceName,
					}
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":    Equal(v1.VirtualMachineFailure),
						"Reason":  Equal("FailedCreate"),
						"Message": ContainSubstring("found existing ControllerRevision with unexpected data"),
					}))

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
				})
			})

			Context("preference", func() {
				var (
					preference        *instancetypev1beta1.VirtualMachinePreference
					clusterPreference *instancetypev1beta1.VirtualMachineClusterPreference
				)

				BeforeEach(func() {
					preferenceSpec := instancetypev1beta1.VirtualMachinePreferenceSpec{
						Firmware: &instancetypev1beta1.FirmwarePreferences{
							PreferredUseEfi: pointer.P(true),
						},
						Devices: &instancetypev1beta1.DevicePreferences{
							PreferredDiskBus:        v1.DiskBusVirtio,
							PreferredInterfaceModel: "virtio",
							PreferredInputBus:       v1.InputBusUSB,
							PreferredInputType:      v1.InputTypeTablet,
						},
					}
					preference = &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "preference",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachinePreference",
						},
						Spec: preferenceSpec,
					}
					_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = preferenceInformerStore.Add(preference)
					Expect(err).NotTo(HaveOccurred())

					clusterPreference = &instancetypev1beta1.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "clusterPreference",
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineClusterPreference",
						},
						Spec: preferenceSpec,
					}
					_, err = virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = clusterPreferenceInformerStore.Add(clusterPreference)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should apply VirtualMachinePreference to VirtualMachineInstance", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					expectedPreferenceRevisionName := instancetype.GetRevisionName(vm.Name, preference.Name, preference.GroupVersionKind().Version, preference.UID, preference.Generation)
					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())
					Expect(vmi.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.PreferenceAnnotation, preference.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(expectedPreferenceRevision.Name))

					preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedPreferenceRevisionName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					preferenceRevisionObj, ok := preferenceRevision.Data.Object.(*instancetypev1beta1.VirtualMachinePreference)
					Expect(ok).To(BeTrue(), "Expected Preference in ControllerRevision")
					Expect(preferenceRevisionObj.Spec).To(Equal(preference.Spec))
				})

				DescribeTable("should apply VirtualMachinePreference from ControllerRevision to VirtualMachineInstance", func(getRevisionData func() []byte) {
					preferenceRevision := &appsv1.ControllerRevision{
						ObjectMeta: metav1.ObjectMeta{
							Name: "crName",
						},
						Data: runtime.RawExtension{
							Raw: getRevisionData(),
						},
					}

					preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name:         preference.Name,
						Kind:         instancetypeapi.SingularPreferenceResourceName,
						RevisionName: preferenceRevision.Name,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())
					Expect(vmi.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.PreferenceAnnotation, preference.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))

				},
					Entry("using v1alpha1 and VirtualMachinePreferenceSpecRevision with APIVersion", func() []byte {
						v1alpha1preferenceSpec := instancetypev1alpha1.VirtualMachinePreferenceSpec{
							Firmware: &instancetypev1alpha1.FirmwarePreferences{
								PreferredUseEfi: pointer.P(true),
							},
							Devices: &instancetypev1alpha1.DevicePreferences{
								PreferredDiskBus:        v1.DiskBusVirtio,
								PreferredInterfaceModel: "virtio",
								PreferredInputBus:       v1.InputBusUSB,
								PreferredInputType:      v1.InputTypeTablet,
							},
						}

						specBytes, err := json.Marshal(&v1alpha1preferenceSpec)
						Expect(err).ToNot(HaveOccurred())

						specRevision := instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{
							APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
							Spec:       specBytes,
						}
						specRevisionBytes, err := json.Marshal(specRevision)
						Expect(err).ToNot(HaveOccurred())

						return specRevisionBytes
					}),
					Entry("using v1alpha1 and VirtualMachinePreferenceSpecRevision without APIVersion", func() []byte {
						v1alpha1preferenceSpec := instancetypev1alpha1.VirtualMachinePreferenceSpec{
							Firmware: &instancetypev1alpha1.FirmwarePreferences{
								PreferredUseEfi: pointer.P(true),
							},
							Devices: &instancetypev1alpha1.DevicePreferences{
								PreferredDiskBus:        v1.DiskBusVirtio,
								PreferredInterfaceModel: "virtio",
								PreferredInputBus:       v1.InputBusUSB,
								PreferredInputType:      v1.InputTypeTablet,
							},
						}

						specBytes, err := json.Marshal(&v1alpha1preferenceSpec)
						Expect(err).ToNot(HaveOccurred())

						specRevision := instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{
							APIVersion: "",
							Spec:       specBytes,
						}
						specRevisionBytes, err := json.Marshal(specRevision)
						Expect(err).ToNot(HaveOccurred())

						return specRevisionBytes
					}),
					Entry("using v1alpha1", func() []byte {
						v1alpha1preference := &instancetypev1alpha1.VirtualMachinePreference{
							TypeMeta: metav1.TypeMeta{
								APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
								Kind:       "VirtualMachinePreference",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: preference.Name,
							},
							Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
								Firmware: &instancetypev1alpha1.FirmwarePreferences{
									PreferredUseEfi: pointer.P(true),
								},
								Devices: &instancetypev1alpha1.DevicePreferences{
									PreferredDiskBus:        v1.DiskBusVirtio,
									PreferredInterfaceModel: "virtio",
									PreferredInputBus:       v1.InputBusUSB,
									PreferredInputType:      v1.InputTypeTablet,
								},
							},
						}
						preferenceBytes, err := json.Marshal(v1alpha1preference)
						Expect(err).ToNot(HaveOccurred())

						return preferenceBytes
					}),
					Entry("using v1alpha2", func() []byte {
						v1alpha2preference := &instancetypev1alpha2.VirtualMachinePreference{
							TypeMeta: metav1.TypeMeta{
								APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
								Kind:       "VirtualMachinePreference",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: preference.Name,
							},
							Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
								Firmware: &instancetypev1alpha2.FirmwarePreferences{
									PreferredUseEfi: pointer.P(true),
								},
								Devices: &instancetypev1alpha2.DevicePreferences{
									PreferredDiskBus:        v1.DiskBusVirtio,
									PreferredInterfaceModel: "virtio",
									PreferredInputBus:       v1.InputBusUSB,
									PreferredInputType:      v1.InputTypeTablet,
								},
							},
						}
						preferenceBytes, err := json.Marshal(v1alpha2preference)
						Expect(err).ToNot(HaveOccurred())

						return preferenceBytes
					}),
					Entry("using v1beta1", func() []byte {
						preferenceBytes, err := json.Marshal(preference)
						Expect(err).ToNot(HaveOccurred())

						return preferenceBytes
					}),
				)

				It("should apply VirtualMachinePreference to VirtualMachineInstance if an existing ControllerRevision is present but not referenced by PreferenceMatcher", func() {
					preferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())
					Expect(vmi.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.PreferenceAnnotation, preference.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceRevision.Name))
				})

				It("should apply VirtualMachineClusterPreference to VirtualMachineInstance", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: clusterPreference.Name,
						Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					expectedPreferenceRevisionName := instancetype.GetRevisionName(vm.Name, clusterPreference.Name, clusterPreference.GroupVersionKind().Version, clusterPreference.UID, clusterPreference.Generation)
					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
					Expect(err).ToNot(HaveOccurred())

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())
					Expect(vmi.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, clusterPreference.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(expectedPreferenceRevision.Name))

					preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedPreferenceRevisionName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					preferenceRevisionObj, ok := preferenceRevision.Data.Object.(*instancetypev1beta1.VirtualMachineClusterPreference)
					Expect(ok).To(BeTrue(), "Expected Preference in ControllerRevision")
					Expect(preferenceRevisionObj.Spec).To(Equal(clusterPreference.Spec))
				})

				It("should apply VirtualMachineClusterPreference from ControllerRevision to VirtualMachineInstance", func() {
					preferenceRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name:         clusterPreference.Name,
						Kind:         instancetypeapi.ClusterSingularPreferenceResourceName,
						RevisionName: preferenceRevision.Name,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())
					Expect(vmi.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, clusterPreference.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))

				})

				It("should apply VirtualMachineClusterPreference to VirtualMachineInstance if an existing ControllerRevision is present but not referenced by PreferenceMatcher", func() {
					preferenceRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: clusterPreference.Name,
						Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())
					Expect(vmi.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
					Expect(vmi.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, clusterPreference.Name))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceRevision.Name))
				})

				It("should reject the request if an invalid PreferenceMatcher Kind is provided", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: "foobar",
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())

					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":    Equal(v1.VirtualMachineFailure),
						"Reason":  Equal("FailedCreate"),
						"Message": ContainSubstring("got unexpected kind in PreferenceMatcher"),
					}))

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
				})

				It("should reject the request if a VirtualMachinePreference cannot be found", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: "foobar",
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())

					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":   Equal(v1.VirtualMachineFailure),
						"Reason": Equal("FailedCreate"),
					}))

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
				})

				It("should reject the request if a VirtualMachineClusterPreference cannot be found", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: "foobar",
						Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())

					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":   Equal(v1.VirtualMachineFailure),
						"Reason": Equal("FailedCreate"),
					}))

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should reject if an existing ControllerRevision is found with unexpected VirtualMachinePreferenceSpec data", func() {
					unexpectedPreference := preference.DeepCopy()
					unexpectedPreference.Spec.Firmware = &instancetypev1beta1.FirmwarePreferences{
						PreferredUseBios: pointer.P(true),
					}

					preferenceRevision, err := instancetype.CreateControllerRevision(vm, unexpectedPreference)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())

					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":    Equal(v1.VirtualMachineFailure),
						"Reason":  Equal("FailedCreate"),
						"Message": ContainSubstring("found existing ControllerRevision with unexpected data"),
					}))

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
				})

				It("should apply preferences to default network interface", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm.Spec.Template.Spec.Domain.Devices.Interfaces = []v1.Interface{}
					vm.Spec.Template.Spec.Networks = []v1.Network{}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Devices.Interfaces[0].Model).To(Equal(preference.Spec.Devices.PreferredInterfaceModel))
					Expect(vmi.Spec.Networks).To(Equal([]v1.Network{*v1.DefaultPodNetwork()}))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(expectedPreferenceRevision.Name))
				})

				It("should apply preferredAutoattachPodInterface and skip adding default network interface", func() {

					autoattachPodInterfacePreference := &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "autoattachPodInterfacePreference",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							Devices: &instancetypev1beta1.DevicePreferences{
								PreferredAutoattachPodInterface: pointer.P(false),
							},
						},
					}

					_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), autoattachPodInterfacePreference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: autoattachPodInterfacePreference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm.Spec.Template.Spec.Domain.Devices.Interfaces = []v1.Interface{}
					vm.Spec.Template.Spec.Networks = []v1.Network{}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, autoattachPodInterfacePreference)
					Expect(err).ToNot(HaveOccurred())

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(*vmi.Spec.Domain.Devices.AutoattachPodInterface).To(BeFalse())
					Expect(vmi.Spec.Domain.Devices.Interfaces).To(BeEmpty())
					Expect(vmi.Spec.Networks).To(BeEmpty())

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(expectedPreferenceRevision.Name))
				})

				It("should apply preferences to default volume disk", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					presentVolumeName := "present-vol"
					missingVolumeName := "missing-vol"
					vm.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{
						{
							Name: presentVolumeName,
							DiskDevice: v1.DiskDevice{
								Disk: &v1.DiskTarget{
									Bus: v1.DiskBusSATA,
								},
							},
						},
					}
					vm.Spec.Template.Spec.Volumes = []v1.Volume{
						{
							Name: presentVolumeName,
						},
						{
							Name: missingVolumeName,
						},
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(2))
					Expect(vmi.Spec.Domain.Devices.Disks[0].Name).To(Equal(presentVolumeName))
					// Assert that the preference hasn't overwritten anything defined by the user
					Expect(vmi.Spec.Domain.Devices.Disks[0].Disk.Bus).To(Equal(v1.DiskBusSATA))
					Expect(vmi.Spec.Domain.Devices.Disks[1].Name).To(Equal(missingVolumeName))
					// Assert that it has however been applied to the newly introduced disk
					Expect(vmi.Spec.Domain.Devices.Disks[1].Disk.Bus).To(Equal(preference.Spec.Devices.PreferredDiskBus))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(expectedPreferenceRevision.Name))
				})

				It("should apply preferences to AutoattachInputDevice attached input device", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm.Spec.Template.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Devices.Inputs).To(HaveLen(1))
					Expect(vmi.Spec.Domain.Devices.Inputs[0].Name).To(Equal("default-0"))
					Expect(vmi.Spec.Domain.Devices.Inputs[0].Type).To(Equal(preference.Spec.Devices.PreferredInputType))
					Expect(vmi.Spec.Domain.Devices.Inputs[0].Bus).To(Equal(preference.Spec.Devices.PreferredInputBus))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(expectedPreferenceRevision.Name))
				})

				It("should apply preferences to preferredAutoattachInputDevice attached input device", func() {

					autoattachInputDevicePreference := &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "autoattachInputDevicePreference",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							Devices: &instancetypev1beta1.DevicePreferences{
								PreferredAutoattachInputDevice: pointer.P(true),
								PreferredInputBus:              v1.InputBusVirtio,
								PreferredInputType:             v1.InputTypeTablet,
							},
						},
					}
					_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), autoattachInputDevicePreference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: autoattachInputDevicePreference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, autoattachInputDevicePreference)
					Expect(err).ToNot(HaveOccurred())

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.Devices.Inputs).To(HaveLen(1))
					Expect(vmi.Spec.Domain.Devices.Inputs[0].Name).To(Equal("default-0"))
					Expect(vmi.Spec.Domain.Devices.Inputs[0].Type).To(Equal(autoattachInputDevicePreference.Spec.Devices.PreferredInputType))
					Expect(vmi.Spec.Domain.Devices.Inputs[0].Bus).To(Equal(autoattachInputDevicePreference.Spec.Devices.PreferredInputBus))

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(expectedPreferenceRevision.Name))
				})

				It("should apply preferredAutoattachInputDevice and skip adding default input device", func() {

					autoattachInputDevicePreference := &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "preferredAutoattachInputDevicePreference",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							Devices: &instancetypev1beta1.DevicePreferences{
								PreferredAutoattachInputDevice: pointer.P(false),
							},
						},
					}

					_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), autoattachInputDevicePreference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: autoattachInputDevicePreference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, autoattachInputDevicePreference)
					Expect(err).ToNot(HaveOccurred())

					sanityExecute(vm)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(*vmi.Spec.Domain.Devices.AutoattachInputDevice).To(BeFalse())
					Expect(vmi.Spec.Domain.Devices.Inputs).To(BeEmpty())

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Spec.Preference.RevisionName).To(Equal(expectedPreferenceRevision.Name))
				})
			})
		})

		DescribeTable("should add the default network interface",
			func(iface string, field gstruct.Fields) {
				vm, _ := DefaultVirtualMachine(true)

				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							NetworkConfiguration: &v1.NetworkConfiguration{
								NetworkInterface: iface,
							},
						},
					},
				})

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				sanityExecute(vm)

				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Networks).To(ContainElements(*v1.DefaultPodNetwork()))
				Expect(vmi.Spec.Domain.Devices.Interfaces).To(ContainElements(gstruct.MatchFields(gstruct.IgnoreExtras,
					gstruct.Fields{
						"InterfaceBindingMethod": gstruct.MatchFields(gstruct.IgnoreExtras, field),
					},
				)))

			},
			Entry("as bridge", "bridge", gstruct.Fields{"Bridge": Not(BeNil())}),
			Entry("as masquerade", "masquerade", gstruct.Fields{"Masquerade": Not(BeNil())}),
		)

		It("should reject adding a default deprecated slirp interface", func() {
			vm, _ := DefaultVirtualMachine(true)

			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						NetworkConfiguration: &v1.NetworkConfiguration{
							NetworkInterface:               string(v1.DeprecatedSlirpInterface),
							DeprecatedPermitSlirpInterface: pointer.P(true),
						},
					},
				},
			})

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		DescribeTable("should not add the default interfaces if", func(interfaces []v1.Interface, networks []v1.Network) {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Domain.Devices.Interfaces = append([]v1.Interface{}, interfaces...)
			vm.Spec.Template.Spec.Networks = append([]v1.Network{}, networks...)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Networks).To(ContainElements(networks))
			Expect(vmi.Spec.Domain.Devices.Interfaces).To(ContainElements(interfaces))
		},
			Entry("interfaces and networks are non-empty", []v1.Interface{{Name: "a"}}, []v1.Network{{Name: "b"}}),
			Entry("interfaces is non-empty", []v1.Interface{{Name: "a"}}, []v1.Network{}),
			Entry("networks is non-empty", []v1.Interface{}, []v1.Network{{Name: "b"}}),
		)

		It("should add a missing volume disk", func() {
			vm, _ := DefaultVirtualMachine(true)
			presentVolumeName := "present-vol"
			missingVolumeName := "missing-vol"
			vm.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: presentVolumeName,
				},
			}
			vm.Spec.Template.Spec.Volumes = []v1.Volume{
				{
					Name: presentVolumeName,
				},
				{
					Name: missingVolumeName,
				},
			}

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.Devices.Disks).To(ContainElements(
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(presentVolumeName)}),
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(missingVolumeName)}),
			))

		})

		DescribeTable("AutoattachInputDevice should ", func(autoAttach *bool, existingInputDevices []v1.Input, matcher gomegatypes.GomegaMatcher) {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Domain.Devices.AutoattachInputDevice = autoAttach
			vm.Spec.Template.Spec.Domain.Devices.Inputs = existingInputDevices

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Spec.Domain.Devices.Inputs).To(matcher)

		},
			Entry("add default input device when enabled in VirtualMachine", pointer.P(true), []v1.Input{},
				ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Name": Equal("default-0"),
				}))),
			Entry("not add default input device when disabled by VirtualMachine", pointer.P(false), []v1.Input{},
				BeEmpty()),
			Entry("not add default input device by default", nil, []v1.Input{},
				BeEmpty()),
			Entry("not add default input device when devices already present in VirtualMachine", pointer.P(true), []v1.Input{{Name: "existing-0"}},
				ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Name": Equal("existing-0"),
				}))),
		)

		Context("Live update features", func() {
			const maxSocketsFromSpec uint32 = 24
			const maxSocketsFromConfig uint32 = 48
			maxGuestFromSpec := resource.MustParse("4Gi")
			maxGuestFromConfig := resource.MustParse("8Gi")

			Context("CPU", func() {
				It("should honour the maximum CPU sockets from VM spec", func() {
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{MaxSockets: maxSocketsFromSpec}

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(maxSocketsFromSpec))
				})

				It("should prefer maximum CPU sockets from VM spec rather than from cluster config", func() {
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{MaxSockets: maxSocketsFromSpec}
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								LiveUpdateConfiguration: &v1.LiveUpdateConfiguration{
									MaxCpuSockets: pointer.P(maxSocketsFromConfig),
								},
								VMRolloutStrategy: &liveUpdate,
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(maxSocketsFromSpec))
				})

				DescribeTable("should patch VMI when CPU hotplug is requested", func(resources v1.ResourceRequirements) {
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.Resources = resources
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						Sockets: 2,
					}

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.CPU = &v1.CPU{
						Sockets:    1,
						MaxSockets: 4,
					}
					vmi.Spec.Domain.Resources = resources

					vcpusDelta := int64(vm.Spec.Template.Spec.Domain.CPU.Sockets - vmi.Spec.Domain.CPU.Sockets)
					resourcesDelta := resource.NewMilliQuantity(vcpusDelta*int64(1000*(1.0/float32(config.GetCPUAllocationRatio()))), resource.DecimalSI)

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = controller.handleCPUChangeRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())

					updatedVMI, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(updatedVMI.Spec.Domain.CPU.Sockets).To(Equal(vm.Spec.Template.Spec.Domain.CPU.Sockets))

					if !resources.Requests.Cpu().IsZero() {
						expectedCpuReq := vmi.Spec.Domain.Resources.Requests.Cpu().DeepCopy()
						expectedCpuReq.Add(*resourcesDelta)

						Expect(updatedVMI.Spec.Domain.Resources.Requests.Cpu().String()).To(Equal(expectedCpuReq.String()))
					}

					if !resources.Limits.Cpu().IsZero() {
						expectedCpuLim := vmi.Spec.Domain.Resources.Limits.Cpu().DeepCopy()
						expectedCpuLim.Add(*resourcesDelta)

						Expect(updatedVMI.Spec.Domain.Resources.Limits.Cpu().String()).To(Equal(expectedCpuLim.String()))
					}
				},
					Entry("with no resources set", v1.ResourceRequirements{}),
					Entry("with cpu request set", v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceCPU: resource.MustParse("100m"),
						},
					}),
					Entry("with cpu request and limits set", v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceCPU: resource.MustParse("100m"),
						},
						Limits: k8sv1.ResourceList{
							k8sv1.ResourceCPU: resource.MustParse("400m"),
						},
					}),
				)

				It("should correctly bump requests and limits when multiple hotplugs are performed", func() {
					resources := v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceCPU: resource.MustParse("100m"),
						},
						Limits: k8sv1.ResourceList{
							k8sv1.ResourceCPU: resource.MustParse("200m"),
						},
					}

					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.Resources = resources
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						Sockets: 2,
					}

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.CPU = &v1.CPU{
						Sockets:    1,
						MaxSockets: 4,
					}
					vmi.Spec.Domain.Resources = resources

					originalVMI, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("first hotplug")
					err = controller.handleCPUChangeRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())

					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(vm.Spec.Template.Spec.Domain.CPU.Sockets))

					By("second hotplug")
					vm.Spec.Template.Spec.Domain.CPU.Sockets = 3
					vcpusDelta := int64(vm.Spec.Template.Spec.Domain.CPU.Sockets - originalVMI.Spec.Domain.CPU.Sockets)
					resourcesDelta := resource.NewMilliQuantity(vcpusDelta*int64(1000*(1.0/float32(config.GetCPUAllocationRatio()))), resource.DecimalSI)

					err = controller.handleCPUChangeRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())

					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(vm.Spec.Template.Spec.Domain.CPU.Sockets))

					expectedCpuReq := originalVMI.Spec.Domain.Resources.Requests.Cpu().DeepCopy()
					expectedCpuReq.Add(*resourcesDelta)
					Expect(vmi.Spec.Domain.Resources.Requests.Cpu().String()).To(Equal(expectedCpuReq.String()))

					expectedCpuLim := originalVMI.Spec.Domain.Resources.Limits.Cpu().DeepCopy()
					expectedCpuLim.Add(*resourcesDelta)
					Expect(vmi.Spec.Domain.Resources.Limits.Cpu().String()).To(Equal(expectedCpuLim.String()))
				})

				It("should set a restartRequired condition if NetworkInterfaceMultiQueue is enabled", func() {
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						Sockets: 2,
					}
					vm.Spec.Template.Spec.Domain.Devices.NetworkInterfaceMultiQueue = pointer.P(true)

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.CPU = &v1.CPU{
						Sockets:    1,
						MaxSockets: 4,
					}

					Expect(controller.handleCPUChangeRequest(vm, vmi)).To(Succeed())

					vmCondManager := virtcontroller.NewVirtualMachineConditionManager()
					cond := vmCondManager.GetCondition(vm, v1.VirtualMachineRestartRequired)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":    Equal(v1.VirtualMachineRestartRequired),
						"Message": ContainSubstring("when NetworkInterfaceMultiQueue is enabled"),
						"Status":  Equal(k8sv1.ConditionTrue),
					}))
				})
			})

			Context("Memory", func() {
				It("should honour the max guest memory from VM spec", func() {
					vm, _ := DefaultVirtualMachine(true)
					guestMemory := resource.MustParse("64Mi")
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
						Guest:    &guestMemory,
						MaxGuest: &maxGuestFromSpec,
					}

					vmi := controller.setupVMIFromVM(vm)
					Expect(*vmi.Spec.Domain.Memory.MaxGuest).To(Equal(maxGuestFromSpec))
				})

				It("should prefer maxGuest from VM spec rather than from cluster config", func() {
					vm, _ := DefaultVirtualMachine(true)
					guestMemory := resource.MustParse("64Mi")
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
						Guest:    &guestMemory,
						MaxGuest: &maxGuestFromSpec,
					}
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								LiveUpdateConfiguration: &v1.LiveUpdateConfiguration{
									MaxGuest: &maxGuestFromConfig,
								},
								VMRolloutStrategy: &liveUpdate,
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(*vmi.Spec.Domain.Memory.MaxGuest).To(Equal(maxGuestFromSpec))
				})

				DescribeTable("should patch VMI when memory hotplug is requested", func(resources v1.ResourceRequirements) {
					vm, _ := DefaultVirtualMachine(true)
					newMemory := resource.MustParse("2Gi")
					vm.Spec.Template.Spec.Domain.Resources = resources
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{Guest: &newMemory}
					vm.Spec.Template.Spec.Architecture = "amd64"

					vmi := api.NewMinimalVMI(vm.Name)
					guestMemory := resource.MustParse("1Gi")
					vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory, MaxGuest: &maxGuestFromSpec}
					vmi.Spec.Domain.Resources = resources

					vmi.Status.Memory = &v1.MemoryStatus{
						GuestAtBoot:    &guestMemory,
						GuestCurrent:   &guestMemory,
						GuestRequested: &guestMemory,
					}
					vmiCondManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionTrue,
					})

					vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					Expect(controller.handleMemoryHotplugRequest(vm, vmi)).To(Succeed())

					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					vmConditionController := virtcontroller.NewVirtualMachineConditionManager()
					Expect(vmConditionController.HasCondition(vm, v1.VirtualMachineRestartRequired)).To(BeFalse(), "No RestartRequired condition should be set")

					Expect(vmi.Spec.Domain.Memory.Guest.Cmp(*vm.Spec.Template.Spec.Domain.Memory.Guest)).To(Equal(0), "The VMI Guest should match VM's")

					if !resources.Requests.Memory().IsZero() {
						expectedMemReq := resources.Requests.Memory().Value() + newMemory.Value() - guestMemory.Value()
						Expect(vmi.Spec.Domain.Resources.Requests.Memory().Value()).To(Equal(expectedMemReq))
					}
				},
					Entry("with memory request set", v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse("1Gi"),
						},
					}),
					Entry("with memory request and limits set", v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse("1Gi"),
						},
						Limits: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse("4Gi"),
						},
					}),
				)

				It("should not patch VMI if memory hotplug is already in progress", func() {
					vm, _ := DefaultVirtualMachine(true)
					newMemory := resource.MustParse("128Mi")
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{Guest: &newMemory}
					vm.Spec.Template.Spec.Architecture = "amd64"

					vmi := api.NewMinimalVMI(vm.Name)
					guestMemory := resource.MustParse("1Gi")
					vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory, MaxGuest: &maxGuestFromSpec}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &v1.MemoryStatus{
						GuestAtBoot:    &guestMemory,
						GuestCurrent:   &guestMemory,
						GuestRequested: &guestMemory,
					}

					vmiCondManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionTrue,
					})

					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceMemoryChange,
						Status: k8sv1.ConditionTrue,
					})

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).To(HaveOccurred())
				})

				It("should not patch VMI if a migration is in progress", func() {
					vm, _ := DefaultVirtualMachine(true)
					newMemory := resource.MustParse("128Mi")
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{Guest: &newMemory}
					vm.Spec.Template.Spec.Architecture = "amd64"

					vmi := api.NewMinimalVMI(vm.Name)
					guestMemory := resource.MustParse("1Gi")
					vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory, MaxGuest: &maxGuestFromSpec}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &v1.MemoryStatus{
						GuestAtBoot:  &guestMemory,
						GuestCurrent: &guestMemory,
					}
					vmi.Spec.Architecture = "amd64"
					vmiCondManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionTrue,
					})

					migrationStart := metav1.Now()
					vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
						StartTimestamp: &migrationStart,
					}

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).To(HaveOccurred())
				})

				It("should not patch VMI if guest memory did not change", func() {
					guestMemory := resource.MustParse("1Gi")
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}
					vm.Spec.Template.Spec.Architecture = "amd64"

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory, MaxGuest: &maxGuestFromSpec}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &v1.MemoryStatus{
						GuestAtBoot:  &guestMemory,
						GuestCurrent: &guestMemory,
					}
					vmi.Spec.Architecture = "amd64"
					vmiCondManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionTrue,
					})

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())
				})

				It("should set a restartRequired condition if the memory decreased from start", func() {
					guestMemory := resource.MustParse("2Gi")
					newMemory := resource.MustParse("1Gi")
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{Guest: &newMemory}
					vm.Spec.Template.Spec.Architecture = "amd64"

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory, MaxGuest: &maxGuestFromSpec}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &v1.MemoryStatus{
						GuestAtBoot:  &guestMemory,
						GuestCurrent: &guestMemory,
					}
					vmi.Spec.Architecture = "amd64"
					vmiCondManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionTrue,
					})

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())

					vmConditionController := virtcontroller.NewVirtualMachineConditionManager()
					Expect(vmConditionController.HasCondition(vm, v1.VirtualMachineRestartRequired)).To(BeTrue())
				})

				It("should set a restartRequired condition if VM does not support memory hotplug", func() {
					guestMemory := resource.MustParse("2Gi")
					newMemory := resource.MustParse("4Gi")
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{Guest: &newMemory}
					vm.Spec.Template.Spec.Architecture = "risc-v"

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory, MaxGuest: &maxGuestFromSpec}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &v1.MemoryStatus{
						GuestAtBoot:  &guestMemory,
						GuestCurrent: &guestMemory,
					}
					vmi.Spec.Architecture = "amd64"
					vmiCondManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionTrue,
					})

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())

					Expect(virtFakeClient.Actions()).To(WithTransform(func(actions []testing.Action) []testing.Action {
						var patchActions []testing.Action
						for _, action := range actions {
							if action.GetVerb() == "patch" && action.GetResource().Resource == "virtualmachineinstances" {
								patchActions = append(patchActions, action)
							}
						}
						return patchActions
					}, BeEmpty()))

					vmCondManager := virtcontroller.NewVirtualMachineConditionManager()
					cond := vmCondManager.GetCondition(vm, v1.VirtualMachineRestartRequired)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":    Equal(v1.VirtualMachineRestartRequired),
						"Message": ContainSubstring("memory hotplug not supported"),
						"Status":  Equal(k8sv1.ConditionTrue),
					}))
				})

				It("should set a restartRequired condition if memory hotplug failed", func() {
					guestMemory := resource.MustParse("1Gi")
					newMemory := resource.MustParse("2Gi")
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{Guest: &newMemory}
					vm.Spec.Template.Spec.Architecture = "amd64"

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory, MaxGuest: &maxGuestFromSpec}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &v1.MemoryStatus{
						GuestAtBoot:  &guestMemory,
						GuestCurrent: &guestMemory,
					}
					vmi.Spec.Architecture = "amd64"
					vmiCondManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionTrue,
					})
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceMemoryChange,
						Status: k8sv1.ConditionFalse,
					})

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())

					Expect(virtFakeClient.Actions()).To(WithTransform(func(actions []testing.Action) []testing.Action {
						var patchActions []testing.Action
						for _, action := range actions {
							if action.GetVerb() == "patch" && action.GetResource().Resource == "virtualmachineinstances" {
								patchActions = append(patchActions, action)
							}
						}
						return patchActions
					}, BeEmpty()))

					vmCondManager := virtcontroller.NewVirtualMachineConditionManager()
					cond := vmCondManager.GetCondition(vm, v1.VirtualMachineRestartRequired)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":    Equal(v1.VirtualMachineRestartRequired),
						"Message": ContainSubstring("memory updated in template spec. Memory-hotplug failed and is not available for this VM configuration"),
						"Status":  Equal(k8sv1.ConditionTrue),
					}))
				})

				It("should set a restartRequired condition if memory hotplug is incompatible with the VM configuration", func() {
					guestMemory := resource.MustParse("1Gi")
					newMemory := resource.MustParse("2Gi")
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{Guest: &newMemory}
					vm.Spec.Template.Spec.Architecture = "risc-v"

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory, MaxGuest: nil}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &v1.MemoryStatus{
						GuestAtBoot:  &guestMemory,
						GuestCurrent: &guestMemory,
					}
					vmiCondManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionTrue,
					})

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())

					Expect(virtFakeClient.Actions()).To(WithTransform(func(actions []testing.Action) []testing.Action {
						var patchActions []testing.Action
						for _, action := range actions {
							if action.GetVerb() == "patch" && action.GetResource().Resource == "virtualmachineinstances" {
								patchActions = append(patchActions, action)
							}
						}
						return patchActions
					}, BeEmpty()))

					vmCondManager := virtcontroller.NewVirtualMachineConditionManager()
					cond := vmCondManager.GetCondition(vm, v1.VirtualMachineRestartRequired)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":    Equal(v1.VirtualMachineRestartRequired),
						"Message": ContainSubstring("memory updated in template spec. Memory-hotplug is not available for this VM configuration"),
						"Status":  Equal(k8sv1.ConditionTrue),
					}))
				})

				It("should set a restartRequired condition if VM is not migratable", func() {
					guestMemory := resource.MustParse("1Gi")
					newMemory := resource.MustParse("2Gi")
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{Guest: &newMemory}
					vm.Spec.Template.Spec.Architecture = "amd64"

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory, MaxGuest: nil}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &v1.MemoryStatus{
						GuestAtBoot:  &guestMemory,
						GuestCurrent: &guestMemory,
					}

					vmiCondManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
					vmiCondManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionFalse,
					})

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())

					Expect(virtFakeClient.Actions()).To(WithTransform(func(actions []testing.Action) []testing.Action {
						var patchActions []testing.Action
						for _, action := range actions {
							if action.GetVerb() == "patch" && action.GetResource().Resource == "virtualmachineinstances" {
								patchActions = append(patchActions, action)
							}
						}
						return patchActions
					}, BeEmpty()))

					vmCondManager := virtcontroller.NewVirtualMachineConditionManager()
					cond := vmCondManager.GetCondition(vm, v1.VirtualMachineRestartRequired)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":    Equal(v1.VirtualMachineRestartRequired),
						"Message": ContainSubstring("memory updated in template spec. Memory-hotplug is only available for migratable VMs"),
						"Status":  Equal(k8sv1.ConditionTrue),
					}))
				})
			})

			Context("Affinity", func() {
				It("should be live-updated", func() {
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &liveUpdate,
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vm, vmi := DefaultVirtualMachine(true)

					affinity := k8sv1.Affinity{
						PodAffinity: &k8sv1.PodAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"test": "nmc",
										},
									},
								},
							},
						},
					}
					vm.Spec.Template.Spec.Affinity = affinity.DeepCopy()

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())

					addVirtualMachine(vm)

					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(controller.vmiIndexer.Add(vmi)).To(Succeed())

					sanityExecute(vm)

					By("Expecting to see patch for the VMI with new affinity")
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmi.Spec.Affinity).To(Not(BeNil()))
					Expect(*vmi.Spec.Affinity).To(Equal(affinity))
				})

			})

			Context("Volumes", func() {
				DescribeTable("should set the restart condition", func(strategy *v1.UpdateVolumesStrategy) {
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &liveUpdate,
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{
										virtconfig.VMLiveUpdateFeaturesGate,
										virtconfig.VolumesUpdateStrategy,
									},
								},
							},
						},
					})
					vm, vmi := DefaultVirtualMachine(true)
					vm.Spec.UpdateVolumesStrategy = strategy
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "vol1"})
					vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{Name: "vol2"})
					controller.handleVolumeUpdateRequest(vm, vmi)
					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineRestartRequired)
					Expect(cond).ToNot(BeNil())
					Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
					Expect(cond.Message).To(Equal("the volumes replacement is effective only after restart"))
				},
					Entry("without the updateVolumeStrategy field", nil),
					Entry("with the replacement updateVolumeStrategy",
						pointer.P(v1.UpdateVolumesStrategyReplacement)),
				)

				It("should set the restart condition with the Migration updateVolumeStrategy if volumes cannot be migrated", func() {
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &liveUpdate,
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{
										virtconfig.VMLiveUpdateFeaturesGate,
										virtconfig.VolumesUpdateStrategy,
										virtconfig.VolumeMigration,
									},
								},
							},
						},
					})
					vm, vmi := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "vol1"})
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "vol3",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "updated-vol3"}},
						},
					})
					vm.Spec.UpdateVolumesStrategy = pointer.P(v1.UpdateVolumesStrategyMigration)
					vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems,
						v1.Filesystem{
							Name:     "vol3",
							Virtiofs: &v1.FilesystemVirtiofs{},
						})
					vmi.Spec.Volumes = append(vmi.Spec.Volumes,
						v1.Volume{Name: "vol2"},
						v1.Volume{
							Name: "vol3",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "vol3"}},
							},
						},
					)
					controller.handleVolumeUpdateRequest(vm, vmi)
					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineRestartRequired)
					Expect(cond).ToNot(BeNil())
					Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
					Expect(cond.Message).To(ContainSubstring("invalid volumes to update with migration:"))
				})
			})

			Context("Instance Types and Preferences", func() {
				const resourceUID types.UID = "9160e5de-2540-476a-86d9-af0081aee68a"
				const resourceGeneration int64 = 1

				var (
					originalVM *v1.VirtualMachine
					updatedVM  *v1.VirtualMachine

					controllerrevisionInformerStore cache.Store

					originalInstancetype *instancetypev1beta1.VirtualMachineInstancetype
				)

				BeforeEach(func() {
					originalVM, _ = DefaultVirtualMachine(true)
					originalVM.Spec.Template.Spec.Domain = v1.DomainSpec{}

					controllerrevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
					controllerrevisionInformerStore = controllerrevisionInformer.GetStore()

					originalInstancetype = &instancetypev1beta1.VirtualMachineInstancetype{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineInstancetype",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:       "instancetype",
							Namespace:  originalVM.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1beta1.CPUInstancetype{
								Guest: uint32(2),
							},
							Memory: instancetypev1beta1.MemoryInstancetype{
								Guest: resource.MustParse("128M"),
							},
						},
					}

					revision, err := instancetype.CreateControllerRevision(originalVM, originalInstancetype)
					Expect(err).ToNot(HaveOccurred())

					err = controllerrevisionInformerStore.Add(revision)
					Expect(err).NotTo(HaveOccurred())

					originalVM.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name:         originalInstancetype.Name,
						Kind:         instancetypeapi.SingularResourceName,
						RevisionName: revision.Name,
					}

					controller.instancetypeMethods = &instancetype.InstancetypeMethods{
						ControllerRevisionStore: controllerrevisionInformerStore,
						Clientset:               virtClient,
					}

					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &liveUpdate,
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})
				})

				It("should not add addRestartRequiredIfNeeded to VM if live-updatable", func() {
					updatedInstancetype := originalInstancetype.DeepCopy()
					updatedInstancetype.Generation = originalInstancetype.Generation + 1
					updatedInstancetype.Spec.CPU.Guest = uint32(2)
					updatedInstancetype.Spec.Memory.Guest = resource.MustParse("256M")

					updatedRevision, err := instancetype.CreateControllerRevision(originalVM, originalInstancetype)
					Expect(err).ToNot(HaveOccurred())

					Expect(controllerrevisionInformerStore.Add(updatedRevision)).To(Succeed())

					updatedVM = originalVM.DeepCopy()
					updatedVM.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name:         updatedInstancetype.Name,
						Kind:         instancetypeapi.SingularResourceName,
						RevisionName: updatedRevision.Name,
					}

					Expect(controller.addRestartRequiredIfNeeded(&originalVM.Spec, updatedVM)).To(BeFalse())
					vmConditionController := virtcontroller.NewVirtualMachineConditionManager()
					Expect(vmConditionController.HasCondition(updatedVM, v1.VirtualMachineRestartRequired)).To(BeFalse())
				})
				It("should add addRestartRequiredIfNeeded to VM if not live-updatable", func() {
					updatedInstancetype := originalInstancetype.DeepCopy()
					updatedInstancetype.Generation = originalInstancetype.Generation + 1
					updatedInstancetype.Spec.CPU.Guest = uint32(2)
					updatedInstancetype.Spec.Memory.Guest = resource.MustParse("256M")

					// Enabling DedicatedCPUPlacement is not a supported live updatable attribute and should require a reboot
					updatedInstancetype.Spec.CPU.DedicatedCPUPlacement = pointer.P(true)

					updatedRevision, err := instancetype.CreateControllerRevision(originalVM, updatedInstancetype)
					Expect(err).ToNot(HaveOccurred())

					Expect(controllerrevisionInformerStore.Add(updatedRevision)).To(Succeed())

					updatedVM = originalVM.DeepCopy()
					updatedVM.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name:         updatedInstancetype.Name,
						Kind:         instancetypeapi.SingularResourceName,
						RevisionName: updatedRevision.Name,
					}

					Expect(controller.addRestartRequiredIfNeeded(&originalVM.Spec, updatedVM)).To(BeTrue())
					vmConditionController := virtcontroller.NewVirtualMachineConditionManager()
					Expect(vmConditionController.HasCondition(updatedVM, v1.VirtualMachineRestartRequired)).To(BeTrue())
				})
			})
		})

		Context("The RestartRequired condition", func() {
			var vm *v1.VirtualMachine
			var vmi *v1.VirtualMachineInstance
			var kv *v1.KubeVirt

			restartRequiredMatcher := func(status k8sv1.ConditionStatus) gomegatypes.GomegaMatcher {
				return ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":   Equal(v1.VirtualMachineRestartRequired),
					"Status": Equal(status),
				}))
			}

			BeforeEach(func() {
				vm, vmi = DefaultVirtualMachine(true)
				vm.ObjectMeta.UID = types.UID(uuid.NewString())
				vmi.ObjectMeta.UID = vm.ObjectMeta.UID
				vm.Generation = 1
				vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
					Cores: 2,
				}
				guest := resource.MustParse("128Mi")
				vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
					Guest: &guest,
				}
				kv = &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							LiveUpdateConfiguration: &v1.LiveUpdateConfiguration{},
							VMRolloutStrategy:       &liveUpdate,
							DeveloperConfiguration: &v1.DeveloperConfiguration{
								FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
							},
						},
					},
				}
			})

			It("should not call update if the condition is set", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.ObjectMeta.UID = types.UID(uuid.NewString())
				vmi.ObjectMeta.UID = vm.ObjectMeta.UID
				vm.Generation = 1
				vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
					Cores: 2,
				}
				guest := resource.MustParse("128Mi")
				vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
					Guest: &guest,
				}
				kv := &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							LiveUpdateConfiguration: &v1.LiveUpdateConfiguration{},
							VMRolloutStrategy:       &liveUpdate,
							DeveloperConfiguration: &v1.DeveloperConfiguration{
								FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
							},
						},
					},
				}
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), kv)

				var maxSockets uint32 = 8

				By("Setting a cluster-wide CPU maxSockets value")
				kv.Spec.Configuration.LiveUpdateConfiguration.MaxCpuSockets = pointer.P(maxSockets)
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), kv)

				By("Creating a VM with CPU sockets set to the cluster maxiumum")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = maxSockets
				controller.crIndexer.Add(createVMRevision(vm))

				By("Creating a VMI with cluster max")
				vmi = controller.setupVMIFromVM(vm)
				vmi.Status.Phase = v1.Running
				controller.vmiIndexer.Add(vmi)

				By("Bumping the VM sockets above the cluster maximum")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 10
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				virtFakeClient.PrependReactor("update", "virtualmachines", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
					if action.GetSubresource() != "" {
						return false, nil, nil
					}

					Fail("Update should not be called")
					return true, &v1.VirtualMachine{}, fmt.Errorf("conflict")
				})

				By("Executing the controller expecting the RestartRequired condition to appear")
				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.Conditions).To(restartRequiredMatcher(k8sv1.ConditionTrue))

			})

			It("should appear when changing a non-live-updatable field", func() {
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), kv)

				By("Creating a VMI with hostname 'a'")
				vm.Spec.Template.Spec.Hostname = "a"
				vmi = controller.setupVMIFromVM(vm)
				controller.vmiIndexer.Add(vmi)

				By("Creating a Controller Revision with the hostname 'a'")
				controller.crIndexer.Add(createVMRevision(vm))

				By("Changing the hostname to 'b'")
				vm.Spec.Template.Spec.Hostname = "b"
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				By("Executing the controller expecting the RestartRequired condition to appear")
				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.Conditions).To(restartRequiredMatcher(k8sv1.ConditionTrue), "restart required")
			})

			It("should appear when VM doesn't specify maxSockets and sockets go above cluster-wide maxSockets", func() {
				var maxSockets uint32 = 8

				By("Creating a VM with CPU sockets set to the cluster maxiumum")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = maxSockets
				vm.Spec.Template.Spec.Domain.CPU.MaxSockets = maxSockets
				controller.crIndexer.Add(createVMRevision(vm))

				By("Creating a VMI with cluster max")
				vmi = controller.setupVMIFromVM(vm)
				controller.vmiIndexer.Add(vmi)

				By("Bumping the VM sockets above the cluster maximum")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 10
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				By("Executing the controller expecting the RestartRequired condition to appear")
				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.Conditions).To(restartRequiredMatcher(k8sv1.ConditionTrue), "restart required")
			})

			It("should appear when VM doesn't specify maxGuest and guest memory goes above cluster-wide maxGuest", func() {
				var maxGuest = resource.MustParse("256Mi")

				By("Creating a VM with guest memory set to the cluster maximum")
				vm.Spec.Template.Spec.Domain.Memory.Guest = &maxGuest
				vm.Spec.Template.Spec.Domain.Memory.MaxGuest = &maxGuest
				controller.crIndexer.Add(createVMRevision(vm))

				By("Creating a VMI")
				vmi = controller.setupVMIFromVM(vm)
				markAsReady(vmi)
				vmi.Status.Memory = &v1.MemoryStatus{
					GuestAtBoot:  &maxGuest,
					GuestCurrent: &maxGuest,
				}
				controller.vmiIndexer.Add(vmi)

				By("Bumping the VM guest memory above the cluster maximum")
				bigGuest := resource.MustParse("257Mi")
				vm.Spec.Template.Spec.Domain.Memory.Guest = &bigGuest

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				By("Executing the controller expecting the RestartRequired condition to appear")
				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.Conditions).To(restartRequiredMatcher(k8sv1.ConditionTrue), "restart required")
			})

			It("should appear when VM sockets count is reduced", func() {
				By("Creating a VM with two sockets")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 2

				vmi = controller.setupVMIFromVM(vm)
				controller.vmiIndexer.Add(vmi)

				By("Creating a Controller Revision with two sockets")
				controller.crIndexer.Add(createVMRevision(vm))

				By("Reducing the sockets count to one")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = vm.Spec.Template.Spec.Domain.CPU.Sockets - 1
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				addVirtualMachine(vm)

				By("Executing the controller expecting the RestartRequired condition to appear")
				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(vm.Status.Conditions).To(restartRequiredMatcher(k8sv1.ConditionTrue), "restart required")
			})

			DescribeTable("when changing a live-updatable field", func(fgs []string, strat *v1.VMRolloutStrategy, matcher gomegatypes.GomegaMatcher) {
				// Add necessary stuff to reflect running VM
				// TODO: This should be done in more places
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.PrintableStatus = "Running"
				virtcontroller.NewVirtualMachineConditionManager().UpdateCondition(vm, &v1.VirtualMachineCondition{
					Type:   v1.VirtualMachineReady,
					Status: k8sv1.ConditionTrue,
				})
				kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = fgs
				kv.Spec.Configuration.VMRolloutStrategy = strat
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), kv)

				By("Creating a VM with CPU sockets set to 2")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 2
				vm.Spec.Template.Spec.Domain.CPU.MaxSockets = 8
				controller.crIndexer.Add(createVMRevision(vm))

				By("Creating a VMI")
				vmi = controller.setupVMIFromVM(vm)
				markAsReady(vmi)
				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				By("Bumping the VM sockets to 4")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 4

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				By("Executing the controller expecting the RestartRequired condition to appear")
				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.Conditions).To(matcher, "restart Required")

				if strat == &liveUpdate && len(fgs) > 0 {
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(4)))
				}
			},
				Entry("should appear if the feature gate is not set",
					[]string{}, &liveUpdate, restartRequiredMatcher(k8sv1.ConditionTrue)),
				Entry("should appear if the VM rollout strategy is not set",
					[]string{virtconfig.VMLiveUpdateFeaturesGate}, nil, restartRequiredMatcher(k8sv1.ConditionTrue)),
				Entry("should appear if the VM rollout strategy is set to Stage",
					[]string{virtconfig.VMLiveUpdateFeaturesGate}, &stage, restartRequiredMatcher(k8sv1.ConditionTrue)),
				Entry("should not appear if both the VM rollout strategy and feature gate are set",
					[]string{virtconfig.VMLiveUpdateFeaturesGate}, &liveUpdate, Not(restartRequiredMatcher(k8sv1.ConditionTrue))),
			)
		})

		Context("RunStrategy", func() {

			It("when starting a VM with RerunOnFailure the VM should get started", func() {
				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyRerunOnFailure)

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				addVirtualMachine(vm)

				sanityExecute(vm)

				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi).ToNot(BeNil())
			})

			It("when starting a VM with Manual and DVs are not ready the start request should not get trimmed", func() {
				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyManual)
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "test1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				})
				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dv1",
					},
				})

				shouldFailDataVolumeCreationNoResourceFound()

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				addVirtualMachine(vm)

				sanityExecute(vm)

				// update VM from the store
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))

				By("Add Start Request")
				startRequest := v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest}
				vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, startRequest)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).UpdateStatus(context.TODO(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				addVirtualMachine(vm)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vm).ToNot(BeNil())

				Expect(vm.Status.StateChangeRequests).ToNot(BeEmpty())
				Expect(vm.Status.StateChangeRequests).To(ContainElement(startRequest))
			})

			DescribeTable("The VM should get started when switching to RerunOnFailure from", func(runStrategy v1.VirtualMachineRunStrategy) {
				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = pointer.P(runStrategy)

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				addVirtualMachine(vm)

				sanityExecute(vm)

				_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))

				By("Change RunStrategy")
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyRerunOnFailure)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Update(context.TODO(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				addVirtualMachine(vm)

				sanityExecute(vm)

				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi).ToNot(BeNil())
			},
				Entry("Halted", v1.RunStrategyHalted),
				Entry("Manual", v1.RunStrategyManual),
			)

			PIt("The VM should get restarted when doing RerunOnFailure -> Halted -> RerunOnFailure", func() {
				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyRerunOnFailure)

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				addVirtualMachine(vm)
				sanityExecute(vm)

				controller.crIndexer.Add(createVMRevision(vm))

				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi).ToNot(BeNil())

				// let the controller pick up the creation
				controller.vmiIndexer.Add(vmi)
				key, err := virtcontroller.KeyFunc(vm)
				Expect(err).To(Not(HaveOccurred()))
				controller.Queue.Add(key)
				sanityExecute(vm)

				By("Change RunStrategy to Halted")
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyHalted)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Update(context.TODO(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())

				addVirtualMachine(vm)
				sanityExecute(vm)

				controller.crIndexer.Delete(createVMRevision(vm))

				// let the controller pick up the deletion
				controller.Queue.Add(key)
				controller.vmiIndexer.Delete(vmi)
				sanityExecute(vm)

				_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))

				By("Change RunStrategy back to RerunOnFailure")
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyRerunOnFailure)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Update(context.TODO(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())

				addVirtualMachine(vm)
				sanityExecute(vm)

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi).ToNot(BeNil())
			})

			clearExpectations := func(vm *v1.VirtualMachine) {
				//Clear all expectations
				key, err := virtcontroller.KeyFunc(vm)
				Expect(err).To(Not(HaveOccurred()))
				controller.expectations.SetExpectations(key, 0, 0)
			}

			It("when issuing a restart a VM with Always should get restarted", func() {
				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)

				key, err := virtcontroller.KeyFunc(vm)
				Expect(err).To(Not(HaveOccurred()))

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				addVirtualMachine(vm)
				sanityExecute(vm)
				clearExpectations(vm)

				controller.crIndexer.Add(createVMRevision(vm))

				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				// let the controller pick up the creation
				controller.vmiIndexer.Add(vmi)

				controller.Queue.Add(key)
				sanityExecute(vm)
				clearExpectations(vm)

				// update VM from the store
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Add a stop and start Request")
				vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID})
				vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest})

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).UpdateStatus(context.TODO(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("VM should get stopped")
				addVirtualMachine(vm)
				sanityExecute(vm)
				clearExpectations(vm)

				_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))

				By("VM should get now restarted")
				// pick up deletion
				controller.crIndexer.Delete(createVMRevision(vm))
				controller.vmiIndexer.Delete(vmi)
				controller.Queue.Add(key)
				sanityExecute(vm)
				clearExpectations(vm)

				// check for VMI existence
				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi).ToNot(BeNil())

				// update VM from Store
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(vm.Status.StateChangeRequests).ToNot(ContainElement(v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID}))
				Expect(vm.Status.StateChangeRequests).ToNot(BeEmpty())

				// let the controller pick up the creation
				controller.vmiIndexer.Add(vmi)
				controller.Queue.Add(key)
				addVirtualMachine(vm)
				sanityExecute(vm)

				// update VM from Store
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vm.Status.StateChangeRequests).To(BeEmpty())

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(vmi).ToNot(BeNil())
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})
	Context("syncConditions", func() {
		var vm *v1.VirtualMachine
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vm, vmi = DefaultVirtualMachineWithNames(false, "test", "test")
		})

		It("should set ready to false when VMI is nil", func() {
			syncConditions(vm, nil, nil)
			Expect(vm.Status.Conditions).To(HaveLen(1))
			Expect(vm.Status.Conditions[0].Type).To(Equal(v1.VirtualMachineReady))
			Expect(vm.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionFalse))
		})

		It("should set ready to false when VMI doesn't have a ready condition", func() {
			syncConditions(vm, vmi, nil)
			Expect(vm.Status.Conditions).To(HaveLen(1))
			Expect(vm.Status.Conditions[0].Type).To(Equal(v1.VirtualMachineReady))
			Expect(vm.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionFalse))
		})

		It("should set ready to false when VMI has a false ready condition", func() {
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{
				Type:   v1.VirtualMachineInstanceReady,
				Status: k8sv1.ConditionFalse,
			}}
			syncConditions(vm, vmi, nil)
			Expect(vm.Status.Conditions).To(HaveLen(1))
			Expect(vm.Status.Conditions[0].Type).To(Equal(v1.VirtualMachineReady))
			Expect(vm.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionFalse))
		})

		It("should sync appropriate conditions and ignore others", func() {
			fromCondList := []v1.VirtualMachineConditionType{
				v1.VirtualMachineReady, v1.VirtualMachineFailure,
				v1.VirtualMachinePaused, v1.VirtualMachineRestartRequired,
			}
			toCondList := []v1.VirtualMachineConditionType{
				v1.VirtualMachineReady, v1.VirtualMachinePaused,
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{}
			for _, cond := range fromCondList {
				vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
					Type:   v1.VirtualMachineInstanceConditionType(cond),
					Status: k8sv1.ConditionTrue,
				})
			}
			syncConditions(vm, vmi, nil)
			Expect(vm.Status.Conditions).To(HaveLen(len(toCondList)))
			for _, cond := range vm.Status.Conditions {
				Expect(toCondList).To(ContainElements(cond.Type))
			}
		})
	})

	Context("Live updates", func() {
		createPVCVol := func(volName, claimName string, hotpluggable bool) v1.Volume {
			return v1.Volume{
				Name: volName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						Hotpluggable: hotpluggable,
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: claimName,
						},
					},
				}}
		}
		createDisk := func(name string) v1.Disk {
			return v1.Disk{
				Name: name,
			}
		}
		DescribeTable("should be validated for volume updates", func(oldVols, newVols []v1.Volume, expectValid bool) {
			oldVm, _ := DefaultVirtualMachine(true)
			newVm := oldVm.DeepCopy()
			oldVm.Spec.Template.Spec.Volumes = oldVols
			newVm.Spec.Template.Spec.Volumes = newVols

			Expect(validLiveUpdateVolumes(&oldVm.Spec, newVm)).To(Equal(expectValid))
		},
			Entry("without changes", []v1.Volume{createPVCVol("vol1", "test1", false)}, []v1.Volume{createPVCVol("vol1", "test1", false)}, true),
			Entry("for container disks", []v1.Volume{{Name: "vol1", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "test1"}}}},
				[]v1.Volume{{Name: "vol1", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "test2"}}}}, false),
			Entry("for a replaced pvc", []v1.Volume{createPVCVol("vol1", "test1", false)}, []v1.Volume{createPVCVol("vol1", "test2", false)}, false),
			Entry("for an added pvc", []v1.Volume{}, []v1.Volume{createPVCVol("vol1", "test1", false)}, false),
			Entry("for a removed pvc", []v1.Volume{createPVCVol("vol1", "test1", false)}, []v1.Volume{}, false),
			Entry("for an updated hotpluggable pvc", []v1.Volume{createPVCVol("vol1", "test1", true)}, []v1.Volume{createPVCVol("vol1", "test2", true)}, true),
			Entry("for an added hotpluggable pvc", []v1.Volume{}, []v1.Volume{createPVCVol("vol1", "test1", true)}, true),
			Entry("for an added hotpluggable pvc as first volume", []v1.Volume{createPVCVol("vol2", "test2", false)}, []v1.Volume{createPVCVol("vol1", "test1", true),
				createPVCVol("vol2", "test2", false)}, true),
			Entry("for a replaced hotpluggable pvc", []v1.Volume{createPVCVol("vol1", "test1", true)}, []v1.Volume{createPVCVol("vol1", "test2", true)}, true),
			Entry("for a removed hotpluggable pvc", []v1.Volume{createPVCVol("vol1", "test1", true)}, []v1.Volume{}, true),
		)
		DescribeTable("should be validated for disk updates", func(oldVols, newVols []v1.Volume, oldDisks, newDisks []v1.Disk, expectValid bool) {
			oldVm, _ := DefaultVirtualMachine(true)
			newVm := oldVm.DeepCopy()
			oldVm.Spec.Template.Spec.Volumes = oldVols
			newVm.Spec.Template.Spec.Volumes = newVols
			oldVm.Spec.Template.Spec.Domain.Devices.Disks = oldDisks
			newVm.Spec.Template.Spec.Domain.Devices.Disks = newDisks
			Expect(validLiveUpdateDisks(&oldVm.Spec, newVm)).To(Equal(expectValid))
		},
			Entry("without changes", []v1.Volume{createPVCVol("vol1", "test1", false)}, []v1.Volume{createPVCVol("vol1", "test1", false)}, []v1.Disk{createDisk("vol1")}, []v1.Disk{createDisk("vol1")}, true),
			Entry("for container disks", []v1.Volume{{Name: "vol1", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "test1"}}}},
				[]v1.Volume{{Name: "vol1", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "test2"}}}},
				[]v1.Disk{createDisk("vol1")}, []v1.Disk{createDisk("vol2")}, false),
			Entry("for a replaced pvc", []v1.Volume{createPVCVol("vol1", "test1", false)}, []v1.Volume{createPVCVol("vol2", "test2", false)},
				[]v1.Disk{createDisk("vol1")}, []v1.Disk{createDisk("vol2")}, false),
			Entry("for an added pvc", []v1.Volume{}, []v1.Volume{createPVCVol("vol1", "test1", false)},
				[]v1.Disk{}, []v1.Disk{createDisk("vol1")}, false),
			Entry("for a removed pvc", []v1.Volume{createPVCVol("vol1", "test1", false)}, []v1.Volume{}, []v1.Disk{createDisk("vol1")}, []v1.Disk{}, false),
			Entry("for a replaced hotpluggable pvc", []v1.Volume{createPVCVol("vol1", "test1", true)}, []v1.Volume{createPVCVol("vol2", "test2", true)},
				[]v1.Disk{createDisk("vol1")}, []v1.Disk{createDisk("vol2")}, true),
			Entry("for an updated hotpluggable pvc", []v1.Volume{createPVCVol("vol1", "test1", true)}, []v1.Volume{createPVCVol("vol1", "test2", true)},
				[]v1.Disk{createDisk("vol1")}, []v1.Disk{createDisk("vol1")}, true),
			Entry("for an added hotpluggable pvc", []v1.Volume{}, []v1.Volume{createPVCVol("vol1", "test1", true)},
				[]v1.Disk{}, []v1.Disk{createDisk("vol1")}, true),
			Entry("for an added hotpluggable pvc as first volume", []v1.Volume{createPVCVol("vol2", "test2", false)}, []v1.Volume{createPVCVol("vol1", "test1", true),
				createPVCVol("vol2", "test2", false)}, []v1.Disk{createDisk("vol2")}, []v1.Disk{createDisk("vol1"), createDisk("vol2")}, true),
			Entry("for a removed hotpluggable pvc", []v1.Volume{createPVCVol("vol1", "test1", true)}, []v1.Volume{},
				[]v1.Disk{createDisk("vol1")}, []v1.Disk{}, true),
		)
	})
})

func VirtualMachineFromVMI(name string, vmi *v1.VirtualMachineInstance, started bool) *v1.VirtualMachine {
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1", UID: vmUID},
		Spec: v1.VirtualMachineSpec{
			Running: &started,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
		Status: v1.VirtualMachineStatus{
			Conditions: []v1.VirtualMachineCondition{
				{
					Type:   v1.VirtualMachineReady,
					Status: k8sv1.ConditionFalse,
					Reason: "VMINotExists",
				},
			},
		},
	}
	return vm
}

func DefaultVirtualMachineWithNames(started bool, vmName string, vmiName string) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	vmi := api.NewMinimalVMI(vmiName)
	vmi.GenerateName = "prettyrandom"
	vmi.Status.Phase = v1.Running
	vmi.Finalizers = append(vmi.Finalizers, v1.VirtualMachineControllerFinalizer)
	vm := VirtualMachineFromVMI(vmName, vmi, started)
	vm.Finalizers = append(vm.Finalizers, v1.VirtualMachineControllerFinalizer)
	vmi.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               v1.VirtualMachineGroupVersionKind.Kind,
		Name:               vm.ObjectMeta.Name,
		UID:                vm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}}
	virtcontroller.SetLatestApiVersionAnnotation(vmi)
	virtcontroller.SetLatestApiVersionAnnotation(vm)
	return vm, vmi
}

func DefaultVirtualMachine(started bool) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	return DefaultVirtualMachineWithNames(started, "testvmi", "testvmi")
}

func markVmAsReady(vm *v1.VirtualMachine) {
	virtcontroller.NewVirtualMachineConditionManager().UpdateCondition(vm, &v1.VirtualMachineCondition{Type: v1.VirtualMachineReady, Status: k8sv1.ConditionTrue})
}

func failVMSpecUpdate(virtFakeClient *fake.Clientset) {
	virtFakeClient.PrependReactor("update", "virtualmachines", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		switch action := action.(type) {
		case testing.UpdateActionImpl:
			if action.GetSubresource() != "" {
				return false, nil, nil
			}
		default:
			return false, nil, nil
		}
		return true, nil, fmt.Errorf("conflict")
	})
}
