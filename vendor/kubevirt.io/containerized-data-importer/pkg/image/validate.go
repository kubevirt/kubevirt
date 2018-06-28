package image

import (
	"strings"
)

const (
	ExtImg   = ".img"
	ExtIso   = ".iso"
	ExtGz    = ".gz"
	ExtQcow2 = ".qcow2"
	ExtTar   = ".tar"
	ExtXz    = ".xz"
	ExtTarXz = ExtTar + ExtXz
	ExtTarGz = ExtTar + ExtGz
)

var SupportedNestedExtensions = []string{
	ExtTarGz, ExtTarXz,
}

var SupportedCompressionExtensions = []string{
	ExtGz, ExtXz,
}

var SupportedArchiveExtensions = []string{
	ExtTar,
}

var SupportedCompressArchiveExtensions = append(
	SupportedCompressionExtensions,
	SupportedArchiveExtensions...,
)

var SupportedImageFormats = []string{
	ExtImg, ExtIso, ExtQcow2,
}

var SupportedFileExtensions = append(
	SupportedImageFormats, append(
		SupportedCompressionExtensions, append(
			SupportedArchiveExtensions,
			SupportedNestedExtensions...,
		)...,
	)...,
)

func IsSupportedType(fn string, exts []string) bool {
	fn = TrimString(fn)
	for _, ext := range exts {
		if strings.HasSuffix(fn, ext) {
			return true
		}
	}
	return false
}

func IsSupportedFileType(fn string) bool {
	return IsSupportedType(fn, SupportedFileExtensions)
}

func IsSupportedCompressionType(fn string) bool {
	return IsSupportedType(fn, SupportedCompressionExtensions)
}

func IsSupportedArchiveType(fn string) bool {
	return IsSupportedType(fn, SupportedArchiveExtensions)
}

func IsSupportedCompressArchiveType(fn string) bool {
	return IsSupportedType(fn, SupportedCompressArchiveExtensions)
}

// Return string as lowercase with all spaces removed.
func TrimString(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
