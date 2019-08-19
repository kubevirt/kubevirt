package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/utils/conditions"
	testsutils "kubevirt.io/machine-remediation-operator/tests/utils"

	mapiv1 "sigs.k8s.io/cluster-api/pkg/apis/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[Feature:MachineDisruptionBudget] MachineDisruptionBudget controller", func() {
	var c client.Client
	var workerNode *corev1.Node
	var workerMachineSet *mapiv1.MachineSet
	var testMdb *mrv1.MachineDisruptionBudget

	mdbName := "test-mdb"
	getCurrentHealthyMachines := func() int32 {
		updateMdb := &mrv1.MachineDisruptionBudget{}
		key := types.NamespacedName{
			Name:      mdbName,
			Namespace: testsutils.NamespaceOpenShiftMachineAPI,
		}
		err := c.Get(context.TODO(), key, updateMdb)
		if err != nil {
			return 0
		}
		return updateMdb.Status.CurrentHealthy
	}

	BeforeEach(func() {
		var err error
		c, err = testsutils.LoadClient()
		Expect(err).ToNot(HaveOccurred())

		By("Getting worker node")
		workerNodes, err := testsutils.GetWorkerNodes(c)
		Expect(err).ToNot(HaveOccurred())

		readyWorkerNodes := testsutils.FilterReadyNodes(workerNodes)
		Expect(readyWorkerNodes).ToNot(BeEmpty())

		workerNode = &readyWorkerNodes[0]
		glog.V(2).Infof("Worker node %s", workerNode.Name)

		By("Getting worker machine")
		workerMachine, err := testsutils.GetMachineFromNode(c, workerNode)
		Expect(err).ToNot(HaveOccurred())
		glog.V(2).Infof("Worker machine %s", workerMachine.Name)

		By("Geting worker machine set")
		workerMachineSet, err = testsutils.GetMachinesSetByMachine(workerMachine)
		Expect(err).ToNot(HaveOccurred())

		glog.V(2).Infof("Create machine health check with label selector: %s", workerMachine.Labels)
		err = testsutils.CreateMachineHealthCheck(testsutils.MachineHealthCheckName, workerMachine.Labels)
		Expect(err).ToNot(HaveOccurred())

		unhealthyConditions := &conditions.UnhealthyConditions{
			Items: []conditions.UnhealthyCondition{
				{
					Name:    "Ready",
					Status:  "Unknown",
					Timeout: "60s",
				},
			},
		}
		glog.V(2).Infof("Create node-unhealthy-conditions configmap")
		err = testsutils.CreateUnhealthyConditionsConfigMap(unhealthyConditions)
		Expect(err).ToNot(HaveOccurred())
	})

	It("updates MDB status", func() {
		minAvailable := int32(3)
		testMdb = testsutils.NewMachineDisruptionBudget(
			mdbName,
			workerMachineSet.Spec.Selector.MatchLabels,
			&minAvailable,
			nil,
		)
		By("Creating MachineDisruptionBudget")
		err := c.Create(context.TODO(), testMdb)
		Expect(err).ToNot(HaveOccurred())

		updateMdb := &mrv1.MachineDisruptionBudget{}
		Eventually(func() int32 {
			key := types.NamespacedName{
				Name:      mdbName,
				Namespace: testsutils.NamespaceOpenShiftMachineAPI,
			}
			err := c.Get(context.TODO(), key, updateMdb)
			if err != nil {
				return 0
			}
			return updateMdb.Status.Total
		}, 120*time.Second, time.Second).Should(Equal(*workerMachineSet.Spec.Replicas))

		currentHealthy := updateMdb.Status.CurrentHealthy
		Expect(currentHealthy).To(Equal(workerMachineSet.Status.ReadyReplicas))

		By(fmt.Sprintf("Stopping kubelet service on the node %s", workerNode.Name))
		err = testsutils.StopKubelet(workerNode.Name)
		Expect(err).ToNot(HaveOccurred())

		// Waiting until worker machine will have unhealthy node
		Eventually(getCurrentHealthyMachines, 6*time.Minute, 10*time.Second).Should(Equal(currentHealthy - 1))

		// Waiting until machine set will create new healthy machine
		Eventually(getCurrentHealthyMachines, 15*time.Minute, 30*time.Second).Should(Equal(currentHealthy))
	})

	AfterEach(func() {
		err := c.Delete(context.TODO(), testMdb)
		Expect(err).ToNot(HaveOccurred())

		err = testsutils.DeleteMachineHealthCheck(testsutils.MachineHealthCheckName)
		Expect(err).ToNot(HaveOccurred())

		err = testsutils.DeleteKubeletKillerPods()
		Expect(err).ToNot(HaveOccurred())

		err = testsutils.DeleteUnhealthyConditionsConfigMap()
		Expect(err).ToNot(HaveOccurred())
	})
})
