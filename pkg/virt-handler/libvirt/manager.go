package libvirt

//go:generate mockgen -source $GOFILE -imports "libvirt=github.com/rgbkrk/libvirt-go" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/xml"
	"github.com/jeevatkm/go-model"
	"github.com/rgbkrk/libvirt-go"
	kubev1 "k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
)

type DomainManager interface {
	SyncVM(*v1.VM) error
	KillVM(*v1.VM) error
}

// TODO: Should we handle libvirt connection errors transparent or panic?
type Connection interface {
	LookupDomainByName(name string) (VirDomain, error)
	DomainDefineXML(xml string) (VirDomain, error)
	CloseConnection() (int, error)
	DomainEventRegister(dom libvirt.VirDomain, eventId int, callback *libvirt.DomainEventCallback, opaque func()) int
	ListAllDomains(flags uint32) ([]VirDomain, error)
}

type LibvirtConnection struct {
	libvirt.VirConnection
}

func (l *LibvirtConnection) LookupDomainByName(name string) (VirDomain, error) {
	dom, err := l.VirConnection.LookupDomainByName(name)
	return &dom, err
}

func (l *LibvirtConnection) DomainDefineXML(xml string) (VirDomain, error) {
	dom, err := l.VirConnection.DomainDefineXML(xml)
	return &dom, err
}

func (l *LibvirtConnection) ListAllDomains(flags uint32) ([]VirDomain, error) {
	virDoms, err := l.VirConnection.ListAllDomains(flags)
	if err != nil {
		return nil, err
	}
	doms := make([]VirDomain, len(virDoms))
	for i, d := range virDoms {
		doms[i] = &d
	}
	return doms, nil
}

type VirDomain interface {
	GetState() ([]int, error)
	Create() error
	Resume() error
	Destroy() error
	GetName() (string, error)
	GetUUIDString() (string, error)
	GetXMLDesc(flags uint32) (string, error)
	Undefine() error
}

type LibvirtDomainManager struct {
	virConn  Connection
	recorder record.EventRecorder
}

func NewConnection(uri string, user string, pass string) (Connection, error) {
	virConn, err := libvirt.NewVirConnectionWithAuth(uri, user, pass)
	if err != nil {
		return nil, err
	}
	return &LibvirtConnection{VirConnection: virConn}, nil
}

func NewLibvirtDomainManager(connection Connection, recorder record.EventRecorder) (DomainManager, error) {
	manager := LibvirtDomainManager{virConn: connection, recorder: recorder}
	return &manager, nil
}

func (l *LibvirtDomainManager) SyncVM(vm *v1.VM) error {
	var wantedSpec DomainSpec
	mappingErrs := model.Copy(&wantedSpec, vm.Spec.Domain)
	if len(mappingErrs) > 0 {
		// TODO: proper aggregation
		return mappingErrs[0]
	}
	dom, err := l.virConn.LookupDomainByName(vm.GetObjectMeta().GetName())
	if err != nil {
		// We need the domain but it does not exist, so create it
		if err.(libvirt.VirError).Code == libvirt.VIR_ERR_NO_DOMAIN {
			xml, err := xml.Marshal(&wantedSpec)
			if err != nil {
				return err
			}
			dom, err = l.virConn.DomainDefineXML(string(xml))
			if err != nil {
				return err
			}
			l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Created.String(), "VM defined")
		}
	}
	domState, err := dom.GetState()
	if err != nil {
		return err
	}
	// TODO Suspend, Pause, ..., for now we only support reaching the running state
	// TODO for migration and error detection we also need the state change reason
	state := LifeCycleTranslationMap[domState[0]]
	switch state {
	case NoState, Shutdown, Shutoff, Crashed:
		err := dom.Create()
		if err != nil {
			return err
		}
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Started.String(), "VM started")
	case Paused:
		// TODO: if state change reason indicates a system error, we could try something smarter
		err := dom.Resume()
		if err != nil {
			return err
		}
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Resumed.String(), "VM resumed")
	default:
		// Nothing to do
		// TODO: blocked state
	}

	// TODO: check if VM Spec and Domain Spec are equal or if we have to sync
	return nil
}

func (l *LibvirtDomainManager) KillVM(vm *v1.VM) error {
	dom, err := l.virConn.LookupDomainByName(vm.GetObjectMeta().GetName())
	if err != nil {
		// If the VM does not exist, we are done
		if err.(libvirt.VirError).Code == libvirt.VIR_ERR_NO_DOMAIN {
			return nil
		} else {
			return err
		}
	}
	// TODO: Graceful shutdown
	domState, err := dom.GetState()
	if err != nil {
		return err
	}

	state := LifeCycleTranslationMap[domState[0]]
	if state == Running || state == Paused {
		err = dom.Destroy()
		if err != nil {
			return err
		}
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Stopped.String(), "VM stopped")
	}

	err = dom.Undefine()
	if err != nil {
		return err
	}
	l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Deleted.String(), "VM undefined")
	return nil
}
