package virthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var _ = Describe("VirtualMachineInstance", func() {

	var (
		launcherClient        *cmdclient.MockLauncherClient
		vmiInterface          *kubecli.MockVirtualMachineInstanceInterface
		virtClient            *kubecli.MockKubevirtClient
		clientTest            *fake.Clientset
		ctrl                  *gomock.Controller
		controller            *GuestStatsController
		vmiSourceInformer     cache.SharedIndexInformer
		mockQueue             *testutils.MockWorkQueue
		mockIsolationDetector *isolation.MockPodIsolationDetector
		mockIsolationResult   *isolation.MockIsolationResult
		virtShareDir          string
		shareDir              string
		podsDir               string
		err                   error
		sockFile              string
		vmiTestUUID           types.UID
		podTestUUID           types.UID
		stop                  chan struct{}

		vmi *v1.VirtualMachineInstance
	)

	newVmi := func() *v1.VirtualMachineInstance {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		return vmi
	}

	BeforeEach(func() {
		shareDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		podsDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())

		virtShareDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())

		cmdclient.SetLegacyBaseDir(shareDir)
		cmdclient.SetPodsBaseDir(podsDir)

		stop = make(chan struct{})
		vmiSourceInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})

		clientTest = fake.NewSimpleClientset()
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().CoreV1().Return(clientTest.CoreV1()).AnyTimes()
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()

		Expect(os.MkdirAll(filepath.Join(virtShareDir, "dev"), 0755)).To(Succeed())
		f, err := os.OpenFile(filepath.Join(virtShareDir, "dev", "kvm"), os.O_CREATE, 0755)
		Expect(err).ToNot(HaveOccurred())
		f.Close()

		mockIsolationResult = isolation.NewMockIsolationResult(ctrl)
		mockIsolationResult.EXPECT().Pid().Return(1).AnyTimes()
		rootDir, err := safepath.JoinAndResolveWithRelativeRoot(virtShareDir)
		Expect(err).ToNot(HaveOccurred())
		mockIsolationResult.EXPECT().MountRoot().Return(rootDir, nil).AnyTimes()

		mockIsolationDetector = isolation.NewMockPodIsolationDetector(ctrl)
		mockIsolationDetector.EXPECT().Detect(gomock.Any()).Return(mockIsolationResult, nil).AnyTimes()
		mockIsolationDetector.EXPECT().AdjustResources(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		podTestUUID = uuid.NewUUID()
		sockFile = cmdclient.SocketFilePathOnHost(string(podTestUUID))
		Expect(os.MkdirAll(filepath.Dir(sockFile), 0755)).To(Succeed())
		f, err = os.Create(sockFile)
		Expect(err).ToNot(HaveOccurred())
		f.Close()

		launcherClient = cmdclient.NewMockLauncherClient(ctrl)
		clientInfo := &virtcache.LauncherClientInfo{
			Client:             launcherClient,
			SocketFile:         sockFile,
			DomainPipeStopChan: make(chan struct{}),
			Ready:              true,
		}

		controller, _ = NewGuestStatsController(virtShareDir, virtClient, vmiSourceInformer)
		mockQueue = testutils.NewMockWorkQueue(controller.vmiQueue)
		controller.vmiQueue = mockQueue
		controller.podIsolationDetector = mockIsolationDetector
		controller.launcherClient.Store(vmiTestUUID, clientInfo)

		go vmiSourceInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmiSourceInformer.HasSynced)).To(BeTrue())

		vmi = newVmi()
	})

	AfterEach(func() {
		close(stop)
		os.RemoveAll(virtShareDir)
		os.RemoveAll(podsDir)
		os.RemoveAll(shareDir)
	})

	newGuestStats := func(sampleCount int64, average, variance float64) v1.GuestStats {
		return v1.GuestStats{
			DirtyRate: &v1.DirtyRateStats{
				SampleCount: sampleCount,
				Average:     average,
				Variance:    variance,
			},
		}
	}

	setGuestStats := func(guestStats v1.GuestStats) {
		launcherClient.EXPECT().GetGuestStats().Return(guestStats, nil).Times(1)
	}

	localVMIPatch := func(patchBytes []byte) {
		marshalledVMI, err := json.Marshal(vmi)
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		patch, err := jsonpatch.DecodePatch(patchBytes)
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		modifiedMarshalledVMI, err := patch.Apply(marshalledVMI)
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		newVmi := &v1.VirtualMachineInstance{}
		err = json.Unmarshal(modifiedMarshalledVMI, newVmi)
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		vmi = newVmi
	}

	expectStatsUpdate := func(stats v1.GuestStats) {
		guestStatsBytes, err := json.Marshal(stats)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		expectedPatchBytes := fmt.Sprintf(`"value": %s`, string(guestStatsBytes))

		vmiInterface.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(
			func(_ context.Context, _ string, _ types.PatchType, patchBody interface{}, _ *metav1.PatchOptions, _ ...string) (interface{}, interface{}) {
				patchBytes := patchBody.([]byte)
				patchStr := string(patchBytes)
				ExpectWithOffset(2, patchStr).To(ContainSubstring(expectedPatchBytes))

				localVMIPatch(patchBytes)
				err = vmiSourceInformer.GetStore().Update(vmi)
				ExpectWithOffset(2, patchStr).To(ContainSubstring(expectedPatchBytes))

				// Return values are ignored
				return nil, nil
			}).Times(1)
	}

	addVmiToQueue := func(vmi *v1.VirtualMachineInstance) {
		key, err := cache.MetaNamespaceKeyFunc(vmi)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		mockQueue.Add(key)
		err = vmiSourceInformer.GetStore().Add(vmi)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	It("Ensure dirty rate is being updated when it's the first sample", func() {
		addVmiToQueue(vmi)

		curGuestStats := newGuestStats(1, 1234, 0)
		setGuestStats(curGuestStats)
		expectStatsUpdate(curGuestStats)

		controller.Execute()
	})

	It("Ensure dirty rate is being updated when average diff is big enough", func() {
		addVmiToQueue(vmi)

		const (
			dirtyRate1 = 1234
			dirtyRate2 = 5678
			variance2  = (dirtyRate1 + dirtyRate2) / 2
		)

		guestStats := newGuestStats(1, dirtyRate1, 0)
		setGuestStats(guestStats)
		expectStatsUpdate(guestStats)

		controller.Execute()
		addVmiToQueue(vmi)

		guestStats = newGuestStats(2, dirtyRate2, variance2)
		setGuestStats(guestStats)
		expectStatsUpdate(guestStats)

		controller.Execute()
	})

	It("Ensure dirty rate is being updated when sample diff is big enough", func() {
		addVmiToQueue(vmi)

		guestStats := newGuestStats(1, 1234, 0)
		setGuestStats(guestStats)
		expectStatsUpdate(guestStats)

		controller.Execute()
		addVmiToQueue(vmi)

		guestStats.DirtyRate.SampleCount = 1234567
		setGuestStats(guestStats)
		expectStatsUpdate(guestStats)

		controller.Execute()
	})

	It("Ensure dirty rate is not being updated otherwise", func() {
		addVmiToQueue(vmi)

		guestStats := newGuestStats(1, 1234, 0)
		setGuestStats(guestStats)
		expectStatsUpdate(guestStats)

		controller.Execute()
		addVmiToQueue(vmi)

		guestStats = newGuestStats(2, 1236, 11223)
		setGuestStats(guestStats)

		controller.Execute()
	})
})
