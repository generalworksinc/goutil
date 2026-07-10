package gw_web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"

	gw_errors "github.com/generalworksinc/goutil/errors"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v3"
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
		reqCtx := ctx.Context()
		defer func() {
			r := recover()
			if r != nil {
				slog.ErrorContext(reqCtx, "panic occurred in CustomHTTPErrorHandler", slog.Any("recover", r))
			}
		}()

		settedCode := ctx.StatusCode()

		code := http.StatusInternalServerError
		message := "error has occured"

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

		userId := getUserIdFunc(ctx)
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

		// 5xx は ERROR、4xx 等は WARN で構造化出力（request_id は context 注入ハンドラが付与）
		level := slog.LevelWarn
		if code >= 500 {
			level = slog.LevelError
		}
		slog.LogAttrs(reqCtx, level, "request error",
			slog.Int("status", code),
			slog.String("error", errorData.Error),
			slog.String("message", message),
			slog.String("url", errorData.Url),
			slog.String("method", errorData.Method),
			slog.String("ip", errorData.Ip),
			slog.String("ua", errorData.Ua),
			slog.String("error_user_id", userId),
			slog.String("version", version),
			slog.String("stack_trace", stackTrace),
		)

		isSentToLogger := gw_errors.CheckSentToLogger(err)
		if !isSentToLogger {
			// if errorData.Ua != "" && errorData.Url != "http:///" && errorData.Message != "Bad Request" {
			if errorData.Url != "http:///" && errorData.Message != "Bad Request" {
				// 安全にSentryにメッセージを送る
				func() {
					defer func() {
						if r := recover(); r != nil {
							slog.ErrorContext(reqCtx, "failed to send error to Sentry", slog.Any("recover", r))
						}
					}()

					errorDataForSentry := errorData
					errorDataForSentry.StackTrace = ""
					errorDataForSentry.Error = ""

					//format json
					errorStr := ""
					errorJsonForSentry, _ := json.Marshal(errorDataForSentry)
					var formatedJsonBytes bytes.Buffer
					jsonErr := json.Indent(&formatedJsonBytes, errorJsonForSentry, "", "  ") // indentは2スペース
					if jsonErr != nil {
						slog.WarnContext(reqCtx, "failed to format error JSON for Sentry", slog.String("error", jsonErr.Error()))
						errorStr = string(errorJsonForSentry)
					} else {
						errorStr = formatedJsonBytes.String()
					}

					// エラーメッセージを安全に作成
					sentryMsg := ""
					func() {
						defer func() {
							if r := recover(); r != nil {
								sentryMsg = "Error creating Sentry message"
								slog.ErrorContext(reqCtx, "failed to format Sentry message", slog.Any("recover", r))
							}
						}()
						sentryMsg = fmt.Sprintf("error on errorhandler:: %s\n\n%s\n\n%s", errorData.Error, errorData.StackTrace, errorStr)
					}()

					sentry.CaptureMessage(sentryMsg)
				}()
			}
		}

		// Return HTTP response
		ctx.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		//return ctx.Status(code).SendString(message)
		return ctx.SendString(message)
	}
}
