package gw_web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"

	gw_errors "github.com/generalworksinc/goutil/errors"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/exp/utf8string"
)

type ErrorData struct {
	Message    string
	StackTrace string
	Body       string
	Code       int
	Error      string
	Url        string
	Method     string
	Protocol   string
	Ip         string
	Ua         string
	UserId     string
	Version    string
	FullString string
}

func CustomHTTPErrorHandler(version string, getUserIdFunc func(ctx *WebCtx) string) func(ctx *WebCtx, err error) error {
	return func(ctx *WebCtx, err error) error {
		defer func() {
			err := recover()
			if err != nil {
				log.Println("panic occured in CustomHTTPErrorHandler.")
				log.Println("Recover!:", err)
			}
		}()

		settedCode := ctx.StatusCode()
		log.Println("settedCode:", settedCode)

		code := http.StatusInternalServerError
		message := "error has occured"
		log.Println("error message:", err.Error())

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		}
		// コードがセットされていないか、デフォルト（正常200）の場合、新たなコードをセットする
		if settedCode == 200 || settedCode == 0 {
			ctx.Status(code)
		}

		//stacktraceは、failureで事前セットされていたら、それを取得、そうでなければrecover時に取得。
		stackTrace, ok := gw_errors.CallStackOf(err)
		if !ok {
			for depth := 0; ; depth++ {
				pc, src, line, ok := runtime.Caller(depth)
				if !ok || depth > 30 { //３０行までしかStacktrace表示しない
					break
				}
				stackTrace += fmt.Sprintf(" -> %d: %s: %s(%d)\n", depth, runtime.FuncForPC(pc).Name(), src, line)
			}
		}

		// userId := ""
		// jsonToken, loginErr := domain.LoginAt(ctx)
		// if loginErr == nil && jsonToken != nil {
		// 	userId = jsonToken.Get("Id")
		// }
		userId := getUserIdFunc(ctx)
		log.Println("user data. ip:", ctx.IP(), "ua:", string(ctx.UserAgent()), "userId:", userId, "version", version)
		bodyStrUtf8 := utf8string.NewString(string(ctx.Body()))
		bodyStr := ""
		if bodyStrUtf8.RuneCount() > 2000 {
			bodyStr = bodyStrUtf8.Slice(0, 2000)
		} else {
			bodyStr = bodyStrUtf8.String()
		}

		errorData := ErrorData{
			Message:    message,
			StackTrace: stackTrace,
			Code:       code,
			Error:      err.Error(),
			Url:        ctx.BaseURL() + ctx.OriginalURL(),
			Method:     ctx.Method(),
			Protocol:   ctx.Protocol(),
			Ip:         ctx.IP(),
			Ua:         string(ctx.UserAgent()),
			UserId:     userId,
			Version:    version,
			Body:       bodyStr,
			// FullString: ctx.String(),
		}

		errorJson, _ := json.Marshal(errorData)
		log.Println(string(errorJson))

		if !gw_errors.CheckSentToLogger(err) {
			if errorData.Ua != "" && errorData.Url != "http:///" && errorData.Message != "Bad Request" {

				errorDataForSentry := errorData
				errorDataForSentry.StackTrace = ""
				errorDataForSentry.Error = ""

				//format json
				errorStr := ""
				errorJsonForSentry, _ := json.Marshal(errorDataForSentry)
				var formatedJsonBytes bytes.Buffer
				err := json.Indent(&formatedJsonBytes, errorJsonForSentry, "", "  ") // indentは2スペース
				if err != nil {
					log.Printf("JSONの整形に失敗しました: %v\n", err)
					errorStr = string(errorJsonForSentry)
				} else {
					errorStr = formatedJsonBytes.String()
				}
				sentry.CaptureMessage(fmt.Sprintf("error on errorhandler:: %s\n\n%s\n\n%s", errorData.Error, errorData.StackTrace, errorStr))
			}
		}

		// Return HTTP response
		ctx.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		//return ctx.Status(code).SendString(message)
		return ctx.SendString(message)
	}
}
