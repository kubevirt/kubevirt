package util

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"syscall"
	"time"

	libvirt "github.com/libvirt/libvirt-go"
	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
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

func SetDomainSpecStr(virConn cli.Connection, vmi *v1.VirtualMachineInstance, wantedSpec string) (cli.VirDomain, error) {
	log.Log.Object(vmi).V(3).With("xml", wantedSpec).Info("Domain XML generated.")
	dom, err := virConn.DomainDefineXML(wantedSpec)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Defining the VirtualMachineInstance failed.")
		return nil, err
	}
	return dom, nil
}

// GetDomainSpecWithRuntimeInfo return the active domain XML with runtime information embedded
func GetDomainSpecWithRuntimeInfo(status libvirt.DomainState, dom cli.VirDomain) (*api.DomainSpec, error) {

	// get libvirt xml with runtime status
	activeSpec, err := GetDomainSpecWithFlags(dom, 0)
	if err != nil {
		log.Log.Reason(err).Error("failed to get domain spec")
		return nil, err
	}

	metadataXML, err := dom.GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG)
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

func StartLibvirt(stopChan chan struct{}) {
	// we spawn libvirt from virt-launcher in order to ensure the libvirtd+qemu process
	// doesn't exit until virt-launcher is ready for it to. Virt-launcher traps signals
	// to perform special shutdown logic. These processes need to live in the same
	// container.

	go func() {
		for {
			exitChan := make(chan struct{})
			cmd := exec.Command("/usr/sbin/libvirtd")

			// connect libvirt's stderr to our own stdout in order to see the logs in the container logs
			reader, err := cmd.StderrPipe()
			if err != nil {
				log.Log.Reason(err).Error("failed to start libvirtd")
				panic(err)
			}

			go func() {
				scanner := bufio.NewScanner(reader)
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

func StartVirtlog(stopChan chan struct{}) {
	go func() {
		for {
			var args []string
			args = append(args, "-f")
			args = append(args, "/etc/libvirt/virtlogd.conf")
			cmd := exec.Command("/usr/sbin/virtlogd", args...)

			exitChan := make(chan struct{})

			err := cmd.Start()
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

func SetupLibvirt() error {

	// TODO: setting permissions and owners is not part of device plugins.
	// Configure these manually right now on "/dev/kvm"
	stats, err := os.Stat("/dev/kvm")
	if err == nil {
		s, ok := stats.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("can't convert file stats to unix/linux stats")
		}
		err := os.Chown("/dev/kvm", int(s.Uid), 107)
		if err != nil {
			return err
		}
		err = os.Chmod("/dev/kvm", 0660)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	qemuConf, err := os.OpenFile("/etc/libvirt/qemu.conf", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer qemuConf.Close()
	// We are in a container, don't try to stuff qemu inside special cgroups
	_, err = qemuConf.WriteString("cgroup_controllers = [ ]\n")
	if err != nil {
		return err
	}

	// If hugepages exist, tell libvirt about them
	_, err = os.Stat("/dev/hugepages")
	if err == nil {
		_, err = qemuConf.WriteString("hugetlbfs_mount = \"/dev/hugepages\"\n")
	} else if !os.IsNotExist(err) {
		return err
	}

	// Let libvirt log to stderr
	libvirtConf, err := os.OpenFile("/etc/libvirt/libvirtd.conf", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer libvirtConf.Close()
	_, err = libvirtConf.WriteString("log_outputs = \"1:stderr\"\n")
	if err != nil {
		return err
	}

	return nil
}
