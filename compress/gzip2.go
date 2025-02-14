package gw_compress

import (
	"bytes"
	"io"
	"strings"
	"unsafe"

	"github.com/dsnet/compress/bzip2"
	gw_errors "github.com/generalworksinc/goutil/errors"
)

func CompressString(str string) (string, error) {
	buf, err := Compress(strings.NewReader(str))
	if err != nil {
		return "", gw_errors.Wrap(err)
	} else {
		return buf.String(), nil
	}
}
func Compress(r io.Reader) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zw, err := bzip2.NewWriter(buf, &bzip2.WriterConfig{Level: bzip2.BestCompression})
	if err != nil {
		return buf, gw_errors.Wrap(err)
	}
	defer zw.Close()

	if _, err := io.Copy(zw, r); err != nil {
		return buf, gw_errors.Wrap(err)
	}
	return buf, nil
}

func ExtractString(str string) (string, error) {
	bytes := []byte{}
	reader, err := bzip2.NewReader(strings.NewReader(str), new(bzip2.ReaderConfig))
	if err != nil {
		return "", gw_errors.Wrap(err)
	}
	defer reader.Close()
	_, err = reader.Read(bytes)
	if err != nil {
		return "", gw_errors.Wrap(err)
	}
	retStr := *(*string)(unsafe.Pointer(&bytes))
	return retStr, nil
}

func Extract(zr io.Reader) (*bzip2.Reader, error) {
	return bzip2.NewReader(zr, new(bzip2.ReaderConfig))
}
