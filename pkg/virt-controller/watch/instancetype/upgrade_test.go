package instancetype

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	newCRName          = "newCR"
	newCRObjectVersion = "v1beta1"
)

type MockUpgrader struct {
	UpgradeFn func(*appsv1.ControllerRevision) (*appsv1.ControllerRevision, error)
}

func (u *MockUpgrader) Upgrade(original *appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
	return u.UpgradeFn(original)
}

var _ instancetype.Upgrader = &MockUpgrader{}

func newMockUpgrader() *MockUpgrader {
	return &MockUpgrader{
		UpgradeFn: func(*appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
			return &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name: newCRName,
					Labels: map[string]string{
						instancetypeapi.ControllerRevisionObjectVersionLabel: newCRObjectVersion,
					},
				},
			}, nil
		},
	}
}

var _ = Describe("UpgradeController", func() {
	var (
		controller *UpgradeController

		vmInformer        cache.SharedIndexInformer
		crInformer        cache.SharedIndexInformer
		crUpgradeInformer cache.SharedIndexInformer
		crUpgradeSource   *framework.FakeControllerSource
		recorder          *record.FakeRecorder
		mockQueue         *testutils.MockWorkQueue

		client *kubevirtfake.Clientset
		ctrl   *gomock.Controller
		stop   chan struct{}
	)

	const (
		crUpgradeName = "controllerrevisionupgrade"
		crName        = "controllerrevision"
	)

	syncCaches := func(stop chan struct{}) {
		go vmInformer.Run(stop)
		go crInformer.Run(stop)
		go crUpgradeInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, crInformer.HasSynced, crUpgradeInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		vmInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		crInformer, _ = testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		crUpgradeInformer, crUpgradeSource = testutils.NewFakeInformerFor(&instancetypev1beta1.ControllerRevisionUpgrade{})

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		ctrl = gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)

		client = kubevirtfake.NewSimpleClientset()
		virtClient.EXPECT().ControllerRevisionUpgrade(util.NamespaceTestDefault).Return(client.InstancetypeV1beta1().ControllerRevisionUpgrades(util.NamespaceTestDefault)).AnyTimes()

		client.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})

		var err error
		controller, err = NewUpgradeController(virtClient, recorder, vmInformer, crInformer, crUpgradeInformer)
		Expect(err).ToNot(HaveOccurred())

		// Overwrite the upgrader within the controller with a mocked version for testing
		controller.upgrader = newMockUpgrader()

		mockQueue = testutils.NewMockWorkQueue(controller.queue)
		controller.queue = mockQueue

		stop = make(chan struct{})
		syncCaches(stop)
	})

	AfterEach(func() {
		close(stop)
	})

	addCRUpgrade := func(crUpgrade *instancetypev1beta1.ControllerRevisionUpgrade) {
		mockQueue.ExpectAdds(1)
		crUpgradeSource.Add(crUpgrade)
		mockQueue.Wait()
	}

	addRunningUpgrade := func() {
		addCRUpgrade(&instancetypev1beta1.ControllerRevisionUpgrade{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: util.NamespaceTestDefault,
				Name:      crUpgradeName,
			},
			Spec: instancetypev1beta1.ControllerRevisionUpgradeSpec{
				TargetName: crName,
			},
			Status: instancetypev1beta1.ControllerRevisionUpgradeStatus{
				Phase: instancetypev1beta1.UpgradeRunning,
			},
		})
	}

	expectUpgradePhase := func(phase instancetypev1beta1.ControllerRevisionUpgradePhase) {
		client.Fake.PrependReactor("update", instancetypeapi.PluralControllerRevisionUpgradeResourceName, func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())

			crUpgrade := update.GetObject().(*instancetypev1beta1.ControllerRevisionUpgrade)
			Expect(crUpgrade.Status.Phase).To(Equal(phase))

			switch phase {
			case instancetypev1beta1.UpgradeSucceeded:
				Expect(crUpgrade.Status.Result).ToNot(BeNil())
				Expect(crUpgrade.Status.Result.Name).To(Equal(newCRName))
				Expect(crUpgrade.Status.Result.Version).To(Equal(newCRObjectVersion))
			case instancetypev1beta1.UpgradeFailed:
				Expect(crUpgrade.Status.Conditions).To(HaveLen(1))
				Expect(crUpgrade.Status.Conditions[0].Type).To(Equal(instancetypev1beta1.ControllerRevisionUpgradeFailure))
				Expect(crUpgrade.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionTrue))
				Expect(crUpgrade.Status.Conditions[0].Reason).To(Equal(upgradeFailureReason))
			}

			return true, update.GetObject(), nil
		})
	}

	assertProcessed := func() {
		Expect(mockQueue.Len()).To(Equal(1))
		Expect(controller.process()).To(BeTrue())
		Expect(mockQueue.Len()).To(Equal(0))
		Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
	}

	Context("ControllerRevisionUpgrade", func() {
		BeforeEach(func() {
			cr := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: util.NamespaceTestDefault,
					Name:      crName,
				},
			}
			Expect(crInformer.GetStore().Add(cr)).To(Succeed())
		})

		It("should not reenqueue on failure to find ControllerRevisionUpgrade", func() {
			mockQueue.Add("non-existing-crUpgrade-key")
			assertProcessed()
		})

		It("should be ignored if phase already successful", func() {
			addCRUpgrade(&instancetypev1beta1.ControllerRevisionUpgrade{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: util.NamespaceTestDefault,
					Name:      crUpgradeName,
				},
				Spec: instancetypev1beta1.ControllerRevisionUpgradeSpec{
					TargetName: crName,
				},
				Status: instancetypev1beta1.ControllerRevisionUpgradeStatus{
					Phase: instancetypev1beta1.UpgradeSucceeded,
				},
			})
			assertProcessed()
		})

		It("should update new upgrade phase to running", func() {
			addCRUpgrade(&instancetypev1beta1.ControllerRevisionUpgrade{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: util.NamespaceTestDefault,
					Name:      crUpgradeName,
				},
				Spec: instancetypev1beta1.ControllerRevisionUpgradeSpec{
					TargetName: crName,
				},
			})
			expectUpgradePhase(instancetypev1beta1.UpgradeRunning)
			assertProcessed()
		})

		It("mark completed upgrade as succeeded", func() {
			addRunningUpgrade()
			expectUpgradePhase(instancetypev1beta1.UpgradeSucceeded)
			assertProcessed()
		})

		It("should mark upgrade as failed when unable to find target ControllerRevision", func() {
			addCRUpgrade(&instancetypev1beta1.ControllerRevisionUpgrade{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: util.NamespaceTestDefault,
					Name:      crUpgradeName,
				},
				Spec: instancetypev1beta1.ControllerRevisionUpgradeSpec{
					TargetName: "non-existing-cr",
				},
				Status: instancetypev1beta1.ControllerRevisionUpgradeStatus{
					Phase: instancetypev1beta1.UpgradeRunning,
				},
			})
			expectUpgradePhase(instancetypev1beta1.UpgradeFailed)
			assertProcessed()
		})

		It("should mark failed upgrade as failed", func() {
			controller.upgrader = &MockUpgrader{
				UpgradeFn: func(original *appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
					return nil, fmt.Errorf("failure")
				},
			}
			addRunningUpgrade()
			expectUpgradePhase(instancetypev1beta1.UpgradeFailed)
			assertProcessed()
		})
	})
})
