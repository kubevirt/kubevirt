package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/machine-remediation-operator/pkg/utils/conditions"
	testsutils "kubevirt.io/machine-remediation-operator/tests/utils"

	mapiv1 "sigs.k8s.io/cluster-api/pkg/apis/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	machineAPIControllers       = "machine-api-controllers"
	machineHealthCheckControler = "machine-healthcheck-controller"
)

var _ = Describe("[TechPreview:Feature:MachineHealthCheck] MachineHealthCheck controller", func() {
	var c client.Client
	var numberOfReadyWorkers int
	var workerNode *corev1.Node
	var workerMachine *mapiv1.Machine

	stopKubeletAndValidateMachineDeletion := func(workerNodeName *corev1.Node, workerMachine *mapiv1.Machine, timeout time.Duration) {
		By(fmt.Sprintf("Stopping kubelet service on the node %s", workerNode.Name))
		err := testsutils.StopKubelet(workerNode.Name)
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("Validating that node %s has 'NotReady' condition", workerNode.Name))
		waitForNodeUnhealthyCondition(workerNode.Name)

		By(fmt.Sprintf("Validating that machine %s is deleted", workerMachine.Name))
		machine := &mapiv1.Machine{}
		key := types.NamespacedName{
			Namespace: workerMachine.Namespace,
			Name:      workerMachine.Name,
		}
		Eventually(func() bool {
			err := c.Get(context.TODO(), key, machine)
			if err != nil {
				if errors.IsNotFound(err) {
					return true
				}
			}
			glog.V(2).Infof("machine deletion timestamp %s still exists", machine.DeletionTimestamp)
			return false
		}, timeout, 5*time.Second).Should(BeTrue())
	}

	BeforeEach(func() {
		var err error
		c, err = testsutils.LoadClient()
		Expect(err).ToNot(HaveOccurred())

		workerNodes, err := testsutils.GetWorkerNodes(c)
		Expect(err).ToNot(HaveOccurred())

		readyWorkerNodes := testsutils.FilterReadyNodes(workerNodes)
		Expect(readyWorkerNodes).ToNot(BeEmpty())

		numberOfReadyWorkers = len(readyWorkerNodes)
		workerNode = &readyWorkerNodes[0]
		glog.V(2).Infof("Worker node %s", workerNode.Name)

		workerMachine, err = testsutils.GetMachineFromNode(c, workerNode)
		Expect(err).ToNot(HaveOccurred())
		glog.V(2).Infof("Worker machine %s", workerMachine.Name)

		glog.V(2).Infof("Create machine health check with label selector: %s", workerMachine.Labels)
		err = testsutils.CreateMachineHealthCheck(testsutils.MachineHealthCheckName, workerMachine.Labels)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("with node-unhealthy-conditions configmap", func() {
		BeforeEach(func() {
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
			err := testsutils.CreateUnhealthyConditionsConfigMap(unhealthyConditions)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete unhealthy machine", func() {
			stopKubeletAndValidateMachineDeletion(workerNode, workerMachine, 2*time.Minute)
		})

		AfterEach(func() {
			glog.V(2).Infof("Delete node-unhealthy-conditions configmap")
			err := testsutils.DeleteUnhealthyConditionsConfigMap()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	It("should delete unhealthy machine", func() {
		stopKubeletAndValidateMachineDeletion(workerNode, workerMachine, 6*time.Minute)
	})

	AfterEach(func() {
		waitForWorkersToGetReady(numberOfReadyWorkers)
		testsutils.DeleteMachineHealthCheck(testsutils.MachineHealthCheckName)
		testsutils.DeleteKubeletKillerPods()
	})
})

func waitForNodeUnhealthyCondition(workerNodeName string) {
	c, err := testsutils.LoadClient()
	Expect(err).ToNot(HaveOccurred())

	key := types.NamespacedName{
		Name:      workerNodeName,
		Namespace: testsutils.NamespaceOpenShiftMachineAPI,
	}
	node := &corev1.Node{}
	glog.Infof("Wait until node %s will have 'Ready' condition with the status %s", node.Name, corev1.ConditionUnknown)
	Eventually(func() bool {
		err := c.Get(context.TODO(), key, node)
		if err != nil {
			return false
		}
		readyCond := conditions.GetNodeCondition(node, corev1.NodeReady)
		glog.V(2).Infof("Node %s has 'Ready' condition with the status %s", node.Name, readyCond.Status)
		return readyCond.Status == corev1.ConditionUnknown
	}, testsutils.WaitLong, 10*time.Second).Should(BeTrue())
}

func waitForWorkersToGetReady(numberOfReadyWorkers int) {
	client, err := testsutils.LoadClient()
	Expect(err).ToNot(HaveOccurred())

	glog.V(2).Infof("Wait until the environment will have %d ready workers", numberOfReadyWorkers)
	Eventually(func() bool {
		workerNodes, err := testsutils.GetWorkerNodes(client)
		if err != nil {
			return false
		}

		readyWorkerNodes := testsutils.FilterReadyNodes(workerNodes)
		glog.V(2).Infof("Number of ready workers %d", len(readyWorkerNodes))
		return len(readyWorkerNodes) == numberOfReadyWorkers
	}, 15*time.Minute, 10*time.Second).Should(BeTrue())
}
