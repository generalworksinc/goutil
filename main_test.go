package main

import (
	"errors"
	"testing"

	gw_errors "github.com/generalworksinc/goutil/errors"
	"github.com/morikuni/failure/v2"
)

func TestErrorsWrap(t *testing.T) {
	// Create a base error
	baseErr := errors.New("base error")

	// Test wrapping the error
	wrappedErr := gw_errors.Wrap(baseErr, "wrapped message")
	t.Errorf("firstError: '%s', code: %v\n", wrappedErr.Error(), failure.CodeOf(wrappedErr))

	// secondError := gw_errors.Wrap(wrappedErr, "second wrapped message")
	secondError := failure.NewFailure(wrappedErr, []failure.Field{failure.WithCode(999)})
	t.Errorf("secondError: '%s', code: %v\n", secondError.Error(), failure.CodeOf(secondError))

	secondError2 := failure.NewFailure(secondError, []failure.Field{failure.WithCode(455)})
	t.Errorf("secondError: '%s', code: %v\n", secondError2.Error(), failure.CodeOf(secondError2))

	//thirdError with no message
	thirdError := gw_errors.Wrap(secondError2, "ぽむ")
	t.Errorf("thirdError: '%v', code: %v\n", thirdError.Error(), failure.CodeOf(thirdError))

	//Code1のみを除外して、再度chainを構築する
	tailError := thirdError
	var fl failure.Failure
	for {
		fl, tailError = failure.UnwrapFailure(tailError)
		t.Errorf("=========== %v,,,,,,,,,,,,%v, code:%v", fl, tailError, failure.CodeOf(tailError))
		if tailError == nil {
			break
		}
	}
	// return newErr
}
