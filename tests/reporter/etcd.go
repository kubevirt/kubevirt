package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
)

const (
	etcdContainerName = "etcd"
	etcdCACert        = "/etc/kubernetes/pki/etcd/ca.crt"
	etcdCert          = "/etc/kubernetes/pki/etcd/server.crt"
	etcdKey           = "/etc/kubernetes/pki/etcd/server.key"
)

type EtcdSnapshot struct {
	DBSizeBytes      int64  `json:"dbSizeBytes"`
	DBSizeInUseBytes int64  `json:"dbSizeInUseBytes"`
	Revision         int64  `json:"revision"`
	TmpfsUsedBytes   int64  `json:"tmpfsUsedBytes"`
	TmpfsTotalBytes  int64  `json:"tmpfsTotalBytes"`
	TmpfsAvailBytes  int64  `json:"tmpfsAvailBytes"`
	WALSizeBytes     int64  `json:"walSizeBytes"`
	SnapSizeBytes    int64  `json:"snapSizeBytes"`
	Error            string `json:"error,omitempty"`
}

type EtcdSpecRecord struct {
	SpecName        string       `json:"specName"`
	Before          EtcdSnapshot `json:"before"`
	After           EtcdSnapshot `json:"after"`
	DeltaDBSize     int64        `json:"deltaDBSizeBytes"`
	DeltaTmpfsUsed  int64        `json:"deltaTmpfsUsedBytes"`
	DeltaRevision   int64        `json:"deltaRevision"`
	Passed          bool         `json:"passed"`
	DurationSeconds float64      `json:"durationSeconds"`
}

type EtcdProfiler struct {
	artifactsDir   string
	mu             sync.Mutex
	records        []EtcdSpecRecord
	currentSnap    *EtcdSnapshot
	etcdPod        *v1.Pod
	virtHandlerPod *v1.Pod
}

func NewEtcdProfiler(artifactsDir string) *EtcdProfiler {
	return &EtcdProfiler{
		artifactsDir: artifactsDir,
	}
}

func (p *EtcdProfiler) getEtcdPod() (*v1.Pod, error) {
	if p.etcdPod != nil {
		return p.etcdPod, nil
	}

	virtCli := kubevirt.Client()
	pods, err := virtCli.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{
		LabelSelector: "component=etcd",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list etcd pods: %v", err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no etcd pods found")
	}

	p.etcdPod = &pods.Items[0]
	return p.etcdPod, nil
}

func (p *EtcdProfiler) collectSnapshot() EtcdSnapshot {
	snap := EtcdSnapshot{}

	pod, err := p.getEtcdPod()
	if err != nil {
		snap.Error = err.Error()
		return snap
	}

	p.collectEndpointStatus(pod, &snap)
	p.collectDfOutput(pod, &snap)
	p.collectDirSizes(pod, &snap)

	return snap
}

func (p *EtcdProfiler) collectEndpointStatus(pod *v1.Pod, snap *EtcdSnapshot) {
	cmd := []string{
		"etcdctl",
		"endpoint", "status",
		"--write-out=json",
		"--cacert=" + etcdCACert,
		"--cert=" + etcdCert,
		"--key=" + etcdKey,
	}

	stdout, err := exec.ExecuteCommandOnPod(pod, etcdContainerName, cmd)
	if err != nil {
		snap.Error = appendError(snap.Error, fmt.Sprintf("etcdctl endpoint status: %v", err))
		return
	}

	// etcdctl returns an array of endpoint statuses
	var statuses []struct {
		Status struct {
			DBSize      int64 `json:"dbSize"`
			DBSizeInUse int64 `json:"dbSizeInUse"`
			Header      struct {
				Revision int64 `json:"revision"`
			} `json:"header"`
		} `json:"Status"`
	}
	if err := json.Unmarshal([]byte(stdout), &statuses); err != nil {
		snap.Error = appendError(snap.Error, fmt.Sprintf("failed to parse etcdctl output: %v", err))
		return
	}

	if len(statuses) > 0 {
		snap.DBSizeBytes = statuses[0].Status.DBSize
		snap.DBSizeInUseBytes = statuses[0].Status.DBSizeInUse
		snap.Revision = statuses[0].Status.Header.Revision
	}
}

func (p *EtcdProfiler) getVirtHandlerPod(nodeName string) (*v1.Pod, error) {
	if p.virtHandlerPod != nil {
		return p.virtHandlerPod, nil
	}

	virtCli := kubevirt.Client()
	pod, err := libnode.GetVirtHandlerPod(virtCli, nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get virt-handler pod on node %s: %v", nodeName, err)
	}

	p.virtHandlerPod = pod
	return p.virtHandlerPod, nil
}

// execOnNode runs a command on the host via nsenter through the virt-handler pod.
func (p *EtcdProfiler) execOnNode(nodeName string, command string) (string, error) {
	pod, err := p.getVirtHandlerPod(nodeName)
	if err != nil {
		return "", err
	}

	cmd := []string{"sh", "-c", "nsenter -t 1 -m -- " + command}
	stdout, err := exec.ExecuteCommandOnPod(pod, virtHandlerName, cmd)
	if err != nil {
		// Invalidate cached pod in case it was recycled by a test
		p.virtHandlerPod = nil
		return "", err
	}
	return stdout, nil
}

func (p *EtcdProfiler) collectDfOutput(etcdPod *v1.Pod, snap *EtcdSnapshot) {
	stdout, err := p.execOnNode(etcdPod.Spec.NodeName, "df -k /var/lib/etcd | tail -1")
	if err != nil {
		snap.Error = appendError(snap.Error, fmt.Sprintf("df: %v", err))
		return
	}

	// Parse df output: Filesystem 1K-blocks Used Available Use% Mounted
	fields := strings.Fields(strings.TrimSpace(stdout))
	if len(fields) >= 4 {
		if total, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
			snap.TmpfsTotalBytes = total * 1024
		}
		if used, err := strconv.ParseInt(fields[2], 10, 64); err == nil {
			snap.TmpfsUsedBytes = used * 1024
		}
		if avail, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
			snap.TmpfsAvailBytes = avail * 1024
		}
	}
}

func (p *EtcdProfiler) collectDirSizes(etcdPod *v1.Pod, snap *EtcdSnapshot) {
	stdout, err := p.execOnNode(etcdPod.Spec.NodeName, "du -sb /var/lib/etcd/member/wal /var/lib/etcd/member/snap 2>/dev/null")
	if err != nil {
		snap.Error = appendError(snap.Error, fmt.Sprintf("du: %v", err))
		return
	}

	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		size, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		if strings.Contains(fields[1], "/wal") {
			snap.WALSizeBytes = size
		} else if strings.Contains(fields[1], "/snap") {
			snap.SnapSizeBytes = size
		}
	}
}

// RecordBeforeSpec captures etcd state before a spec runs.
func (p *EtcdProfiler) RecordBeforeSpec() {
	snap := p.collectSnapshot()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentSnap = &snap
}

// RecordSpec captures etcd state after a spec runs and stores the record.
func (p *EtcdProfiler) RecordSpec(specReport types.SpecReport) {
	after := p.collectSnapshot()

	p.mu.Lock()
	defer p.mu.Unlock()

	before := EtcdSnapshot{}
	if p.currentSnap != nil {
		before = *p.currentSnap
		p.currentSnap = nil
	}

	record := EtcdSpecRecord{
		SpecName:        specReport.FullText(),
		Before:          before,
		After:           after,
		DeltaDBSize:     after.DBSizeBytes - before.DBSizeBytes,
		DeltaTmpfsUsed:  after.TmpfsUsedBytes - before.TmpfsUsedBytes,
		DeltaRevision:   after.Revision - before.Revision,
		Passed:          !specReport.Failed(),
		DurationSeconds: specReport.RunTime.Seconds(),
	}

	p.records = append(p.records, record)

	printInfo("etcd profiler: spec=%q dbSize=%d tmpfsUsed=%d/%d deltaDB=%d deltaTmpfs=%d rev=%d walSize=%d snapSize=%d",
		specReport.FullText(),
		after.DBSizeBytes,
		after.TmpfsUsedBytes,
		after.TmpfsTotalBytes,
		record.DeltaDBSize,
		record.DeltaTmpfsUsed,
		after.Revision,
		after.WALSizeBytes,
		after.SnapSizeBytes,
	)
}

// Finalize writes the collected records to the artifacts directory.
func (p *EtcdProfiler) Finalize() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.records) == 0 {
		return
	}

	if err := os.MkdirAll(p.artifactsDir, 0777); err != nil {
		printError("etcd profiler: failed to create artifacts dir: %v", err)
		return
	}

	// Write summary
	summary := p.buildSummary()
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		printError("etcd profiler: failed to marshal summary: %v", err)
		return
	}

	outPath := filepath.Join(p.artifactsDir, "etcd-storage-profile.json")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		printError("etcd profiler: failed to write %s: %v", outPath, err)
		return
	}

	printInfo("etcd profiler: wrote %d records to %s", len(p.records), outPath)
}

type etcdProfileSummary struct {
	CollectedAt     string           `json:"collectedAt"`
	TotalSpecs      int              `json:"totalSpecs"`
	FinalDBSize     int64            `json:"finalDBSizeBytes"`
	FinalTmpfsUsed  int64            `json:"finalTmpfsUsedBytes"`
	FinalTmpfsTotal int64            `json:"finalTmpfsTotalBytes"`
	FinalWALSize    int64            `json:"finalWALSizeBytes"`
	FinalSnapSize   int64            `json:"finalSnapSizeBytes"`
	PeakTmpfsUsed   int64            `json:"peakTmpfsUsedBytes"`
	PeakDBSize      int64            `json:"peakDBSizeBytes"`
	Records         []EtcdSpecRecord `json:"records"`
}

func (p *EtcdProfiler) buildSummary() etcdProfileSummary {
	s := etcdProfileSummary{
		CollectedAt: time.Now().UTC().Format(time.RFC3339),
		TotalSpecs:  len(p.records),
		Records:     p.records,
	}

	for _, r := range p.records {
		if r.After.TmpfsUsedBytes > s.PeakTmpfsUsed {
			s.PeakTmpfsUsed = r.After.TmpfsUsedBytes
		}
		if r.After.DBSizeBytes > s.PeakDBSize {
			s.PeakDBSize = r.After.DBSizeBytes
		}
	}

	if len(p.records) > 0 {
		last := p.records[len(p.records)-1].After
		s.FinalDBSize = last.DBSizeBytes
		s.FinalTmpfsUsed = last.TmpfsUsedBytes
		s.FinalTmpfsTotal = last.TmpfsTotalBytes
		s.FinalWALSize = last.WALSizeBytes
		s.FinalSnapSize = last.SnapSizeBytes
	}

	return s
}

func appendError(existing, new string) string {
	if existing == "" {
		return new
	}
	return existing + "; " + new
}
