package rest

import (
	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"github.com/libvirt/libvirt-go"
	"io"
	"k8s.io/client-go/pkg/types"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Console struct {
	connection virtwrap.Connection
}

func NewConsoleResource(connection virtwrap.Connection) *Console {
	return &Console{connection: connection}
}

func (t *Console) Console(request *restful.Request, response *restful.Response) {
	console := request.HeaderParameter("console")
	vmName := request.PathParameter("name")
	vm := v1.NewVMReferenceFromName(vmName)
	log := logging.DefaultLogger().Object(vm)
	domain, err := t.connection.LookupDomainByName(vmName)
	if err != nil {
		if err.(libvirt.Error).Code == libvirt.ERR_NO_DOMAIN {
			log.Error().Reason(err).Msg("Domain not found.")
			response.WriteError(http.StatusNotFound, err)
			return
		} else {
			response.WriteError(http.StatusInternalServerError, err)
			log.Error().Reason(err).Msg("Failed to look up domain.")
			return
		}
	}

	uid, err := domain.GetUUIDString()
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		log.Error().Reason(err).Msg("Failed to look up domain UID.")
		return
	}
	vm.GetObjectMeta().SetUID(types.UID(uid))
	log = logging.DefaultLogger().Object(vm)

	log.Info().Msgf("Opening connection to console %s", console)

	consoleStream, err := t.connection.NewStream(0)
	if err != nil {
		log.Error().Reason(err).Msg("Creating a consoleStream failed.")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	defer consoleStream.Close()

	log.Info().V(3).Msg("Stream created.")

	err = domain.OpenConsole(console, consoleStream.UnderlyingStream(), libvirt.DOMAIN_CONSOLE_FORCE)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		log.Error().Reason(err).Msg("Failed to open console.")
		return
	}
	log.Info().V(3).Msg("Connection to console created.")

	errorChan := make(chan error)

	ws, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		log.Error().Reason(err).Msg("Failed to upgrade websocket connection.")
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	defer ws.Close()

	wsReadWriter := &TextReadWriter{ws}

	go func() {
		_, err := io.Copy(consoleStream, wsReadWriter)
		errorChan <- err
	}()

	go func() {
		_, err := io.Copy(wsReadWriter, consoleStream)
		errorChan <- err
	}()

	err = <-errorChan

	if err != nil {
		log.Error().Reason(err).Msg("Proxying data between libvirt and the websocket failed.")
	}

	log.Info().V(3).Msg("Done.")
	response.WriteHeader(http.StatusOK)
}

type TextReadWriter struct {
	*websocket.Conn
}

func (s *TextReadWriter) Write(p []byte) (int, error) {
	err := s.Conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, s.err(err)
	}
	return len(p), nil
}

func (s *TextReadWriter) Read(p []byte) (int, error) {
	_, r, err := s.Conn.NextReader()
	if err != nil {
		return 0, s.err(err)
	}
	n, err := r.Read(p)
	return n, s.err(err)
}

func (s *TextReadWriter) err(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*websocket.CloseError); ok {
		if e.Code == websocket.CloseNormalClosure {
			return io.EOF
		}
	}
	return err
}
