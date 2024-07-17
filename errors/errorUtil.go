package gw_errors

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/getsentry/sentry-go"

	//"reflect"
	"runtime"

	"github.com/morikuni/failure/v2"
)

func errorLog(err error, objList ...interface{}) {
	//error message, relational data
	errorMessageList := []string{"err: " + err.Error()}
	for ind, obj := range objList {
		relationalStr := fmt.Sprintf("error relational data %v: %v\n", ind, obj)
		errorMessageList = append(errorMessageList, relationalStr)
	}
	//stacktrace
	pc, fileName, line, ok := runtime.Caller(2)
	stackTraceStr := ""
	if ok {
		stackTraceStr = fmt.Sprintf("memory address: %v, file: %v, line: %v \n", pc, fileName, line)
	} else {
		stackTraceStr = fmt.Sprintf("can't get line data \n")
	}
	errorMessageList = append(errorMessageList, "", stackTraceStr)

	//logging & send to sentry server
	log.Println(strings.Join(errorMessageList, "\n"))
	sentry.CaptureMessage(strings.Join(errorMessageList, "\n"))
}

func New(errStr string) error {
	return Wrap(errors.New(errStr))
}
func Errorf(format string, a ...interface{}) error {
	return Wrap(fmt.Errorf(format, a...))
}
func CallStackOf(err error) (stackTrace string, ok bool) {
	stack := failure.CallStackOf(err)
	if stack == nil || len(stack.Frames()) == 0 {
		return "", false
	} else {
		out := &bytes.Buffer{}
		for _, f := range stack.Frames() {
			p := f.Path()
			fmt.Fprintf(out, "%s:%d [%s.%s]\n", p, f.Line(), f.Pkg(), f.Func())
		}
		return out.String(), ok
	}
}
func Wrap(err error, objList ...interface{}) error {
	const KEY_WRAP_COUNT = "wrapCnt"
	if err == nil {
		return err
	}
	//errに、すでにwrapされた回数があれば、それを取得して、+1する
	wrapCount, _ := failure.OriginValueAs[int, string](err, KEY_WRAP_COUNT)
	wrapCount += 1
	failureCtx := failure.Context{KEY_WRAP_COUNT: fmt.Sprintf("%v", wrapCount)}
	objStrList := []string{}
	for _, obj := range objList {
		// relationalStr := fmt.Sprintf("error relational data %v: %v\n", ind, obj)
		objStrList = append(objStrList, fmt.Sprintf("%v", obj))
	}
	if objStrList != nil && len(objStrList) > 0 {
		failureCtx[fmt.Sprintf("param_%v", wrapCount)] = fmt.Sprintf("%v", objStrList)
	}
	return failure.Wrap(err, failureCtx)
	//stacktrace
	// pc, fileName, line, ok := runtime.Caller(1)
	// stackTraceStr := ""
	// if ok {
	// 	stackTraceStr = fmt.Sprintf("memory address: %v, file: %v, line: %v \n", pc, fileName, line)
	// } else {
	// 	stackTraceStr = fmt.Sprintf("can't get line data \n")
	// }
	// errorMessageList = append(errorMessageList, "", stackTraceStr)
	// return pkg_errors.Wrap(err, strings.Join(errorMessageList, "\n"))
}

func ReturnError(err error, objList ...interface{}) error {
	if err != nil {
		errorLog(err, objList)
	}
	return err
}
func ReturnErrorStr(errStr string) error {
	if errStr != "" {
		err := errors.New(errStr)
		errorLog(err)
	}
	return nil
}

func PanicError(err error, objList ...interface{}) {
	if err != nil {
		errorLog(err, objList)
		panic(err)
	}
}
func PanicErrorStr(errStr string, objList ...interface{}) {
	if errStr != "" {
		err := errors.New(errStr)
		errorLog(err, objList)
		panic(err)
	}
}
func PanicErrorWithFunc(err error, f func(), objList ...interface{}) {
	if err != nil {
		errorLog(err, objList)
		//c.Status(status)
		f()
		panic(err)
	}
}

func PrintError(err error, objList ...interface{}) {
	if err != nil {
		errorLog(err, objList)
	}
}
func PrintErrorStr(errStr string, objList ...interface{}) {
	if errStr != "" {
		err := errors.New(errStr)
		errorLog(err, objList)
	}
}
