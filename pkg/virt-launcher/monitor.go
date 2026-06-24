/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package virtlauncher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"

	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
)

// proctreeSnapshotInterval is how often a full process-tree snapshot is logged
// at V(3). It is intentionally slower than the 1s monitor refresh to keep
// log volume low; a hung process is sticky, so 30s never loses the signal.
const proctreeSnapshotInterval = 30 * time.Second

// dstateEscalationInterval throttles D-state escalation logs so a long hang
// doesn't flood the logs. A hang lasts minutes/hours; one line per 10s is
// plenty.
const dstateEscalationInterval = 10 * time.Second

type OnShutdownCallback func(pid int)
type OnGracefulShutdownCallback func()

type monitor struct {
	timeout                  time.Duration
	pid                      int
	pidDir                   string
	domainName               string
	start                    time.Time
	isDone                   bool
	gracePeriod              int
	gracePeriodStartTime     int64
	finalShutdownCallback    OnShutdownCallback
	gracefulShutdownCallback OnGracefulShutdownCallback

	// dStateSince is set (Unix ns) the first time we observe the qemu pid in
	// D-state. Used to throttle escalation logs and to report how long the
	// process has been stuck.
	dStateSince time.Time
	// lastDStateLog throttles D-state escalation logs.
	lastDStateLog time.Time
	// lastProctreeSnapshot throttles the periodic process-tree snapshot.
	lastProctreeSnapshot time.Time
}

type ProcessMonitor interface {
	RunForever(startTimeout time.Duration, signalStopChan chan struct{})
}

func InitializePrivateDirectories(baseDir string) error {
	if err := util.MkdirAllWithNosec(baseDir); err != nil {
		return err
	}
	if err := diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(baseDir); err != nil {
		return err
	}
	return nil
}

func InitializeConsoleLogFile(baseDir string) error {
	logPath := filepath.Join(baseDir, "virt-serial0-log")

	_, err := os.Stat(logPath)
	if errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(logPath)
		if err != nil {
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err = diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(logPath); err != nil {
		return err
	}
	return nil
}

func InitializeDisksDirectories(baseDir string) error {
	err := os.MkdirAll(baseDir, 0750)
	if err != nil {
		return err
	}

	// #nosec G302: Poor file permissions used with chmod. Using the safe permission setting for a directory.
	err = os.Chmod(baseDir, 0750)
	if err != nil {
		return err
	}
	err = diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(baseDir)
	if err != nil {
		return err
	}
	return nil
}

func NewProcessMonitor(domainName string,
	pidDir string,
	gracePeriod int,
	finalShutdownCallback OnShutdownCallback,
	gracefulShutdownCallback OnGracefulShutdownCallback) ProcessMonitor {
	return &monitor{
		domainName:               domainName,
		pidDir:                   pidDir,
		gracePeriod:              gracePeriod,
		finalShutdownCallback:    finalShutdownCallback,
		gracefulShutdownCallback: gracefulShutdownCallback,
	}
}

func (mon *monitor) isGracePeriodExpired() bool {
	if mon.gracePeriodStartTime != 0 {
		now := time.Now().UTC().Unix()
		if (now - mon.gracePeriodStartTime) > int64(mon.gracePeriod) {
			return true
		}
	}
	return false
}

func (mon *monitor) refresh() {
	if mon.isDone {
		log.Log.Error("Called refresh after done!")
		return
	} else if cmdserver.ReceivedEarlyExitSignal() {
		log.Log.Infof("received early exit signal - stop waiting for %s", mon.domainName)
		mon.isDone = true
		return
	}

	log.Log.V(4).Infof("Refreshing. domainName %s pid %d", mon.domainName, mon.pid)

	expired := mon.isGracePeriodExpired()

	// is the process there?
	if mon.pid == 0 {
		var err error

		mon.pid, err = FindPid(mon.domainName, mon.pidDir)
		if err != nil {

			log.Log.Infof("Still missing PID for %s, %v", mon.domainName, err)
			// check to see if we've timed out looking for the process
			elapsed := time.Since(mon.start)
			if mon.timeout > 0 && elapsed >= mon.timeout {
				log.Log.Infof("%s not found after timeout", mon.domainName)
				mon.isDone = true
			} else if expired {
				log.Log.Infof("%s not found after grace period expired", mon.domainName)
				mon.isDone = true
			} else if mon.gracePeriodStartTime != 0 {
				log.Log.Infof("%s not found after shutdown initiated", mon.domainName)
				mon.isDone = true
			}
			return
		}

		log.Log.Infof("Found PID for %s: %d", mon.domainName, mon.pid)
	}

	exists, state, err := pidState(mon.pid)
	if err != nil {
		log.Log.Reason(err).Errorf("Error detecting pid (%d) status.", mon.pid)
		return
	}
	if exists == false {
		log.Log.Infof("Process %s and pid %d is gone!", mon.domainName, mon.pid)
		mon.pid = 0
		mon.isDone = true
		return
	}

	isZombie := state == "Z"
	if isZombie {
		log.Log.Infof("Process %s and pid %d is a zombie, sending SIGCHLD to pid 1 to reap process", mon.domainName, mon.pid)
		syscall.Kill(1, syscall.SIGCHLD)
		mon.pid = 0
		mon.isDone = true
		return
	}

	// D-state escalation: an uninterruptible-sleep qemu is the classic cause
	// of a pod stuck in Terminating (SIGKILL is ignored until the process
	// leaves the kernel syscall). Surface wchan + kernel stack so the root
	// cause (e.g. DRBD, NFS, CSI) is visible from `kubectl logs` alone.
	if state == "D" {
		mon.handleDState()
	} else {
		// process is healthy again — reset the D-state tracker so a future
		// hang starts a fresh escalation.
		if !mon.dStateSince.IsZero() {
			log.Log.Infof("qemu pid %d left D-state", mon.pid)
			mon.dStateSince = time.Time{}
		}
	}

	if expired {
		log.Log.Infof("Grace Period expired, shutting down.")
		mon.finalShutdownCallback(mon.pid)
	}

	return
}

func (mon *monitor) monitorLoop(startTimeout time.Duration, signalStopChan chan struct{}) {
	// random value, no real rationale
	rate := 1 * time.Second

	timeoutRepr := fmt.Sprintf("%v", startTimeout)
	if startTimeout == 0 {
		timeoutRepr = "disabled"
	}
	log.Log.Infof("Monitoring loop: rate %v start timeout %s", rate, timeoutRepr)

	ticker := time.NewTicker(rate)
	defer ticker.Stop()
	mon.isDone = false
	mon.timeout = startTimeout
	mon.start = time.Now()

	for !mon.isDone {
		select {
		case <-ticker.C:
			mon.refresh()
			mon.maybeDumpProcessTree()
		case <-signalStopChan:
			if mon.gracePeriodStartTime != 0 {
				continue
			}

			mon.gracefulShutdownCallback()
			mon.gracePeriodStartTime = time.Now().UTC().Unix()
		}
	}

}

func (mon *monitor) RunForever(startTimeout time.Duration, signalStopChan chan struct{}) {
	mon.monitorLoop(startTimeout, signalStopChan)
}

// pidState reads /proc/<pid>/status and returns whether the process exists
// plus its single-letter kernel state (R, D, S, Z, T, ...).
//
// 'D' (uninterruptible disk sleep) is the state that blocks SIGKILL and causes
// pods to hang in Terminating; 'Z' is a zombie.
func pidState(pid int) (exists bool, state string, err error) {
	pathCmdline := fmt.Sprintf("/proc/%d/cmdline", pid)
	pathStatus := fmt.Sprintf("/proc/%d/status", pid)

	exists, err = diskutils.FileExists(pathCmdline)
	if err != nil {
		return false, "", err
	}
	if exists == false {
		return false, "", nil
	}

	dataBytes, err := os.ReadFile(pathStatus)
	if err != nil {
		return false, "", err
	}

	state = parseProcState(string(dataBytes))
	return exists, state, nil
}

// parseProcState extracts the single-letter process state from /proc/<pid>/status.
// The "State:" line looks like:  State:	D (disk sleep)
func parseProcState(status string) string {
	for _, line := range strings.Split(status, "\n") {
		if !strings.HasPrefix(line, "State:") {
			continue
		}
		fields := strings.Fields(line)
		// ["State:", "D", "(disk", "sleep)"]
		if len(fields) >= 2 {
			return fields[1]
		}
	}
	return ""
}

// pidExists is retained for backward compatibility with existing callers/tests.
// It wraps pidState and reports only the zombie bit.
func pidExists(pid int) (exists bool, isZombie bool, err error) {
	ex, st, err := pidState(pid)
	if err != nil {
		return false, false, err
	}
	return ex, st == "Z", nil
}

// handleDState logs a one-line escalation when the qemu pid is in D-state,
// including wchan and a best-effort kernel stack so the root cause is
// diagnosable from pod logs without node SSH. Throttled to one line per
// dstateEscalationInterval.
func (mon *monitor) handleDState() {
	now := time.Now()
	if mon.dStateSince.IsZero() {
		mon.dStateSince = now
	}

	if !mon.lastDStateLog.IsZero() && now.Sub(mon.lastDStateLog) < dstateEscalationInterval {
		return
	}
	mon.lastDStateLog = now

	duration := now.Sub(mon.dStateSince).Round(time.Second)
	wchan := readProcFile(mon.pid, "wchan")
	stack := readKernelStack(mon.pid)

	if stack == "" {
		log.Log.Warningf("qemu pid %d D-state %s wchan=%s", mon.pid, duration, wchan)
	} else {
		log.Log.Warningf("qemu pid %d D-state %s wchan=%s stack=%s", mon.pid, duration, wchan, stack)
	}
}

// maybeDumpProcessTree emits a periodic one-line process-tree snapshot at
// V(3). The snapshot covers every process in the container's pid namespace
// (tini → virt-launcher → virtqemud → qemu) plus qemu's thread states, so a
// stuck vCPU thread is visible too.
func (mon *monitor) maybeDumpProcessTree() {
	now := time.Now()
	if !mon.lastProctreeSnapshot.IsZero() && now.Sub(mon.lastProctreeSnapshot) < proctreeSnapshotInterval {
		return
	}
	mon.lastProctreeSnapshot = now

	tree := buildProcessTree()
	if tree == "" {
		return
	}
	log.Log.V(3).Infof("proctree: %s", tree)
}

// procInfo is a minimal /proc/<pid>/stat view for tree rendering.
type procInfo struct {
	pid   int
	ppid  int
	state string
	comm  string
}

// readProcDir returns procInfo for every numeric /proc/<pid> entry reachable
// in the current pid namespace.
func readProcDir() []procInfo {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		log.Log.Reason(err).Error("failed to read /proc")
		return nil
	}
	var procs []procInfo
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		info, ok := readProcStat(pid)
		if !ok {
			continue
		}
		procs = append(procs, info)
	}
	return procs
}

// readProcStat parses /proc/<pid>/stat for pid, comm, state, ppid.
// Returns ok=false if the file can't be read (process exited between readdir
// and read — common, not an error).
func readProcStat(pid int) (procInfo, bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return procInfo{}, false
	}
	// /proc/<pid>/stat fields are space-separated, but comm is wrapped in
	// parentheses and may contain spaces, so parse from the last ')'.
	s := string(data)
	end := strings.LastIndexByte(s, ')')
	if end < 0 {
		return procInfo{}, false
	}
	comm := s[strings.IndexByte(s, '(')+1 : end]
	rest := strings.Fields(s[end+1:])
	if len(rest) < 2 {
		return procInfo{}, false
	}
	state := rest[0]
	ppid, _ := strconv.Atoi(rest[1])
	return procInfo{pid: pid, ppid: ppid, state: state, comm: comm}, true
}

// readProcFile reads a single /proc/<pid>/<name> file and returns its
// trimmed content, or "" on any error.
func readProcFile(pid int, name string) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/%s", pid, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// readKernelStack returns the qemu pid's kernel stack joined with "→" so it
// fits on one log line. Best-effort: returns "" if /proc/<pid>/stack is
// unreadable (e.g. EPERM on hardened kernels without CAP_SYS_ADMIN).
func readKernelStack(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stack", pid))
	if err != nil {
		return ""
	}
	var frames []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// strip the leading "[<0>] " prefix
		line = strings.TrimPrefix(line, "[<0>] ")
		frames = append(frames, line)
	}
	return strings.Join(frames, "→")
}

// buildProcessTree renders the container's process tree as a single line:
//
//	tini(1,S)→vl-monitor(2,S) virt-launcher(3,S) virtqemud(4,S)→qemu(5,R)[vcpu0:R,vcpu1:R]
//
// Children follow their parent via '→'; siblings (same ppid) are separated by
// a space. QEMU threads from /proc/<qemu>/task are appended in [...].
func buildProcessTree() string {
	procs := readProcDir()
	if len(procs) == 0 {
		return ""
	}
	sort.Slice(procs, func(i, j int) bool { return procs[i].pid < procs[j].pid })

	children := make(map[int][]procInfo)
	for _, p := range procs {
		children[p.ppid] = append(children[p.ppid], p)
	}

	// find the qemu pid (the one whose comm starts with "qemu-system" or
	// "qemu-kvm") for thread rendering.
	var qemuPid int
	for _, p := range procs {
		if strings.HasPrefix(p.comm, "qemu-system") || p.comm == "qemu-kvm" {
			qemuPid = p.pid
			break
		}
	}

	// roots: pid 1, or any process whose ppid is not in the set.
	var root procInfo
	found := false
	for _, p := range procs {
		if p.pid == 1 || p.ppid == 0 {
			root = p
			found = true
			break
		}
	}
	if !found {
		// no pid 1; pick the lowest pid as root.
		root = procs[0]
	}

	var sb strings.Builder
	renderNode(&sb, root, children)

	if qemuPid != 0 {
		threads := readQemuThreads(qemuPid)
		if threads != "" {
			sb.WriteString("[" + threads + "]")
		}
	}
	return sb.String()
}

// renderNode writes one node as comm(pid,state) and recurses into children
// joined by '→'.
func renderNode(sb *strings.Builder, p procInfo, children map[int][]procInfo) {
	fmt.Fprintf(sb, "%s(%d,%s)", p.comm, p.pid, p.state)
	kids := children[p.pid]
	for i, kid := range kids {
		if i > 0 {
			sb.WriteString(" ")
		} else {
			sb.WriteString("→")
		}
		renderNode(sb, kid, children)
	}
}

// readQemuThreads returns a comma-separated list of qemu thread states
// (e.g. "vcpu0:R,vcpu1:D,io:R"). Threads live under /proc/<pid>/task/<tid>.
func readQemuThreads(pid int) string {
	taskDir := fmt.Sprintf("/proc/%d/task", pid)
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		return ""
	}
	var parts []string
	for _, entry := range entries {
		tid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		info, ok := readProcStat(tid)
		if !ok {
			continue
		}
		name := info.comm
		if name == "" {
			name = entry.Name()
		}
		parts = append(parts, fmt.Sprintf("%s:%s", name, info.state))
	}
	return strings.Join(parts, ",")
}

func FindPid(domainName string, pidDir string) (int, error) {
	content, err := os.ReadFile(filepath.Join(pidDir, domainName+".pid"))
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(content))
}
