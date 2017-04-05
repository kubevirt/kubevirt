package virtwrap

//go:generate mockgen -source $GOFILE -imports "libvirt=github.com/libvirt/libvirt-go" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/xml"
	"io"

	"github.com/jeevatkm/go-model"
	"github.com/libvirt/libvirt-go"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
)

type DomainManager interface {
	SyncVM(*v1.VM) error
	KillVM(*v1.VM) error
}

// TODO: Should we handle libvirt connection errors transparent or panic?
type Connection interface {
	LookupDomainByName(name string) (VirDomain, error)
	DomainDefineXML(xml string) (VirDomain, error)
	Close() (int, error)
	DomainEventLifecycleRegister(dom *libvirt.Domain, callback libvirt.DomainEventLifecycleCallback) (int, error)
	ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]VirDomain, error)
	NewStream(flags libvirt.StreamFlags) (Stream, error)
}

type Stream interface {
	io.ReadWriteCloser
	UnderlyingStream() *libvirt.Stream
}

type VirStream struct {
	*libvirt.Stream
}

type LibvirtConnection struct {
	libvirt.Connect
	user  string
	pass  string
	uri   string
	alive bool
}

func (s *VirStream) Write(p []byte) (n int, err error) {
	return s.Stream.Send(p)
}

func (s *VirStream) Read(p []byte) (n int, err error) {
	return s.Stream.Recv(p)
}

/*
Close the stream and free its resources. Since closing a stream involves multiple calls with errors,
the first error occured will be returned. The stream will always be freed.
*/
func (s *VirStream) Close() (e error) {
	e = s.Finish()
	if e != nil {
		return s.Free()
	}
	s.Free()
	return e
}

func (s *VirStream) UnderlyingStream() *libvirt.Stream {
	return s.Stream
}

func (l *LibvirtConnection) NewStream(flags libvirt.StreamFlags) (Stream, error) {
	s, err := l.Connect.NewStream(flags)
	if err != nil {
		return nil, err
	}
	return &VirStream{Stream: s}, nil
}

func (l *LibvirtConnection) LookupDomainByName(name string) (VirDomain, error) {
	libvirt.EventRegisterDefaultImpl()
	if !l.alive {
		conn, err := newConnection(l.uri, l.user, l.pass)
		if err != nil {
			logging.DefaultLogger().Error().Reason(err).Msg("Connection to libvirt lost.")
			return nil, err
		}
		l.alive = true
		l.Connect = *conn
	}

	dom, err := l.Connect.LookupDomainByName(name)
	if err != nil && err.(libvirt.Error).Code != libvirt.ERR_NO_DOMAIN {
		l.alive = false
		logging.DefaultLogger().Error().Reason(err).Msg("Connection to libvirt lost.")
	}
	return dom, err
}

func (l *LibvirtConnection) DomainDefineXML(xml string) (VirDomain, error) {
	dom, err := l.Connect.DomainDefineXML(xml)
	return dom, err
}

func (l *LibvirtConnection) ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]VirDomain, error) {
	virDoms, err := l.Connect.ListAllDomains(flags)
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
	GetState() (libvirt.DomainState, int, error)
	Create() error
	Resume() error
	Destroy() error
	GetName() (string, error)
	GetUUIDString() (string, error)
	GetXMLDesc(flags libvirt.DomainXMLFlags) (string, error)
	Undefine() error
	OpenConsole(devname string, stream *libvirt.Stream, flags libvirt.DomainConsoleFlags) error
}

type LibvirtDomainManager struct {
	virConn  Connection
	recorder record.EventRecorder
}

func NewConnection(uri string, user string, pass string) (Connection, error) {
	virConn, err := newConnection(uri, user, pass)
	if err != nil {
		return nil, err
	}
	lvConn := &LibvirtConnection{Connect: *virConn, user: user, pass: pass, uri: uri, alive: true}
	return lvConn, nil
}

// TODO: needs a functional test.
func authWithPassword(uri string, user string, pass string) (*libvirt.Connect, error) {
	callback := func(creds []*libvirt.ConnectCredential) {
		for _, cred := range creds {
			if cred.Type == libvirt.CRED_AUTHNAME {
				cred.Result = user
				cred.ResultLen = len(cred.Result)
			} else if cred.Type == libvirt.CRED_PASSPHRASE {
				cred.Result = pass
				cred.ResultLen = len(cred.Result)
			}
		}
	}
	auth := &libvirt.ConnectAuth{
		CredType: []libvirt.ConnectCredentialType{
			libvirt.CRED_AUTHNAME, libvirt.CRED_PASSPHRASE,
		},
		Callback: callback,
	}
	virConn, err := libvirt.NewConnectWithAuth(uri, auth, 0)

	return virConn, err
}

func newConnection(uri string, user string, pass string) (*libvirt.Connect, error) {
	var virConn *libvirt.Connect
	var err error
	if user == "" {
		virConn, err = libvirt.NewConnect(uri)
	} else {
		virConn, err = authWithPassword(uri, user, pass)
	}
	return virConn, err
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
		if err.(libvirt.Error).Code == libvirt.ERR_NO_DOMAIN {
			xmlStr, err := xml.Marshal(&wantedSpec)
			if err != nil {
				logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Generating the domain xmlStr failed.")
				return err
			}
			logging.DefaultLogger().Object(vm).Info().V(3).Msg("Domain XML generated.")
			dom, err = l.virConn.DomainDefineXML(string(xmlStr))
			if err != nil {
				logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Defining the VM failed.")
				return err
			}
			logging.DefaultLogger().Object(vm).Info().Msg("Domain defined.")
			l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Created.String(), "VM defined")
		} else {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain failed.")
			return err
		}
	}
	domState, _, err := dom.GetState()
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain state failed.")
		return err
	}
	// TODO Suspend, Pause, ..., for now we only support reaching the running state
	// TODO for migration and error detection we also need the state change reason
	//state := LifeCycleTranslationMap[domState[0]]
	switch domState {
	case libvirt.DOMAIN_NOSTATE, libvirt.DOMAIN_SHUTDOWN, libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_CRASHED:
		err := dom.Create()
		if err != nil {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Starting the VM failed.")
			return err
		}
		logging.DefaultLogger().Object(vm).Info().Msg("Domain started.")
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Started.String(), "VM started.")
	case libvirt.DOMAIN_PAUSED:
		// TODO: if state change reason indicates a system error, we could try something smarter
		err := dom.Resume()
		if err != nil {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Resuming the VM failed.")
			return err
		}
		logging.DefaultLogger().Object(vm).Info().Msg("Domain resumed.")
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
		if err.(libvirt.Error).Code == libvirt.ERR_NO_DOMAIN {
			return nil
		} else {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain failed.")
			return err
		}
	}
	// TODO: Graceful shutdown
	domState, _, err := dom.GetState()
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED {
		err = dom.Destroy()
		if err != nil {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Destroying the domain state failed.")
			return err
		}
		logging.DefaultLogger().Object(vm).Info().Msg("Domain stopped.")
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Stopped.String(), "VM stopped")
	}

	err = dom.Undefine()
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Undefining the domain state failed.")
		return err
	}
	logging.DefaultLogger().Object(vm).Info().Msg("Domain undefined.")
	l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Deleted.String(), "VM undefined")
	return nil
}
