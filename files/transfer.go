package gw_files

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	gw_common "github.com/generalworksinc/goutil/common"
	gw_errors "github.com/generalworksinc/goutil/errors"
)

// 指定したURLからファイルをダウンロードする関数
// downloadPath: default current directory
func DownloadFile(urlStr string, downloadPath *string, fileName *string) (string, error) {
	// HTTP GETリクエストを送信します
	resp, err := http.Get(urlStr)
	if err != nil {
		return "", gw_errors.Wrap(err)
	}
	defer resp.Body.Close()

	// レスポンスが成功したことを確認します
	if resp.StatusCode != http.StatusOK {
		return "", gw_errors.New(fmt.Sprintf("サーバーからエラー応答がありました: %v", resp.Status))
	}

	// ファイルを作成します
	var f string
	if fileName == nil {
		parsedUrl, err := url.Parse(urlStr)
		if err != nil {
			return "", gw_errors.Wrap(err)
		}
		// パス部分を取得し、ファイル名を抽出
		fileName = gw_common.Pointer(path.Base(parsedUrl.Path))
	} else {
		f = *fileName
	}
	if downloadPath == nil {
	} else {
		f = path.Join(*downloadPath, f)
	}
	out, err := os.Create(f)
	if err != nil {
		return "", gw_errors.Wrap(err)
	}
	defer out.Close()

	// レスポンスの内容をファイルに書き込みます
	_, err = io.Copy(out, resp.Body)
	return f, gw_errors.Wrap(err)
}
