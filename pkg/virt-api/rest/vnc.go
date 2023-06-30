package rest

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"time"

	"kubevirt.io/kubevirt/pkg/virt-api/definitions"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/api"

	"github.com/mitchellh/go-vnc"
)

func (app *SubresourceAPIApp) VNCRequestHandler(request *restful.Request, response *restful.Response) {
	activeConnectionMetric := apimetrics.NewActiveVNCConnection(request.PathParameter("namespace"), request.PathParameter("name"))
	defer activeConnectionMetric.Dec()

	streamer := NewRawStreamer(
		app.FetchVirtualMachineInstance,
		validateVMIForVNC,
		app.virtHandlerDialer(func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			return conn.VNCURI(vmi)
		}),
	)

	streamer.Handle(request, response)
}

// VNCScreenshotRequestHandler opens a websocket based VNC connection to virt-handler and creates a screenshot in PNG format
// which it returns to the caller. No websocket connection will be forwarded to the client.
// This is inspired by https://raw.githubusercontent.com/hexylena/vnc-screenshot/9f609b72518d6d6ab5149502a6be1dd3c5b015c8/vnc-screenshot.go.
func (app *SubresourceAPIApp) VNCScreenshotRequestHandler(request *restful.Request, response *restful.Response) {
	activeConnectionMetric := apimetrics.NewActiveVNCConnection(request.PathParameter("namespace"), request.PathParameter("name"))
	defer activeConnectionMetric.Dec()

	dialer := NewDirectDialer(
		app.FetchVirtualMachineInstance,
		validateVMIForVNC,
		app.virtHandlerDialer(func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			return conn.VNCURI(vmi)
		}),
	)
	namespace := request.PathParameter(definitions.NamespaceParamName)
	name := request.PathParameter(definitions.NameParamName)
	moveCursor := request.QueryParameter(definitions.MoveCursorParamName)

	nc, statusErr := dialer.Dial(namespace, name)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}

	done := make(chan struct{})
	streamer := kubecli.NewWebsocketStreamer(nc, done)
	defer close(done)

	ch := make(chan vnc.ServerMessage)
	c, err := vnc.Client(streamer.AsConn(), &vnc.ClientConfig{
		Exclusive:       false,
		ServerMessageCh: ch,
		ServerMessages:  []vnc.ServerMessage{new(vnc.FramebufferUpdateMessage)},
	})
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}
	defer c.Close()
	log.DefaultLogger().Infof("Connected to VNC desktop: %s [res:%dx%d]\n", c.DesktopName, c.FrameBufferWidth, c.FrameBufferHeight)

	// Try to wake up the screen
	if moveCursor == "true" {
		_ = c.PointerEvent(0, 0, 0)
		_ = c.PointerEvent(0, 1, 1)
	}

	// Then send a buffer update request
	err = c.FramebufferUpdateRequest(false, 0, 0, c.FrameBufferWidth, c.FrameBufferHeight)
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	var msg vnc.ServerMessage
	select {
	case msg = <-ch:
	case <-time.After(2 * time.Second):
		writeError(errors.NewInternalError(fmt.Errorf("timed out waiting for VNC server messages")), response)
		return
	}

	fbMsg, ok := msg.(*vnc.FramebufferUpdateMessage)

	if !ok || len(fbMsg.Rectangles) == 0 {
		writeError(errors.NewInternalError(fmt.Errorf("failed to retrieve the VNC screen")), response)
		return
	}
	rects := fbMsg.Rectangles

	w := int(rects[0].Width)
	h := int(rects[0].Height)
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	enc := rects[0].Enc.(*vnc.RawEncoding)
	i := 0
	x := 0
	y := 0
	for _, v := range enc.Colors {
		x = i % w
		y = i / w
		r := uint8(v.R)
		g := uint8(v.G)
		b := uint8(v.B)

		img.Set(x, y, color.RGBA{r, g, b, 255})
		i++
	}

	pipeReader, pipeWriter := io.Pipe()
	encodeErrChan := make(chan error, 1)
	copyErrChan := make(chan error, 1)
	go func() {
		encodeErrChan <- png.Encode(pipeWriter, img)
		defer pipeWriter.Close()
	}()

	response.AddHeader("Content-Type", "image/png")
	go func() {
		_, err := io.Copy(response, pipeReader)
		copyErrChan <- err
	}()

	encodeErr := <-encodeErrChan
	if encodeErr != nil {
		writeError(errors.NewInternalError(fmt.Errorf("failed to encode the image: %v", encodeErr)), response)
		return
	}

	copyErr := <-copyErrChan
	if copyErr != nil {
		writeError(errors.NewInternalError(fmt.Errorf("failed to stream the image: %v", copyErr)), response)
		return
	}
}

func validateVMIForVNC(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	// If there are no graphics devices present, we can't proceed
	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice != nil && *vmi.Spec.Domain.Devices.AutoattachGraphicsDevice == false {
		err := fmt.Errorf("No graphics devices are present.")
		log.Log.Object(vmi).Reason(err).Error("Can't establish VNC connection.")
		return errors.NewBadRequest(err.Error())
	}
	if !vmi.IsRunning() {
		return errors.NewBadRequest(vmiNotRunning)
	}
	return nil
}
