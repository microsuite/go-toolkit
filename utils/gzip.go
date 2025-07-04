package utils

import (
	"bytes"
	"compress/gzip"
	"io"
)

func Decompress(buf []byte) ([]byte, error) {
	var body []byte

	gzr, err := gzip.NewReader(bytes.NewBuffer(buf))
	if err != nil {
		return []byte{}, err
	}
	defer gzr.Close()

	body, err = io.ReadAll(gzr)
	if err != nil {
		return []byte{}, err
	}
	return body, nil
}

func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)

	if _, err := gzw.Write(data); err != nil {
		return nil, err
	}

	if err := gzw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
