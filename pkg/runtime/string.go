package runtime

import (
	"strings"
	"unsafe"
)

func ToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	b := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&b))
}

func ToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// IndexOf get the index of target string in array. if pos is negative which means
// target string is not found in array, otherwise pos is the index of target string.
func IndexOf(target string, arr []string) (pos int) {
	pos = -1
	for idx, v := range arr {
		if strings.Compare(v, target) == 0 {
			pos = idx
			break
		}
	}

	return
}
