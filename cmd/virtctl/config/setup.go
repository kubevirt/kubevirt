package config

import (
	"fmt"
	"os/exec"
)

func detectViewers() []string {
	found := []string{}

	for _, v := range SupportedViewers {
		if _, err := exec.LookPath(v); err == nil {
			found = append(found, v)
		}
	}
	return found
}

func isSupported(viewer string) bool {
	for _, v := range SupportedViewers {
		if v == viewer {
			return true
		}
	}
	return false
}

func RunSetup(vncViewer string) error {
	var selected string

	if vncViewer != "" {
		if !isSupported(vncViewer) {
			return fmt.Errorf("unsupported viewer %q, supported: %v", vncViewer, SupportedViewers)
		}
		selected = vncViewer
	} else {
		fmt.Println("Detecting VNC viewers...")

		found := detectViewers()

		if len(found) == 0 {
			return fmt.Errorf("no VNC viewers found in PATH")
		}

		fmt.Println("Select preferred VNC viewer:")

		for i, v := range found {
			fmt.Printf("%d) %s\n", i+1, v)
		}

		var choice int
		fmt.Print("Enter choice: ")

		_, err := fmt.Scanln(&choice)
		if err != nil || choice < 1 || choice > len(found) {
			return fmt.Errorf("invalid selection")
		}

		selected = found[choice-1]
	}

	cfg := &Config{}
	cfg.VNC.Viewer = selected

	if err := SaveConfig(cfg); err != nil {
		return err
	}

	fmt.Println("Configuration saved successfully.")
	return nil
}
