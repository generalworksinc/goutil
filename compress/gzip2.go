package gw_compress

import (
	"bytes"
	"io"
	"strings"

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
	// bzip2 compressed string を展開
	reader, err := bzip2.NewReader(strings.NewReader(str), new(bzip2.ReaderConfig))
	if err != nil {
		return "", gw_errors.Wrap(err)
	}
	defer reader.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return "", gw_errors.Wrap(err)
	}
	return buf.String(), nil
}

func Extract(zr io.Reader) (*bzip2.Reader, error) {
	return bzip2.NewReader(zr, new(bzip2.ReaderConfig))
}
