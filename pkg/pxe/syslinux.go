package pxe

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/emicklei/go-restful"
)

const (
	menu = `
# Use the simple menu
UI menu.c32

# Time out and use the default menu option. Defined as tenths of a second.
TIMEOUT 2600 

# Prompt the user. Set to '1' to automatically choose the default option. This
# is really meant for files matched to MAC addresses.
PROMPT 0

# Set the boot menu to be 1024x768 with a nice background image. Be careful to
# ensure that all your user's can see this resolution! Default is 640x480.
MENU RESOLUTION 1024 768

# These do not need to be set. I set them here to show how you can customize or
# localize your PXE server's dialogue.
MENU TITLE    PXE Boot Server

### Now define the menu options
LABEL next
	MENU LABEL Boot local 
	MENU DEFAULT
	localboot

LABEL hdt
MENU LABEL Run Hardware Detection Tool
COM32 /hdt.c32

LABEL reboot
MENU LABEL Reboot
COM32 /reboot.c32

LABEL poweroff
MENU LABEL Power Off
COM32 /poweroff.c32
`
)

const (
	ConfigDir          = "/pxelinux.cfg/"
	DefaultSYSLINUXDir = "/usr/share/syslinux"
	DefaultConfig      = "default"
)

func SYSLINUXConfigServer(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	namespace := req.PathParameter("namespace")
	config := req.PathParameter("config")

	fmt.Println("name: " + name)
	fmt.Println("namespace: " + namespace)
	fmt.Println("config: " + config)
	if config == DefaultConfig {
		resp.Header().Set(restful.HEADER_ContentType, "text/plain")
		resp.WriteHeader(http.StatusOK)
		_, err := resp.Write([]byte(menu))
		if err != nil {
			fmt.Println(err)
		}
	} else {
		resp.WriteErrorString(http.StatusNotFound, "config file does not exist")
	}
}

func SYSLINUXServer(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	namespace := req.PathParameter("namespace")
	fmt.Println("name: " + name)
	fmt.Println("namespace: " + namespace)
	file := req.PathParameter("filepath")
	fmt.Println("syslinux: " + file)
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(DefaultSYSLINUXDir, file))
}
