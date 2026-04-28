package vmlogchecker

import (
	"regexp"
	"strings"
)

// SIGMask is a bitmask identifying which SIG areas are affected by an allowlist entry.
// Entries may affect multiple SIGs; combine with |, e.g. SIGNetwork | SIGStorage.
type SIGMask uint8

const (
	SIGCompute     SIGMask = 1 << iota // 0x01
	SIGNetwork                         // 0x02
	SIGOperator                        // 0x04
	SIGPerformance                     // 0x08
	SIGStorage                         // 0x10
	SIGMonitoring                      // 0x20
)

// AllowlistEntry describes a known/expected error pattern in virt-launcher logs.
// When adding a new entry, always use the last entry's ID + 1.
// Never reuse an ID after deletion.
type AllowlistEntry struct {
	// ID is a stable unique identifier. Set to last entry's ID + 1 on insert.
	ID int
	// Regex is matched against the full log line.
	Regex *regexp.Regexp
	// SIGs is the bitmask of affected SIG areas for triage routing.
	SIGs SIGMask
}

// VirtLauncherErrorAllowlist lists known error patterns that are expected and
// should not fail tests. Add new entries at the end with ID = last ID + 1.
var VirtLauncherErrorAllowlist = []AllowlistEntry{
	{
		ID:    1,
		Regex: regexp.MustCompile(`"level":"error","msg":"Fetching guest info failed:.*(The command guest-get-load has not been found|virError\(Code=.*, Domain=.*, Message='(Requested operation is not valid: domain is not running|guest agent command timed out: Guest agent disappeared while executing command)'\))`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage | SIGMonitoring,
	},
	{
		ID:    2,
		Regex: regexp.MustCompile(`"level":"error","msg":"(Guest agent is not responding: (QEMU guest agent is not connected|Guest agent disappeared while executing command)|guest agent command timed out: Guest agent disappeared while executing command)","pos":"qemu(DomainAgentAvailable|AgentCommandFull)`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    3,
		Regex: regexp.MustCompile(`"level":"error","msg":"(failed to get fs status before freeze vmi|Failed to (freeze|unfreeze) vmi).*Guest agent is not responding: QEMU guest agent is not connected`),
		SIGs:  SIGCompute,
	},
	{
		ID:    4,
		Regex: regexp.MustCompile(`"level":"error","msg":"Fetching guest info failed:.*unable to execute QEMU agent command.*the agent is in frozen state`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    5,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to freeze vmi.*virError\(Code=.*, Domain=.*, Message='(internal error: unable to execute|guest agent command failed: unable to execute) QEMU agent command 'guest-fsfreeze-freeze':.*Permission denied'\)`),
		SIGs:  SIGStorage,
	},
	{
		ID:    6,
		Regex: regexp.MustCompile(`"level":"error","msg":"End of file while reading data: Input/output error","pos":"virNetSocketReadWire`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    7,
		Regex: regexp.MustCompile(`"level":"error","msg":"internal error: client socket is closed","pos":"virNetClientSendInternal`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    8,
		Regex: regexp.MustCompile(`"level":"error","msg":"internal error: connection closed due to keepalive timeout","pos":"virKeepAliveTimerInternal`),
		SIGs:  SIGCompute | SIGNetwork,
	},
	{
		ID:    9,
		Regex: regexp.MustCompile(`"level":"error","msg":"Cannot recv data: Connection reset by peer","pos":"virNetSocketReadWire`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    10,
		Regex: regexp.MustCompile(`"level":"error","msg":"Cannot write data: Broken pipe","pos":"virNetSocketWriteWire`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    11,
		Regex: regexp.MustCompile(`"level":"error","msg":"Connection to libvirt lost\.",".*"reason":".*(Connection reset by peer|End of file while reading data: Input/output error)`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    12,
		Regex: regexp.MustCompile(`"level":"error","msg":"(Connection to libvirt lost\.|Getting the domain failed\.)",".*"reason":"virError\(Code=.*, Domain=.*, Message='internal error: client socket is closed'\)`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    13,
		Regex: regexp.MustCompile(`"level":"error","msg":"virtqemud exited, restarting","pos":"libvirt_helper.go`),
		SIGs:  SIGCompute,
	},
	{
		ID:    14,
		Regex: regexp.MustCompile(`"level":"error","msg":"Re-registered domain and agent callbacks for new connection","pos":"libvirt.go`),
		SIGs:  SIGCompute,
	},
	{
		ID:    15,
		Regex: regexp.MustCompile(`"level":"error","msg":"failed to read libvirt logs","pos":"libvirt_helper.go.*"reason":"read \|0: file already closed"`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    16,
		Regex: regexp.MustCompile(`"level":"error","msg":"packet \d+ bytes received from server too large, want \d+","pos":"virNetMessageDecodeLength`),
		SIGs:  SIGCompute,
	},
	{
		ID:    17,
		Regex: regexp.MustCompile(`"level":"error","msg":"Could not fetch the Domain\.",".*"reason":".*(Connection reset by peer|Failed to connect socket.*Connection refused|virError\(Code=.*, Domain=.*, Message='(internal error: client socket is closed|End of file while reading data: Input/output error)'\))`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    18,
		Regex: regexp.MustCompile(`"level":"error","msg":"Error updating cache: failed to get domain stats:.*(domain is not running|virError\(Code=.*, Domain=.*, Message='Domain not found:)`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    19,
		Regex: regexp.MustCompile(`"level":"error","msg":"failed to get domain spec",".*"reason":".*Domain not found`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    20,
		Regex: regexp.MustCompile(`"level":"error","msg":"Domain lookup failed: virError\(Code=.*, Domain=.*, Message='Domain not found:`),
		SIGs:  SIGCompute,
	},
	{
		ID:    21,
		Regex: regexp.MustCompile(`"level":"error","msg":"unpausing the VirtualMachineInstance failed\.",".*"reason":"virError\(Code=.*, Domain=.*, Message='Requested operation is not valid: domain is already running'\)"`),
		SIGs:  SIGStorage,
	},
	{
		ID:    22,
		Regex: regexp.MustCompile(`"level":"error","msg":"(internal error: Child process|Hook script execution failed).*cannot touch '/run/kubevirt-private/backend-storage-meta/migrated'`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    23,
		Regex: regexp.MustCompile(`"level":"error","msg":"Operation not supported: migration statistics are available only on the source host","pos":"qemuDomainGetJobStatsInternal`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    24,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to abort live migration",".*"reason":"failed to cancel migration - vmi is not migrating"`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    25,
		Regex: regexp.MustCompile(`"level":"error","msg":"(migration failed with error|Live migration failed\.|Received a live migration error\. Will check the latest migration status\.).*virError\(Code=.*, Domain=.*, Message='internal error: (client socket is closed|unable to execute QEMU command 'nbd-server-start': Failed to bind socket to .* No such file or directory)'\)`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    26,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to migrate vmi",".*"reason":"migration job .* already executed, finished at .*, failed: true, abortStatus: "`),
		SIGs:  SIGCompute,
	},
	{
		ID:    27,
		Regex: regexp.MustCompile(`"level":"error","msg":"(migration successfully aborted|operation aborted: migration out: canceled by client)","pos":"qemuMigration(DstFinish|SrcNBDStorageCopy)`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    28,
		Regex: regexp.MustCompile(`"level":"error","msg":"Live migration abort detected with reason: Live migration is not completed after \d+ seconds and has been aborted"`),
		SIGs:  SIGCompute,
	},
	{
		ID:    29,
		Regex: regexp.MustCompile(`"level":"error","msg":"(migration failed with error|Live migration failed\.|Received a live migration error\. Will check the latest migration status\.).*virError\(Code=.*, Domain=.*, Message='internal error: (process exited while connecting to monitor|QEMU unexpectedly closed the monitor).*The sum of offset.*has to be smaller or equal to the  actual size of the containing file`),
		SIGs:  SIGStorage,
	},
	{
		ID:    30,
		Regex: regexp.MustCompile(`"level":"error","msg":"Cannot access storage file '/var/run/kubevirt/container-disks/disk_0\.img'.*No such file or directory","pos":"virStorageSourceReportBrokenChain`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    31,
		Regex: regexp.MustCompile(`"level":"error","msg":"could not read data from source.*is a directory`),
		SIGs:  SIGCompute,
	},
	{
		ID:    32,
		Regex: regexp.MustCompile(`"level":"error","msg":"No disk capacity","pos":"manager.go`),
		SIGs:  SIGCompute | SIGNetwork | SIGPerformance | SIGStorage,
	},
	{
		ID:    33,
		Regex: regexp.MustCompile(`"level":"error","msg":"(Failed to get block info|invalid argument: invalid path.*not assigned to domain)`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    34,
		Regex: regexp.MustCompile(`"level":"error","msg":"Direct IO check failed for.*(permission denied|"reason":"open : no such file or directory")`),
		SIGs:  SIGStorage,
	},
	{
		ID:    35,
		Regex: regexp.MustCompile(`"level":"error","msg":"(failed to generate libvirt domain from VMI spec|Failed to sync vmi)",".*"reason":"failed to get container disk info: failed to invoke qemu-img: signal: segmentation fault"`),
		SIGs:  SIGCompute | SIGMonitoring,
	},
	{
		ID:    36,
		Regex: regexp.MustCompile(`"level":"error","msg":"Unable to read from monitor: Connection reset by peer","pos":"qemuMonitorIORead`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    37,
		Regex: regexp.MustCompile(`"level":"error","msg":"internal error: (QEMU unexpectedly closed the monitor|process exited while connecting to monitor).*(The sum of offset.*has to be smaller or equal to the  actual size of the containing file|Permission denied|Could not open '/var/run/kubevirt/container-disks/disk_0\.img': No such file or directory)`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    38,
		Regex: regexp.MustCompile(`"level":"error","msg":"Timed out during operation: cannot acquire state change lock.*held by monitor=`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    39,
		Regex: regexp.MustCompile(`"level":"error","msg":"(Defining the VirtualMachineInstance failed|failed to allocate hotplug ports|Failed to sync vmi|XML error:).*Invalid PCI address.*slot must be <=`),
		SIGs:  SIGCompute,
	},
	{
		ID:    40,
		Regex: regexp.MustCompile(`"level":"error","msg":"(failed to format domain cputune\.|Failed to sync vmi)".*"reason":"not enough exclusive threads provided, could not fit`),
		SIGs:  SIGCompute,
	},
	{
		ID:    41,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to sync vmi",".*"reason":"virError\(Code=.*, Domain=.*, Message='(internal error: client socket is closed|Requested operation is not valid: domain is already running)'\)"`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    42,
		Regex: regexp.MustCompile(`"level":"error","msg":"Conversion failed\.",".*"pos":"manager.go`),
		SIGs:  SIGCompute,
	},
	{
		ID:    43,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to start VirtualMachineInstance.*Permission denied'\)`),
		SIGs:  SIGStorage,
	},
	{
		ID:    44,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to sync vmi",".*"reason":"virError\(Code=.*, Domain=.*, Message='internal error: process exited while connecting to monitor:.*Permission denied'\)"`),
		SIGs:  SIGStorage,
	},
	{
		ID:    45,
		Regex: regexp.MustCompile(`"level":"error","msg":"Break reap loop","pos":"virt-launcher-monitor.go`),
		SIGs:  SIGCompute | SIGNetwork | SIGStorage,
	},
	{
		ID:    46,
		Regex: regexp.MustCompile(`"level":"error","msg":"received signal terminated but can't signal virt-launcher to shut down",".*"reason":"os: process already finished"`),
		SIGs:  SIGCompute,
	},
	{
		ID:    47,
		Regex: regexp.MustCompile(`"level":"error","msg":"dirty virt-launcher shutdown: exit-code 2","pos":"virt-launcher-monitor.go`),
		SIGs:  SIGCompute | SIGPerformance | SIGStorage,
	},
	{
		ID:    48,
		Regex: regexp.MustCompile(`"level":"error","msg":"failed to read qemu log directory","pos":"virt-launcher-monitor.go`),
		SIGs:  SIGCompute | SIGPerformance | SIGStorage,
	},
	{
		ID:    49,
		Regex: regexp.MustCompile(`"level":"error","msg":"(Failed to connect to notify server|Could not send domain notify event\.)","pos":"client.go.*"reason":"context deadline exceeded"`),
		SIGs:  SIGCompute | SIGPerformance | SIGStorage,
	},
	{
		ID:    50,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to send domain notify event\. closing connection\.","pos":"client.go.*"reason":"rpc error: code = Unavailable desc = connection error:.*(connection reset by peer|connection refused)`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    51,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to run backup job",".*"reason":"backup .* already executed, finished at .*, completed: true"`),
		SIGs:  SIGCompute,
	},
	{
		ID:    52,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to redefine checkpoint .*",".*"reason":"no disks found with checkpoint bitmap .*"`),
		SIGs:  SIGStorage,
	},
	{
		ID:    53,
		Regex: regexp.MustCompile(`"level":"error","msg":"Live migration failed\. Failure is forced by functional tests suite\.",".*"pos":"live-migration-source.go`),
		SIGs:  SIGCompute,
	},
	{
		ID:    54,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to prepare migration target pod",".*"reason":"Blocking preparation of migration target in order to satisfy a functional test condition"`),
		SIGs:  SIGCompute,
	},
	{
		ID:    55,
		Regex: regexp.MustCompile(`"level":"error","msg":"Error updating cache: empty DomainStats","pos":"time-defined-cache\.go`),
		SIGs:  SIGCompute | SIGNetwork | SIGOperator | SIGPerformance | SIGStorage | SIGMonitoring,
	},
	{
		ID:    56,
		Regex: regexp.MustCompile(`"level":"error","msg":"internal error: unable to execute QEMU command 'migrate-start-postcopy': Postcopy must be started after migration has been started","pos":"qemuMonitorJSONCheckErrorFull`),
		SIGs:  SIGCompute,
	},
	{
		ID:    57,
		Regex: regexp.MustCompile(`"level":"error","msg":"failed to start post migration".*"pos":"live-migration-source\.go`),
		SIGs:  SIGCompute,
	},
	{
		ID:    58,
		Regex: regexp.MustCompile(`"level":"error","msg":"internal error: Child process \(dmidecode .*Can't read memory from /dev/mem","pos":"virCommandWait`),
		SIGs:  SIGCompute,
	},
	{
		ID:    59,
		Regex: regexp.MustCompile(`"level":"error","msg":"At least one cgroup controller is required: No such device or address","pos":"virCgroupDetectControllers`),
		SIGs:  SIGCompute,
	},
	{
		ID:    60,
		Regex: regexp.MustCompile(`"level":"error","msg":"Error updating cache: failed to get domain stats: virError\(Code=.*Domain=.*Message='(Timed out during operation: cannot acquire state change lock \(held by monitor=.*\)|internal error: client socket is closed|Cannot recv data: Connection reset by peer)'\)","pos":"time-defined-cache\.go`),
		SIGs:  SIGCompute | SIGNetwork,
	},
	{
		ID:    61,
		Regex: regexp.MustCompile(`"level":"error","msg":"Guest agent is not responding: guest agent didn't respond to synchronization within '5' seconds","pos":"qemuAgentSend`),
		SIGs:  SIGCompute,
	},
	{
		ID:    62,
		Regex: regexp.MustCompile(`"level":"error","msg":"backup tunnel stopped with terminal error","pos":"backup_tunnel\.go`),
		SIGs:  SIGStorage,
	},
	{
		ID:    63,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to run backup job".*"pos":"server\.go`),
		SIGs:  SIGStorage,
	},
	{
		ID:    64,
		Regex: regexp.MustCompile(`"level":"error","msg":"Fetching guest info failed: virError\(Code=86, Domain=10, Message='Guest agent is not responding: (guest agent didn't respond to synchronization within '5' seconds|QEMU guest agent is not connected)'\)","pos":"agent_poller\.go`),
		SIGs:  SIGCompute,
	},
	{
		ID:    65,
		Regex: regexp.MustCompile(`"level":"error","msg":"internal error: unable to execute QEMU command 'nbd-server-stop': NBD server not running","pos":"qemuMonitorJSONCheckErrorFull`),
		SIGs:  SIGStorage,
	},
	{
		ID:    66,
		Regex: regexp.MustCompile(`"level":"error","msg":"guest agent command timed out: guest agent didn't respond to command within '5' seconds","pos":"qemuAgentSend`),
		SIGs:  SIGCompute,
	},
	{
		ID:    67,
		Regex: regexp.MustCompile(`"level":"error","msg":"Unable to write to monitor: Broken pipe","pos":"qemuMonitorIOWrite`),
		SIGs:  SIGCompute,
	},
	{
		ID:    68,
		Regex: regexp.MustCompile(`"level":"error","msg":"Operation not supported: cannot set time: qemu doesn't support rtc-reset-reinjection command","pos":"qemuDomainSetTime`),
		SIGs:  SIGCompute,
	},
	{
		ID:    69,
		Regex: regexp.MustCompile(`"level":"error","msg":"internal error: Missing monitor reply object","pos":"qemuMonitorJSONCommandWithFd`),
		SIGs:  SIGCompute,
	},
	{
		ID:    70,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to send domain notify event\. closing connection\.".*"pos":"client\.go.*"reason":"rpc error: code = Unavailable desc = connection error:.*failed to write client preface: write unix .*domain-notify-pipe\.sock: write: broken pipe.*`),
		SIGs:  SIGCompute,
	},
	{
		ID:    71,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to sync vmi",".*"reason":"virError\(.*Message='(Cannot access storage file '/var/run/kubevirt/container-disks/disk_[0-9]+\.img'.*No such file or directory|internal error: (QEMU unexpectedly closed the monitor|process exited while connecting to monitor).*Could not open '/var/run/kubevirt-private/vmi-disks/disk[0-9]+/disk\.img': Permission denied)'\)"`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    72,
		Regex: regexp.MustCompile(`"level":"error","msg":"Connection to libvirt lost\.?","pos":"libvirt\.go`),
		SIGs:  SIGCompute | SIGStorage,
	},
	{
		ID:    73,
		Regex: regexp.MustCompile(`"level":"error","msg":"Could not fetch the Domain\.","pos":"client\.go.*"reason":"virError\(.*Message='Cannot write data: Broken pipe'\)"`),
		SIGs:  SIGStorage,
	},
	{
		ID:    74,
		Regex: regexp.MustCompile(`"level":"error","msg":"migration successfully aborted","pos":"virNetClientProgramDispatchError`),
		SIGs:  SIGStorage,
	},
	{
		ID:    75,
		Regex: regexp.MustCompile(`"level":"error","msg":"error encountered during MigrateToURI3 libvirt api call: virError\(Code=.*Domain=10, Message='(operation aborted: migration out: canceled by client|internal error: process exited while connecting to monitor: .*The sum of offset.*actual size of the containing file.*)'\)".*"pos":"live-migration-source\.go`),
		SIGs:  SIGStorage,
	},
	{
		ID:    76,
		Regex: regexp.MustCompile(`"level":"error","msg":"Live migration failed\.".*"pos":"live-migration-source\.go.*"reason":"virError\(.*Message='operation aborted: migration out: canceled by client'\)"`),
		SIGs:  SIGStorage,
	},
	{
		ID:    77,
		Regex: regexp.MustCompile(`"level":"error","msg":"No filesystem overhead found for disk \{disk\s+file \{ /var/run/kubevirt-private/vmi-disks/disk0/disk\.img.*","pos":"manager\.go`),
		SIGs:  SIGCompute,
	},
	{
		ID:    78,
		Regex: regexp.MustCompile(`"level":"error","msg":"internal error: End of file from qemu monitor \(vm='kubevirt-test-.*'\)","pos":"qemuMonitorSend`),
		SIGs:  SIGCompute,
	},
	{
		ID:    79,
		Regex: regexp.MustCompile(`"level":"error","msg":"operation failed: domain 'kubevirt-test-.*' already exists with uuid [0-9a-f-]+","pos":"virDomainObjListAddLocked`),
		SIGs:  SIGCompute,
	},
	{
		ID:    80,
		Regex: regexp.MustCompile(`"level":"error","msg":"Failed to start VirtualMachineInstance with flags [0-9]+\.",".*"reason":"virError\(.*Message='Cannot access storage file '/var/run/kubevirt/container-disks/disk_[0-9]+\.img'.*No such file or directory'\)"`),
		SIGs:  SIGCompute,
	},
	{
		ID:    81,
		Regex: regexp.MustCompile(`"level":"error","msg":"Timed out during operation: cannot acquire state change lock \(held by agent=remoteDispatchDomainGetGuestInfo\)","pos":"virDomainObjBeginJobInternal`),
		SIGs:  SIGCompute,
	},
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

	if matchAllowlist(line) != nil {
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

// matchAllowlist returns the first AllowlistEntry whose Regex matches the given
// line, or nil if the line is not allowlisted.
func matchAllowlist(errorLine string) *AllowlistEntry {
	for i := range VirtLauncherErrorAllowlist {
		if VirtLauncherErrorAllowlist[i].Regex.MatchString(errorLine) {
			return &VirtLauncherErrorAllowlist[i]
		}
	}
	return nil
}
