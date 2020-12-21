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
func Error(err error, objList ...interface{}) error {
	errorLog(err, objList)
	return err
}
func CheckErrorWithFunc(err error, f func(), objList ...interface{}) {
	if err != nil {
		errorLog(err, objList)
		//c.Status(status)
		f()
		panic(err)
	}
}
func CheckError(err error, objList ...interface{}) {
	if err != nil {
		errorLog(err, objList)
		panic(err)
	}
}

func ErrorPrint(err error, objList ...interface{}) {
	errorLog(err, objList)
}

func ErrorStr(errStr string) error {
	err := errors.New(errStr)
	errorLog(err)
	return err
}

func ErrorStrPrint(errStr string) {
	err := errors.New(errStr)
	errorLog(err)
}
