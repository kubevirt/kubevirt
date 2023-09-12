package reporter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"kubevirt.io/kubevirt/tests/libnode"

	"k8s.io/client-go/util/flowcontrol"

	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v12 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	apicdi "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	failedCreateDirectoryFmt     = "failed to create directory: %v\n"
	failedOpenFileFmt            = "failed to open the file: %v\n"
	failedGetVirtHandlerPodFmt   = "failed to get virt-handler pod on node %s: %v\n"
	virtHandlerName              = "virt-handler"
	computeContainer             = "compute"
	virtLauncherNameFmt          = "%s=virt-launcher"
	failedCreateLogsDirectoryFmt = "failed to create directory %s: %v\n"
	logFileNameFmt               = "%d_%s_%s.log"
	ipAddrName                   = "ip address"
	ipLinkName                   = "ip link"
	ipRouteShowTableAll          = "ip route show table all"
	ipNeighShow                  = "ip neigh show"
	bridgeJVlanShow              = "bridge -j vlan show"
	bridgeFdb                    = "bridge fdb"
	devVFio                      = "ls -lsh -Z -St /dev/vfio"
	failedExecuteCmdFmt          = "failed to execute command %s on %s, stdout: %s, stderr: %s, error: %v\n"
	failedExecuteCmdOnNodeFmt    = "failed to execute command %s on node %s, stdout: %s, error: %v\n"
)

const (
	k8sCNICNCFEntityURLTemplate       = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/%s/"
	networkAttachmentDefinitionEntity = "network-attachment-definitions"
)

type KubernetesReporter struct {
	failureCount int
	artifactsDir string
	maxFails     int
}

type commands struct {
	command        string
	fileNameSuffix string
}

func NewKubernetesReporter(artifactsDir string, maxFailures int) *KubernetesReporter {
	return &KubernetesReporter{
		failureCount: 0,
		artifactsDir: artifactsDir,
		maxFails:     maxFailures,
	}
}

func (r *KubernetesReporter) JustAfterEach(specReport types.SpecReport) {
	fmt.Fprintf(GinkgoWriter, "On failure, artifacts will be collected in %s/%d_*\n", r.artifactsDir, r.failureCount+1)

	if r.failureCount > r.maxFails {
		return
	}
	if specReport.Failed() {
		r.failureCount++
	} else {
		return
	}

	// If we got not directory, print to stderr
	if r.artifactsDir == "" {
		return
	}
	By("Collecting Logs for failed test")
	r.DumpTestNamespaces(specReport.RunTime)
}

func (r *KubernetesReporter) DumpTestNamespaces(duration time.Duration) {
	r.dumpNamespaces(duration, testsuite.TestNamespaces)
}

func (r *KubernetesReporter) DumpAllNamespaces(duration time.Duration) {
	r.dumpNamespaces(duration, []string{v1.NamespaceAll})
}

func (r *KubernetesReporter) dumpNamespaces(duration time.Duration, vmiNamespaces []string) {
	cfg, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get client config: %v\n", err)
		return
	}
	// we fetch quite some stuff, this can take ages if we don't increase the default rate limit
	cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 100)
	virtCli, err := kubecli.GetKubevirtClientFromRESTConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create client: %v\n", err)
		return
	}

	if err := os.MkdirAll(r.artifactsDir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, failedCreateDirectoryFmt, err)
		return
	}

	nodesDir := r.createNodesDir()
	podsDir := r.createPodsDir()
	networkPodsDir := r.createNetworkPodsDir()

	duration += 5 * time.Second
	since := time.Now().Add(-duration)

	nodes := getNodeList(virtCli)
	nodesWithTestPods := getNodesRunningTests(virtCli)
	pods := getPodList(virtCli)
	virtHandlerPods := getVirtHandlerList(virtCli)
	vmis := getVMIList(virtCli)
	vmims := getVMIMList(virtCli)

	r.logClusterOverview()
	r.logEvents(virtCli, since)
	r.logNamespaces(virtCli)
	r.logPVCs(virtCli)
	r.logPVs(virtCli)
	r.logStorageClasses(virtCli)
	r.logCSIDrivers(virtCli)
	r.logAPIServices(virtCli)
	r.logServices(virtCli)
	r.logEndpoints(virtCli)
	r.logConfigMaps(virtCli)
	r.logSecrets(virtCli)
	r.logNetworkAttachmentDefinitionInfo(virtCli)
	r.logKubeVirtCR(virtCli)
	r.logNodes(virtCli, nodes)
	r.logPods(virtCli, pods)
	r.logVMs(virtCli)
	r.logVMSnapshot(virtCli)
	r.logVMRestore(virtCli)
	r.logDVs(virtCli)
	r.logVMExports(virtCli)
	r.logDeployments(virtCli)
	r.logDaemonsets(virtCli)
	r.logVolumeSnapshots(virtCli)
	r.logVolumeSnapshotContents(virtCli)
	r.logVolumeSnapshotClasses(virtCli)
	r.logVirtualMachineSnapshots(virtCli)
	r.logVirtualMachineSnapshotContents(virtCli)

	r.logAuditLogs(virtCli, nodesDir, nodesWithTestPods, since)
	r.logDMESG(virtCli, nodesDir, nodesWithTestPods, since)
	r.logJournal(virtCli, nodesDir, nodesWithTestPods, duration, "")
	r.logJournal(virtCli, nodesDir, nodesWithTestPods, duration, "kubelet")

	r.logLogs(virtCli, podsDir, pods, since)

	r.logVMIs(virtCli, vmis)
	r.logDomainXMLs(virtCli, vmis)

	r.logVMIMs(virtCli, vmims)

	r.logNodeCommands(virtCli, nodesWithTestPods)
	r.logVirtLauncherCommands(virtCli, networkPodsDir)
	r.logVirtLauncherPrivilegedCommands(virtCli, networkPodsDir, virtHandlerPods)
	r.logVMICommands(virtCli, vmiNamespaces)

	r.logCloudInit(virtCli, vmiNamespaces)
	r.logVirtualMachinePools(virtCli)
}

// Cleanup cleans up the current content of the artifactsDir
func (r *KubernetesReporter) Cleanup() {
	// clean up artifacts from previous run
	if r.artifactsDir != "" {
		os.RemoveAll(r.artifactsDir)
	}
}

func (r *KubernetesReporter) logDomainXMLs(virtCli kubecli.KubevirtClient, vmis *v12.VirtualMachineInstanceList) {

	if vmis == nil {
		fmt.Fprintf(os.Stderr, "vmi list is empty, skipping logDomainXMLs\n")
		return
	}

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_domains.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, failedOpenFileFmt, err)
		return
	}
	defer f.Close()

	for _, vmi := range vmis.Items {
		if vmi.IsFinal() {
			continue
		}
		domxml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtCli, &vmi)
		if err == nil {
			fmt.Fprintln(f, domxml)
		}
	}
}

func (r *KubernetesReporter) logVMs(virtCli kubecli.KubevirtClient) {
	vms, err := virtCli.VirtualMachine(v1.NamespaceAll).List(context.Background(), &metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vms: %v\n", err)
		return
	}
	r.logObjects(virtCli, vms, "vms")
}

func (r *KubernetesReporter) logVMIs(virtCli kubecli.KubevirtClient, vmis *v12.VirtualMachineInstanceList) {
	r.logObjects(virtCli, vmis, "vmis")
}

func (r *KubernetesReporter) logVMIMs(virtCli kubecli.KubevirtClient, vmims *v12.VirtualMachineInstanceMigrationList) {
	r.logObjects(virtCli, vmims, "vmims")
}

func (r *KubernetesReporter) logVMSnapshot(virtCli kubecli.KubevirtClient) {
	snapshots, err := virtCli.VirtualMachineSnapshot(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vmsnapshots: %v\n", err)
		return
	}
	r.logObjects(virtCli, snapshots, "virtualmachinesnapshots")
}

func (r *KubernetesReporter) logVMRestore(virtCli kubecli.KubevirtClient) {
	restores, err := virtCli.VirtualMachineRestore(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vmrestores: %v\n", err)
		return
	}
	r.logObjects(virtCli, restores, "virtualmachinerestores")
}

func (r *KubernetesReporter) logDMESG(virtCli kubecli.KubevirtClient, logsdir string, nodes []string, since time.Time) {

	if logsdir == "" {
		fmt.Fprintf(os.Stderr, "logsdir is empty, skipping logDMESG\n")
		return
	}

	timestampRexp := regexp.MustCompile(`\[([^]]+)]`)
	for _, node := range nodes {
		func() {
			fileName := fmt.Sprintf("%d_dmesg_%s.log", r.failureCount, node)
			f, err := os.OpenFile(filepath.Join(logsdir, fileName),
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open the file %s: %v\n", fileName, err)
				return
			}
			defer f.Close()
			pod, err := libnode.GetVirtHandlerPod(virtCli, node)
			if err != nil {
				fmt.Fprintf(os.Stderr, failedGetVirtHandlerPodFmt, node, err)
				return
			}

			commands := []string{
				virt_chroot.GetChrootBinaryPath(),
				"--mount",
				virt_chroot.GetChrootMountNamespace(),
				"exec",
				"--",
				"/proc/1/root/bin/dmesg",
				"--kernel",
				"--ctime",
				"--userspace",
				"--decode",
			}

			// TODO may need to be improved, in case that the auditlog is really huge, since stdout is in memory
			stdout, _, err := exec.ExecuteCommandOnPodWithResults(virtCli, pod, virtHandlerName, commands)
			if err != nil {
				fmt.Fprintf(
					os.Stderr,
					failedExecuteCmdOnNodeFmt,
					commands,
					node, stdout, err,
				)
				return
			}
			scanner := bufio.NewScanner(bytes.NewBufferString(stdout))
			add := false
			for scanner.Scan() {
				line := scanner.Text()
				if !add {
					matches := timestampRexp.FindStringSubmatch(line)
					if len(matches) == 0 {
						continue
					}
					timestamp, err := time.Parse("Mon Jan 2 15:04:05 2006", matches[1])
					if err != nil {
						fmt.Fprintf(os.Stderr, "failed to convert iso timestamp: %v\n", err)
						continue
					}
					if !timestamp.UTC().Before(since.UTC()) {
						f.WriteString(line + "\n")
						add = true
					}
				} else {
					f.WriteString(line + "\n")
				}
			}
		}()
	}
}

func (r *KubernetesReporter) logAuditLogs(virtCli kubecli.KubevirtClient, logsdir string, nodes []string, since time.Time) {

	if logsdir == "" {
		fmt.Fprintf(os.Stderr, "logsdir is empty, skipping logAuditLogs\n")
		return
	}

	timestampRexp := regexp.MustCompile(`audit\(([0-9]+)[0-9.:]+\)`)
	for _, node := range nodes {
		func() {
			fileName := fmt.Sprintf("%d_auditlog_%s.log", r.failureCount, node)
			f, err := os.OpenFile(filepath.Join(logsdir, fileName),
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open the file %s: %v\n", fileName, err)
				return
			}
			defer f.Close()
			pod, err := libnode.GetVirtHandlerPod(virtCli, node)
			if err != nil {
				fmt.Fprintf(os.Stderr, failedGetVirtHandlerPodFmt, node, err)
				return
			}
			// TODO may need to be improved, in case that the auditlog is really huge, since stdout is in memory
			getAuditLogCmd := []string{"cat", "/proc/1/root/var/log/audit/audit.log"}
			stdout, _, err := exec.ExecuteCommandOnPodWithResults(virtCli, pod, virtHandlerName, getAuditLogCmd)
			if err != nil {
				fmt.Fprintf(
					os.Stderr,
					failedExecuteCmdOnNodeFmt,
					getAuditLogCmd, node, stdout, err,
				)
				return
			}
			scanner := bufio.NewScanner(bytes.NewBufferString(stdout))
			add := false
			for scanner.Scan() {
				line := scanner.Text()
				if !add {
					matches := timestampRexp.FindStringSubmatch(line)
					if len(matches) == 0 {
						continue
					}
					timestamp, err := strconv.ParseInt(matches[1], 10, 64)
					if err != nil {
						fmt.Fprintf(os.Stderr, "failed to convert string to unix timestamp: %v\n", err)
						continue
					}
					if !time.Unix(timestamp, 0).Before(since) {
						f.WriteString(line + "\n")
						add = true
					}
				} else {
					f.WriteString(line + "\n")
				}
			}
		}()
	}
}

func (r *KubernetesReporter) logVMICommands(virtCli kubecli.KubevirtClient, vmiNamespaces []string) {
	runningVMIs := getRunningVMIs(virtCli, vmiNamespaces)

	if len(runningVMIs) < 1 {
		return
	}

	logsDir := filepath.Join(r.artifactsDir, "network", "vmis")
	if err := os.MkdirAll(logsDir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, failedCreateDirectoryFmt, err)
		return
	}

	for _, vmi := range runningVMIs {
		vmiType, err := getVmiType(vmi)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping vmi %s/%s: failed to get vmi type: %v\n", vmi.Namespace, vmi.Name, err)
			continue
		}

		r.executeVMICommands(vmi, logsDir, vmiType)
	}
}

func (r *KubernetesReporter) logCloudInit(virtCli kubecli.KubevirtClient, vmiNamespaces []string) {
	runningVMIs := getRunningVMIs(virtCli, vmiNamespaces)

	if len(runningVMIs) < 1 {
		return
	}

	logsDir := filepath.Join(r.artifactsDir, "cloud-init")
	if err := os.MkdirAll(logsDir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory %s: %v\n", logsDir, err)
		return
	}

	for _, vmi := range runningVMIs {
		vmiType, err := getVmiType(vmi)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping vmi %s/%s: failed to get vmi type: %v\n", vmi.Namespace, vmi.Name, err)
			continue
		}

		r.executeCloudInitCommands(vmi, logsDir, vmiType)
	}
}

func (r *KubernetesReporter) logVirtLauncherPrivilegedCommands(virtCli kubecli.KubevirtClient, logsdir string, virtHandlerPods *v1.PodList) {

	if logsdir == "" {
		fmt.Fprintf(os.Stderr, "logsdir is empty, skipping logVirtLauncherPrivilegedCommands\n")
		return
	}

	if virtHandlerPods == nil {
		fmt.Fprintf(os.Stderr, "virt-handler pod list is empty, skipping logVirtLauncherPrivilegedCommands\n")
		return
	}

	nodeMap := map[string]v1.Pod{}
	for _, virtHandlerPod := range virtHandlerPods.Items {
		if virtHandlerPod.Status.Phase != "Running" {
			fmt.Fprintf(os.Stderr, "skipping virt-handler %s, phase is not Running\n", virtHandlerPod.ObjectMeta.Name)
			continue
		}

		nodeMap[virtHandlerPod.Spec.NodeName] = virtHandlerPod
	}

	virtLauncherPods, err := virtCli.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf(virtLauncherNameFmt, v12.AppLabel)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch virt-launcher pods: %v\n", err)
		return
	}

	for _, virtLauncherPod := range virtLauncherPods.Items {
		if virtHandlerPod, ok := nodeMap[virtLauncherPod.Spec.NodeName]; ok {
			labels := virtLauncherPod.GetLabels()
			if uid, ok := labels["kubevirt.io/created-by"]; ok {
				pid, err := getVirtLauncherMonitorPID(virtCli, &virtHandlerPod, uid)
				if err != nil {
					continue
				}

				r.executePriviledgedVirtLauncherCommands(virtCli, &virtHandlerPod, logsdir, pid, virtLauncherPod.ObjectMeta.Name)
			}
		}
	}
}

func (r *KubernetesReporter) logVirtLauncherCommands(virtCli kubecli.KubevirtClient, logsdir string) {

	if logsdir == "" {
		fmt.Fprintf(os.Stderr, "logsdir is empty, skipping logVirtLauncherCommands\n")
		return
	}

	virtLauncherPods, err := virtCli.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf(virtLauncherNameFmt, v12.AppLabel)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch virt-launcher pods: %v\n", err)
		return
	}

	for _, pod := range virtLauncherPods.Items {
		if pod.Status.Phase != "Running" {
			fmt.Fprintf(os.Stderr, "skipping pod %s, phase is not Running\n", pod.ObjectMeta.Name)
			continue
		}

		if !isContainerReady(pod.Status.ContainerStatuses, computeContainer) {
			fmt.Fprintf(os.Stderr, "could not find healty compute container for pod %s\n", pod.ObjectMeta.Name)
			continue
		}

		r.executeVirtLauncherCommands(virtCli, logsdir, pod)
	}
}

func isContainerReady(containerStatuses []v1.ContainerStatus, containerName string) bool {
	for _, containerStatus := range containerStatuses {
		if containerStatus.Name == containerName {
			return containerStatus.Ready
		}
	}

	return false
}

func (r *KubernetesReporter) logNodeCommands(virtCli kubecli.KubevirtClient, nodes []string) {
	logsdir := filepath.Join(r.artifactsDir, "network", "nodes")
	if err := os.MkdirAll(logsdir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, failedCreateLogsDirectoryFmt, logsdir, err)
		return
	}

	for _, node := range nodes {
		pod, err := libnode.GetVirtHandlerPod(virtCli, node)
		if err != nil {
			fmt.Fprintf(os.Stderr, failedGetVirtHandlerPodFmt, node, err)
			continue
		}

		if pod.Status.Phase != "Running" {
			fmt.Fprintf(os.Stderr, "skipping node's pod %s, phase is not Running\n", pod.ObjectMeta.Name)
			continue
		}

		r.executeNodeCommands(virtCli, logsdir, pod)
	}
}

func (r *KubernetesReporter) logJournal(virtCli kubecli.KubevirtClient, logsdir string, nodes []string, duration time.Duration, unit string) {

	if logsdir == "" {
		fmt.Fprintf(os.Stderr, "logsdir is empty, skipping logJournal\n")
		return
	}

	var component string = "journal"
	var unitCommandArgs []string

	if unit != "" {
		component += "_" + unit
		unitCommandArgs = append(unitCommandArgs, "-u", unit)
	}

	logDuration := strconv.FormatInt(int64(duration/time.Second), 10)

	for _, node := range nodes {
		pod, err := libnode.GetVirtHandlerPod(virtCli, node)
		if err != nil {
			fmt.Fprintf(os.Stderr, failedGetVirtHandlerPodFmt, node, err)
			continue
		}

		commands := []string{
			virt_chroot.GetChrootBinaryPath(),
			"--mount",
			virt_chroot.GetChrootMountNamespace(),
			"exec",
			"--",
			"/usr/bin/journalctl",
			"--since",
			"-" + logDuration + "s",
		}
		commands = append(commands, unitCommandArgs...)

		stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtCli, pod, virtHandlerName, commands)
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"failed to execute command %s on node %s, stdout: %s, stderr: %s, error: %v\n",
				commands, node, stdout, stderr, err,
			)
			continue
		}

		fileName := fmt.Sprintf(logFileNameFmt, r.failureCount, component, node)
		err = writeStringToFile(filepath.Join(logsdir, fileName), stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write node %s logs: %v\n", node, err)
			continue
		}
	}
}

func (r *KubernetesReporter) logPods(virtCli kubecli.KubevirtClient, pods *v1.PodList) {
	r.logObjects(virtCli, pods, "pods")
}

func (r *KubernetesReporter) logServices(virtCli kubecli.KubevirtClient) {
	services, err := virtCli.CoreV1().Services(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch services: %v\n", err)
		return
	}

	r.logObjects(virtCli, services, "services")
}

func (r *KubernetesReporter) logAPIServices(virtCli kubecli.KubevirtClient) {
	result, err := virtCli.RestClient().Get().RequestURI("/apis/apiregistration.k8s.io/v1/").Resource("apiservices").Do(context.Background()).Raw()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch apiServices: %v\n", err)
		return
	}
	apiServices := apiregv1.APIServiceList{}
	err = json.Unmarshal(result, &apiServices)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to unmarshal raw result to apiServicesList: %v\n", err)
	}

	r.logObjects(virtCli, apiServices, "apiServices")
}

func (r *KubernetesReporter) logEndpoints(virtCli kubecli.KubevirtClient) {
	endpoints, err := virtCli.CoreV1().Endpoints(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch endpointss: %v\n", err)
		return
	}

	r.logObjects(virtCli, endpoints, "endpoints")
}

func (r *KubernetesReporter) logConfigMaps(virtCli kubecli.KubevirtClient) {
	configmaps, err := virtCli.CoreV1().ConfigMaps(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch configmaps: %v\n", err)
		return
	}

	r.logObjects(virtCli, configmaps, "configmaps")
}

func (r *KubernetesReporter) logKubeVirtCR(virtCli kubecli.KubevirtClient) {
	kvs, err := virtCli.KubeVirt(flags.KubeVirtInstallNamespace).List(&metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch kubevirts: %v\n", err)
		return
	}

	r.logObjects(virtCli, kvs, "kubevirtCR")
}

func (r *KubernetesReporter) logSecrets(virtCli kubecli.KubevirtClient) {
	secrets, err := virtCli.CoreV1().Secrets(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch secrets: %v\n", err)
		return
	}

	r.logObjects(virtCli, secrets, "secrets")
}

func (r *KubernetesReporter) logNamespaces(virtCli kubecli.KubevirtClient) {
	namespaces, err := virtCli.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch Namespaces: %v\n", err)
		return
	}

	r.logObjects(virtCli, namespaces, "namespaces")
}

func (r *KubernetesReporter) logNodes(virtCli kubecli.KubevirtClient, nodes *v1.NodeList) {
	r.logObjects(virtCli, nodes, "nodes")
}

func (r *KubernetesReporter) logPVs(virtCli kubecli.KubevirtClient) {
	pvs, err := virtCli.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch pvs: %v\n", err)
		return
	}

	r.logObjects(virtCli, pvs, "pvs")
}

func (r *KubernetesReporter) logStorageClasses(virtCli kubecli.KubevirtClient) {
	storageClasses, err := virtCli.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch storageclasses: %v\n", err)
		return
	}

	r.logObjects(virtCli, storageClasses, "storageclasses")
}

func (r *KubernetesReporter) logCSIDrivers(virtCli kubecli.KubevirtClient) {
	csiDrivers, err := virtCli.StorageV1().CSIDrivers().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch csidrivers: %v\n", err)
		return
	}

	r.logObjects(virtCli, csiDrivers, "csidrivers")
}

func (r *KubernetesReporter) logPVCs(virtCli kubecli.KubevirtClient) {
	pvcs, err := virtCli.CoreV1().PersistentVolumeClaims(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch pvcs: %v\n", err)
		return
	}

	r.logObjects(virtCli, pvcs, "pvcs")
}

func (r *KubernetesReporter) logDeployments(virtCli kubecli.KubevirtClient) {
	deployments, err := virtCli.AppsV1().Deployments(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch deployments: %v\n", err)
		return
	}

	r.logObjects(virtCli, deployments, "deployments")
}

func (r *KubernetesReporter) logDaemonsets(virtCli kubecli.KubevirtClient) {
	daemonsets, err := virtCli.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch daemonsets: %v\n", err)
		return
	}

	r.logObjects(virtCli, daemonsets, "daemonsets")
}

func (r *KubernetesReporter) logVolumeSnapshots(virtCli kubecli.KubevirtClient) {
	volumeSnapshots, err := virtCli.KubernetesSnapshotClient().SnapshotV1().VolumeSnapshots(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch volume snapshots: %v\n", err)
		return
	}

	r.logObjects(virtCli, volumeSnapshots, "volumesnapshots")
}

func (r *KubernetesReporter) logVolumeSnapshotContents(virtCli kubecli.KubevirtClient) {
	volumeSnapshotContents, err := virtCli.KubernetesSnapshotClient().SnapshotV1().VolumeSnapshotContents().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch volume snapshot contents: %v\n", err)
		return
	}

	r.logObjects(virtCli, volumeSnapshotContents, "volumesnapshotcontents")
}

func (r *KubernetesReporter) logVolumeSnapshotClasses(virtCli kubecli.KubevirtClient) {
	volumeSnapshotClasses, err := virtCli.KubernetesSnapshotClient().SnapshotV1().VolumeSnapshotClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch volume snapshot classes: %v\n", err)
		return
	}

	r.logObjects(virtCli, volumeSnapshotClasses, "volumesnapshotclasses")
}

func (r *KubernetesReporter) logVirtualMachineSnapshots(virtCli kubecli.KubevirtClient) {
	volumeSnapshots, err := virtCli.VirtualMachineSnapshot(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch virtual machine snapshots: %v\n", err)
		return
	}

	r.logObjects(virtCli, volumeSnapshots, "virtualmachinesnapshots")
}

func (r *KubernetesReporter) logVirtualMachineSnapshotContents(virtCli kubecli.KubevirtClient) {
	volumeSnapshotContents, err := virtCli.VirtualMachineSnapshotContent(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch virtual machine snapshot contents: %v\n", err)
		return
	}

	r.logObjects(virtCli, volumeSnapshotContents, "virtualmachinenapshotcontents")
}

func (r *KubernetesReporter) logDVs(virtCli kubecli.KubevirtClient) {
	dvEnabled, _ := isDataVolumeEnabled(virtCli)
	if !dvEnabled {
		return
	}
	dvs, err := virtCli.CdiClient().CdiV1beta1().DataVolumes(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch dvs: %v\n", err)
		return
	}

	r.logObjects(virtCli, dvs, "dvs")
}

func (r *KubernetesReporter) logVMExports(virtCli kubecli.KubevirtClient) {
	vmexports, err := virtCli.VirtualMachineExport(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vm exports: %v\n", err)
		return
	}

	r.logObjects(virtCli, vmexports, "vmexports")
}

func (r *KubernetesReporter) logObjects(virtCli kubecli.KubevirtClient, elements interface{}, name string) {
	if elements == nil {
		fmt.Fprintf(os.Stderr, "%s list is empty, skipping\n", name)
		return
	}

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_%s.log", r.failureCount, name)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, failedOpenFileFmt, err)
		return
	}
	defer f.Close()

	j, err := json.MarshalIndent(elements, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal %s", name)
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logLogs(virtCli kubecli.KubevirtClient, logsdir string, pods *v1.PodList, since time.Time) {

	if logsdir == "" {
		fmt.Fprintf(os.Stderr, "logsdir is empty, skipping logLogs\n")
		return
	}

	if pods == nil {
		fmt.Fprintf(os.Stderr, "pod list is empty, skipping logLogs\n")
		return
	}

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			current, err := os.OpenFile(filepath.Join(logsdir, fmt.Sprintf("%d_%s_%s-%s.log", r.failureCount, pod.Namespace, pod.Name, container.Name)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, failedOpenFileFmt, err)
				return
			}
			defer current.Close()

			previous, err := os.OpenFile(filepath.Join(logsdir, fmt.Sprintf("%d_%s_%s-%s_previous.log", r.failureCount, pod.Namespace, pod.Name, container.Name)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, failedOpenFileFmt, err)
				return
			}
			defer previous.Close()

			logStart := metav1.NewTime(since)
			logs, err := virtCli.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{SinceTime: &logStart, Container: container.Name}).DoRaw(context.Background())
			if err == nil {
				fmt.Fprintln(current, string(logs))
			}

			logs, err = virtCli.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{SinceTime: &logStart, Container: container.Name, Previous: true}).DoRaw(context.Background())
			if err == nil {
				fmt.Fprintln(previous, string(logs))
			}
		}
	}
}

func getVirtHandlerList(virtCli kubecli.KubevirtClient) *v1.PodList {

	pods, err := virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=virt-handler", v12.AppLabel)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch virt-handler pods: %v\n", err)
		return nil
	}

	return pods
}

func getVMIList(virtCli kubecli.KubevirtClient) *v12.VirtualMachineInstanceList {

	vmis, err := virtCli.VirtualMachineInstance(v1.NamespaceAll).List(context.Background(), &metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vmis: %v\n", err)
		return nil
	}

	return vmis
}

func getVMIMList(virtCli kubecli.KubevirtClient) *v12.VirtualMachineInstanceMigrationList {

	vmims, err := virtCli.VirtualMachineInstanceMigration(v1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vmims: %v\n", err)
		return nil
	}

	return vmims
}

func getRunningVMIs(virtCli kubecli.KubevirtClient, namespace []string) []v12.VirtualMachineInstance {
	runningVMIs := []v12.VirtualMachineInstance{}

	for _, ns := range namespace {
		nsVMIs, err := virtCli.VirtualMachineInstance(ns).List(context.Background(), &metav1.ListOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get vmis from namespace %s: %v\n", ns, err)
			continue
		}

		for _, vmi := range nsVMIs.Items {
			if vmi.Status.Phase != v12.Running {
				fmt.Fprintf(os.Stderr, "skipping vmi %s/%s: phase is not Running\n", vmi.Namespace, vmi.Name)
				continue
			}

			isPaused := false
			for _, cond := range vmi.Status.Conditions {
				if cond.Type == v12.VirtualMachineInstancePaused && cond.Status == v1.ConditionTrue {
					isPaused = true
					break
				}
			}
			if isPaused {
				fmt.Fprintf(os.Stderr, "skipping paused vmi %s\n", vmi.ObjectMeta.Name)
				continue
			}

			vmiType, err := getVmiType(vmi)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skipping vmi %s/%s: failed to get vmi type: %v\n", vmi.Namespace, vmi.Name, err)
				continue
			}

			if err := prepareVmiConsole(vmi, vmiType); err != nil {
				fmt.Fprintf(os.Stderr, "skipping vmi %s/%s: failed to login: %v\n", vmi.Namespace, vmi.Name, err)
				continue
			}
			runningVMIs = append(runningVMIs, vmi)
		}
	}

	return runningVMIs
}

func getNodeList(virtCli kubecli.KubevirtClient) *v1.NodeList {

	nodes, err := virtCli.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch nodes: %v\n", err)
		return nil
	}

	return nodes
}

func getPodList(virtCli kubecli.KubevirtClient) *v1.PodList {

	pods, err := virtCli.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch pods: %v\n", err)
		return nil
	}

	return pods
}

func (r *KubernetesReporter) createNetworkPodsDir() string {

	logsdir := filepath.Join(r.artifactsDir, "network", "pods")
	if err := os.MkdirAll(logsdir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, failedCreateLogsDirectoryFmt, logsdir, err)
		return ""
	}

	return logsdir
}

func (r *KubernetesReporter) createNodesDir() string {

	logsdir := filepath.Join(r.artifactsDir, "nodes")
	if err := os.MkdirAll(logsdir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, failedCreateLogsDirectoryFmt, logsdir, err)
		return ""
	}

	return logsdir
}

func (r *KubernetesReporter) createPodsDir() string {

	logsdir := filepath.Join(r.artifactsDir, "pods")
	if err := os.MkdirAll(logsdir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, failedCreateLogsDirectoryFmt, logsdir, err)
		return ""
	}

	return logsdir
}

func (r *KubernetesReporter) logEvents(virtCli kubecli.KubevirtClient, since time.Time) {
	events, err := virtCli.CoreV1().Events(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to fetch events")
		return
	}

	e := events.Items
	sort.Slice(e, func(i, j int) bool {
		return e[i].LastTimestamp.After(e[j].LastTimestamp.Time)
	})

	eventsToPrint := v1.EventList{}
	for _, event := range e {
		if event.LastTimestamp.Time.After(since) {
			eventsToPrint.Items = append(eventsToPrint.Items, event)
		}
	}

	r.logObjects(virtCli, eventsToPrint, "events")
}

func (r *KubernetesReporter) logNetworkAttachmentDefinitionInfo(virtCli kubecli.KubevirtClient) {
	r.logNetworkAttachmentDefinition(virtCli, r.artifactsDir)
}

func (r *KubernetesReporter) logNetworkAttachmentDefinition(virtCli kubecli.KubevirtClient, outputFolder string) {
	networkAttachmentDefinitionsPath := filepath.Join(outputFolder, fmt.Sprintf("%d_networkAttachmentDefinitions.log", r.failureCount))
	r.dumpK8sEntityToFile(virtCli, networkAttachmentDefinitionEntity, v1.NamespaceAll, k8sCNICNCFEntityURLTemplate, networkAttachmentDefinitionsPath)
}

func (r *KubernetesReporter) dumpK8sEntityToFile(virtCli kubecli.KubevirtClient, entityName string, namespace string, entityURITemplate string, outputFilePath string) {
	requestURI := fmt.Sprintf(entityURITemplate, namespace, entityName)
	f, err := os.OpenFile(outputFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open file: %v\n", err)
		return
	}
	defer f.Close()

	response, err := virtCli.RestClient().Get().RequestURI(requestURI).Do(context.Background()).Raw()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dump %s: %v\n", entityName, err)
		return
	}

	var prettyJson bytes.Buffer
	err = json.Indent(&prettyJson, response, "", "    ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshall [%s] state objects\n", entityName)
		return
	}
	fmt.Fprintln(f, prettyJson.String())
}

func (r *KubernetesReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
	if setupSummary.State.Is(types.SpecStateFailureStates) {
		r.failureCount++
		r.DumpTestNamespaces(setupSummary.RunTime)
	}
}

func (r *KubernetesReporter) logClusterOverview() {
	binary := ""
	if flags.KubeVirtKubectlPath != "" {
		binary = "kubectl"
	} else if flags.KubeVirtOcPath != "" {
		binary = "oc"
	} else {
		return
	}

	stdout, stderr, err := clientcmd.RunCommandWithNS("", binary, "get", "all", "--all-namespaces", "-o", "wide")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch cluster overview: %v, %s\n", err, stderr)
		return
	}
	filePath := filepath.Join(r.artifactsDir, fmt.Sprintf("%d_overview.log", r.failureCount))
	err = writeStringToFile(filePath, stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write cluster overview: %v\n", err)
		return
	}
}

// getNodesRunningTests returns all node used by pods on test namespaces
func getNodesRunningTests(virtCli kubecli.KubevirtClient) []string {
	nodeMap := map[string]struct{}{}
	for _, testNamespace := range testsuite.TestNamespaces {
		pods, err := virtCli.CoreV1().Pods(testNamespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to fetch pods: %v\n", err)
			return nil
		}

		for _, pod := range pods.Items {
			if pod.Spec.NodeName != "" {
				nodeMap[pod.Spec.NodeName] = struct{}{}
			}
		}
	}

	nodes := []string{}
	for k := range nodeMap {
		nodes = append(nodes, k)
	}

	return nodes
}

func writeStringToFile(filePath string, data string) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer f.Close()

	_, err = f.WriteString(data)
	return err
}

func getVmiType(vmi v12.VirtualMachineInstance) (string, error) {
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.ContainerDisk == nil {
			continue
		}

		image := volume.VolumeSource.ContainerDisk.Image
		if strings.Contains(image, "fedora") {
			return "fedora", nil
		} else if strings.Contains(image, "cirros") {
			return "cirros", nil
		} else if strings.Contains(image, "alpine") {
			return "alpine", nil
		}
	}

	if vmiJson, err := json.Marshal(vmi); err == nil {
		return "", fmt.Errorf("unknown type, vmi %s", vmiJson)
	} else {
		return "", fmt.Errorf("%w: unknown type, vmi %s", err, vmi.ObjectMeta.Name)
	}
}

func prepareVmiConsole(vmi v12.VirtualMachineInstance, vmiType string) error {
	switch vmiType {
	case "fedora":
		return console.LoginToFedora(&vmi)
	case "cirros":
		return console.LoginToCirros(&vmi)
	case "alpine":
		return console.LoginToAlpine(&vmi)
	default:
		return fmt.Errorf("unknown vmi %s type", vmi.ObjectMeta.Name)
	}
}

func (r *KubernetesReporter) executeNodeCommands(virtCli kubecli.KubevirtClient, logsdir string, pod *v1.Pod) {
	const networkPrefix = "nsenter -t 1 -n -- "

	cmds := []commands{
		{command: networkPrefix + ipAddrName, fileNameSuffix: "ipaddress"},
		{command: networkPrefix + ipLinkName, fileNameSuffix: "iplink"},
		{command: networkPrefix + ipRouteShowTableAll, fileNameSuffix: "iproute"},
		{command: networkPrefix + ipNeighShow, fileNameSuffix: "ipneigh"},
		{command: networkPrefix + bridgeJVlanShow, fileNameSuffix: "brvlan"},
		{command: networkPrefix + bridgeFdb, fileNameSuffix: "brfdb"},
		{command: networkPrefix + "nft list ruleset", fileNameSuffix: "nftlist"},
	}

	if checks.IsRunningOnKindInfra() {
		cmds = append(cmds, []commands{{command: devVFio, fileNameSuffix: "vfio-devices"}}...)
	}

	r.executeContainerCommands(virtCli, logsdir, pod, virtHandlerName, cmds)
}

func (r *KubernetesReporter) executeVirtLauncherCommands(virtCli kubecli.KubevirtClient, logsdir string, pod v1.Pod) {
	cmds := []commands{
		{command: ipAddrName, fileNameSuffix: "ipaddress"},
		{command: ipLinkName, fileNameSuffix: "iplink"},
		{command: ipRouteShowTableAll, fileNameSuffix: "iproute"},
		{command: ipNeighShow, fileNameSuffix: "ipneigh"},
		{command: bridgeJVlanShow, fileNameSuffix: "brvlan"},
		{command: bridgeFdb, fileNameSuffix: "brfdb"},
		{command: "env", fileNameSuffix: "env"},
		{command: "[ -f /var/run/kubevirt/passt.log ] && cat /var/run/kubevirt/passt.log", fileNameSuffix: "passt"},
	}

	if checks.IsRunningOnKindInfra() {
		cmds = append(cmds, []commands{{command: devVFio, fileNameSuffix: "vfio-devices"}}...)
	}

	r.executeContainerCommands(virtCli, logsdir, &pod, computeContainer, cmds)
}

func (r *KubernetesReporter) executeContainerCommands(virtCli kubecli.KubevirtClient, logsdir string, pod *v1.Pod, container string, cmds []commands) {
	target := pod.ObjectMeta.Name
	if container == virtHandlerName {
		target = pod.Spec.NodeName
	}

	for _, cmd := range cmds {
		command := strings.Split(cmd.command, " ")

		stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtCli, pod, container, command)
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				failedExecuteCmdFmt,
				command, target, stdout, stderr, err,
			)

			pod, err := virtCli.CoreV1().Pods(pod.ObjectMeta.Namespace).Get(context.Background(), pod.ObjectMeta.Name, metav1.GetOptions{})
			if errors.IsNotFound(err) || (err == nil && (pod.Status.Phase != "Running" || !isContainerReady(pod.Status.ContainerStatuses, container))) {
				break
			}
			continue
		}

		fileName := fmt.Sprintf(logFileNameFmt, r.failureCount, target, cmd.fileNameSuffix)
		err = writeStringToFile(filepath.Join(logsdir, fileName), stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write %s %s output: %v\n", target, cmd.fileNameSuffix, err)
			continue
		}
	}
}

func (r *KubernetesReporter) executeVMICommands(vmi v12.VirtualMachineInstance, logsdir string, vmiType string) {
	cmds := []commands{
		{command: ipAddrName, fileNameSuffix: "ipaddress"},
		{command: ipLinkName, fileNameSuffix: "iplink"},
		{command: ipRouteShowTableAll, fileNameSuffix: "iproute"},
		{command: "dmesg", fileNameSuffix: "dmesg"},
	}

	if vmiType == "fedora" {
		cmds = append(cmds, []commands{
			{command: ipNeighShow, fileNameSuffix: "ipneigh"},
			{command: bridgeJVlanShow, fileNameSuffix: "brvlan"},
			{command: bridgeFdb, fileNameSuffix: "brfdb"},
			{command: "nmcli connection", fileNameSuffix: "nmcon"},
			{command: "nmcli device", fileNameSuffix: "nmdev"}}...)
	} else if vmiType == "cirros" || vmiType == "alpine" {
		cmds = append(cmds, []commands{
			{command: "lspci", fileNameSuffix: "lspci"},
			{command: "arp", fileNameSuffix: "arp"}}...)
	}

	for _, cmd := range cmds {
		res, err := console.SafeExpectBatchWithResponse(&vmi, []expect.Batcher{
			&expect.BSnd{S: cmd.command + "\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: console.RetValue("0")},
		}, 10)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed console vmi %s: %v\n", vmi.ObjectMeta.Name, err)
			continue
		}

		fileName := fmt.Sprintf(logFileNameFmt, r.failureCount, vmi.ObjectMeta.Name, cmd.fileNameSuffix)
		err = writeStringToFile(filepath.Join(logsdir, fileName), res[0].Output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write vmi %s %s output: %v\n", vmi.ObjectMeta.Name, cmd.fileNameSuffix, err)
			continue
		}
	}
}

func (r *KubernetesReporter) executePriviledgedVirtLauncherCommands(virtCli kubecli.KubevirtClient, virtHandlerPod *v1.Pod, logsdir, pid, target string) {
	nftCommand := strings.Split(fmt.Sprintf("nsenter -t %s -n -- nft list ruleset", pid), " ")

	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtCli, virtHandlerPod, virtHandlerName, nftCommand)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			failedExecuteCmdFmt,
			nftCommand, target, stdout, stderr, err,
		)
		return
	}

	fileName := fmt.Sprintf(logFileNameFmt, r.failureCount, target, "nftlist")
	err = writeStringToFile(filepath.Join(logsdir, fileName), stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s %s output: %v\n", target, "nftlist", err)
		return
	}
}

func (r *KubernetesReporter) executeCloudInitCommands(vmi v12.VirtualMachineInstance, path string, vmiType string) {
	var cmds []commands

	if vmiType == "fedora" {
		cmds = append(cmds, []commands{
			{command: "cat /var/log/cloud-init.log", fileNameSuffix: "cloud-init-log"},
			{command: "cat /var/log/cloud-init-output.log", fileNameSuffix: "cloud-init-output"},
			{command: "cat /var/run/cloud-init/status.json", fileNameSuffix: "cloud-init-status"},
		}...)
	}
	for _, cmd := range cmds {
		res, err := console.SafeExpectBatchWithResponse(&vmi, []expect.Batcher{
			&expect.BSnd{S: cmd.command + "\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: console.RetValue("0")},
		}, 10)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed console vmi %s/%s: %v\n", vmi.Namespace, vmi.Name, err)
			continue
		}

		fileName := fmt.Sprintf("%d_%s_%s_%s.log", r.failureCount, vmi.Namespace, vmi.Name, cmd.fileNameSuffix)
		err = writeStringToFile(filepath.Join(path, fileName), res[0].Output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write vmi %s/%s %s output: %v\n", vmi.Namespace, vmi.Name, cmd.fileNameSuffix, err)
			continue
		}
	}
}

func getVirtLauncherMonitorPID(virtCli kubecli.KubevirtClient, virtHandlerPod *v1.Pod, uid string) (string, error) {
	command := []string{
		"/bin/bash",
		"-c",
		fmt.Sprintf("pgrep -f \"monitor.*uid %s\"", uid),
	}

	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtCli, virtHandlerPod, virtHandlerName, command)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			failedExecuteCmdFmt,
			command, virtHandlerPod.ObjectMeta.Name, stdout, stderr, err,
		)
		return "", err
	}

	return strings.TrimSuffix(stdout, "\n"), nil
}

func isDataVolumeEnabled(clientset kubecli.KubevirtClient) (bool, error) {
	_, apis, err := clientset.DiscoveryClient().ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return false, err
	}

	for _, api := range apis {
		if api.GroupVersion == apicdi.SchemeGroupVersion.String() {
			for _, resource := range api.APIResources {
				if resource.Name == "datavolumes" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (r *KubernetesReporter) logVirtualMachinePools(virtCli kubecli.KubevirtClient) {
	pools, err := virtCli.VirtualMachinePool(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vm exports: %v\n", err)
		return
	}

	r.logObjects(virtCli, pools, "virtualmachinepools")
}
