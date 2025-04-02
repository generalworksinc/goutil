package gw_errors

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/getsentry/sentry-go"

	//"reflect"
	"runtime"

	"github.com/morikuni/failure/v2"
)

type ErrorCode string

const (
	GenericError        ErrorCode = "GenericError" //一般的なエラー。デフォルトでこれを指定
	InternalServerError ErrorCode = "InternalServerError"
	BadRequest          ErrorCode = "BadRequest"
	NotFound            ErrorCode = "NotFound"
	Forbidden           ErrorCode = "Forbidden"
	UnknownError        ErrorCode = "UnknownError" // どれにも当てはまらないエラー

	FAILER_CODE_FLG_SENT_TO_LOGGER = 9999
)

func CheckSentToLogger(err error) bool {
	isSent := false
	tailError := err
	var fl failure.Failure
	ind := 0
	for {
		fl, tailError = failure.UnwrapFailure(tailError)
		if fl != nil && failure.CodeOf(fl) == FAILER_CODE_FLG_SENT_TO_LOGGER {
			isSent = true
			break
		}
		if tailError == nil {
			break
		}
		//failuerのネストが深い場合は、15回で打ち切る
		if ind > 15 {
			break
		}
		ind++
	}
	return isSent
}
func errorLog(err error, sendLogger bool, objList ...interface{}) error {
	// Recover from any panics during error logging
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in errorLog: %v", r)
		}
	}()

	//error message, relational data
	errorMessageList := []string{"err: " + err.Error()}
	for ind, obj := range objList {
		objStr := "nil"
		if obj != nil {
			// Safely convert object to string
			func() {
				defer func() {
					if r := recover(); r != nil {
						objStr = fmt.Sprintf("[Error converting object to string: %v]", r)
					}
				}()
				objStr = fmt.Sprintf("%v", obj)
			}()
		}
		relationalStr := fmt.Sprintf("error relational data %v: %v\n", ind, objStr)
		errorMessageList = append(errorMessageList, relationalStr)
	}
	//stacktrace
	stackTraceStr, ok := CallStackOf(err)
	if !ok {
		pc, fileName, line, ok := runtime.Caller(2)

		if ok {
			stackTraceStr = fmt.Sprintf("memory address: %v, file: %v, line: %v \n", pc, fileName, line)
		} else {
			stackTraceStr = "can't get line data \n"
		}
	}
	errorMessageList = append(errorMessageList, "", stackTraceStr)

	//logging & send to sentry server
	log.Println(strings.Join(errorMessageList, "\n"))
	if sendLogger && !CheckSentToLogger(err) {
		// Safely send to Sentry
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Failed to send error to Sentry: %v", r)
				}
			}()
			log.Println("sentry.CaptureMessage on errorLog start!")
			sentry.CaptureMessage(strings.Join(errorMessageList, "\n"))
			log.Println("sentry.CaptureMessage on errorLog end!")
		}()
		err = LoggerSentFlagOn(err)
	}
	return err
}
func LoggerSentFlagOn(err error) error {
	return failure.NewFailure(err, []failure.Field{failure.WithCode(FAILER_CODE_FLG_SENT_TO_LOGGER)})
}

func New(errStr string) error {
	return failure.New(GenericError, failure.Message(errStr))
}
func Errorf(format string, a ...interface{}) error {
	return Wrap(fmt.Errorf(format, a...))
}
func HasStack(err error) (failure.CallStack, bool) {
	stack := failure.CallStackOf(err)
	if stack == nil || len(stack.Frames()) == 0 {
		return nil, false
	} else {
		return stack, true
	}
}
func CallStackOf(err error) (stackTrace string, ok bool) {
	if stack, ok := HasStack(err); ok {
		out := &bytes.Buffer{}
		for _, f := range stack.Frames() {
			p := f.Path()
			fmt.Fprintf(out, "%s:%d [%s.%s]\n", p, f.Line(), f.Pkg(), f.Func())
		}
		return out.String(), ok
	} else {
		return "", false
	}
}
func Wrap(err error, objList ...interface{}) error {
	if err == nil {
		return err
	}

	failerErrorCode := failure.CodeOf(err)
	// failer.WithCode(err, failerErrorCode)
	// log.Println("failerErrorCode: ", failerErrorCode)

	var failureCtx *failure.Context
	objStrList := []string{}
	for _, obj := range objList {
		// relationalStr := fmt.Sprintf("error relational data %v: %v\n", ind, obj)
		objStr := "nil"
		if obj != nil {
			// 安全な文字列変換のためのブロック
			func() {
				defer func() {
					if r := recover(); r != nil {
						objStr = fmt.Sprintf("[Error converting object to string: %v]", r)
					}
				}()
				objStr = fmt.Sprintf("%v", obj)
			}()
		}
		objStrList = append(objStrList, objStr)
	}
	if len(objStrList) > 0 {
		failureCtx = &failure.Context{"params": fmt.Sprintf("%v", objStrList)}
	}

	//If not wrapped with failer yet, then create new Failer Error
	var newError error
	if failerErrorCode == nil {
		if failureCtx != nil {
			newError = failure.New(err, failureCtx)
		} else {
			newError = failure.New(err)
		}
	} else {
		//If wrapped with failer already, and has context, then add new context
		if failureCtx != nil {
			newError = failure.Wrap(err, failureCtx)
		} else {
			//If failer wrapped already, and has no context, then return the original error
			newError = err
		}
	}
	return newError
}

func CatchPanic(errPt *error, sendLogger bool) {
	var err error
	if r := recover(); r != nil {
		var ok bool
		if err, ok = r.(error); !ok {
			// Set error that will call the global error handler
			err = Errorf("%v", r)
		}
		stackTrace, ok := CallStackOf(err)
		if !ok {
			//stacktraceを出力
			for depth := 0; ; depth++ {
				pc, src, line, ok := runtime.Caller(depth)
				if !ok || depth > 30 { //３０行までしかStacktrace表示しない
					break
				}
				stackTrace += fmt.Sprintf(" -> %d: %s: %s(%d)\n", depth, runtime.FuncForPC(pc).Name(), src, line)
			}
		}
		log.Println("panic capture. message:" + fmt.Sprintf("%v", r) + "\n\n" + stackTrace)
		if sendLogger && !CheckSentToLogger(err) {
			//sentryに送信
			log.Println("sentry.CaptureMessage on CatchPanic start!")
			sentry.CaptureMessage("panic capture. message:" + fmt.Sprintf("%v", r) + "\n\n" + stackTrace)
			log.Println("sentry.CaptureMessage on CatchPanic end!")
			err = LoggerSentFlagOn(err)
		}
		*errPt = Wrap(err)
	}
}
func ReturnError(err error, objList ...interface{}) error {
	if err != nil {
		err = errorLog(err, false, objList...)
	}
	return Wrap(err, objList...)
}
func ReturnErrorStr(errStr string) error {
	if errStr != "" {
		err := New(errStr)
		err = errorLog(err, false)
		return err
	}
	return nil
}

func PanicError(err error, objList ...interface{}) {
	if err != nil {
		err = Wrap(err, objList...)
		err = errorLog(err, false, objList...)
		panic(err)
	}
}
func PanicErrorStr(errStr string, objList ...interface{}) {
	if errStr != "" {
		err := New(errStr)
		err = errorLog(err, false, objList...)
		panic(err)
	}
}
func PanicErrorWithFunc(err error, f func(), objList ...interface{}) {
	if err != nil {
		err = Wrap(err, objList...)
		err = errorLog(err, false, objList...)
		//c.Status(status)
		f()
		panic(err)
	}
}

func PrintError(err error, objList ...interface{}) {
	if err != nil {
		err = Wrap(err, objList...)
		_ = errorLog(err, true, objList)
	}
}
func PrintErrorStr(errStr string, objList ...interface{}) {
	if errStr != "" {
		err := New(errStr)
		_ = errorLog(err, true, objList...)
	}
}
