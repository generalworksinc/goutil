package gw_files

import (
	"bytes"
	"encoding/base64"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	gw_crypto "github.com/generalworksinc/goutil/crypto"
	gw_errors "github.com/generalworksinc/goutil/errors"
)

func ConvFileToByte(uri string, data string, isEncrypt bool, encryptKey []byte) (io.Reader, string, error) {
	var imageBody io.Reader
	prefix := ""
	fileContentType := ""
	if uri != "" {
		response, err := http.Get(uri)
		if err != nil {
			return nil, "", gw_errors.Wrap(err)
		}
		defer response.Body.Close()
		fileContentType = response.Header.Get("Content-Type")

		if isEncrypt {
			imageBodyBytes, err := io.ReadAll(response.Body)
			if err != nil {
				return nil, "", gw_errors.Wrap(err)
			}
			ciphertext, err := gw_crypto.EncryptAESGCM(encryptKey, imageBodyBytes)
			if err != nil {
				return nil, "", gw_errors.Wrap(err)
			}
			imageBody = bytes.NewReader(ciphertext)
		} else {
			imageBody = response.Body
		}

	} else if data != "" && strings.Index(data, "data:") == 0 {
		data64Ind := strings.Index(data, ";base64")
		if data64Ind == -1 {
			return nil, "", gw_errors.New("not contains ';base64'")
		}
		mimeType := data[len("data:"):data64Ind]
		log.Println("mimeType:", mimeType)
		ext, err := mime.ExtensionsByType(mimeType)
		if err != nil {
			return nil, "", gw_errors.Wrap(err)
		}

		imageBodyBytes, err := base64.StdEncoding.DecodeString(data[strings.Index(data, ";base64,")+8:])
		if err != nil {
			return nil, "", gw_errors.Wrap(err)
		}

		if isEncrypt {
			//暗号化時間を図ってlogに出力
			start := time.Now()
			ciphertext, err := gw_crypto.EncryptAESGCM(encryptKey, imageBodyBytes)
			log.Println("EncryptAESGCM time:", time.Since(start))
			if err != nil {
				return nil, "", gw_errors.Wrap(err)
			}

			imageBody = bytes.NewReader(ciphertext)
		} else {
			imageBody = bytes.NewReader(imageBodyBytes)
		}

		if len(ext) > 0 {
			prefix = ext[0]
			//bug対応(なぜかjpegのprefixがjpe、が設定されている場合があるため)
			if prefix == ".jpe" {
				prefix = ".jpg"
			}
		}
	}

	uriTmp := uri

	if prefix == "" {
		queryIndex := strings.LastIndex(uri, "?")
		if queryIndex >= 0 {
			uriTmp = uriTmp[:queryIndex]
		}
		slashIndex := strings.LastIndex(uri, "/")
		if slashIndex >= 0 {
			uriTmp = uriTmp[slashIndex+1:]
		}
		dotIndex := strings.LastIndex(uriTmp, ".")
		if dotIndex >= 0 {
			prefix = uriTmp[dotIndex:]
		}
	}
	if prefix == "" {
		if fileContentType == "image/jpeg" {
			prefix = ".jpg"
		} else if fileContentType == "image/png" {
			prefix = ".png"
		} else if fileContentType == "image/gif" {
			prefix = ".gif"
		} else if fileContentType == "image/x-icon" {
			prefix = ".ico"
		} else if fileContentType == "image/webp" {
			prefix = ".webp"
		} else if fileContentType == "image/bmp" {
			prefix = ".bmp"
		}
	}

	return imageBody, prefix, nil
}
