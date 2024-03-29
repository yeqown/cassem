// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: acl.proto

package concept

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
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
	_ = sort.Sort
)

// Validate checks the field values on User with the rules defined in the proto
// definition for this message. If any rules are violated, the first error
// encountered is returned, or nil if there are no violations.
func (m *User) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on User with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in UserMultiError, or nil if none found.
func (m *User) ValidateAll() error {
	return m.validate(true)
}

func (m *User) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if err := m._validateEmail(m.GetAccount()); err != nil {
		err = UserValidationError{
			field:  "Account",
			reason: "value must be a valid email address",
			cause:  err,
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if l := utf8.RuneCountInString(m.GetNickname()); l < 1 || l > 64 {
		err := UserValidationError{
			field:  "Nickname",
			reason: "value length must be between 1 and 64 runes, inclusive",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if l := utf8.RuneCountInString(m.GetHashedPassword()); l < 6 || l > 12 {
		err := UserValidationError{
			field:  "HashedPassword",
			reason: "value length must be between 6 and 12 runes, inclusive",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if utf8.RuneCountInString(m.GetSalt()) != 8 {
		err := UserValidationError{
			field:  "Salt",
			reason: "value length must be 8 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)

	}

	if _, ok := User_Status_name[int32(m.GetStatus())]; !ok {
		err := UserValidationError{
			field:  "Status",
			reason: "value must be one of the defined enum values",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return UserMultiError(errors)
	}
	return nil
}

func (m *User) _validateHostname(host string) error {
	s := strings.ToLower(strings.TrimSuffix(host, "."))

	if len(host) > 253 {
		return errors.New("hostname cannot exceed 253 characters")
	}

	for _, part := range strings.Split(s, ".") {
		if l := len(part); l == 0 || l > 63 {
			return errors.New("hostname part must be non-empty and cannot exceed 63 characters")
		}

		if part[0] == '-' {
			return errors.New("hostname parts cannot begin with hyphens")
		}

		if part[len(part)-1] == '-' {
			return errors.New("hostname parts cannot end with hyphens")
		}

		for _, r := range part {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return fmt.Errorf("hostname parts can only contain alphanumeric characters or hyphens, got %q", string(r))
			}
		}
	}

	return nil
}

func (m *User) _validateEmail(addr string) error {
	a, err := mail.ParseAddress(addr)
	if err != nil {
		return err
	}
	addr = a.Address

	if len(addr) > 254 {
		return errors.New("email addresses cannot exceed 254 characters")
	}

	parts := strings.SplitN(addr, "@", 2)

	if len(parts[0]) > 64 {
		return errors.New("email address local phrase cannot exceed 64 characters")
	}

	return m._validateHostname(parts[1])
}

// UserMultiError is an error wrapping multiple validation errors returned by
// User.ValidateAll() if the designated constraints aren't met.
type UserMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m UserMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m UserMultiError) AllErrors() []error { return m }

// UserValidationError is the validation error returned by User.Validate if the
// designated constraints aren't met.
type UserValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e UserValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e UserValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e UserValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e UserValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e UserValidationError) ErrorName() string { return "UserValidationError" }

// Error satisfies the builtin error interface
func (e UserValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sUser.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = UserValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = UserValidationError{}

// Validate checks the field values on Casbin with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Casbin) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Casbin with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in CasbinMultiError, or nil if none found.
func (m *Casbin) ValidateAll() error {
	return m.validate(true)
}

func (m *Casbin) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	for idx, item := range m.GetPolicies() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, CasbinValidationError{
						field:  fmt.Sprintf("Policies[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, CasbinValidationError{
						field:  fmt.Sprintf("Policies[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return CasbinValidationError{
					field:  fmt.Sprintf("Policies[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return CasbinMultiError(errors)
	}
	return nil
}

// CasbinMultiError is an error wrapping multiple validation errors returned by
// Casbin.ValidateAll() if the designated constraints aren't met.
type CasbinMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m CasbinMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m CasbinMultiError) AllErrors() []error { return m }

// CasbinValidationError is the validation error returned by Casbin.Validate if
// the designated constraints aren't met.
type CasbinValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e CasbinValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e CasbinValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e CasbinValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e CasbinValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e CasbinValidationError) ErrorName() string { return "CasbinValidationError" }

// Error satisfies the builtin error interface
func (e CasbinValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCasbin.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = CasbinValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = CasbinValidationError{}

// Validate checks the field values on Casbin_Policy with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Casbin_Policy) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Casbin_Policy with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in Casbin_PolicyMultiError, or
// nil if none found.
func (m *Casbin_Policy) ValidateAll() error {
	return m.validate(true)
}

func (m *Casbin_Policy) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Ptype

	// no validation rules for V0

	// no validation rules for V1

	// no validation rules for V2

	// no validation rules for V3

	// no validation rules for V4

	// no validation rules for V5

	if len(errors) > 0 {
		return Casbin_PolicyMultiError(errors)
	}
	return nil
}

// Casbin_PolicyMultiError is an error wrapping multiple validation errors
// returned by Casbin_Policy.ValidateAll() if the designated constraints
// aren't met.
type Casbin_PolicyMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m Casbin_PolicyMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m Casbin_PolicyMultiError) AllErrors() []error { return m }

// Casbin_PolicyValidationError is the validation error returned by
// Casbin_Policy.Validate if the designated constraints aren't met.
type Casbin_PolicyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e Casbin_PolicyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e Casbin_PolicyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e Casbin_PolicyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e Casbin_PolicyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e Casbin_PolicyValidationError) ErrorName() string { return "Casbin_PolicyValidationError" }

// Error satisfies the builtin error interface
func (e Casbin_PolicyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCasbin_Policy.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = Casbin_PolicyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = Casbin_PolicyValidationError{}
