// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: cassemagent.api.proto

package agent

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

// Validate checks the field values on GetElementReq with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *GetElementReq) Validate() error {
	if m == nil {
		return nil
	}

	if l := utf8.RuneCountInString(m.GetApp()); l < 3 || l > 30 {
		return GetElementReqValidationError{
			field:  "App",
			reason: "value length must be between 3 and 30 runes, inclusive",
		}
	}

	if l := utf8.RuneCountInString(m.GetEnv()); l < 3 || l > 30 {
		return GetElementReqValidationError{
			field:  "Env",
			reason: "value length must be between 3 and 30 runes, inclusive",
		}
	}

	if l := len(m.GetKeys()); l < 1 || l > 100 {
		return GetElementReqValidationError{
			field:  "Keys",
			reason: "value must contain between 1 and 100 items, inclusive",
		}
	}

	_GetElementReq_Keys_Unique := make(map[string]struct{}, len(m.GetKeys()))

	for idx, item := range m.GetKeys() {
		_, _ = idx, item

		if _, exists := _GetElementReq_Keys_Unique[item]; exists {
			return GetElementReqValidationError{
				field:  fmt.Sprintf("Keys[%v]", idx),
				reason: "repeated value must contain unique items",
			}
		} else {
			_GetElementReq_Keys_Unique[item] = struct{}{}
		}

		// no validation rules for Keys[idx]
	}

	return nil
}

// GetElementReqValidationError is the validation error returned by
// GetElementReq.Validate if the designated constraints aren't met.
type GetElementReqValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetElementReqValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetElementReqValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetElementReqValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetElementReqValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetElementReqValidationError) ErrorName() string { return "GetElementReqValidationError" }

// Error satisfies the builtin error interface
func (e GetElementReqValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetElementReq.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetElementReqValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetElementReqValidationError{}

// Validate checks the field values on GetElementResp with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *GetElementResp) Validate() error {
	if m == nil {
		return nil
	}

	for idx, item := range m.GetElems() {
		_, _ = idx, item

		if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return GetElementRespValidationError{
					field:  fmt.Sprintf("Elems[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	return nil
}

// GetElementRespValidationError is the validation error returned by
// GetElementResp.Validate if the designated constraints aren't met.
type GetElementRespValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetElementRespValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetElementRespValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetElementRespValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetElementRespValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetElementRespValidationError) ErrorName() string { return "GetElementRespValidationError" }

// Error satisfies the builtin error interface
func (e GetElementRespValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetElementResp.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetElementRespValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetElementRespValidationError{}

// Validate checks the field values on UnregisterReq with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *UnregisterReq) Validate() error {
	if m == nil {
		return nil
	}

	if l := utf8.RuneCountInString(m.GetClientId()); l < 5 || l > 64 {
		return UnregisterReqValidationError{
			field:  "ClientId",
			reason: "value length must be between 5 and 64 runes, inclusive",
		}
	}

	if ip := net.ParseIP(m.GetClientIp()); ip == nil {
		return UnregisterReqValidationError{
			field:  "ClientIp",
			reason: "value must be a valid IP address",
		}
	}

	return nil
}

// UnregisterReqValidationError is the validation error returned by
// UnregisterReq.Validate if the designated constraints aren't met.
type UnregisterReqValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e UnregisterReqValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e UnregisterReqValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e UnregisterReqValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e UnregisterReqValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e UnregisterReqValidationError) ErrorName() string { return "UnregisterReqValidationError" }

// Error satisfies the builtin error interface
func (e UnregisterReqValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sUnregisterReq.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = UnregisterReqValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = UnregisterReqValidationError{}

// Validate checks the field values on RegisterReq with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *RegisterReq) Validate() error {
	if m == nil {
		return nil
	}

	if l := utf8.RuneCountInString(m.GetClientId()); l < 5 || l > 64 {
		return RegisterReqValidationError{
			field:  "ClientId",
			reason: "value length must be between 5 and 64 runes, inclusive",
		}
	}

	if ip := net.ParseIP(m.GetClientIp()); ip == nil {
		return RegisterReqValidationError{
			field:  "ClientIp",
			reason: "value must be a valid IP address",
		}
	}

	if len(m.GetWatching()) > 0 {

		for idx, item := range m.GetWatching() {
			_, _ = idx, item

			if v, ok := interface{}(item).(interface{ Validate() error }); ok {
				if err := v.Validate(); err != nil {
					return RegisterReqValidationError{
						field:  fmt.Sprintf("Watching[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					}
				}
			}

		}

	}

	return nil
}

// RegisterReqValidationError is the validation error returned by
// RegisterReq.Validate if the designated constraints aren't met.
type RegisterReqValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RegisterReqValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RegisterReqValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RegisterReqValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RegisterReqValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RegisterReqValidationError) ErrorName() string { return "RegisterReqValidationError" }

// Error satisfies the builtin error interface
func (e RegisterReqValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRegisterReq.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RegisterReqValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RegisterReqValidationError{}

// Validate checks the field values on EmptyResp with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *EmptyResp) Validate() error {
	if m == nil {
		return nil
	}

	return nil
}

// EmptyRespValidationError is the validation error returned by
// EmptyResp.Validate if the designated constraints aren't met.
type EmptyRespValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e EmptyRespValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e EmptyRespValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e EmptyRespValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e EmptyRespValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e EmptyRespValidationError) ErrorName() string { return "EmptyRespValidationError" }

// Error satisfies the builtin error interface
func (e EmptyRespValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sEmptyResp.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = EmptyRespValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = EmptyRespValidationError{}

// Validate checks the field values on WatchReq with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *WatchReq) Validate() error {
	if m == nil {
		return nil
	}

	if len(m.GetWatching()) > 0 {

		for idx, item := range m.GetWatching() {
			_, _ = idx, item

			if v, ok := interface{}(item).(interface{ Validate() error }); ok {
				if err := v.Validate(); err != nil {
					return WatchReqValidationError{
						field:  fmt.Sprintf("Watching[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					}
				}
			}

		}

	}

	if l := utf8.RuneCountInString(m.GetClientId()); l < 5 || l > 64 {
		return WatchReqValidationError{
			field:  "ClientId",
			reason: "value length must be between 5 and 64 runes, inclusive",
		}
	}

	if ip := net.ParseIP(m.GetClientIp()); ip == nil {
		return WatchReqValidationError{
			field:  "ClientIp",
			reason: "value must be a valid IP address",
		}
	}

	return nil
}

// WatchReqValidationError is the validation error returned by
// WatchReq.Validate if the designated constraints aren't met.
type WatchReqValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e WatchReqValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e WatchReqValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e WatchReqValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e WatchReqValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e WatchReqValidationError) ErrorName() string { return "WatchReqValidationError" }

// Error satisfies the builtin error interface
func (e WatchReqValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sWatchReq.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = WatchReqValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = WatchReqValidationError{}

// Validate checks the field values on WatchResp with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *WatchResp) Validate() error {
	if m == nil {
		return nil
	}

	if v, ok := interface{}(m.GetElem()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return WatchRespValidationError{
				field:  "Elem",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	return nil
}

// WatchRespValidationError is the validation error returned by
// WatchResp.Validate if the designated constraints aren't met.
type WatchRespValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e WatchRespValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e WatchRespValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e WatchRespValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e WatchRespValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e WatchRespValidationError) ErrorName() string { return "WatchRespValidationError" }

// Error satisfies the builtin error interface
func (e WatchRespValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sWatchResp.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = WatchRespValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = WatchRespValidationError{}

// Validate checks the field values on DispatchReq with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *DispatchReq) Validate() error {
	if m == nil {
		return nil
	}

	for idx, item := range m.GetElems() {
		_, _ = idx, item

		if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return DispatchReqValidationError{
					field:  fmt.Sprintf("Elems[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	return nil
}

// DispatchReqValidationError is the validation error returned by
// DispatchReq.Validate if the designated constraints aren't met.
type DispatchReqValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DispatchReqValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DispatchReqValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DispatchReqValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DispatchReqValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DispatchReqValidationError) ErrorName() string { return "DispatchReqValidationError" }

// Error satisfies the builtin error interface
func (e DispatchReqValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDispatchReq.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DispatchReqValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DispatchReqValidationError{}

// Validate checks the field values on DispatchResp with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *DispatchResp) Validate() error {
	if m == nil {
		return nil
	}

	return nil
}

// DispatchRespValidationError is the validation error returned by
// DispatchResp.Validate if the designated constraints aren't met.
type DispatchRespValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DispatchRespValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DispatchRespValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DispatchRespValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DispatchRespValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DispatchRespValidationError) ErrorName() string { return "DispatchRespValidationError" }

// Error satisfies the builtin error interface
func (e DispatchRespValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDispatchResp.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DispatchRespValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DispatchRespValidationError{}