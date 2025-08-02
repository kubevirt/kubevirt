package cmdclient

import "path/filepath"

func SocketOnGuest() string {
	sockFile := StandardLauncherSocketFileName
	return filepath.Join(SocketsDirectory(), sockFile)
}

func UninitializedSocketOnGuest() string {
	sockFile := StandardInitLauncherSocketFileName
	return filepath.Join(SocketsDirectory(), sockFile)
}
