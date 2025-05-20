package util

import (
	"bufio"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	"golang.org/x/sys/unix"
	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

const QEMUSeaBiosDebugPipe = converter.QEMUSeaBiosDebugPipe
const (
	qemuConfPath        = "/etc/libvirt/qemu.conf"
	virtqemudConfPath   = "/etc/libvirt/virtqemud.conf"
	libvirtRuntimePath  = "/var/run/libvirt"
	libvirtHomePath     = "/var/run/kubevirt-private/libvirt"
	qemuNonRootConfPath = libvirtHomePath + "/qemu.conf"
)

var LifeCycleTranslationMap = map[libvirt.DomainState]api.LifeCycle{
	libvirt.DOMAIN_NOSTATE:     api.NoState,
	libvirt.DOMAIN_RUNNING:     api.Running,
	libvirt.DOMAIN_BLOCKED:     api.Blocked,
	libvirt.DOMAIN_PAUSED:      api.Paused,
	libvirt.DOMAIN_SHUTDOWN:    api.Shutdown,
	libvirt.DOMAIN_SHUTOFF:     api.Shutoff,
	libvirt.DOMAIN_CRASHED:     api.Crashed,
	libvirt.DOMAIN_PMSUSPENDED: api.PMSuspended,
}

var ShutdownReasonTranslationMap = map[libvirt.DomainShutdownReason]api.StateChangeReason{
	libvirt.DOMAIN_SHUTDOWN_UNKNOWN: api.ReasonUnknown,
	libvirt.DOMAIN_SHUTDOWN_USER:    api.ReasonUser,
}

var ShutoffReasonTranslationMap = map[libvirt.DomainShutoffReason]api.StateChangeReason{
	libvirt.DOMAIN_SHUTOFF_UNKNOWN:       api.ReasonUnknown,
	libvirt.DOMAIN_SHUTOFF_SHUTDOWN:      api.ReasonShutdown,
	libvirt.DOMAIN_SHUTOFF_DESTROYED:     api.ReasonDestroyed,
	libvirt.DOMAIN_SHUTOFF_CRASHED:       api.ReasonCrashed,
	libvirt.DOMAIN_SHUTOFF_MIGRATED:      api.ReasonMigrated,
	libvirt.DOMAIN_SHUTOFF_SAVED:         api.ReasonSaved,
	libvirt.DOMAIN_SHUTOFF_FAILED:        api.ReasonFailed,
	libvirt.DOMAIN_SHUTOFF_FROM_SNAPSHOT: api.ReasonFromSnapshot,
}

var CrashedReasonTranslationMap = map[libvirt.DomainCrashedReason]api.StateChangeReason{
	libvirt.DOMAIN_CRASHED_UNKNOWN:  api.ReasonUnknown,
	libvirt.DOMAIN_CRASHED_PANICKED: api.ReasonPanicked,
}

var PausedReasonTranslationMap = map[libvirt.DomainPausedReason]api.StateChangeReason{
	libvirt.DOMAIN_PAUSED_UNKNOWN:         api.ReasonPausedUnknown,
	libvirt.DOMAIN_PAUSED_USER:            api.ReasonPausedUser,
	libvirt.DOMAIN_PAUSED_MIGRATION:       api.ReasonPausedMigration,
	libvirt.DOMAIN_PAUSED_SAVE:            api.ReasonPausedSave,
	libvirt.DOMAIN_PAUSED_DUMP:            api.ReasonPausedDump,
	libvirt.DOMAIN_PAUSED_IOERROR:         api.ReasonPausedIOError,
	libvirt.DOMAIN_PAUSED_WATCHDOG:        api.ReasonPausedWatchdog,
	libvirt.DOMAIN_PAUSED_FROM_SNAPSHOT:   api.ReasonPausedFromSnapshot,
	libvirt.DOMAIN_PAUSED_SHUTTING_DOWN:   api.ReasonPausedShuttingDown,
	libvirt.DOMAIN_PAUSED_SNAPSHOT:        api.ReasonPausedSnapshot,
	libvirt.DOMAIN_PAUSED_CRASHED:         api.ReasonPausedCrashed,
	libvirt.DOMAIN_PAUSED_STARTING_UP:     api.ReasonPausedStartingUp,
	libvirt.DOMAIN_PAUSED_POSTCOPY:        api.ReasonPausedPostcopy,
	libvirt.DOMAIN_PAUSED_POSTCOPY_FAILED: api.ReasonPausedPostcopyFailed,
}

var getHookManager = hooks.GetManager

type LibvirtWrapper struct {
	user uint32
}

func NewLibvirtWrapper(nonRoot bool) *LibvirtWrapper {
	if nonRoot {
		return &LibvirtWrapper{
			user: util.NonRootUID,
		}
	}
	return &LibvirtWrapper{
		user: util.RootUser,
	}
}

func ConvState(status libvirt.DomainState) api.LifeCycle {
	return LifeCycleTranslationMap[status]
}

func ConvReason(status libvirt.DomainState, reason int) api.StateChangeReason {
	switch status {
	case libvirt.DOMAIN_SHUTDOWN:
		return ShutdownReasonTranslationMap[libvirt.DomainShutdownReason(reason)]
	case libvirt.DOMAIN_SHUTOFF:
		return ShutoffReasonTranslationMap[libvirt.DomainShutoffReason(reason)]
	case libvirt.DOMAIN_CRASHED:
		return CrashedReasonTranslationMap[libvirt.DomainCrashedReason(reason)]
	case libvirt.DOMAIN_PAUSED:
		return PausedReasonTranslationMap[libvirt.DomainPausedReason(reason)]
	default:
		return api.ReasonUnknown
	}
}

// base64.StdEncoding.EncodeToString
func SetDomainSpecStr(virConn cli.Connection, vmi *v1.VirtualMachineInstance, wantedSpec string) (cli.VirDomain, error) {
	log.Log.Object(vmi).V(2).Infof("Domain XML generated. Base64 dump %s", base64.StdEncoding.EncodeToString([]byte(wantedSpec)))
	dom, err := virConn.DomainDefineXML(wantedSpec)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Defining the VirtualMachineInstance failed.")
		return nil, err
	}
	return dom, nil
}

func SetDomainSpecStrWithHooks(virConn cli.Connection, vmi *v1.VirtualMachineInstance, wantedSpec *api.DomainSpec) (cli.VirDomain, error) {
	hooksManager := getHookManager()
	domainSpec, err := hooksManager.OnDefineDomain(wantedSpec, vmi)
	if err != nil {
		return nil, err
	}

	// update wantedSpec to reflect changes made to domain spec by hooks
	domainSpecObj := &api.DomainSpec{}
	if err = xml.Unmarshal([]byte(domainSpec), domainSpecObj); err != nil {
		return nil, err
	}
	domainSpecObj.DeepCopyInto(wantedSpec)

	return SetDomainSpecStr(virConn, vmi, domainSpec)
}

// GetDomainSpecWithRuntimeInfo return the active domain XML with runtime information embedded
func GetDomainSpecWithRuntimeInfo(dom cli.VirDomain) (*api.DomainSpec, error) {

	// get libvirt xml with runtime status
	activeSpec, err := GetDomainSpecWithFlags(dom, 0)
	if err != nil {
		log.Log.Reason(err).Error("failed to get domain spec")
		return nil, err
	}

	return activeSpec, nil
}

// GetDomainSpec return the domain XML without runtime information.
// The result XML is merged from inactive XML and migratable XML.
func GetDomainSpec(status libvirt.DomainState, dom cli.VirDomain) (*api.DomainSpec, error) {

	var spec, inactiveSpec *api.DomainSpec
	var err error

	inactiveSpec, err = GetDomainSpecWithFlags(dom, libvirt.DOMAIN_XML_INACTIVE)

	if err != nil {
		return nil, err
	}

	spec = inactiveSpec
	// libvirt (the whole server) sometimes block indefinitely if a guest-shutdown was performed
	// and we immediately ask it after the successful shutdown for a migratable xml.
	if !cli.IsDown(status) {
		spec, err = GetDomainSpecWithFlags(dom, libvirt.DOMAIN_XML_MIGRATABLE)
		if err != nil {
			return nil, err
		}
	}

	return spec, nil
}

func GetDomainSpecWithFlags(dom cli.VirDomain, flags libvirt.DomainXMLFlags) (*api.DomainSpec, error) {
	domain := &api.DomainSpec{}
	domxml, err := dom.GetXMLDesc(flags)
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal([]byte(domxml), domain)
	if err != nil {
		return nil, err
	}

	return domain, nil
}

func (l LibvirtWrapper) StartVirtqemud(stopChan chan struct{}) {
	// we spawn libvirt from virt-launcher in order to ensure the virtqemud+qemu process
	// doesn't exit until virt-launcher is ready for it to. Virt-launcher traps signals
	// to perform special shutdown logic. These processes need to live in the same
	// container.

	go func() {
		for {
			exitChan := make(chan struct{})
			args := []string{"-f", "/var/run/libvirt/virtqemud.conf", "--no-admin-srv", "--no-ro-srv"}
			cmd := exec.Command("/usr/sbin/virtqemud", args...)
			if l.user != 0 {
				cmd.SysProcAttr = &syscall.SysProcAttr{
					AmbientCaps: []uintptr{unix.CAP_NET_BIND_SERVICE},
				}
			}

			// connect libvirt's stderr to our own stdout in order to see the logs in the container logs
			reader, err := cmd.StderrPipe()
			if err != nil {
				log.Log.Reason(err).Error("failed to start virtqemud")
				panic(err)
			}

			go func() {
				scanner := bufio.NewScanner(reader)
				scanner.Buffer(make([]byte, 1024), 512*1024)
				for scanner.Scan() {
					log.LogLibvirtLogLine(log.Log, scanner.Text())
				}

				if err := scanner.Err(); err != nil {
					log.Log.Reason(err).Error("failed to read libvirt logs")
				}
			}()

			err = cmd.Start()
			if err != nil {
				log.Log.Reason(err).Error("failed to start virtqemud")
				panic(err)
			}

			go func() {
				defer close(exitChan)
				cmd.Wait()
			}()

			select {
			case <-stopChan:
				cmd.Process.Kill()
				return
			case <-exitChan:
				log.Log.Errorf("virtqemud exited, restarting")
			}

			// this sleep is to avoid consuming all resources in the
			// event of a virtqemud crash loop.
			time.Sleep(time.Second)
		}
	}()
}

func startVirtlogdLogging(stopChan chan struct{}, domainName string, nonRoot bool) {
	for {
		cmd := exec.Command("/usr/sbin/virtlogd", "-f", "/etc/libvirt/virtlogd.conf", "--no-admin-srv")

		exitChan := make(chan struct{})

		err := cmd.Start()
		if err != nil {
			log.Log.Reason(err).Error("failed to start virtlogd")
			panic(err)
		}

		go func() {
			logfile := fmt.Sprintf("/var/log/libvirt/qemu/%s.log", domainName)
			if nonRoot {
				logfile = filepath.Join("/var", "run", "kubevirt-private", "libvirt", "qemu", "log", fmt.Sprintf("%s.log", domainName))
			}

			// It can take a few seconds to the log file to be created
			for {
				_, err = os.Stat(logfile)
				if !errors.Is(err, os.ErrNotExist) {
					break
				}
				time.Sleep(time.Second)
			}
			// #nosec No risk for path injection. logfile has a static basedir
			file, err := os.Open(logfile)
			if err != nil {
				errMsg := fmt.Sprintf("failed to open logfile in path: \"%s\"", logfile)
				log.Log.Reason(err).Error(errMsg)
				return
			}
			defer util.CloseIOAndCheckErr(file, nil)

			scanner := bufio.NewScanner(file)
			scanner.Buffer(make([]byte, 1024), 512*1024)
			for scanner.Scan() {
				log.LogQemuLogLine(log.Log, scanner.Text())
			}

			if err := scanner.Err(); err != nil {
				log.Log.Reason(err).Error("failed to read virtlogd logs")
			}
		}()

		go func() {
			defer close(exitChan)
			_ = cmd.Wait()
		}()

		select {
		case <-stopChan:
			_ = cmd.Process.Kill()
			return
		case <-exitChan:
			log.Log.Errorf("virtlogd exited, restarting")
		}

		// this sleep is to avoid consuming all resources in the
		// event of a virtlogd crash loop.
		time.Sleep(time.Second)
	}
}

func startQEMUSeaBiosLogging(stopChan chan struct{}) {
	const QEMUSeaBiosDebugPipeMode uint32 = 0666
	const logLinePrefix = "[SeaBios]:"

	err := syscall.Mkfifo(QEMUSeaBiosDebugPipe, QEMUSeaBiosDebugPipeMode)
	if err != nil {
		log.Log.Reason(err).Error(fmt.Sprintf("%s failed creating a pipe for sea bios debug logs", logLinePrefix))
		return
	}

	// Chmod is needed since umask is 0018. Therefore Mkfifo does not actually create a pipe with proper permissions.
	err = syscall.Chmod(QEMUSeaBiosDebugPipe, QEMUSeaBiosDebugPipeMode)
	if err != nil {
		log.Log.Reason(err).Error(fmt.Sprintf("%s failed executing chmod on pipe for sea bios debug logs.", logLinePrefix))
		return
	}

	QEMUPipe, err := os.OpenFile(QEMUSeaBiosDebugPipe, os.O_RDONLY, 0604)

	if err != nil {
		log.Log.Reason(err).Error(fmt.Sprintf("%s failed to open %s", logLinePrefix, QEMUSeaBiosDebugPipe))
		return
	}
	defer QEMUPipe.Close()

	scanner := bufio.NewScanner(QEMUPipe)
	for {
		for scanner.Scan() {
			logLine := fmt.Sprintf("%s %s", logLinePrefix, scanner.Text())

			log.LogQemuLogLine(log.Log, logLine)

			select {
			case <-stopChan:
				return
			default:
			}
		}

		if err := scanner.Err(); err != nil {
			log.Log.Reason(err).Error(fmt.Sprintf("%s reader failed with an error", logLinePrefix))
			return
		}

		log.Log.Errorf(fmt.Sprintf("%s exited, restarting", logLinePrefix))
	}
}

func StartVirtlog(stopChan chan struct{}, domainName string, nonRoot bool) {
	go startVirtlogdLogging(stopChan, domainName, nonRoot)
	go startQEMUSeaBiosLogging(stopChan)
}

// returns the namespace and name that is encoded in the
// domain name.
func SplitVMINamespaceKey(domainName string) (namespace, name string) {
	splitName := strings.SplitN(domainName, "_", 2)
	if len(splitName) == 1 {
		return k8sv1.NamespaceDefault, splitName[0]
	}
	return splitName[0], splitName[1]
}

// VMINamespaceKeyFunc constructs the domain name with a namespace prefix i.g.
// namespace_name.
func VMINamespaceKeyFunc(vmi *v1.VirtualMachineInstance) string {
	return DomainFromNamespaceName(vmi.Namespace, vmi.Name)
}

func DomainFromNamespaceName(namespace, name string) string {
	return fmt.Sprintf("%s_%s", namespace, name)
}

func NewDomain(dom cli.VirDomain) (*api.Domain, error) {

	name, err := dom.GetName()
	if err != nil {
		return nil, err
	}
	namespace, name := SplitVMINamespaceKey(name)

	domain := api.NewDomainReferenceFromName(namespace, name)
	domain.GetObjectMeta().SetUID(domain.Spec.Metadata.KubeVirt.UID)
	return domain, nil
}

func NewDomainFromName(name string, vmiUID types.UID) *api.Domain {
	namespace, name := SplitVMINamespaceKey(name)

	domain := api.NewDomainReferenceFromName(namespace, name)
	domain.Spec.Metadata.KubeVirt.UID = vmiUID
	domain.GetObjectMeta().SetUID(domain.Spec.Metadata.KubeVirt.UID)
	return domain
}

func configureQemuConf(qemuFilename string) (err error) {
	qemuConf, err := os.OpenFile(qemuFilename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(qemuConf, &err)

	// If hugepages exist, tell libvirt about them
	_, err = os.Stat("/dev/hugepages")
	if err == nil {
		_, err = qemuConf.WriteString("hugetlbfs_mount = \"/dev/hugepages\"\n")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if debugLogsStr, ok := os.LookupEnv("VIRTIOFSD_DEBUG_LOGS"); ok && (debugLogsStr == "1") {
		_, err = qemuConf.WriteString("virtiofsd_debug = 1\n")
		if err != nil {
			return err
		}
	}

	if pathsStr, ok := os.LookupEnv(services.ENV_VAR_SHARED_FILESYSTEM_PATHS); ok {
		paths := strings.Split(pathsStr, ":")
		formatted := strings.Join(paths, "\", \"")
		sharedFsEntry := fmt.Sprintf("shared_filesystems = [ \"%s\" ]\n", formatted)
		_, err = qemuConf.WriteString(sharedFsEntry)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyFile(from, to string) error {
	f, err := os.OpenFile(from, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(f, &err)
	newFile, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(newFile, &err)

	_, err = io.Copy(newFile, f)
	return err
}

func copyDir(src, dest string) error {
	sourceDirInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if _, err = os.Stat(dest); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dest, sourceDirInfo.Mode())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, destPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(srcPath, destPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

const (
	etlLibvirtInit = "/etc/libvirt-init"
	etcLibvirt     = "/etc/libvirt"
)

func (l LibvirtWrapper) SetupLibvirt(customLogFilters *string) (err error) {
	if _, err = os.Stat(etlLibvirtInit); err == nil {
		if err = copyDir(etlLibvirtInit, etcLibvirt); err != nil {
			return fmt.Errorf("failed to copy %q to %q: %w", etlLibvirtInit, etcLibvirt, err)
		}
	}

	runtimeQemuConfPath := qemuConfPath
	if !l.root() {
		runtimeQemuConfPath = qemuNonRootConfPath

		if err := os.MkdirAll(libvirtHomePath, 0755); err != nil {
			return err
		}
		if err := copyFile(qemuConfPath, runtimeQemuConfPath); err != nil {
			return err
		}
	}

	if err := configureQemuConf(runtimeQemuConfPath); err != nil {
		return err
	}

	runtimeVirtqemudConfPath := path.Join(libvirtRuntimePath, "virtqemud.conf")
	if err := copyFile(virtqemudConfPath, runtimeVirtqemudConfPath); err != nil {
		return err
	}

	var libvirtLogVerbosityEnvVar *string
	if envVarValue, envVarDefined := os.LookupEnv(services.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY); envVarDefined {
		libvirtLogVerbosityEnvVar = &envVarValue
	}
	_, libvirtDebugLogsEnvVarDefined := os.LookupEnv(services.ENV_VAR_LIBVIRT_DEBUG_LOGS)

	if logFilters, enableDebugLogs := getLibvirtLogFilters(customLogFilters, libvirtLogVerbosityEnvVar, libvirtDebugLogsEnvVarDefined); enableDebugLogs {
		virtqemudConf, err := os.OpenFile(runtimeVirtqemudConfPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer util.CloseIOAndCheckErr(virtqemudConf, &err)

		log.Log.Infof("Enabling libvirt log filters: %s", logFilters)
		_, err = virtqemudConf.WriteString(fmt.Sprintf("log_filters=\"%s\"\n", logFilters))
		if err != nil {
			return err
		}
	}

	return nil
}

// getLibvirtLogFilters returns libvirt debug log filters that should be enabled if enableDebugLogs is true.
// The decision is based on the following logic:
//   - If custom log filters are defined - they should be enabled and used.
//   - If verbosity is defined and beyond threshold then debug logs would be enabled and determined by verbosity level
//   - If verbosity level is below threshold but debug logs environment variable is defined, debug logs would be enabled
//     and set to the highest verbosity level.
//   - If verbosity level is below threshold and debug logs environment variable is not defined - debug logs are disabled.
func getLibvirtLogFilters(customLogFilters, libvirtLogVerbosityEnvVar *string, libvirtDebugLogsEnvVarDefined bool) (logFilters string, enableDebugLogs bool) {

	if customLogFilters != nil && *customLogFilters != "" {
		return *customLogFilters, true
	}

	var libvirtLogVerbosity int
	var err error

	if libvirtLogVerbosityEnvVar != nil {
		libvirtLogVerbosity, err = strconv.Atoi(*libvirtLogVerbosityEnvVar)
		if err != nil {
			log.Log.Infof("cannot apply %s value %s - must be a number", services.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY, *libvirtLogVerbosityEnvVar)
			libvirtLogVerbosity = -1
		}
	} else {
		libvirtLogVerbosity = -1
	}

	const verbosityThreshold = services.EXT_LOG_VERBOSITY_THRESHOLD

	if libvirtLogVerbosity < verbosityThreshold {
		if libvirtDebugLogsEnvVarDefined {
			libvirtLogVerbosity = verbosityThreshold + 5
		} else {
			return "", false
		}
	}

	// Higher log level means higher verbosity
	const logsLevel4 = "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access"
	const logsLevel3 = logsLevel4 + " 3:util.threadjob 3:cpu.cpu"
	const logsLevel2 = logsLevel3 + " 3:qemu.qemu_monitor"
	const logsLevel1 = logsLevel2 + " 3:qemu.qemu_monitor_json 3:conf.domain_addr"
	const allowAllOtherCategories = " 1:*"

	switch libvirtLogVerbosity {
	case verbosityThreshold:
		logFilters = logsLevel1
	case verbosityThreshold + 1:
		logFilters = logsLevel2
	case verbosityThreshold + 2:
		logFilters = logsLevel3
	case verbosityThreshold + 3:
		logFilters = logsLevel4
	default:
		logFilters = logsLevel4
	}

	return logFilters + allowAllOtherCategories, true
}

func (l LibvirtWrapper) root() bool {
	return l.user == 0
}
