package reporter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v12 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

const (
	sriovEntityURITemplate            = "/apis/sriovnetwork.openshift.io/v1/namespaces/%s/%s/"
	sriovNetworksEntity               = "sriovnetworks"
	sriovNodeNetworkPolicyEntity      = "sriovnetworknodepolicies"
	sriovNodeStateEntity              = "sriovnetworknodestates"
	sriovOperatorConfigsEntity        = "sriovoperatorconfigs"
	k8sCNICNCFEntityURLTemplate       = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/%s/"
	networkAttachmentDefinitionEntity = "network-attachment-definitions"
)

type KubernetesReporter struct {
	failureCount int
	artifactsDir string
	maxFails     int
}

func NewKubernetesReporter(artifactsDir string, maxFailures int) *KubernetesReporter {
	return &KubernetesReporter{
		failureCount: 0,
		artifactsDir: artifactsDir,
		maxFails:     maxFailures,
	}
}

func (r *KubernetesReporter) SpecSuiteWillBegin(_ config.GinkgoConfigType, _ *types.SuiteSummary) {

}

func (r *KubernetesReporter) BeforeSuiteDidRun(_ *types.SetupSummary) {
	r.Cleanup()
}

func (r *KubernetesReporter) SpecWillRun(_ *types.SpecSummary) {
	fmt.Fprintf(ginkgo.GinkgoWriter, "On failure, artifacts will be collected in %s/%d_*\n", r.artifactsDir, r.failureCount+1)
}

func (r *KubernetesReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	if r.failureCount > r.maxFails {
		return
	}
	if specSummary.HasFailureState() {
		r.failureCount++
	} else {
		return
	}

	// If we got not directory, print to stderr
	if r.artifactsDir == "" {
		return
	}
	r.Dump(specSummary.RunTime)
}

// Dump dumps the current state of the cluster. The relevant logs are collected starting
// from the since parameter.
func (r *KubernetesReporter) Dump(duration time.Duration) {
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get client: %v\n", err)
		return
	}

	if err := os.MkdirAll(r.artifactsDir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory: %v\n", err)
		return
	}

	nodesDir := r.createNodesDir()
	podsDir := r.createPodsDir()

	duration += 5 * time.Second
	since := time.Now().Add(-duration)

	nodes := getNodeList(virtCli)
	nodesWithVirtLauncher := getNodesWithVirtLauncher(virtCli)
	pods := getPodList(virtCli)
	vmis := getVMIList(virtCli)

	r.logClusterOverview()
	r.logEvents(virtCli, since)
	r.logNamespaces(virtCli)
	r.logPVCs(virtCli)
	r.logPVs(virtCli)
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

	r.logAuditLogs(virtCli, nodesDir, nodesWithVirtLauncher, since)
	r.logDMESG(virtCli, nodesDir, nodesWithVirtLauncher, since)
	r.logJournal(virtCli, nodesDir, nodesWithVirtLauncher, duration, "")
	r.logJournal(virtCli, nodesDir, nodesWithVirtLauncher, duration, "kubelet")

	r.logLogs(virtCli, podsDir, pods, since)

	r.logVMIs(virtCli, vmis)
	r.logDomainXMLs(virtCli, vmis)

	r.logSRIOVInfo(virtCli)
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
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
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

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_vms.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	vms, err := virtCli.VirtualMachine(v1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vms: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(vms, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal vms")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logVMIs(virtCli kubecli.KubevirtClient, vmis *v12.VirtualMachineInstanceList) {

	if vmis == nil {
		fmt.Fprintf(os.Stderr, "vmi list is empty, skipping logVMIs\n")
		return
	}

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_vmis.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	j, err := json.MarshalIndent(vmis, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal vmis")
		return
	}
	fmt.Fprintln(f, string(j))
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
			pod, err := kubecli.NewVirtHandlerClient(virtCli).Namespace(flags.KubeVirtInstallNamespace).ForNode(node).Pod()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get virt-handler pod on node %s: %v\n", node, err)
				return
			}
			// TODO may need to be improved, in case that the auditlog is really huge, since stdout is in memory
			stdout, _, err := tests.ExecuteCommandOnPodV2(virtCli, pod, "virt-handler", []string{"/proc/1/root/bin/dmesg", "--kernel", "--ctime", "--userspace", "--decode"})
			if err != nil {
				fmt.Fprintf(
					os.Stderr,
					"failed to execute command %s on node %s, stdout: %s, error: %v\n",
					[]string{"/proc/1/root/bin/dmesg", "--kernel", "--ctime", "--userspace", "--decode"},
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
			pod, err := kubecli.NewVirtHandlerClient(virtCli).Namespace(flags.KubeVirtInstallNamespace).ForNode(node).Pod()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get virt-handler pod on node %s: %v\n", node, err)
				return
			}
			// TODO may need to be improved, in case that the auditlog is really huge, since stdout is in memory
			getAuditLogCmd := []string{"cat", "/proc/1/root/var/log/audit/audit.log"}
			stdout, _, err := tests.ExecuteCommandOnPodV2(virtCli, pod, "virt-handler", getAuditLogCmd)
			if err != nil {
				fmt.Fprintf(
					os.Stderr,
					"failed to execute command %s on node %s, stdout: %s, error: %v\n",
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
		pod, err := kubecli.NewVirtHandlerClient(virtCli).Namespace(flags.KubeVirtInstallNamespace).ForNode(node).Pod()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get virt-handler pod on node %s: %v\n", node, err)
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

		stdout, stderr, err := tests.ExecuteCommandOnPodV2(virtCli, pod, "virt-handler", commands)
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"failed to execute command %s on node %s, stdout: %s, stderr: %s, error: %v\n",
				commands, node, stdout, stderr, err,
			)
			continue
		}

		fileName := fmt.Sprintf("%d_%s_%s.log", r.failureCount, component, node)
		err = writeStringToFile(filepath.Join(logsdir, fileName), stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write node %s logs: %v\n", node, err)
			continue
		}
	}
}

func (r *KubernetesReporter) logPods(virtCli kubecli.KubevirtClient, pods *v1.PodList) {

	if pods == nil {
		fmt.Fprintf(os.Stderr, "pod list is empty, skipping logPods\n")
		return
	}

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_pods.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	j, err := json.MarshalIndent(pods, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal pods")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logServices(virtCli kubecli.KubevirtClient) {
	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_services.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	services, err := virtCli.CoreV1().Services(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch services: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(services, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal services")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logAPIServices(virtCli kubecli.KubevirtClient) {
	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_apiServices.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

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

	j, err := json.MarshalIndent(apiServices, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal apiServices")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logEndpoints(virtCli kubecli.KubevirtClient) {
	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_endpoints.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	endpoints, err := virtCli.CoreV1().Endpoints(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch endpointss: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(endpoints, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal endpoints")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logConfigMaps(virtCli kubecli.KubevirtClient) {
	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_configmaps.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	configmaps, err := virtCli.CoreV1().ConfigMaps(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch configmaps: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(configmaps, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal configmaps")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logKubeVirtCR(virtCli kubecli.KubevirtClient) {
	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_kubevirtCR.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	kvs, err := virtCli.KubeVirt(flags.KubeVirtInstallNamespace).List(&metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch kubevirts: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(kvs, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal kubevirt custom resource")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logSecrets(virtCli kubecli.KubevirtClient) {
	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_secrets.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	secrets, err := virtCli.CoreV1().Secrets(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch secrets: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(secrets, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal secrets")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logNamespaces(virtCli kubecli.KubevirtClient) {
	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_namespaces.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	namespaces, err := virtCli.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch Namespaces: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(namespaces, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal Namespaces")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logNodes(virtCli kubecli.KubevirtClient, nodes *v1.NodeList) {

	if nodes == nil {
		fmt.Fprintf(os.Stderr, "node list is empty, skipping logNodes\n")
		return
	}

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_nodes.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	j, err := json.MarshalIndent(nodes, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal nodes")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logPVs(virtCli kubecli.KubevirtClient) {

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_pvs.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	pvs, err := virtCli.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch pvs: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(pvs, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal pvs")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logPVCs(virtCli kubecli.KubevirtClient) {

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_pvcs.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	pvcs, err := virtCli.CoreV1().PersistentVolumeClaims(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch pvcs: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(pvcs, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal pvcs")
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
				fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
				return
			}
			defer current.Close()

			previous, err := os.OpenFile(filepath.Join(logsdir, fmt.Sprintf("%d_%s_%s-%s_previous.log", r.failureCount, pod.Namespace, pod.Name, container.Name)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
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

func getVMIList(virtCli kubecli.KubevirtClient) *v12.VirtualMachineInstanceList {

	vmis, err := virtCli.VirtualMachineInstance(v1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vmis: %v\n", err)
		return nil
	}

	return vmis
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

func (r *KubernetesReporter) createNodesDir() string {

	logsdir := filepath.Join(r.artifactsDir, "nodes")
	if err := os.MkdirAll(logsdir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory %s: %v\n", logsdir, err)
		return ""
	}

	return logsdir
}

func (r *KubernetesReporter) createPodsDir() string {

	logsdir := filepath.Join(r.artifactsDir, "pods")
	if err := os.MkdirAll(logsdir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory %s: %v\n", logsdir, err)
		return ""
	}

	return logsdir
}

func (r *KubernetesReporter) logEvents(virtCli kubecli.KubevirtClient, since time.Time) {

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%d_events.log", r.failureCount)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

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

	j, err := json.MarshalIndent(eventsToPrint, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal events")
		return
	}
	fmt.Fprintln(f, string(j))
}

func (r *KubernetesReporter) logSRIOVInfo(virtCli kubecli.KubevirtClient) {
	sriovOutputDir := filepath.Join(r.artifactsDir, "sriov")
	if err := os.MkdirAll(sriovOutputDir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory: %v\n", err)
		return
	}

	r.logSRIOVNodeState(virtCli, sriovOutputDir)
	r.logSRIOVNodeNetworkPolicies(virtCli, sriovOutputDir)
	r.logSRIOVNetworks(virtCli, sriovOutputDir)
	r.logSRIOVOperatorConfigs(virtCli, sriovOutputDir)
}

func (r *KubernetesReporter) logSRIOVNodeState(virtCli kubecli.KubevirtClient, outputFolder string) {
	nodeStateLogPath := filepath.Join(outputFolder, fmt.Sprintf("%d_nodestate.log", r.failureCount))
	r.dumpK8sEntityToFile(virtCli, sriovNodeStateEntity, v1.NamespaceAll, sriovEntityURITemplate, nodeStateLogPath)
}

func (r *KubernetesReporter) logSRIOVNodeNetworkPolicies(virtCli kubecli.KubevirtClient, outputFolder string) {
	nodeNetworkPolicyLogPath := filepath.Join(outputFolder, fmt.Sprintf("%d_nodenetworkpolicies.log", r.failureCount))
	r.dumpK8sEntityToFile(virtCli, sriovNodeNetworkPolicyEntity, v1.NamespaceAll, sriovEntityURITemplate, nodeNetworkPolicyLogPath)
}

func (r *KubernetesReporter) logSRIOVNetworks(virtCli kubecli.KubevirtClient, outputFolder string) {
	networksPath := filepath.Join(outputFolder, fmt.Sprintf("%d_networks.log", r.failureCount))
	r.dumpK8sEntityToFile(virtCli, sriovNetworksEntity, v1.NamespaceAll, sriovEntityURITemplate, networksPath)
}

func (r *KubernetesReporter) logSRIOVOperatorConfigs(virtCli kubecli.KubevirtClient, outputFolder string) {
	operatorConfigPath := filepath.Join(outputFolder, fmt.Sprintf("%d_operatorconfigs.log", r.failureCount))
	r.dumpK8sEntityToFile(virtCli, sriovOperatorConfigsEntity, v1.NamespaceAll, sriovEntityURITemplate, operatorConfigPath)
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
		fmt.Fprintf(os.Stderr, "failed to dump entity named [%s]: %v\n", entityName, err)
		return
	}

	var prettyJson bytes.Buffer
	err = json.Indent(&prettyJson, response, "", "    ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshall [%s] state objects\n", entityName)
		return
	}
	fmt.Fprintln(f, string(prettyJson.Bytes()))
}

func (r *KubernetesReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
	if setupSummary.State.IsFailure() {
		r.failureCount++
		r.Dump(setupSummary.RunTime)
	}
}

func (r *KubernetesReporter) SpecSuiteDidEnd(_ *types.SuiteSummary) {

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

	stdout, stderr, err := tests.RunCommandWithNS("", binary, "get", "all", "--all-namespaces", "-o", "wide")
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

//getNodesWithVirtLauncher returns all node where a virt-launcher pod ran (finished) or still runs
func getNodesWithVirtLauncher(virtCli kubecli.KubevirtClient) []string {
	pods, err := virtCli.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=virt-launcher", v12.AppLabel)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch pods: %v\n", err)
		return nil
	}

	nodeMap := map[string]struct{}{}
	for _, pod := range pods.Items {
		if pod.Spec.NodeName != "" {
			nodeMap[pod.Spec.NodeName] = struct{}{}
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
