package vm

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
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/api"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"
	instancetypeclientset "kubevirt.io/client-go/kubevirt/typed/instancetype/v1beta1"
	"kubevirt.io/client-go/log"
	kvtesting "kubevirt.io/client-go/testing"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	"kubevirt.io/kubevirt/pkg/instancetype"
	instancetypecontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/vm"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
	watchtesting "kubevirt.io/kubevirt/pkg/virt-controller/watch/testing"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"

	gomegatypes "github.com/onsi/gomega/types"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
)

var (
	vmUID      types.UID = "vm-uid"
	t                    = true
	stage                = v1.VMRolloutStrategyStage
	liveUpdate           = v1.VMRolloutStrategyLiveUpdate
)

var _ = Describe("VirtualMachine", func() {

	Context("One valid VirtualMachine controller given", func() {

		var controller *Controller
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue[string]
		var cdiClient *cdifake.Clientset
		var k8sClient *k8sfake.Clientset
		var virtClient *kubecli.MockKubevirtClient
		var config *virtconfig.ClusterConfig
		var kvStore cache.Store
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

			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true

			config, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

			controller, _ = NewController(vmiInformer,
				vmInformer,
				dataVolumeInformer,
				dataSourceInformer,
				namespaceInformer.GetStore(),
				pvcInformer,
				crInformer,
				recorder,
				virtClient,
				config,
				nil,
				instancetypecontroller.NewMockController(),
			)

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
					const finalizerPath = "/metadata/finalizers"
					patches, err := patch.New(
						patch.WithTest(finalizerPath, v1.VirtualMachineControllerFinalizer),
						patch.WithReplace(finalizerPath, []string{}),
					).GeneratePayload()
					Expect(err).ToNot(HaveOccurred(), "Failed to generate patch payload")
					Expect(action.Patch).To(Equal(patches))
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
			controllertesting.SanityExecute(controller, []cache.Store{
				controller.vmiIndexer, controller.vmIndexer, controller.dataSourceStore, controller.dataVolumeStore,
				controller.namespaceStore, controller.pvcStore, controller.crIndexer,
			}, Default)
		}

		It("should update conditions when failed creating DataVolume for virtualMachineInstance", func() {
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					case kvtesting.PutAction[*v1.AddVolumeOptions]:
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
					case kvtesting.PutAction[*v1.RemoveVolumeOptions]:
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
					case kvtesting.PutAction[*v1.AddVolumeOptions]:
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
					case kvtesting.PutAction[*v1.RemoveVolumeOptions]:
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

				vm, vmi := watchtesting.DefaultVirtualMachine(isRunning)
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
					watchtesting.MarkAsReady(vmi)
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
					vm, vmi := watchtesting.DefaultVirtualMachine(true)
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

					watchtesting.MarkAsReady(vmi)
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
					vm, vmi := watchtesting.DefaultVirtualMachine(true)
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

					watchtesting.MarkAsReady(vmi)
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
					vm, _ := watchtesting.DefaultVirtualMachine(false)
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
					vm, _ := watchtesting.DefaultVirtualMachine(false)
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
					vm, vmi := watchtesting.DefaultVirtualMachine(true)
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

					watchtesting.MarkAsReady(vmi)
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
					vm, _ := watchtesting.DefaultVirtualMachine(false)
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
					vm, _ := watchtesting.DefaultVirtualMachine(false)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(isRunning)
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
					watchtesting.MarkAsReady(vmi)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(isRunning)
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
					watchtesting.MarkAsReady(vmi)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(isRunning)
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
					watchtesting.MarkAsReady(vmi)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
		},
			Entry("with runStrategy set to Always", v1.RunStrategyAlways),
			Entry("with runStrategy set to Once", v1.RunStrategyOnce),
			Entry("with runStrategy set to RerunOnFailure", v1.RunStrategyRerunOnFailure),
		)

		It("should start VMI once DataVolumes (not templates) are complete", func() {

			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)

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

			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())
			// TODO // expect update status is called
			Expect(vm.Status.Created).To(BeFalse())
			Expect(vm.Status.Ready).To(BeFalse())

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should Not delete Datavolumes when VMI is stopped", func() {
			vm, vmi := watchtesting.DefaultVirtualMachine(false)
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
			testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		DescribeTable("should create multiple DataVolumes for VirtualMachineInstance when running", func(running bool, anno string, shouldCreate bool) {
			vm, _ := watchtesting.DefaultVirtualMachine(running)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			Entry("when PVC has owned by annotation and CDI does not support adoption", map[string]string{}, 0, shouldFailDataVolumeCreationClaimAlreadyExists),
		)

		DescribeTable("should properly set priority class", func(dvPriorityClass, vmPriorityClass, expectedPriorityClass string) {
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
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

				testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())

				Expect(vm.Status.StartFailure).ToNot(BeNil())
				Expect(vm.Status.StartFailure.RetryAfterTimestamp).ToNot(BeNil())
				Expect(vm.Status.StartFailure.LastFailedVMIUID).To(Equal(vmi.UID))
				Expect(vm.Status.StartFailure.ConsecutiveFailCount).To(Equal(1))
			})

			It("should track a new start failures when a new VMI fails without hitting running state", func() {
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
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

				testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())

				Expect(vm.Status.StartFailure).ToNot(BeNil())
				Expect(vm.Status.StartFailure.RetryAfterTimestamp).ToNot(BeNil())
				Expect(vm.Status.StartFailure.RetryAfterTimestamp.Time).ToNot(Equal(oldRetry))
				Expect(vm.Status.StartFailure.LastFailedVMIUID).To(Equal(vmi.UID))
				Expect(vm.Status.StartFailure.ConsecutiveFailCount).To(Equal(2))
			})

			It("should clear start failures when VMI hits running state", func() {
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
					testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
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
				vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(vmRevision.Name))
		})

		It("should delete older vmRevision and create VMI with new one", func() {
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(vmRevision.Name))
		})

		Context("VM generation tests", func() {

			DescribeTable("should add the generation annotation onto the VMI", func(startingAnnotations map[string]string, endAnnotations map[string]string) {
				_, vmi := watchtesting.DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = startingAnnotations

				annotations := endAnnotations
				setGenerationAnnotationOnVmi(6, vmi)
				Expect(vmi.ObjectMeta.Annotations).To(Equal(annotations))
			},
				Entry("with previous annotations", map[string]string{"test": "test"}, map[string]string{"test": "test", v1.VirtualMachineGenerationAnnotation: "6"}),
				Entry("without previous annotations", map[string]string{}, map[string]string{v1.VirtualMachineGenerationAnnotation: "6"}),
			)

			DescribeTable("should add generation annotation during VMI creation", func(runStrategy v1.VirtualMachineRunStrategy) {
				vm, _ := watchtesting.DefaultVirtualMachine(true)

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

				testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)

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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = nil
				addVirtualMachine(vm)

				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				_, err = controller.patchVmGenerationAnnotationOnVmi(4, vmi)
				Expect(err).ToNot(HaveOccurred())

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Annotations).To(HaveKeyWithValue("kubevirt.io/vm-generation", "4"))
			})

			DescribeTable("should get the generation annotation from the vmi", func(annotations map[string]string, desiredGeneration *string, desiredErr error) {
				_, vmi := watchtesting.DefaultVirtualMachine(true)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)

				BeforeEach(func() {
					// Reset every time
					vm, vmi = watchtesting.DefaultVirtualMachine(true)
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

					_, err = controller.conditionallyBumpGenerationAnnotationOnVmi(vm, vmi)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
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

				_, err = controller.syncGenerationInfo(vm, vmi, log.DefaultLogger())
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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)

				BeforeEach(func() {
					// Reset every time
					vm, vmi = watchtesting.DefaultVirtualMachine(true)
				})

				type testCase struct {
					initialAnnotations        map[string]string
					desiredAnnotations        map[string]string
					revisionVmSpec            v1.VirtualMachineSpec
					newVMSpec                 v1.VirtualMachineSpec
					revisionVmGeneration      int64
					vmGeneration              int64
					desiredErr                error
					expectPatch               bool
					desiredObservedGeneration int64
					desiredDesiredGeneration  int64
				}

				DescribeTable("should update annotations and sync during Execute()", func(tcase testCase) {
					vmi.ObjectMeta.Annotations = tcase.initialAnnotations
					vm.Generation = tcase.revisionVmGeneration
					vm.Spec = tcase.revisionVmSpec

					crName, err := controller.createVMRevision(vm)
					Expect(err).ToNot(HaveOccurred())

					vmi.Status.VirtualMachineRevisionName = crName

					vm.Generation = tcase.vmGeneration
					vm.Spec = tcase.newVMSpec

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					controller.vmiIndexer.Add(vmi)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())

					Expect(vm.Status.ObservedGeneration).To(Equal(tcase.desiredObservedGeneration), "Observed annotation should be")
					Expect(vm.Status.DesiredGeneration).To(Equal(tcase.desiredDesiredGeneration))

					// TODO should not patch if expectPatch == false
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmi.Annotations).To(Equal(tcase.desiredAnnotations))
				},
					Entry(
						"with annotation existing, new changes in VM spec", testCase{
							initialAnnotations: map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
							// Expect no patch on vmi annotations, and vm status to be correct
							desiredAnnotations: map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
							revisionVmSpec: v1.VirtualMachineSpec{
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
							newVMSpec: v1.VirtualMachineSpec{
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
							revisionVmGeneration:      2,
							vmGeneration:              3,
							desiredErr:                nil,
							expectPatch:               false,
							desiredObservedGeneration: 2,
							desiredDesiredGeneration:  3,
						},
					),
					Entry(
						// Expect a patch on vmi annotations, and vm status to be correct
						"with annotation existing, no new changes in VM spec", testCase{
							initialAnnotations: map[string]string{v1.VirtualMachineGenerationAnnotation: "2"},
							desiredAnnotations: map[string]string{v1.VirtualMachineGenerationAnnotation: "3"},
							revisionVmSpec: v1.VirtualMachineSpec{
								Running: pointer.P(true),
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
							newVMSpec: v1.VirtualMachineSpec{
								Running: pointer.P(true),
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
							revisionVmGeneration:      2,
							vmGeneration:              3,
							desiredErr:                nil,
							expectPatch:               true,
							desiredObservedGeneration: 3,
							desiredDesiredGeneration:  3,
						},
					),
				)
			})
		})

		DescribeTable("should create missing VirtualMachineInstance", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm, _ := watchtesting.DefaultVirtualMachine(true)

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

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		},

			Entry("with run strategy Always", v1.RunStrategyAlways),
			Entry("with run strategy Once", v1.RunStrategyOnce),
			Entry("with run strategy RerunOnFailure", v1.RunStrategyRerunOnFailure),
		)

		It("should ignore the name of a VirtualMachineInstance templates", func() {
			vm, _ := watchtesting.DefaultVirtualMachineWithNames(true, "vmname", "vminame")

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(Succeed())

			//TODO expect update status is called
			Expect(vm.Status.Created).To(BeFalse())
			Expect(vm.Status.Ready).To(BeFalse())

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should update status to created if the vmi exists", func() {
			vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
			vm, vmi := watchtesting.DefaultVirtualMachine(true)
			watchtesting.MarkAsReady(vmi)

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
			vm1, _ := watchtesting.DefaultVirtualMachineWithNames(true, "testvm1", "testvmi1")
			vmi1 := controller.setupVMIFromVM(vm1)

			// intentionally use the same names
			vm2, _ := watchtesting.DefaultVirtualMachineWithNames(true, "testvm1", "testvmi1")
			vmi2 := controller.setupVMIFromVM(vm2)
			Expect(vmi1.Spec.Domain.Firmware.UUID).To(Equal(vmi2.Spec.Domain.Firmware.UUID))

			// now we want different names
			vm3, _ := watchtesting.DefaultVirtualMachineWithNames(true, "testvm3", "testvmi3")
			vmi3 := controller.setupVMIFromVM(vm3)
			Expect(vmi1.Spec.Domain.Firmware.UUID).NotTo(Equal(vmi3.Spec.Domain.Firmware.UUID))
		})

		It("should honour any firmware UUID present in the template", func() {
			uid := uuid.NewString()
			vm1, _ := watchtesting.DefaultVirtualMachineWithNames(true, "testvm1", "testvmi1")
			vm1.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{UUID: types.UID(uid)}

			vmi1 := controller.setupVMIFromVM(vm1)
			Expect(string(vmi1.Spec.Domain.Firmware.UUID)).To(Equal(uid))
		})

		It("should delete VirtualMachineInstance when stopped", func() {
			vm, vmi := watchtesting.DefaultVirtualMachine(false)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		It("should add controller finalizer if VirtualMachine does not have it", func() {
			vm, _ := watchtesting.DefaultVirtualMachine(false)
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
			//watchtesting.DefaultVirtualMachine already set finalizer
			vm, _ := watchtesting.DefaultVirtualMachine(false)
			Expect(vm.Finalizers).To(HaveLen(1))
			Expect(vm.Finalizers[0]).To(BeEquivalentTo(v1.VirtualMachineControllerFinalizer))

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			//TODO Expect only update status, not Patch on vmInterface
			sanityExecute(vm)
		})

		It("should delete VirtualMachineInstance when VirtualMachine marked for deletion", func() {
			vm, vmi := watchtesting.DefaultVirtualMachine(true)
			vm.DeletionTimestamp = pointer.P(metav1.Now())

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		It("should remove controller finalizer once VirtualMachineInstance is gone", func() {
			//watchtesting.DefaultVirtualMachine already set finalizer
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
			vm, vmi := watchtesting.DefaultVirtualMachine(false)
			vmi.DeletionTimestamp = pointer.P(metav1.Now())

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)
		})

		It("should ignore non-matching VMIs", func() {
			vm, _ := watchtesting.DefaultVirtualMachine(true)

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

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should detect that a VirtualMachineInstance already exists and adopt it", func() {
			vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
			vm, _ := watchtesting.DefaultVirtualMachine(false)
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
			vm, vmi := watchtesting.DefaultVirtualMachine(true)

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)
		})

		It("should add a fail condition if start up fails", func() {
			vm, _ := watchtesting.DefaultVirtualMachine(true)

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

			testutils.ExpectEvents(recorder, common.FailedCreateVirtualMachineReason)

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		It("should add a fail condition if deletion fails", func() {
			vm, vmi := watchtesting.DefaultVirtualMachine(false)

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

			testutils.ExpectEvents(recorder, common.FailedDeleteVirtualMachineReason)
		})

		unmarkReady := func(vmi *v1.VirtualMachineInstance) {
			virtcontroller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, v1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
		}

		DescribeTable("should add ready condition when VMI exists", func(setup func(vmi *v1.VirtualMachineInstance), status k8sv1.ConditionStatus) {
			vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
			Entry("VMI Ready condition is True", watchtesting.MarkAsReady, k8sv1.ConditionTrue),
			Entry("VMI Ready condition is False", watchtesting.MarkAsNonReady, k8sv1.ConditionFalse),
			Entry("VMI Ready condition doesn't exist", unmarkReady, k8sv1.ConditionFalse),
		)

		It("should sync VMI conditions", func() {
			vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			vm, vmi := watchtesting.DefaultVirtualMachine(true)
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			watchtesting.MarkAsReady(vmi)
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
			vm, vmi := watchtesting.DefaultVirtualMachine(true)
			vm.Status.Conditions = append(vm.Status.Conditions, v1.VirtualMachineCondition{
				Type:   v1.VirtualMachinePaused,
				Status: k8sv1.ConditionTrue,
			})
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			watchtesting.MarkAsReady(vmi)
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
			vm, vmi := watchtesting.DefaultVirtualMachine(false)

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
			testutils.ExpectEvents(recorder, common.FailedDeleteVirtualMachineReason)
		})

		It("should copy annotations from spec.template to vmi", func() {
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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

		It("should not copy annotations from vm to vmi", func() {
			vm, _ := watchtesting.DefaultVirtualMachine(true)
			vm.Annotations["kubevirt.io/test"] = "test"

			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			addVirtualMachine(vm)

			sanityExecute(vm)

			vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Annotations).ToNot(HaveKey("kubevirt.io/test"))
		})

		It("should copy kubevirt ignitiondata annotation from spec.template to vmi", func() {
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			const testPVCName = "testPVC"

			It("should add memory dump volume and update vmi volumes", func() {
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpAssociating,
				}

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				watchtesting.MarkAsReady(vmi)
				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(testPVCName))

				vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(vmi.Spec.Volumes[0].Name).To(Equal(testPVCName))
			})

			It("should remove memory dump volume from vm volumes list", func() {
				// No need to add vmi - can do this action even if vm not running
				vm, _ := watchtesting.DefaultVirtualMachine(false)
				vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpDissociating,
				}

				memoryDumpVol := v1.Volume{
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

				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, memoryDumpVol)
				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
			})

			DescribeTable("should not setup vmi with memory dump if memory dump", func(phase v1.MemoryDumpPhase) {
				vm, _ := watchtesting.DefaultVirtualMachine(true)
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
				vm, _ := watchtesting.DefaultVirtualMachine(false)

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				sanityExecute(vm)

				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStopped))
			})

			DescribeTable("should set a Stopped status when VMI exists but stopped", func(phase v1.VirtualMachineInstancePhase, deletionTimestamp *metav1.Time) {
				vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
				vm, _ := watchtesting.DefaultVirtualMachine(true)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
					vm, vmi = watchtesting.DefaultVirtualMachine(true)
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
					if running {
						vm.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
					} else {
						vm.Spec.RunStrategy = pointer.P(v1.RunStrategyHalted)
					}

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

				It("should set a Provisioning status and add a Failure condition if DV PVC creation fails due to exceeded quota", func() {
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
					dv.Status.Phase = cdiv1.Pending
					dv.Status.Conditions = append(dv.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8sv1.ConditionUnknown,
						Reason: "ErrExceededQuota",
						Message: "persistentvolumeclaims \"dv1\" is forbidden: exceeded quota: storage, " +
							"requested: requests.storage=2Gi, used: requests.storage=0, limited: requests.storage=1Gi",
					})
					controller.dataVolumeStore.Add(dv)

					sanityExecute(vm)

					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusProvisioning))

					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Type":   Equal(v1.VirtualMachineFailure),
						"Reason": Equal("FailedCreate"),
						"Message": And(
							ContainSubstring("Error encountered while creating DataVolumes"),
							ContainSubstring("importer is not running due to an error"),
							ContainSubstring("forbidden: exceeded quota: storage"),
						),
					}))
				})
			})

			Context("VM with PersistentVolumeClaims", func() {
				var vm *v1.VirtualMachine

				BeforeEach(func() {
					vm, _ = watchtesting.DefaultVirtualMachine(true)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
					vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
					vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
					vm, _ := watchtesting.DefaultVirtualMachine(running)

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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)

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

					vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
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

				instancetypeObj *instancetypev1beta1.VirtualMachineInstancetype
				preference      *instancetypev1beta1.VirtualMachinePreference

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
				vm, _ = watchtesting.DefaultVirtualMachine(true)

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

				controller.instancetypeController = instancetypecontroller.New(
					instancetypeInformerStore,
					clusterInstancetypeInformerStore,
					preferenceInformerStore,
					clusterPreferenceInformerStore,
					controllerrevisionInformerStore,
					virtClient,
					config,
					controller.recorder,
				)

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
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: uint32(2),
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128M"),
						},
					},
				}
				_, err := virtClient.VirtualMachineInstancetype(vm.Namespace).Create(context.Background(), instancetypeObj, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				err = instancetypeInformerStore.Add(instancetypeObj)
				Expect(err).NotTo(HaveOccurred())

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
					Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
						Firmware: &instancetypev1beta1.FirmwarePreferences{
							DeprecatedPreferredUseEfi: pointer.P(true),
						},
						Devices: &instancetypev1beta1.DevicePreferences{
							PreferredDiskBus:        v1.DiskBusVirtio,
							PreferredInterfaceModel: "virtio",
							PreferredInputBus:       v1.InputBusUSB,
							PreferredInputType:      v1.InputTypeTablet,
						},
					},
				}
				_, err = virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				err = preferenceInformerStore.Add(preference)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should capture instance type and preference RevisionNames in VirtualMachine ControllerRevision", func() {
				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: instancetypeObj.Name,
					Kind: instancetypeapi.SingularResourceName,
				}
				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: instancetypeapi.SingularPreferenceResourceName,
				}
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				addVirtualMachine(vm)
				sanityExecute(vm)

				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(revision.HasControllerRevisionRef(vm.Status.InstancetypeRef)).To(BeTrue())
				Expect(revision.HasControllerRevisionRef(vm.Status.PreferenceRef)).To(BeTrue())

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.VirtualMachineRevisionName).ToNot(BeEmpty())

				vmRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
					context.Background(), vmi.Status.VirtualMachineRevisionName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmRevision).ToNot(BeNil())

				revisionData := &VirtualMachineRevisionData{}
				Expect(json.Unmarshal(vmRevision.Data.Raw, revisionData)).To(Succeed())
				Expect(revisionData.Spec.Instancetype).ToNot(BeNil())
				Expect(revisionData.Spec.Instancetype.RevisionName).To(Equal(vm.Status.InstancetypeRef.ControllerRevisionRef.Name))
				Expect(revisionData.Spec.Preference).ToNot(BeNil())
				Expect(revisionData.Spec.Preference.RevisionName).To(Equal(vm.Status.PreferenceRef.ControllerRevisionRef.Name))
			})

			Context("preference", func() {
				var (
					clusterPreference *instancetypev1beta1.VirtualMachineClusterPreference
				)

				BeforeEach(func() {
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
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							Firmware: &instancetypev1beta1.FirmwarePreferences{
								DeprecatedPreferredUseEfi: pointer.P(true),
							},
							Devices: &instancetypev1beta1.DevicePreferences{
								PreferredDiskBus:        v1.DiskBusVirtio,
								PreferredInterfaceModel: "virtio",
								PreferredInputBus:       v1.InputBusUSB,
								PreferredInputType:      v1.InputTypeTablet,
							},
						},
					}
					_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = clusterPreferenceInformerStore.Add(clusterPreference)
					Expect(err).NotTo(HaveOccurred())
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
					Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(expectedPreferenceRevision.Name))
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
					Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(expectedPreferenceRevision.Name))
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
					Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(expectedPreferenceRevision.Name))
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
					Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(expectedPreferenceRevision.Name))
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
					Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(expectedPreferenceRevision.Name))
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
					Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(expectedPreferenceRevision.Name))
				})
			})
		})

		DescribeTable("should add the default network interface",
			func(iface string, field gstruct.Fields) {
				vm, _ := watchtesting.DefaultVirtualMachine(true)

				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)

			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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

		DescribeTable("call to synchronizer", func(syncer testSynchronizer, expectCond v1.VirtualMachineCondition, expectCondCount int) {
			vm, vmi := watchtesting.DefaultVirtualMachine(true)
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			addVirtualMachine(vm)
			controller.netSynchronizer = syncer

			watchtesting.MarkAsReady(vmi)
			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			controller.vmiIndexer.Add(vmi)

			sanityExecute(vm)

			vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Status.Conditions).To(ContainElement(
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":    Equal(expectCond.Type),
					"Status":  Equal(expectCond.Status),
					"Reason":  Equal(expectCond.Reason),
					"Message": Equal(expectCond.Message),
				}),
			))
			Expect(vm.Status.Conditions).To(HaveLen(expectCondCount))

			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry(
				"succeeds",
				testSynchronizer{},
				v1.VirtualMachineCondition{Type: v1.VirtualMachineReady, Status: k8sv1.ConditionTrue}, 1,
			),
			Entry(
				"fails with proper error",
				testSynchronizer{err: common.NewSyncError(fmt.Errorf("good error"), "good reason")},
				v1.VirtualMachineCondition{Type: v1.VirtualMachineFailure, Status: k8sv1.ConditionTrue, Reason: "good reason", Message: "good error"}, 2,
			),
			Entry(
				"fails with unsupported error",
				testSynchronizer{err: fmt.Errorf("bad error")},
				v1.VirtualMachineCondition{Type: v1.VirtualMachineFailure, Status: k8sv1.ConditionTrue, Reason: "UnsupportedSyncError", Message: "unsupported error: bad error"}, 2,
			),
		)

		It("should add a missing volume disk", func() {
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
			vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{MaxSockets: maxSocketsFromSpec}

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(maxSocketsFromSpec))
				})

				It("should prefer maximum CPU sockets from VM spec rather than from cluster config", func() {
					vm, _ := watchtesting.DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{MaxSockets: maxSocketsFromSpec}
					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								LiveUpdateConfiguration: &v1.LiveUpdateConfiguration{
									MaxCpuSockets: pointer.P(maxSocketsFromConfig),
								},
								VMRolloutStrategy: &liveUpdate,
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(maxSocketsFromSpec))
				})

				DescribeTable("should patch VMI when CPU hotplug is requested", func(resources v1.ResourceRequirements) {
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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

					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
					guestMemory := resource.MustParse("64Mi")
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
						Guest:    &guestMemory,
						MaxGuest: &maxGuestFromSpec,
					}

					vmi := controller.setupVMIFromVM(vm)
					Expect(*vmi.Spec.Domain.Memory.MaxGuest).To(Equal(maxGuestFromSpec))
				})

				It("should prefer maxGuest from VM spec rather than from cluster config", func() {
					vm, _ := watchtesting.DefaultVirtualMachine(true)
					guestMemory := resource.MustParse("64Mi")
					vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
						Guest:    &guestMemory,
						MaxGuest: &maxGuestFromSpec,
					}
					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								LiveUpdateConfiguration: &v1.LiveUpdateConfiguration{
									MaxGuest: &maxGuestFromConfig,
								},
								VMRolloutStrategy: &liveUpdate,
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(*vmi.Spec.Domain.Memory.MaxGuest).To(Equal(maxGuestFromSpec))
				})

				DescribeTable("should patch VMI when memory hotplug is requested", func(resources v1.ResourceRequirements) {
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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
					vm, _ := watchtesting.DefaultVirtualMachine(true)
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

			Context("Tolerations", func() {
				DescribeTable("should be live-updated", func(existingTolerations, updatedTolerations []k8sv1.Toleration) {
					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &liveUpdate,
							},
						},
					})

					vm, vmi := watchtesting.DefaultVirtualMachine(true)

					vm.Spec.Template.Spec.Tolerations = updatedTolerations
					vmi.Spec.Tolerations = existingTolerations

					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())

					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(controller.vmiIndexer.Add(vmi)).To(Succeed())

					addVirtualMachine(vm)

					sanityExecute(vm)

					Expect(kvtesting.FilterActions(&virtFakeClient.Fake, "patch", "virtualmachineinstances")).To(HaveLen(1))

					By("Expecting to see the updated VMI with the added tolerations")
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmi.Spec.Tolerations).To(Equal(updatedTolerations))
				},
					Entry("when adding a toleration from an empty set",
						nil,
						[]k8sv1.Toleration{{
							Effect:   k8sv1.TaintEffectNoExecute,
							Key:      "testTaint",
							Operator: k8sv1.TolerationOpExists,
						}},
					),
					Entry("when adding a new toleration",
						[]k8sv1.Toleration{{
							Effect:   k8sv1.TaintEffectNoExecute,
							Key:      "testTaint",
							Operator: k8sv1.TolerationOpExists,
						}},
						[]k8sv1.Toleration{
							{
								Effect:   k8sv1.TaintEffectNoExecute,
								Key:      "testTaint",
								Operator: k8sv1.TolerationOpExists,
							},

							{
								Effect:   k8sv1.TaintEffectNoExecute,
								Key:      "testTaint2",
								Operator: k8sv1.TolerationOpExists,
							},
						},
					),
					Entry("when removing a toleration",
						[]k8sv1.Toleration{
							{
								Effect:   k8sv1.TaintEffectNoExecute,
								Key:      "testTaint",
								Operator: k8sv1.TolerationOpExists,
							},

							{
								Effect:   k8sv1.TaintEffectNoExecute,
								Key:      "testTaint2",
								Operator: k8sv1.TolerationOpExists,
							},
						},
						[]k8sv1.Toleration{{
							Effect:   k8sv1.TaintEffectNoExecute,
							Key:      "testTaint",
							Operator: k8sv1.TolerationOpExists,
						}},
					),
					Entry("when removing all tolerations",
						[]k8sv1.Toleration{{
							Effect:   k8sv1.TaintEffectNoExecute,
							Key:      "testTaint",
							Operator: k8sv1.TolerationOpExists,
						}},
						nil,
					),
				)
			})

			Context("Affinity", func() {
				It("should be live-updated", func() {
					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &liveUpdate,
							},
						},
					})

					vm, vmi := watchtesting.DefaultVirtualMachine(true)

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
					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &liveUpdate,
							},
						},
					})
					vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &liveUpdate,
							},
						},
					})
					vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
					vmi        *v1.VirtualMachineInstance

					controllerrevisionInformerStore cache.Store

					originalInstancetype *instancetypev1beta1.VirtualMachineInstancetype
				)

				BeforeEach(func() {
					originalVM, vmi = watchtesting.DefaultVirtualMachine(true)
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

					controller.instancetypeController = instancetypecontroller.New(
						nil,
						nil,
						nil,
						nil,
						controllerrevisionInformerStore,
						virtClient,
						config,
						controller.recorder,
					)

					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &liveUpdate,
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
					Expect(controller.addRestartRequiredIfNeeded(&originalVM.Spec, updatedVM, vmi)).To(BeFalse())
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

					Expect(controller.addRestartRequiredIfNeeded(&originalVM.Spec, updatedVM, vmi)).To(BeTrue())
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
				vm, vmi = watchtesting.DefaultVirtualMachine(true)
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
						},
					},
				}
			})

			It("should not call update if the condition is set", func() {
				vm, vmi := watchtesting.DefaultVirtualMachine(true)
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
						},
					},
				}
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

				var maxSockets uint32 = 8

				By("Setting a cluster-wide CPU maxSockets value")
				kv.Spec.Configuration.LiveUpdateConfiguration.MaxCpuSockets = pointer.P(maxSockets)
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

				By("Creating a VM with CPU sockets set to the cluster maxiumum")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = maxSockets
				controller.crIndexer.Add(createVMRevision(vm))

				By("Creating a VMI with cluster max")
				vmi = controller.setupVMIFromVM(vm)
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
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

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
				watchtesting.MarkAsReady(vmi)
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

			DescribeTable("when changing a live-updatable field", func(strat *v1.VMRolloutStrategy, matcher gomegatypes.GomegaMatcher) {
				// Add necessary stuff to reflect running VM
				// TODO: This should be done in more places
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.PrintableStatus = "Running"
				virtcontroller.NewVirtualMachineConditionManager().UpdateCondition(vm, &v1.VirtualMachineCondition{
					Type:   v1.VirtualMachineReady,
					Status: k8sv1.ConditionTrue,
				})
				kv.Spec.Configuration.VMRolloutStrategy = strat
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

				By("Creating a VM with CPU sockets set to 2")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 2
				vm.Spec.Template.Spec.Domain.CPU.MaxSockets = 8
				controller.crIndexer.Add(createVMRevision(vm))

				By("Creating a VMI")
				vmi = controller.setupVMIFromVM(vm)
				watchtesting.MarkAsReady(vmi)
				vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				controller.vmiIndexer.Add(vmi)

				By("Bumping the VM sockets to 4")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 4

				vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				addVirtualMachine(vm)

				By("Executing the controller expecting the RestartRequired condition to appear as needed")
				sanityExecute(vm)
				vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(vm.Status.Conditions).To(matcher, "restart Required")

				if strat == &liveUpdate {
					vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(4)))
				}
			},
				Entry("should not appear if the VM rollout strategy is not set",
					nil, Not(restartRequiredMatcher(k8sv1.ConditionTrue))),
				Entry("should appear if the VM rollout strategy is set to Stage",
					&stage, restartRequiredMatcher(k8sv1.ConditionTrue)),
				Entry("should not appear if VM rollout strategy is set to LiveUpdate",
					&liveUpdate, Not(restartRequiredMatcher(k8sv1.ConditionTrue))),
			)

			Context("for hotplugged volumes", func() {
				BeforeEach(func() {
					kv.Spec.Configuration.VMRolloutStrategy = pointer.P(stage)
					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

				})

				addVolume := func() {
					hpSource := v1.HotplugVolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name:         "hotplug-dv",
							Hotpluggable: true,
						},
					}
					disk := v1.Disk{
						Name: "hotplug",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{Bus: v1.DiskBusSCSI},
						}}
					vm.Status.VolumeRequests = append(vm.Status.VolumeRequests, v1.VirtualMachineVolumeRequest{
						AddVolumeOptions: &v1.AddVolumeOptions{
							Name:         "hotplug",
							VolumeSource: &hpSource,
							Disk:         &disk,
						},
					})
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

				}

				It("shouldn't appear with the stage rollout strategy and when a volume is added with a hotplug volume request", func() {
					By("Creating a VM")
					vmi = controller.setupVMIFromVM(vm)
					controller.vmiIndexer.Add(vmi)

					By("Creating a Controller Revision")
					controller.crIndexer.Add(createVMRevision(vm))

					By("Adding an hotplugged volume to the VM")
					addVolume()

					By("Executing the controller expecting the RestartRequired not to appear")
					sanityExecute(vm)
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(virtcontroller.NewVirtualMachineConditionManager().HasCondition(vm,
						v1.VirtualMachineRestartRequired)).To(BeFalse())
				})

				It("should appear when the volume is directly added to the spec", func() {
					By("Creating a VM")
					vmi = controller.setupVMIFromVM(vm)
					controller.vmiIndexer.Add(vmi)

					By("Creating a Controller Revision")
					controller.crIndexer.Add(createVMRevision(vm))

					By("Adding an hotplugged volume to the VM")
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "hotplug",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name:         "hotplug-dv",
								Hotpluggable: true,
							},
						},
					})
					vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
						Name: "hotplug",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{Bus: v1.DiskBusSCSI},
						}})
					vm, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					addVirtualMachine(vm)

					By("Executing the controller expecting the RestartRequired not to appear")
					sanityExecute(vm)
					vm, err = virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(virtcontroller.NewVirtualMachineConditionManager().HasCondition(vm,
						v1.VirtualMachineRestartRequired)).To(BeTrue())
				})
			})
		})

		Context("RunStrategy", func() {

			It("when starting a VM with RerunOnFailure the VM should get started", func() {
				vm, _ := watchtesting.DefaultVirtualMachine(true)
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
				vm, _ := watchtesting.DefaultVirtualMachine(true)
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
				vm, _ := watchtesting.DefaultVirtualMachine(true)
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
				vm, _ := watchtesting.DefaultVirtualMachine(true)
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
				vm, _ := watchtesting.DefaultVirtualMachine(true)
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

		Context("startVMI", func() {
			It("should not start the VMI if the VM has the ManualRecoveryRequiredCondition set", func() {
				vm := libvmi.NewVirtualMachine(libvmi.New(libvmi.WithNamespace(metav1.NamespaceDefault)), libvmistatus.WithVMStatus(libvmistatus.NewVMStatus(libvmistatus.WithVMCondition(v1.VirtualMachineCondition{
					Type:   v1.VirtualMachineManualRecoveryRequired,
					Status: k8sv1.ConditionTrue,
				}))))
				vmCopy, err := controller.startVMI(vm)
				Expect(err).NotTo(HaveOccurred())
				Expect(vm).To(Equal(vmCopy))
				vmiList, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmiList.Items).To(BeEmpty())
			})
		})

	})
	Context("syncConditions", func() {
		var vm *v1.VirtualMachine
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vm, vmi = watchtesting.DefaultVirtualMachineWithNames(false, "test", "test")
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
			oldVm, _ := watchtesting.DefaultVirtualMachine(true)
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
			oldVm, _ := watchtesting.DefaultVirtualMachine(true)
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

	Context("syncVolumeMigration", func() {
		const (
			volName = "disk0"
		)
		withMigVols := func(volName, src, dst string) []v1.StorageMigratedVolumeInfo {
			return []v1.StorageMigratedVolumeInfo{
				{
					VolumeName:         volName,
					SourcePVCInfo:      &v1.PersistentVolumeClaimInfo{ClaimName: src},
					DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{ClaimName: dst},
				}}
		}
		withVMCondVolumeMigInProgress := func(inProgress k8sv1.ConditionStatus, reason string) v1.VirtualMachineCondition {
			return v1.VirtualMachineCondition{
				Type:   v1.VirtualMachineConditionType(v1.VirtualMachineInstanceVolumesChange),
				Status: inProgress,
				Reason: reason,
			}
		}
		withVMICondVolumeMigInProgress := func(inProgress k8sv1.ConditionStatus, reason string) v1.VirtualMachineInstanceCondition {
			return v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstanceVolumesChange,
				Status: inProgress,
				Reason: reason,
			}
		}
		DescribeTable("and UpdateVolumesStrategy set to Migration", func(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, expectedVolumeMigrationState *v1.VolumeMigrationState, expectCond bool) {
			vm.Spec.UpdateVolumesStrategy = pointer.P(v1.UpdateVolumesStrategyMigration)
			syncVolumeMigration(vm, vmi)
			Expect(vm.Status.VolumeUpdateState.VolumeMigrationState).To(Equal(expectedVolumeMigrationState))
			Expect(controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm,
				v1.VirtualMachineManualRecoveryRequired, k8sv1.ConditionTrue)).To(Equal(expectCond))
		},
			Entry("without any volume migration in progress", libvmi.NewVirtualMachine(libvmi.New(), libvmistatus.WithVMStatus(libvmistatus.NewVMStatus(libvmistatus.WithVMVolumeUpdateState(&v1.VolumeUpdateState{})))), libvmi.New(), nil, false),
			Entry("with volume migration in progress but no vmi", libvmi.NewVirtualMachine(libvmi.New(), libvmistatus.WithVMStatus(
				libvmistatus.NewVMStatus(libvmistatus.WithVMVolumeUpdateState(&v1.VolumeUpdateState{VolumeMigrationState: &v1.VolumeMigrationState{
					MigratedVolumes: withMigVols(volName, "dv0", "dv1")}},
				), libvmistatus.WithVMCondition(withVMCondVolumeMigInProgress(k8sv1.ConditionTrue, ""))),
			)),
				nil, nil, false),
			Entry("with recovered volumes", libvmi.NewVirtualMachine(libvmi.New(libvmi.WithDataVolume(volName, "dv0")), libvmistatus.WithVMStatus(
				libvmistatus.NewVMStatus(libvmistatus.WithVMVolumeUpdateState(
					&v1.VolumeUpdateState{VolumeMigrationState: &v1.VolumeMigrationState{MigratedVolumes: withMigVols(volName, "dv0", "dv1")}},
				)))),
				libvmi.New(libvmi.WithDataVolume(volName, "dv0")), nil, false),
			Entry("with volumes still to be recovered", libvmi.NewVirtualMachine(libvmi.New(libvmi.WithDataVolume(volName, "dv1")), libvmistatus.WithVMStatus(
				libvmistatus.NewVMStatus(
					libvmistatus.WithVMCondition(v1.VirtualMachineCondition{
						Type:   v1.VirtualMachineManualRecoveryRequired,
						Status: k8sv1.ConditionTrue,
					}),
					libvmistatus.WithVMVolumeUpdateState(
						&v1.VolumeUpdateState{VolumeMigrationState: &v1.VolumeMigrationState{
							MigratedVolumes: withMigVols(volName, "dv0", "dv1"),
						}},
					)))),
				libvmi.New(libvmi.WithDataVolume(volName, "dv0")), &v1.VolumeMigrationState{
					MigratedVolumes: withMigVols(volName, "dv0", "dv1"),
				}, true,
			),
			Entry("with volume change", libvmi.NewVirtualMachine(libvmi.New(libvmi.WithDataVolume(volName, "dv1")),
				libvmistatus.WithVMStatus(
					libvmistatus.NewVMStatus(libvmistatus.WithVMVolumeUpdateState(&v1.VolumeUpdateState{VolumeMigrationState: &v1.VolumeMigrationState{
						MigratedVolumes: withMigVols(volName, "dv0", "dv1"),
					}}), libvmistatus.WithVMCondition(withVMCondVolumeMigInProgress(k8sv1.ConditionTrue, "")),
					))),
				libvmi.New(libvmi.WithDataVolume(volName, "dv0"),
					libvmistatus.WithStatus(
						libvmistatus.New(libvmistatus.WithCondition(
							withVMICondVolumeMigInProgress(k8sv1.ConditionTrue, "")))),
				),
				&v1.VolumeMigrationState{MigratedVolumes: withMigVols(volName, "dv0", "dv1")}, false,
			),
			Entry("with volume change cancellation", libvmi.NewVirtualMachine(libvmi.New(libvmi.WithDataVolume(volName, "dv1")),
				libvmistatus.WithVMStatus(
					libvmistatus.NewVMStatus(libvmistatus.WithVMVolumeUpdateState(&v1.VolumeUpdateState{VolumeMigrationState: &v1.VolumeMigrationState{
						MigratedVolumes: withMigVols(volName, "dv0", "dv1"),
					}}), libvmistatus.WithVMCondition(withVMCondVolumeMigInProgress(k8sv1.ConditionFalse, v1.VirtualMachineInstanceReasonVolumesChangeCancellation)),
					))),
				libvmi.New(libvmi.WithDataVolume(volName, "dv0"),
					libvmistatus.WithStatus(
						libvmistatus.New(libvmistatus.WithCondition(
							withVMICondVolumeMigInProgress(k8sv1.ConditionFalse, v1.VirtualMachineInstanceReasonVolumesChangeCancellation)))),
				),
				nil, false,
			),
			Entry("with a failed volume migration", libvmi.NewVirtualMachine(libvmi.New(libvmi.WithDataVolume(volName, "dv1")),
				libvmistatus.WithVMStatus(
					libvmistatus.NewVMStatus(libvmistatus.WithVMVolumeUpdateState(&v1.VolumeUpdateState{VolumeMigrationState: &v1.VolumeMigrationState{
						MigratedVolumes: withMigVols(volName, "dv0", "dv1"),
					}}), libvmistatus.WithVMCondition(withVMCondVolumeMigInProgress(k8sv1.ConditionFalse, "")),
					))),
				libvmi.New(libvmi.WithDataVolume(volName, "dv0"),
					libvmistatus.WithStatus(
						libvmistatus.New(libvmistatus.WithCondition(
							withVMICondVolumeMigInProgress(k8sv1.ConditionFalse, "")))),
				), &v1.VolumeMigrationState{
					MigratedVolumes: withMigVols(volName, "dv0", "dv1"),
				}, false,
			),
			Entry("with volume migration in progress and vmi disappeared", libvmi.NewVirtualMachine(libvmi.New(libvmi.WithDataVolume(volName, "dv1")),
				libvmistatus.WithVMStatus(
					libvmistatus.NewVMStatus(libvmistatus.WithVMVolumeUpdateState(&v1.VolumeUpdateState{VolumeMigrationState: &v1.VolumeMigrationState{
						MigratedVolumes: withMigVols(volName, "dv0", "dv1"),
					}}), libvmistatus.WithVMCondition(withVMCondVolumeMigInProgress(k8sv1.ConditionTrue, "")),
					))),
				nil, &v1.VolumeMigrationState{
					MigratedVolumes: withMigVols(volName, "dv0", "dv1"),
				}, true,
			),
		)
	})
})

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

type testSynchronizer struct {
	err error
}

func (t testSynchronizer) Sync(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) (*v1.VirtualMachine, error) {
	return vm, t.err
}
