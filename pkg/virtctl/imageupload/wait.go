package imageupload

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
)

func (c *command) waitUploadServerReady() error {
	loggedStatus := false

	err := virtwait.PollImmediately(uploadReadyWaitInterval, time.Duration(c.uploadPodWaitSecs)*time.Second, func(ctx context.Context) (bool, error) {
		pvc, err := c.client.CoreV1().PersistentVolumeClaims(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
		if err != nil {
			// DataVolume controller may not have created the PVC yet
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		// upload controller sets this to true when uploadserver pod is ready to receive data
		podReady := pvc.Annotations[podReadyAnnotation]
		done, _ := strconv.ParseBool(podReady)

		if !done {
			// We check events to provide user with pertinent error messages
			if err := c.handleEventErrors(c.name, c.name); err != nil {
				return false, err
			}
			if !loggedStatus {
				c.cmd.Printf("Waiting for PVC %s upload pod to be ready...\n", c.name)
				loggedStatus = true
			}
		}

		if done && loggedStatus {
			c.cmd.Printf("Pod now ready\n")
		}

		return done, nil
	})

	return err
}

func (c *command) waitDvUploadScheduled() error {
	loggedStatus := false
	err := virtwait.PollImmediately(uploadReadyWaitInterval, time.Duration(c.uploadPodWaitSecs)*time.Second, func(ctx context.Context) (bool, error) {
		dv, err := c.client.CdiClient().CdiV1beta1().DataVolumes(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
		if err != nil {
			// DataVolume controller may not have created the DV yet
			if k8serrors.IsNotFound(err) {
				c.cmd.Printf("DV %s not found... \n", c.name)
				return false, nil
			}

			return false, err
		}

		if (dv.Status.Phase == cdiv1.WaitForFirstConsumer || dv.Status.Phase == cdiv1.PendingPopulation) && !c.forceBind {
			return false, fmt.Errorf("cannot upload to DataVolume in %s phase, make sure the PVC is Bound, or use force-bind flag", string(dv.Status.Phase))
		}

		done := dv.Status.Phase == cdiv1.UploadReady
		if !done {
			// We check events to provide user with pertinent error messages
			if err := c.handleEventErrors(dv.Status.ClaimName, c.name); err != nil {
				return false, err
			}
			if !loggedStatus {
				c.cmd.Printf("Waiting for PVC %s upload pod to be ready...\n", c.name)
				loggedStatus = true
			}
		}

		if done && loggedStatus {
			c.cmd.Printf("Pod now ready\n")
		}

		return done, nil
	})

	return err
}

func waitUploadProcessingComplete(client kubernetes.Interface, cmd *cobra.Command, namespace, name string, interval, timeout time.Duration) error {
	err := virtwait.PollImmediately(interval, timeout, func(ctx context.Context) (bool, error) {
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// upload controller sets this to true when uploadserver pod is ready to receive data
		podPhase := pvc.Annotations[podPhaseAnnotation]

		if podPhase == string(v1.PodSucceeded) {
			cmd.Printf("Processing completed successfully\n")
		}

		return podPhase == string(v1.PodSucceeded), nil
	})

	return err
}
