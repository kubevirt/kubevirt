package vmlogchecker

import (
	"regexp"
	"strings"
)

var VirtLauncherErrorAllowlist = []*regexp.Regexp{
	// Guest agent errors
	regexp.MustCompile(`"level":"error","msg":"Fetching guest info failed:.*(The command guest-get-load has not been found|virError\(Code=.*, Domain=.*, Message='(Requested operation is not valid: domain is not running|guest agent command timed out: Guest agent disappeared while executing command)'\))`),
	regexp.MustCompile(`"level":"error","msg":"(Guest agent is not responding: (QEMU guest agent is not connected|Guest agent disappeared while executing command)|guest agent command timed out: Guest agent disappeared while executing command)","pos":"qemu(DomainAgentAvailable|AgentCommandFull)`),
	regexp.MustCompile(`"level":"error","msg":"(failed to get fs status before freeze vmi|Failed to (freeze|unfreeze) vmi).*Guest agent is not responding: QEMU guest agent is not connected`),
	regexp.MustCompile(`"level":"error","msg":"Fetching guest info failed:.*unable to execute QEMU agent command.*the agent is in frozen state`),
	regexp.MustCompile(`"level":"error","msg":"Failed to freeze vmi.*virError\(Code=.*, Domain=.*, Message='(internal error: unable to execute|guest agent command failed: unable to execute) QEMU agent command 'guest-fsfreeze-freeze':.*Permission denied'\)`),

	// Libvirt connection and socket errors
	regexp.MustCompile(`"level":"error","msg":"End of file while reading data: Input/output error","pos":"virNetSocketReadWire`),
	regexp.MustCompile(`"level":"error","msg":"internal error: client socket is closed","pos":"virNetClientSendInternal`),
	regexp.MustCompile(`"level":"error","msg":"internal error: connection closed due to keepalive timeout","pos":"virKeepAliveTimerInternal`),
	regexp.MustCompile(`"level":"error","msg":"Cannot recv data: Connection reset by peer","pos":"virNetSocketReadWire`),
	regexp.MustCompile(`"level":"error","msg":"Cannot write data: Broken pipe","pos":"virNetSocketWriteWire`),
	regexp.MustCompile(`"level":"error","msg":"Connection to libvirt lost\.",".*"reason":".*(Connection reset by peer|End of file while reading data: Input/output error)`),
	regexp.MustCompile(`"level":"error","msg":"(Connection to libvirt lost\.|Getting the domain failed\.)",".*"reason":"virError\(Code=.*, Domain=.*, Message='internal error: client socket is closed'\)`),
	regexp.MustCompile(`"level":"error","msg":"virtqemud exited, restarting","pos":"libvirt_helper.go`),
	regexp.MustCompile(`"level":"error","msg":"Re-registered domain and agent callbacks for new connection","pos":"libvirt.go`),
	regexp.MustCompile(`"level":"error","msg":"failed to read libvirt logs","pos":"libvirt_helper.go.*"reason":"read \|0: file already closed"`),
	regexp.MustCompile(`"level":"error","msg":"packet \d+ bytes received from server too large, want \d+","pos":"virNetMessageDecodeLength`),

	// Domain lookup and lifecycle errors
	regexp.MustCompile(`"level":"error","msg":"Could not fetch the Domain\.",".*"reason":".*(Connection reset by peer|Failed to connect socket.*Connection refused|virError\(Code=.*, Domain=.*, Message='(internal error: client socket is closed|End of file while reading data: Input/output error)'\))`),
	regexp.MustCompile(`"level":"error","msg":"Error updating cache: failed to get domain stats:.*(domain is not running|virError\(Code=.*, Domain=.*, Message='Domain not found:)`),
	regexp.MustCompile(`"level":"error","msg":"failed to get domain spec",".*"reason":".*Domain not found`),
	regexp.MustCompile(`"level":"error","msg":"Domain lookup failed: virError\(Code=.*, Domain=.*, Message='Domain not found:`),
	regexp.MustCompile(`"level":"error","msg":"unpausing the VirtualMachineInstance failed\.",".*"reason":"virError\(Code=.*, Domain=.*, Message='Requested operation is not valid: domain is already running'\)"`),

	// Migration errors
	regexp.MustCompile(`"level":"error","msg":"(internal error: Child process|Hook script execution failed).*cannot touch '/run/kubevirt-private/backend-storage-meta/migrated'`),
	regexp.MustCompile(`"level":"error","msg":"internal error: unable to execute QEMU command 'blockdev-add': Failed to connect to '/var/run/kubevirt/migrationproxy/.*\.sock': No such file or directory","pos":"qemuMonitorJSONCheckErrorFull`),
	regexp.MustCompile(`"level":"error","msg":"internal error: unable to execute QEMU command 'blockdev-add': Failed to read initial magic: Unexpected end-of-file before all data were read","pos":"qemuMonitorJSONCheckErrorFull`),
	regexp.MustCompile(`"level":"error","msg":"Operation not supported: migration statistics are available only on the source host","pos":"qemuDomainGetJobStatsInternal`),
	regexp.MustCompile(`"level":"error","msg":"Failed to abort live migration",".*"reason":"failed to cancel migration - vmi is not migrating"`),
	regexp.MustCompile(`"level":"error","msg":"(migration failed with error|Live migration failed\.|Received a live migration error\. Will check the latest migration status\.).*virError\(Code=.*, Domain=.*, Message='internal error: (client socket is closed|unable to execute QEMU command 'nbd-server-start': Failed to bind socket to .* No such file or directory)'\)`),
	regexp.MustCompile(`"level":"error","msg":"Failed to migrate vmi",".*"reason":"migration job .* already executed, finished at .*, failed: true, abortStatus: "`),
	regexp.MustCompile(`"level":"error","msg":"(migration successfully aborted|operation aborted: migration out: canceled by client)","pos":"qemuMigration(DstFinish|SrcNBDStorageCopy)`),
	regexp.MustCompile(`"level":"error","msg":"internal error: unable to execute QEMU command 'nbd-server-start': Failed to bind socket to.*No such file or directory","pos":"(qemuMonitorJSONCheckErrorFull|virNetClientProgramDispatchError)`),
	regexp.MustCompile(`"level":"error","msg":"Live migration abort detected with reason: Live migration is not completed after \d+ seconds and has been aborted"`),
	regexp.MustCompile(`"level":"error","msg":"(migration failed with error|Live migration failed\.|Received a live migration error\. Will check the latest migration status\.).*virError\(Code=.*, Domain=.*, Message='internal error: (process exited while connecting to monitor|QEMU unexpectedly closed the monitor).*The sum of offset.*has to be smaller or equal to the  actual size of the containing file`),

	// Storage and disk errors
	regexp.MustCompile(`"level":"error","msg":"Cannot access storage file '/var/run/kubevirt/container-disks/disk_0\.img'.*No such file or directory","pos":"virStorageSourceReportBrokenChain`),
	regexp.MustCompile(`"level":"error","msg":"could not read data from source.*is a directory`),
	regexp.MustCompile(`"level":"error","msg":"No disk capacity","pos":"manager.go`),
	regexp.MustCompile(`"level":"error","msg":"(Failed to get block info|invalid argument: invalid path.*not assigned to domain)`),
	regexp.MustCompile(`"level":"error","msg":"internal error: unable to execute QEMU command 'block_resize': Cannot grow device files","pos":"qemuMonitorJSONCheckErrorFull`),
	regexp.MustCompile(`"level":"error","msg":"libvirt failed to expand disk image.*(Cannot grow device files|domain is not running)'\)"`),
	regexp.MustCompile(`"level":"error","msg":"Direct IO check failed for.*(permission denied|"reason":"open : no such file or directory")`),
	regexp.MustCompile(`"level":"error","msg":"(failed to generate libvirt domain from VMI spec|Failed to sync vmi)",".*"reason":"failed to get container disk info: failed to invoke qemu-img: signal: segmentation fault"`),
	regexp.MustCompile(`"level":"error","msg":"Deleting QCOW2 overlay .*due to failure signal: killed","pos":"cbt.go`),
	regexp.MustCompile(`"level":"error","msg":"(failed to apply CBT|Failed to sync vmi)",".*"reason":"failed to create QCOW2 overlay.*signal: killed`),

	// QEMU monitor and process errors
	regexp.MustCompile(`"level":"error","msg":"internal error: unable to execute QEMU command 'cont': Resetting the Virtual Machine is required","pos":"qemuMonitorJSONCheckErrorFull`),
	regexp.MustCompile(`"level":"error","msg":"Unable to read from monitor: Connection reset by peer","pos":"qemuMonitorIORead`),
	regexp.MustCompile(`"level":"error","msg":"internal error: (QEMU unexpectedly closed the monitor|process exited while connecting to monitor).*(The sum of offset.*has to be smaller or equal to the  actual size of the containing file|Permission denied|Could not open '/var/run/kubevirt/container-disks/disk_0\.img': No such file or directory)`),
	regexp.MustCompile(`"level":"error","msg":"Timed out during operation: cannot acquire state change lock.*held by monitor=`),

	// VMI sync and configuration errors
	regexp.MustCompile(`"level":"error","msg":"(unpausing the VirtualMachineInstance failed|Failed to sync vmi).*Resetting the Virtual Machine is required`),
	regexp.MustCompile(`"level":"error","msg":"(Defining the VirtualMachineInstance failed|failed to allocate hotplug ports|Failed to sync vmi|XML error:).*Invalid PCI address.*slot must be <=`),
	regexp.MustCompile(`"level":"error","msg":"(failed to format domain cputune\.|Failed to sync vmi)".*"reason":"not enough exclusive threads provided, could not fit`),
	regexp.MustCompile(`"level":"error","msg":"Failed to sync vmi",".*"reason":"virError\(Code=.*, Domain=.*, Message='(internal error: client socket is closed|Requested operation is not valid: domain is already running)'\)"`),
	regexp.MustCompile(`"level":"error","msg":"Conversion failed\.",".*"pos":"manager.go:\d+"`),

	// Permission and access errors
	regexp.MustCompile(`"level":"error","msg":"Unable to open /dev/kvm: Permission denied","pos":"virHostCPUGetCPUID`),
	regexp.MustCompile(`"level":"error","msg":"Failed to start VirtualMachineInstance.*Permission denied'\)`),
	regexp.MustCompile(`"level":"error","msg":"Failed to sync vmi",".*"reason":"virError\(Code=.*, Domain=.*, Message='internal error: process exited while connecting to monitor:.*Permission denied'\)"`),

	// Virt-launcher lifecycle errors
	regexp.MustCompile(`"level":"error","msg":"Break reap loop","pos":"virt-launcher-monitor.go`),
	regexp.MustCompile(`"level":"error","msg":"received signal terminated but can't signal virt-launcher to shut down",".*"reason":"os: process already finished"`),
	regexp.MustCompile(`"level":"error","msg":"dirty virt-launcher shutdown: exit-code 2","pos":"virt-launcher-monitor.go`),
	regexp.MustCompile(`"level":"error","msg":"failed to read qemu log directory","pos":"virt-launcher-monitor.go`),

	// Notify server and RPC errors
	regexp.MustCompile(`"level":"error","msg":"(Failed to connect to notify server|Could not send domain notify event\.)","pos":"client.go.*"reason":"context deadline exceeded"`),
	regexp.MustCompile(`"level":"error","msg":"Failed to send domain notify event\. closing connection\.","pos":"client.go.*"reason":"rpc error: code = Unavailable desc = connection error:.*(connection reset by peer|connection refused)`),

	// Backup job errors
	regexp.MustCompile(`"level":"error","msg":"Failed to run backup job",".*"reason":"backup .* already executed, finished at .*, completed: true"`),
	regexp.MustCompile(`"level":"error","msg":"Failed to redefine checkpoint .*",".*"reason":"no disks found with checkpoint bitmap .*"`),

	// Test-injected errors (intentional failures for functional testing)
	regexp.MustCompile(`"level":"error","msg":"Live migration failed\. Failure is forced by functional tests suite\.",".*"pos":"live-migration-source.go`),
	regexp.MustCompile(`"level":"error","msg":"Failed to prepare migration target pod",".*"reason":"Blocking preparation of migration target in order to satisfy a functional test condition"`),
}

// errorKeywordPatterns provides broad keyword-based error detection for the
// CLI tool's --all-levels mode, which scans lines regardless of JSON level.
// The e2e reporter pre-filters on "level":"error" via IsErrorLevel instead.
var errorKeywordPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\berror\b`),
	regexp.MustCompile(`\bfailed\b`),
	regexp.MustCompile(`\bpanic\b`),
	regexp.MustCompile(`\bfatal\b`),
}

type ErrorClassification int

const (
	NotAnError ErrorClassification = iota
	AllowlistedError
	UnexpectedError
)

// IsErrorLevel returns true if the log line contains a JSON "level":"error" field.
// Use this to pre-filter lines before classification when only error-level lines matter.
func IsErrorLevel(line string) bool {
	return strings.Contains(line, `"level":"error"`)
}

func ClassifyLogLine(line string) ErrorClassification {
	if line == "" || !containsErrorKeyword(line) {
		return NotAnError
	}

	if IsAllowlisted(line) {
		return AllowlistedError
	}

	return UnexpectedError
}

func containsErrorKeyword(line string) bool {
	lineLower := strings.ToLower(line)
	for _, pattern := range errorKeywordPatterns {
		if pattern.MatchString(lineLower) {
			return true
		}
	}
	return false
}

func IsAllowlisted(errorLine string) bool {
	for _, pattern := range VirtLauncherErrorAllowlist {
		if pattern.MatchString(errorLine) {
			return true
		}
	}
	return false
}
