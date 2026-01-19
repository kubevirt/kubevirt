package reporter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/vmlogchecker"
)

const testAnnotationKey = "kubevirt.io/created-by-test"

var failOnVMLogErrors = os.Getenv("FAIL_ON_VM_LOG_ERRORS") != "false"

// CheckVMLogsAfterTest is intended to be called from a JustAfterEach block.
func CheckVMLogsAfterTest(specReport types.SpecReport) {
	if specReport.Failed() || specReport.State.Is(types.SpecStateSkipped) {
		return
	}

	testName := specReport.FullText()
	foundErrors := getVMILogErrors(testName)
	if len(foundErrors) == 0 {
		return
	}

	if failOnVMLogErrors {
		ginkgo.Fail(fmt.Sprintf("VM logs contain unexpected errors:\n%s", strings.Join(foundErrors, "\n")))
	} else {
		saveVMLogErrors(testName, foundErrors)
	}
}

func getVMILogErrors(testName string) []string {
	virtCli := kubevirt.Client()

	var foundErrors []string

	for _, namespace := range testsuite.TestNamespaces {
		vmis, err := virtCli.VirtualMachineInstance(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Failed to list VMIs in namespace %s", namespace)
			continue
		}

		for _, vmi := range vmis.Items {
			if vmi.Annotations == nil {
				continue
			}
			createdBy, ok := vmi.Annotations[testAnnotationKey]
			if !ok || createdBy != testName {
				continue
			}

			labelSelector := fmt.Sprintf("%s=%s", virtv1.CreatedByLabel, string(vmi.GetUID()))
			pods, err := virtCli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Failed to list pods for VMI %s/%s", namespace, vmi.Name)
				continue
			}
			if len(pods.Items) == 0 {
				continue
			}

			for _, pod := range pods.Items {
				if pod.DeletionTimestamp != nil {
					continue
				}

				logsRaw, err := virtCli.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{
					Container: "compute",
				}).DoRaw(context.Background())
				if err != nil {
					log.DefaultLogger().Reason(err).Errorf("Failed to get logs for pod %s/%s", pod.Namespace, pod.Name)
					continue
				}

				errors := findDisallowedErrors(string(logsRaw), vmi.Name)
				foundErrors = append(foundErrors, errors...)
			}
		}
	}

	return foundErrors
}

func saveVMLogErrors(testName string, errors []string) {
	artifactsDir := flags.ArtifactsDir
	if artifactsDir == "" {
		return
	}

	filename := filepath.Join(artifactsDir, "vm-log-errors.log")

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to open VM log errors file: %s", filename)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "=== Test: %s ===\n", testName)
	fmt.Fprintln(f, strings.Join(errors, "\n"))
}

func findDisallowedErrors(logs string, vmiName string) []string {
	var disallowedErrors []string

	for _, line := range strings.Split(logs, "\n") {
		if !vmlogchecker.IsErrorLevel(line) {
			continue
		}

		if vmlogchecker.ClassifyLogLine(line) == vmlogchecker.UnexpectedError {
			disallowedErrors = append(disallowedErrors, formatErrorLine(vmiName, line))
		}
	}

	return disallowedErrors
}

func formatErrorLine(vmiName string, line string) string {
	if vmiName != "" {
		return fmt.Sprintf("[%s] %s", vmiName, line)
	}
	return line
}
