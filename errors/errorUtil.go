package gw_errors

import (
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"log"
	"strings"
	//"reflect"
	"runtime"
)

func errorLog(err error, objList ...interface{}) {
	//error message, relational data
	errorMessageList := []string{"err: " + err.Error()}
	for ind, obj := range objList {
		relationalStr := fmt.Sprintf("error relational data %v: %v\n", ind, obj)
		errorMessageList = append(errorMessageList, relationalStr)
	}
	//stacktrace
	pc, fileName, line, _ := runtime.Caller(2)
	stackTraceStr := fmt.Sprintf("memory address: %v, file: %v, line: %v \n", pc, fileName, line)
	errorMessageList = append(errorMessageList, "", stackTraceStr)

	//logging & send to sentry server
	log.Println(strings.Join(errorMessageList, "\n"))
	sentry.CaptureMessage(strings.Join(errorMessageList, "\n"))
}

func ReturnError(err error, objList ...interface{}) error {
	errorLog(err, objList)
	return err
}
func ReturnErrorStr(errStr string) error {
	err := errors.New(errStr)
	errorLog(err)
	return err
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
