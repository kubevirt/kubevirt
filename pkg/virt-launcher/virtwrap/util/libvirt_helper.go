package util

import (
	"bufio"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

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
	libvirdConfPath     = "/etc/libvirt/libvirtd.conf"
	libvirtRuntimePath  = "/var/run/libvirt"
	qemuNonRootConfPath = libvirtRuntimePath + "/qemu.conf"
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
	spec := wantedSpec.DeepCopy()
	hooksManager := hooks.GetManager()

	domainSpec, err := hooksManager.OnDefineDomain(spec, vmi)
	if err != nil {
		return nil, err
	}
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

	// use different flag with GetMetadata for transient domains
	domainModificationImpactFlag, err := getDomainModificationImpactFlag(dom)
	if err != nil {
		return activeSpec, err
	}

	metadataXML, err := dom.GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", domainModificationImpactFlag)
	if err != nil {
		log.Log.Reason(err).Error("failed to get domain metadata")
		return activeSpec, err
	}

	metadata := &api.KubeVirtMetadata{}
	err = xml.Unmarshal([]byte(metadataXML), metadata)
	if err != nil {
		log.Log.Reason(err).Error("failed to unmarshal domain metadata")
		return activeSpec, err
	}

	activeSpec.Metadata.KubeVirt = *metadata
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

	if !reflect.DeepEqual(spec.Metadata, inactiveSpec.Metadata) {
		// Metadata is updated on offline config only. As a result,
		// We have to merge updates to metadata into the domain spec.
		metadata := &inactiveSpec.Metadata
		metadata.DeepCopyInto(&spec.Metadata)
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

func (l LibvirtWrapper) StartLibvirt(stopChan chan struct{}) {
	// we spawn libvirt from virt-launcher in order to ensure the libvirtd+qemu process
	// doesn't exit until virt-launcher is ready for it to. Virt-launcher traps signals
	// to perform special shutdown logic. These processes need to live in the same
	// container.

	go func() {
		for {
			exitChan := make(chan struct{})
			args := []string{"-f", "/var/run/libvirt/libvirtd.conf"}
			cmd := exec.Command("/usr/sbin/libvirtd", args...)
			if l.user != 0 {
				cmd.SysProcAttr = &syscall.SysProcAttr{
					AmbientCaps: []uintptr{unix.CAP_NET_BIND_SERVICE},
				}
			}

			// connect libvirt's stderr to our own stdout in order to see the logs in the container logs
			reader, err := cmd.StderrPipe()
			if err != nil {
				log.Log.Reason(err).Error("failed to start libvirtd")
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
				log.Log.Reason(err).Error("failed to start libvirtd")
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
				log.Log.Errorf("libvirtd exited, restarting")
			}

			// this sleep is to avoid consumming all resources in the
			// event of a libvirtd crash loop.
			time.Sleep(time.Second)
		}
	}()
}

func startVirtlogdLogging(stopChan chan struct{}, domainName string, nonRoot bool) {
	for {
		cmd := exec.Command("/usr/sbin/virtlogd", "-f", "/etc/libvirt/virtlogd.conf")

		exitChan := make(chan struct{})

		err := cmd.Start()
		if err != nil {
			log.Log.Reason(err).Error("failed to start virtlogd")
			panic(err)
		}

		go func() {
			logfile := fmt.Sprintf("/var/log/libvirt/qemu/%s.log", domainName)
			if nonRoot {
				logfile = filepath.Join("/var", "run", "libvirt", "qemu", "log", fmt.Sprintf("%s.log", domainName))
			}

			// It can take a few seconds to the log file to be created
			for {
				_, err = os.Stat(logfile)
				if !os.IsNotExist(err) {
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

		// this sleep is to avoid consumming all resources in the
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
	} else if !os.IsNotExist(err) {
		return err
	}

	if envVarValue, ok := os.LookupEnv("VIRTIOFSD_DEBUG_LOGS"); ok && (envVarValue == "1") {
		_, err = qemuConf.WriteString("virtiofsd_debug = 1\n")
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

func (l LibvirtWrapper) SetupLibvirt() (err error) {
	runtimeQemuConfPath := qemuConfPath
	if !l.root() {
		runtimeQemuConfPath = qemuNonRootConfPath

		if err := copyFile(qemuConfPath, runtimeQemuConfPath); err != nil {
			return err
		}
	}

	if err := configureQemuConf(runtimeQemuConfPath); err != nil {
		return err
	}

	runtimeLibvirtdConfPath := path.Join(libvirtRuntimePath, "libvirtd.conf")
	if err := copyFile(libvirdConfPath, runtimeLibvirtdConfPath); err != nil {
		return err
	}

	if envVarValue, ok := os.LookupEnv("LIBVIRT_DEBUG_LOGS"); ok && (envVarValue == "1") {
		libvirdDConf, err := os.OpenFile(runtimeLibvirtdConfPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer util.CloseIOAndCheckErr(libvirdDConf, &err)

		// see https://libvirt.org/kbase/debuglogs.html for details
		_, err = libvirdDConf.WriteString("log_filters=\"3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*\"\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func getDomainModificationImpactFlag(dom cli.VirDomain) (libvirt.DomainModificationImpact, error) {
	isDomainPersistent, err := dom.IsPersistent()
	if err != nil {
		log.Log.Reason(err).Error("failed to query a domain")
		return libvirt.DOMAIN_AFFECT_CONFIG, err
	}
	if !isDomainPersistent {
		log.Log.V(3).Info("domain is transient")
		return libvirt.DOMAIN_AFFECT_LIVE, nil
	}
	log.Log.V(3).Info("domain is persistent")
	return libvirt.DOMAIN_AFFECT_CONFIG, nil
}

func (l LibvirtWrapper) root() bool {
	return l.user == 0
}
