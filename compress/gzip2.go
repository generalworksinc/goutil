package compress

import (
	"bytes"
	"github.com/dsnet/compress/bzip2"
	"io"
	"strings"
	"unsafe"
)

func CompressString(str string) (string, error) {
	buf, err := Compress(strings.NewReader(str))
	if err != nil {
		return "", err
	} else {
		return buf.String(), nil
	}
}
func Compress(r io.Reader) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zw, err := bzip2.NewWriter(buf, &bzip2.WriterConfig{Level: bzip2.BestCompression})
	if err != nil {
		return buf, err
	}
	defer zw.Close()

	if _, err := io.Copy(zw, r); err != nil {
		return buf, err
	}
	return buf, nil
}

func ExtractString(str string) (string, error) {
	bytes := []byte{}
	reader, err := bzip2.NewReader(strings.NewReader(str), new(bzip2.ReaderConfig))
	defer reader.Close()
	if err != nil {
		return "", err
	} else {
		reader.Read(bytes)
		retStr := *(*string)(unsafe.Pointer(&bytes))
		return retStr, nil
	}
}

func Extract(zr io.Reader) (io.Reader, error) {
	return bzip2.NewReader(zr, new(bzip2.ReaderConfig))
}
