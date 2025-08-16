package gw_encode

import (
	"bytes"
	"io"
	"strings"

	gw_errors "github.com/generalworksinc/goutil/errors"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// Conversion
func Conversion(inStream io.Reader, outStream io.Writer) error {
	//reader from stream (Shift-JIS to UTF-8)
	reader := transform.NewReader(inStream, japanese.ShiftJIS.NewDecoder())

	//writer to stream (UTF-8 to EUC-JP)
	writer := transform.NewWriter(outStream, japanese.EUCJP.NewEncoder())

	//Copy
	_, err := io.Copy(writer, reader)
	return gw_errors.Wrap(err)
}
func Utf8ToSjis(str string) (string, error) {
	ret, err := io.ReadAll(transform.NewReader(strings.NewReader(str), japanese.ShiftJIS.NewEncoder()))
	if err != nil {
		return "", gw_errors.Wrap(err)
	}
	return string(ret), nil
}

// ShiftJIS から UTF-8
func SjisToUtf8(str string) (string, error) {
	ret, err := io.ReadAll(transform.NewReader(strings.NewReader(str), japanese.ShiftJIS.NewDecoder()))
	if err != nil {
		return "", gw_errors.Wrap(err)
	}
	return string(ret), nil
}

// UTF-8 から EUC-JP
func Utf8ToEucjp(str string) (string, error) {
	ret, err := io.ReadAll(transform.NewReader(strings.NewReader(str), japanese.EUCJP.NewEncoder()))
	if err != nil {
		return "", gw_errors.Wrap(err)
	}
	return string(ret), nil
}

// EUC-JP から UTF-8
func EucjpToUtf8(str string) (string, error) {
	ret, err := io.ReadAll(transform.NewReader(strings.NewReader(str), japanese.EUCJP.NewDecoder()))
	if err != nil {
		return "", gw_errors.Wrap(err)
	}
	return string(ret), nil
}

func Utf8ByteToSjisByte(str []byte) ([]byte, error) {
	ret, err := io.ReadAll(transform.NewReader(bytes.NewReader(str), japanese.ShiftJIS.NewEncoder()))
	if err != nil {
		return []byte{}, gw_errors.Wrap(err)
	}
	return ret, nil
}

// ShiftJIS から UTF-8
func SjisByteToUtf8Byte(str []byte) ([]byte, error) {
	ret, err := io.ReadAll(transform.NewReader(bytes.NewReader(str), japanese.ShiftJIS.NewDecoder()))
	if err != nil {
		return []byte{}, gw_errors.Wrap(err)
	}
	return ret, nil
}

// UTF-8 から EUC-JP
func Utf8ByteToEucjpByte(str []byte) ([]byte, error) {
	ret, err := io.ReadAll(transform.NewReader(bytes.NewReader(str), japanese.EUCJP.NewEncoder()))
	if err != nil {
		return []byte{}, gw_errors.Wrap(err)
	}
	return ret, nil
}

// EUC-JP から UTF-8
func EucjpByteToUtf8Byte(str []byte) ([]byte, error) {
	ret, err := io.ReadAll(transform.NewReader(bytes.NewReader(str), japanese.EUCJP.NewDecoder()))
	if err != nil {
		return []byte{}, gw_errors.Wrap(err)
	}
	return ret, nil
}
