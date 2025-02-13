package gw_files

import (
	"bytes"

	gw_errors "github.com/generalworksinc/goutil/errors"
	"github.com/yeka/zip"
)

func CreateZipBuffer(fileName string, content []byte, password string) (*bytes.Buffer, error) {
	//filename := "sample.txt"
	//content := "ファイル内容"
	//password := "long-long-password"

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Create の代わりに Encrypt を使う
	f, err := w.Encrypt(fileName, password, zip.AES256Encryption)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}

	_, err = f.Write(content)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}

	err = w.Close()
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}

	return buf, nil
}
