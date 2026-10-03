package core

import (
	"errors"
	"fmt"
	"strings"
)

// ExitCode is the v1.0 exit contract.
type ExitCode int

const (
	ExitOK           ExitCode = 0
	ExitFailure      ExitCode = 1 // validation / operational failure
	ExitIncompatible ExitCode = 2 // CLI/Bootstrap incompatibility
	ExitUsage        ExitCode = 3 // invalid invocation
	ExitCancelled    ExitCode = 4 // user cancellation
	ExitConflict     ExitCode = 5 // project state conflict
	ExitIntegrity    ExitCode = 6 // Bootstrap integrity failure
	ExitUnavailable  ExitCode = 7 // external dependency unavailable
)

// Error is a user-facing error: what happened, why, and what to do next.
type Error struct {
	Code       ExitCode
	What       string
	Why        string
	Next       string
	Unmodified bool // true when the project is known to be untouched
	Err        error
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString(e.What)
	if e.Why != "" {
		b.WriteString("\n\nReason:\n" + e.Why)
	}
	if e.Unmodified {
		b.WriteString("\n\nNo project files were modified.")
	}
	if e.Next != "" {
		b.WriteString("\n\nNext step:\n" + e.Next)
	}
	return b.String()
}

func (e *Error) Unwrap() error { return e.Err }

// Errorf builds an Error with only a code and a message.
func Errorf(code ExitCode, format string, args ...any) *Error {
	return &Error{Code: code, What: fmt.Sprintf(format, args...)}
}

// CodeOf maps any error to its exit code.
func CodeOf(err error) ExitCode {
	if err == nil {
		return ExitOK
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ExitFailure
}

// ErrCancelled is returned when the human declines.
var ErrCancelled = &Error{Code: ExitCancelled, What: "Cancelled."}
