//go:build includenovnc

package novnc

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"golang.org/x/tools/godoc/vfs"
	"golang.org/x/tools/godoc/vfs/zipfs"
	"kubevirt.io/client-go/kubecli"
)

//go:embed novnc.zip
var novnc []byte

var mimetypes = map[string]string{
	".html": "text/html",
	".js":   "text/javascript",
	".svg":  "image/svg+xml",
	".css":  "text/css",
	".woff": "application/x-font-woff",
	".oga":  "audio/ogg",
	".json": "application/json",
	".ico":  "image/x-icon",
}

func RunNOVNCWebserver(address, port string, vnc net.Conn) error {

	reader, err := zip.NewReader(bytes.NewReader(novnc), int64(len(novnc)))
	if err != nil {
		panic(err)
	}

	rc := &zip.ReadCloser{
		Reader: *reader,
	}
	fs := zipfs.New(rc, "novnc")

	dirs, err := fs.ReadDir("/")
	if err != nil {
		return err
	}
	if len(dirs) != 1 {
		return fmt.Errorf("exprected only one base directory, but got %v", len(dirs))
	}
	baseDir := strings.TrimSpace(dirs[0].Name())

	writeChan := make(chan struct{})
	readChan := make(chan struct{})

	lnAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", address, port))
	if err != nil {
		return fmt.Errorf("Can't resolve the address: %s", err.Error())
	}

	// Listen early to be sure the http server is ready when we open the browser
	l, err := net.ListenTCP("tcp", lnAddr)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveNOVNC(baseDir, fs))
	mux.HandleFunc("/websockify", func(writer http.ResponseWriter, request *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			glog.Errorf("failed to upgrade the websocket connection %v", err)
			writer.WriteHeader(500)
			return
		}
		defer conn.Close()
		done := make(chan struct{})
		defer close(done)
		wsConn := kubecli.NewWebsocketStreamer(conn, done).AsConn()

		go func() {
			_, err := io.Copy(vnc, wsConn)
			if err != nil {
				glog.Errorf("failed to copy content from server to novnc: %v", err)
			}
			close(writeChan)
		}()
		go func() {
			_, err := io.Copy(wsConn, vnc)
			if err != nil {
				glog.Errorf("failed to copy content from novnc to server: %v", err)
			}
			close(readChan)
		}()

		select {
		case <-writeChan:
		case <-readChan:
		}

	})

	errChan := make(chan error, 1)

	go func() {
		errChan <- http.Serve(l, mux)
	}()

	fmt.Printf("Open http://%v:%v?autoconnect=true to connect to the virtual machine.\n", l.Addr().(*net.TCPAddr).IP, l.Addr().(*net.TCPAddr).Port)
	select {
	case <-writeChan:
	case <-readChan:
	case err := <-errChan:
		return err
	}
	return nil
}

func serveNOVNC(baseDir string, fs vfs.FileSystem) func(writer http.ResponseWriter, request *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "no-cache")

		path := request.URL.Path
		if path == "/" {
			path = "/vnc.html"
		}
		if ext := filepath.Ext(path); ext != "" {
			if _, exists := mimetypes[ext]; !exists {
				glog.Warningf("unknown content-type for file with extension: %v", ext)
			}
			writer.Header().Set("Content-Type", mimetypes[ext])
		}
		path = filepath.Join("/", baseDir, path)
		reader, err := fs.Open(path)
		if err != nil {
			glog.Errorf("failed to read path %v: %v", path, err)
			writer.WriteHeader(500)
			return
		}
		defer reader.Close()
		if _, err := io.Copy(writer, reader); err != nil {
			glog.Errorf("failed to serve path content %v: %v", path, err)
			writer.WriteHeader(500)
			return
		}
	}
}
