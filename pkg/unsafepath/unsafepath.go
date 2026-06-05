package unsafepath

import "path/filepath"

type Path struct {
	rootBase     string
	relativePath string
}

func New(rootBase string, relativePath string) *Path {
	return &Path{
		rootBase:     rootBase,
		relativePath: relativePath,
	}
}

func UnsafeAbsolute(path *Path) string {
	return filepath.Join(path.rootBase, path.relativePath)
}

func UnsafeRelative(path *Path) string {
	return path.relativePath
}

func UnsafeRoot(path *Path) string {
	return path.rootBase
}
