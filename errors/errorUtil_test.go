package gw_errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/morikuni/failure/v2"
	"github.com/stretchr/testify/assert"
)

// Mock Sentry Transport for testing
type mockTransport struct {
	capturedMessages []string
	capturedEvents   []*sentry.Event
}

func (t *mockTransport) Configure(options sentry.ClientOptions) {}
func (t *mockTransport) SendEvent(event *sentry.Event) {
	t.capturedEvents = append(t.capturedEvents, event)
	if event.Message != "" {
		t.capturedMessages = append(t.capturedMessages, event.Message)
	}
}
func (t *mockTransport) Flush(timeout time.Duration) bool {
	return true
}

// Close implements sentry.Transport. It's a no-op for the mock.
func (t *mockTransport) Close() {
	// No-op for mock
}
func (t *mockTransport) Events() []*sentry.Event {
	return t.capturedEvents
}
func (t *mockTransport) Messages() []string {
	return t.capturedMessages
}
func (t *mockTransport) Reset() {
	t.capturedMessages = []string{}
	t.capturedEvents = []*sentry.Event{}
}

var transport *mockTransport

func setupSentryTest() {
	transport = &mockTransport{}
	sentry.Init(sentry.ClientOptions{
		Transport: transport,
		// Set a dummy DSN to avoid errors, or configure based on your needs
		// Dsn: "YOUR_SENTRY_DSN",
	})
}

func TestErrorLogSendLoggerFalse(t *testing.T) {
	setupSentryTest()
	err := New("test error for sendLogger=false")
	_ = errorLog(err, false) // sendLogger = false

	assert.Empty(t, transport.Messages(), "Sentry should not capture message when sendLogger is false")
}

func TestErrorLogSendLoggerTrue(t *testing.T) {
	setupSentryTest()
	err := New("test error for sendLogger=true")
	_ = errorLog(err, true) // sendLogger = true

	assert.NotEmpty(t, transport.Messages(), "Sentry should capture message when sendLogger is true")
	assert.True(t, strings.Contains(transport.Messages()[0], "test error for sendLogger=true"), "Sentry message content mismatch")
}

func TestReturnErrorNoSentry(t *testing.T) {
	setupSentryTest()
	originalErr := errors.New("original error for ReturnError")
	err := Wrap(originalErr)

	assert.Error(t, err)
	assert.Empty(t, transport.Messages(), "Sentry should not capture message from ReturnError")
	// Check if there is a trace of the original error
	assert.Contains(t, err.Error(), "original error for ReturnError", "ReturnError should include the original error message")
}

func TestReturnErrorStrNoSentry(t *testing.T) {
	setupSentryTest()
	err := New("error string for ReturnErrorStr")

	assert.Error(t, err)
	assert.Empty(t, transport.Messages(), "Sentry should not capture message from ReturnErrorStr")
	assert.True(t, strings.Contains(err.Error(), "error string for ReturnErrorStr"), "ReturnErrorStr message content mismatch")
}

func TestPrintErrorSendsSentry(t *testing.T) {
	setupSentryTest()
	originalErr := errors.New("original error for PrintError")
	PrintError(originalErr)

	assert.NotEmpty(t, transport.Messages(), "Sentry should capture message from PrintError")
	assert.True(t, strings.Contains(transport.Messages()[0], "original error for PrintError"), "PrintError Sentry message content mismatch")
}

func TestPrintErrorStrSendsSentry(t *testing.T) {
	setupSentryTest()
	PrintErrorStr("error string for PrintErrorStr")

	assert.NotEmpty(t, transport.Messages(), "Sentry should capture message from PrintErrorStr")
	assert.True(t, strings.Contains(transport.Messages()[0], "error string for PrintErrorStr"), "PrintErrorStr Sentry message content mismatch")
}

func TestLoggerSentFlag(t *testing.T) {
	setupSentryTest()
	err := New("test error for flag")
	assert.False(t, CheckSentToLogger(err), "Error should not have sent flag initially")

	// Simulate sending to logger
	err = errorLog(err, true)
	assert.True(t, CheckSentToLogger(err), "Error should have sent flag after errorLog(true)")

	// Reset and test LoggerSentFlagOn directly
	err = New("another test error")
	err = LoggerSentFlagOn(err)
	assert.True(t, CheckSentToLogger(err), "Error should have sent flag after LoggerSentFlagOn")
}

func TestDoubleSendPrevention(t *testing.T) {
	setupSentryTest()
	err := New("test error for double send")

	// First send
	err = errorLog(err, true)
	assert.Len(t, transport.Messages(), 1, "Sentry should capture the first message")
	assert.True(t, CheckSentToLogger(err), "Flag should be set after first send")

	// Attempt second send
	err = errorLog(err, true)
	assert.Len(t, transport.Messages(), 1, "Sentry should not capture the second message due to flag")

	// Test with PrintError
	transport.Reset()
	err = New("test error for PrintError double send")
	PrintError(err) // First send via PrintError

	// Since PrintError doesn't return the error, we need to simulate what it does
	// by manually applying the flag to our err variable
	err = LoggerSentFlagOn(err)

	assert.Len(t, transport.Messages(), 1, "Sentry should capture the first PrintError message")
	assert.True(t, CheckSentToLogger(err), "Flag should be set after first PrintError")

	PrintError(err) // Second send via PrintError
	assert.Len(t, transport.Messages(), 1, "Sentry should not capture the second PrintError message")
}

func TestCatchPanicSendsSentry(t *testing.T) {
	setupSentryTest()
	var err error
	func() {
		defer CatchPanic(&err, true) // sendLogger = true
		panic("simulated panic")
	}()

	assert.Error(t, err, "Error should be set after panic")
	assert.NotEmpty(t, transport.Messages(), "Sentry should capture message from CatchPanic")
	assert.True(t, strings.Contains(transport.Messages()[0], "panic capture"), "CatchPanic Sentry message should contain 'panic capture'")
	assert.True(t, strings.Contains(transport.Messages()[0], "simulated panic"), "CatchPanic Sentry message should contain panic message")
	assert.True(t, CheckSentToLogger(err), "Error from CatchPanic should have sent flag")
}

func TestCatchPanicNoSend(t *testing.T) {
	setupSentryTest()
	var err error
	func() {
		defer CatchPanic(&err, false) // sendLogger = false
		panic("simulated panic no send")
	}()

	assert.Error(t, err, "Error should be set after panic")
	assert.Empty(t, transport.Messages(), "Sentry should not capture message when sendLogger is false in CatchPanic")
	assert.False(t, CheckSentToLogger(err), "Error from CatchPanic(false) should not have sent flag")
}

func TestWrapNilError(t *testing.T) {
	err := Wrap(nil)
	assert.Nil(t, err, "Wrapping nil should return nil")
}

func TestWrapBasicError(t *testing.T) {
	originalErr := errors.New("basic error")
	wrappedErr := Wrap(originalErr)

	assert.Error(t, wrappedErr)
	// Check if it's a failure error
	_, ok := wrappedErr.(failure.Failure)
	assert.True(t, ok, "Wrapped error should be a failure.Failure")

	// Check if original error message is preserved
	assert.Contains(t, wrappedErr.Error(), "basic error", "Wrapped error should contain the original error message")
}

func TestWrapWithErrorContext(t *testing.T) {
	originalErr := errors.New("error with context")
	param1 := "value1"
	param2 := 123
	wrappedErr := Wrap(originalErr, param1, param2)

	assert.Error(t, wrappedErr)
	_, ok := wrappedErr.(failure.Failure)
	assert.True(t, ok, "Wrapped error should be a failure.Failure")

	// Check if original error message is preserved
	assert.Contains(t, wrappedErr.Error(), "error with context", "Wrapped error should contain the original error message")

	// Check context (Note: failure library doesn't provide easy context retrieval, check message)
	errMsg := fmt.Sprintf("%v", wrappedErr) // failure includes context in its string representation
	assert.Contains(t, errMsg, fmt.Sprintf("%v", param1), "Wrapped error message should contain context param1")
	assert.Contains(t, errMsg, fmt.Sprintf("%d", param2), "Wrapped error message should contain context param2")
}

func TestWrapAlreadyFailureError(t *testing.T) {
	originalErr := failure.New(GenericError, failure.Message("already failure"))
	wrappedErr := Wrap(originalErr) // No context added

	assert.Error(t, wrappedErr)
	// Should return the original failure error instance if no context is added
	assert.Equal(t, originalErr, wrappedErr, "Wrapping a failure error without context should return the original")
}

func TestWrapAlreadyFailureErrorWithContext(t *testing.T) {
	originalErr := failure.New(GenericError, failure.Message("already failure with context"))
	param1 := "new context"
	wrappedErr := Wrap(originalErr, param1) // Add context

	assert.Error(t, wrappedErr)
	_, ok := wrappedErr.(failure.Failure)
	assert.True(t, ok, "Wrapped error should still be a failure.Failure")
	assert.NotEqual(t, originalErr, wrappedErr, "Wrapping a failure error with context should return a new instance")

	// Check if the error code is preserved
	assert.Equal(t, failure.CodeOf(originalErr), failure.CodeOf(wrappedErr), "Wrapped error should preserve the original error code")

	// Check context
	errMsg := fmt.Sprintf("%v", wrappedErr)
	assert.Contains(t, errMsg, fmt.Sprintf("%v", param1), "Wrapped error message should contain the new context")
	assert.Contains(t, errMsg, "already failure with context", "Wrapped error message should contain the original message")
}

func TestPanicError(t *testing.T) {
	setupSentryTest()
	originalErr := errors.New("error to panic") // Missing variable declaration
	defer func() {
		r := recover()
		assert.NotNil(t, r, "PanicError should cause a panic")
		recoveredErr, ok := r.(error)
		assert.True(t, ok, "Recovered value should be an error")

		// 内容をチェック
		assert.Contains(t, recoveredErr.Error(), "error to panic", "Panic should contain the original error message")

		// PanicError calls errorLog with sendLogger=false
		assert.Empty(t, transport.Messages(), "Sentry should not capture message from PanicError's internal errorLog call")
		// Check if the error was wrapped before panic
		_, failureOk := recoveredErr.(failure.Failure)
		assert.True(t, failureOk, "Error should be wrapped by failure before panic")
	}()
	PanicError(originalErr)
}

func TestPanicErrorStr(t *testing.T) {
	setupSentryTest()
	errMsg := "string error to panic"
	defer func() {
		r := recover()
		assert.NotNil(t, r, "PanicErrorStr should cause a panic")
		recoveredErr, ok := r.(error)
		assert.True(t, ok, "Recovered value should be an error")
		assert.True(t, strings.Contains(recoveredErr.Error(), errMsg), "Panic should contain the original error string")
		// PanicErrorStr calls errorLog with sendLogger=false
		assert.Empty(t, transport.Messages(), "Sentry should not capture message from PanicErrorStr's internal errorLog call")
		// Check if the error was wrapped before panic
		_, failureOk := recoveredErr.(failure.Failure)
		assert.True(t, failureOk, "Error should be wrapped by failure before panic")
	}()
	PanicErrorStr(errMsg)
}

// Helper function to introduce a delay for Sentry flushing if needed in real scenarios
// import "time"
// func waitForSentryFlush() {
// 	sentry.Flush(2 * time.Second)
// }
