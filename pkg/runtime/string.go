package runtime

import (
	"unsafe"
)

// ToBytes converts a string to a byte slice without copying.
// WARNING: This uses unsafe operations and must be used carefully.
// The returned byte slice must not be modified, and the string must
// not be used after the byte slice is modified.
func ToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	b := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&b))
}

// ToString converts a byte slice to a string without copying.
// WARNING: This uses unsafe operations and must be used carefully.
// The byte slice must not be modified after conversion.
func ToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
