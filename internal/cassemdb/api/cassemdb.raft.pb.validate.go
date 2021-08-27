// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: cassemdb.raft.proto

package api

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
)

// Validate checks the field values on LogEntry with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *LogEntry) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for Action

	if v, ok := interface{}(m.GetCommand()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return LogEntryValidationError{
				field:  "Command",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	return nil
}

// LogEntryValidationError is the validation error returned by
// LogEntry.Validate if the designated constraints aren't met.
type LogEntryValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e LogEntryValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e LogEntryValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e LogEntryValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e LogEntryValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e LogEntryValidationError) ErrorName() string { return "LogEntryValidationError" }

// Error satisfies the builtin error interface
func (e LogEntryValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sLogEntry.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = LogEntryValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = LogEntryValidationError{}

// Validate checks the field values on SetCommand with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *SetCommand) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for DeleteKey

	// no validation rules for IsDir

	// no validation rules for SetKey

	// no validation rules for Value

	return nil
}

// SetCommandValidationError is the validation error returned by
// SetCommand.Validate if the designated constraints aren't met.
type SetCommandValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e SetCommandValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e SetCommandValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e SetCommandValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e SetCommandValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e SetCommandValidationError) ErrorName() string { return "SetCommandValidationError" }

// Error satisfies the builtin error interface
func (e SetCommandValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sSetCommand.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = SetCommandValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = SetCommandValidationError{}

// Validate checks the field values on ChangeCommand with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *ChangeCommand) Validate() error {
	if m == nil {
		return nil
	}

	if v, ok := interface{}(m.GetChange()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ChangeCommandValidationError{
				field:  "Change",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	return nil
}

// ChangeCommandValidationError is the validation error returned by
// ChangeCommand.Validate if the designated constraints aren't met.
type ChangeCommandValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ChangeCommandValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ChangeCommandValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ChangeCommandValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ChangeCommandValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ChangeCommandValidationError) ErrorName() string { return "ChangeCommandValidationError" }

// Error satisfies the builtin error interface
func (e ChangeCommandValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sChangeCommand.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ChangeCommandValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ChangeCommandValidationError{}
