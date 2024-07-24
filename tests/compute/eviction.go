package compute

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"

	k8sv1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = SIGDescribe("Eviction", func() {

	Context("without PDBs", func() {

		It("should not shutdown VM", func() {
			vmi := libvmifact.NewAlpine(
				libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			ctx := context.Background()
			_, cancel := context.WithCancel(ctx)
			defer cancel()
			errChan := make(chan error)

			go func() {
				virtClient := kubevirt.Client()

				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
					}

					err := virtClient.CoreV1().Pods(pod.Namespace).EvictV1(context.TODO(), &policy.Eviction{
						ObjectMeta: metav1.ObjectMeta{
							Name:      pod.Name,
							Namespace: pod.Namespace,
						},
						DeleteOptions: &metav1.DeleteOptions{
							Preconditions: &metav1.Preconditions{
								UID: &pod.UID,
							},
							GracePeriodSeconds: pointer.P(int64(0)),
						},
					})
					if !k8serrors.IsTooManyRequests(err) {
						errChan <- err
						return
					}

					time.Sleep(time.Second)
				}

			}()

			Eventually(matcher.ThisPod(pod)).WithTimeout(time.Minute).WithPolling(5 * time.Second).
				Should(Or(matcher.BeInPhase(k8sv1.PodSucceeded), matcher.BeGone()))
			Expect(matcher.ThisVMI(vmi)()).To(matcher.BeRunning())
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			Expect(errChan).To(BeEmpty())
		})
	})

})
