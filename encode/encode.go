package gw_encode

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"strings"

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
	return err
}
func Utf8ToSjis(str string) (string, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(strings.NewReader(str), japanese.ShiftJIS.NewEncoder()))
	if err != nil {
		return "", err
	}
	return string(ret), err
}

// ShiftJIS から UTF-8
func SjisToUtf8(str string) (string, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(strings.NewReader(str), japanese.ShiftJIS.NewDecoder()))
	if err != nil {
		return "", err
	}
	return string(ret), err
}

// UTF-8 から EUC-JP
func Utf8ToEucjp(str string) (string, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(strings.NewReader(str), japanese.EUCJP.NewEncoder()))
	if err != nil {
		return "", err
	}
	return string(ret), err
}

// EUC-JP から UTF-8
func EucjpToUtf8(str string) (string, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(strings.NewReader(str), japanese.EUCJP.NewDecoder()))
	if err != nil {
		return "", err
	}
	return string(ret), err
}

func Utf8ByteToSjisByte(str []byte) ([]byte, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(bytes.NewReader(str), japanese.ShiftJIS.NewEncoder()))
	if err != nil {
		return []byte{}, err
	}
	return ret, err
}

// ShiftJIS から UTF-8
func SjisByteToUtf8Byte(str []byte) ([]byte, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(bytes.NewReader(str), japanese.ShiftJIS.NewDecoder()))
	if err != nil {
		return []byte{}, err
	}
	return ret, err
}

// UTF-8 から EUC-JP
func Utf8ByteToEucjpByte(str []byte) ([]byte, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(bytes.NewReader(str), japanese.EUCJP.NewEncoder()))
	if err != nil {
		return []byte{}, err
	}
	return ret, err
}

// EUC-JP から UTF-8
func EucjpByteToUtf8Byte(str []byte) ([]byte, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(bytes.NewReader(str), japanese.EUCJP.NewDecoder()))
	if err != nil {
		return []byte{}, err
	}
	return ret, err
}

func GenerateAESKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256のために32バイトのキーを作成
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}
