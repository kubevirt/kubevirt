package config

import "fmt"

var SupportedViewers = []string{
	"remote-viewer",
	"virt-viewer",
	"vncviewer",
}

type Config struct {
	VNC VNCConfig `yaml:"vnc"`
}

type VNCConfig struct {
	Viewer string `yaml:"viewer"`
}

func (c *Config) Validate() error {
	if c.VNC.Viewer == "" {
		return nil
	}
	for _, v := range SupportedViewers {
		if c.VNC.Viewer == v {
			return nil
		}
	}
	return fmt.Errorf("unsupported VNC viewer %q, supported: %v", c.VNC.Viewer, SupportedViewers)
}
