package appregistry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
)

type blobDecoder interface {
	// Decode decodes package blob into plain unencrypted byte array
	Decode(encoded []byte) ([]byte, error)
}

type blobDecoderImpl struct {
}

func (*blobDecoderImpl) Decode(encoded []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewBuffer(encoded))
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	decoded, err := extractManifest(gzipReader)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func extractManifest(r io.Reader) ([]byte, error) {
	reader := tar.NewReader(r)

	writer := &bytes.Buffer{}
	for true {
		header, err := reader.Next()
		if err != nil && err != io.EOF {
			return nil, errors.New(fmt.Sprintf("extraction of tar ball failed - %s", err.Error()))
		}

		if err == io.EOF {
			break
		}

		switch header.Typeflag {
		case tar.TypeReg:
			io.Copy(writer, reader)
			break
		}
	}

	return writer.Bytes(), nil
}
