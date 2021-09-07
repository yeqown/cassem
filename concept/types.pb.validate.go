// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: types.proto

package concept

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

// Validate checks the field values on Element with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *Element) Validate() error {
	if m == nil {
		return nil
	}

	if v, ok := interface{}(m.GetMetadata()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ElementValidationError{
				field:  "Metadata",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for Raw

	// no validation rules for Version

	// no validation rules for Published

	return nil
}

// ElementValidationError is the validation error returned by Element.Validate
// if the designated constraints aren't met.
type ElementValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ElementValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ElementValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ElementValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ElementValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ElementValidationError) ErrorName() string { return "ElementValidationError" }

// Error satisfies the builtin error interface
func (e ElementValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sElement.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ElementValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ElementValidationError{}

// Validate checks the field values on ElementMetadata with the rules defined
// in the proto definition for this message. If any rules are violated, an
// error is returned.
func (m *ElementMetadata) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for Key

	// no validation rules for App

	// no validation rules for Env

	// no validation rules for LatestVersion

	// no validation rules for UnpublishedVersion

	// no validation rules for UsingVersion

	// no validation rules for UsingFingerprint

	// no validation rules for ContentType

	return nil
}

// ElementMetadataValidationError is the validation error returned by
// ElementMetadata.Validate if the designated constraints aren't met.
type ElementMetadataValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ElementMetadataValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ElementMetadataValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ElementMetadataValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ElementMetadataValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ElementMetadataValidationError) ErrorName() string { return "ElementMetadataValidationError" }

// Error satisfies the builtin error interface
func (e ElementMetadataValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sElementMetadata.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ElementMetadataValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ElementMetadataValidationError{}

// Validate checks the field values on AppMetadata with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *AppMetadata) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for Id

	// no validation rules for Description

	// no validation rules for CreatedAt

	// no validation rules for Creator

	// no validation rules for Owner

	// no validation rules for Status

	// no validation rules for Secrets

	return nil
}

// AppMetadataValidationError is the validation error returned by
// AppMetadata.Validate if the designated constraints aren't met.
type AppMetadataValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AppMetadataValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AppMetadataValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AppMetadataValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AppMetadataValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AppMetadataValidationError) ErrorName() string { return "AppMetadataValidationError" }

// Error satisfies the builtin error interface
func (e AppMetadataValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAppMetadata.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AppMetadataValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AppMetadataValidationError{}

// Validate checks the field values on ElementOperation with the rules defined
// in the proto definition for this message. If any rules are violated, an
// error is returned.
func (m *ElementOperation) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for Operator

	// no validation rules for OperatedAt

	// no validation rules for OperatedKey

	// no validation rules for Op

	// no validation rules for LastVersion

	// no validation rules for CurrentVersion

	// no validation rules for Remark

	return nil
}

// ElementOperationValidationError is the validation error returned by
// ElementOperation.Validate if the designated constraints aren't met.
type ElementOperationValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ElementOperationValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ElementOperationValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ElementOperationValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ElementOperationValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ElementOperationValidationError) ErrorName() string { return "ElementOperationValidationError" }

// Error satisfies the builtin error interface
func (e ElementOperationValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sElementOperation.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ElementOperationValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ElementOperationValidationError{}

// Validate checks the field values on Instance with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *Instance) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for ClientId

	// no validation rules for AgentId

	// no validation rules for ClientIp

	// no validation rules for App

	// no validation rules for Env

	// no validation rules for LastRenewTimestamp

	return nil
}

// InstanceValidationError is the validation error returned by
// Instance.Validate if the designated constraints aren't met.
type InstanceValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e InstanceValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e InstanceValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e InstanceValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e InstanceValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e InstanceValidationError) ErrorName() string { return "InstanceValidationError" }

// Error satisfies the builtin error interface
func (e InstanceValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sInstance.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = InstanceValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = InstanceValidationError{}

// Validate checks the field values on AgentInstance with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *AgentInstance) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for AgentId

	// no validation rules for Addr

	// no validation rules for Annotations

	return nil
}

// AgentInstanceValidationError is the validation error returned by
// AgentInstance.Validate if the designated constraints aren't met.
type AgentInstanceValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AgentInstanceValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AgentInstanceValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AgentInstanceValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AgentInstanceValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AgentInstanceValidationError) ErrorName() string { return "AgentInstanceValidationError" }

// Error satisfies the builtin error interface
func (e AgentInstanceValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAgentInstance.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AgentInstanceValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AgentInstanceValidationError{}

// Validate checks the field values on AgentInstanceChange with the rules
// defined in the proto definition for this message. If any rules are
// violated, an error is returned.
func (m *AgentInstanceChange) Validate() error {
	if m == nil {
		return nil
	}

	if v, ok := interface{}(m.GetIns()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return AgentInstanceChangeValidationError{
				field:  "Ins",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for Op

	return nil
}

// AgentInstanceChangeValidationError is the validation error returned by
// AgentInstanceChange.Validate if the designated constraints aren't met.
type AgentInstanceChangeValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AgentInstanceChangeValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AgentInstanceChangeValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AgentInstanceChangeValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AgentInstanceChangeValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AgentInstanceChangeValidationError) ErrorName() string {
	return "AgentInstanceChangeValidationError"
}

// Error satisfies the builtin error interface
func (e AgentInstanceChangeValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAgentInstanceChange.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AgentInstanceChangeValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AgentInstanceChangeValidationError{}