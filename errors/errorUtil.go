package gw_errors

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/getsentry/sentry-go"

	//"reflect"
	"runtime"

	pkg_errors "github.com/pkg/errors"
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
	return Info(errors.New(errStr))
}
func Errorf(format string, a ...interface{}) error {
	return Info(fmt.Errorf(format, a...))
}
func Info(err error, objList ...interface{}) error {
	errorMessageList := []string{"err: " + err.Error()}
	for ind, obj := range objList {
		relationalStr := fmt.Sprintf("error relational data %v: %v\n", ind, obj)
		errorMessageList = append(errorMessageList, relationalStr)
	}
	//stacktrace
	pc, fileName, line, ok := runtime.Caller(1)
	stackTraceStr := ""
	if ok {
		stackTraceStr = fmt.Sprintf("memory address: %v, file: %v, line: %v \n", pc, fileName, line)
	} else {
		stackTraceStr = fmt.Sprintf("can't get line data \n")
	}
	errorMessageList = append(errorMessageList, "", stackTraceStr)
	return pkg_errors.Wrap(err, strings.Join(errorMessageList, "\n"))
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
