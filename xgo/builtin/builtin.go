package builtin

// don't use other packages, only C is supported

/*
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

func keep() {}

// func panic()   {}
func panicln() {}
func fatal()   {}
func fatalln() {}

func malloc3(sz int) voidptr {
	ptr := C.cxmalloc(sz)
	return ptr
}
func realloc3(ptr voidptr, sz int) voidptr {
	ptr2 := C.cxrealloc(ptr, sz)
	return ptr2
}
func free3(ptr voidptr) {
	C.cxfree(ptr)
}

//[nomangle]
func assert()
func sizeof() int
func alignof() int
func offsetof() int

//export hehe_exped
func hehe(a int, b string) int {
	return 0
}

type mirstring struct {
	ptr voidptr
	len int
}

func (s string) cstr() byteptr {
	return s.ptr
}
func (s string) split(sep string) []string {
	res := []string{}
	pos := 0
	slen := len(s)
	seplen := len(sep)
	for i := 0; i < slen; i++ {
		if i+seplen > slen {
			break
		}
		if s[i:i+seplen] == sep {
			t := s[pos:i]
			res = append(res, t)
			pos = i + seplen
		}
	}
	if pos < slen {
		t := s[pos:slen]
		res = append(res, t)
	}
	return res
}
func (s string) trimsp() string {
	return s
}

/*
func (a []string) join(sep string) string {
	s := ""
		for i := 0; i < len(a); i++ {
			s += a[i]
			if i < len(a)-1 {
				s += sep
			}
		}
		return s
}
*/

func (s string) index(sep string) int {
	res := -1
	slen := len(s)
	seplen := len(sep)
	for i := 0; i < slen; i++ {
		if i+seplen > slen {
			break
		}
		//s1 := s[i : i+seplen]
		if s[i:i+seplen] == sep {
			res = i
			break
		}
	}
	return res
}

func (s string) left(sep string) string {
	pos := s.index(sep)
	if pos < 0 {
		return ""
	}
	return s[0:pos]
}
func (s string) right(sep string) string {
	pos := s.index(sep)
	if pos < 0 {
		return ""
	}
	seplen := len(sep)
	return s[pos+seplen:]
}
func (s string) prefixed(sub string) bool {
	pos := s.index(sub)
	return pos == 0
}
func (s string) suffixed(sub string) bool {
	pos := s.index(sub)
	return pos+len(sub) == len(s)
}
func (s string) repeat(count int) string {
	ns := ""
	for i := 0; i < count; i++ {
		ns += s
	}
	return ns
}

func (s string) replace(old string, new string, n int) string {
	pos := 0
	ns := ""
	for cnt := 0; pos < len(s) && (n <= 0 || (n > 0 && cnt < n)); cnt++ {
		s1 := s[pos:]
		idx := s1.index(old)
		if idx < 0 {
			break
		}
		ns += s[pos : pos+idx]
		ns += new
		pos = pos + idx + len(old)
	}
	ns += s[pos:]
	return ns
}
func (s string) replaceall(old string, new string) string {
	return s.replace(old, new, -1)
}

func (s string) toupper() string {
	return s
}
func (s string) tolower() string {
	return s
}
func (s string) totitle() string {
	return s
}

func (s string) tomd5() string {
	return s
}
func (s string) tosha1() string {
	return s
}
func (s string) tosha256() string {
	return s
}
func (s string) tohex() string {
	return s
}

func (s string) toint() int {
	return 0
}
func (s string) tof32() f32 {
	return 0
}
func (s string) tof64() f64 {
	return 0
}
func (s string) tobool() bool {
	return s == "true"
}

func (i int) repr() string {
	return ""
}

func (i float64) repr() string {
	return ""
}
func (i float32) repr() string {
	return ""
}

func (i int16) repr() string {
	return ""
}
func (i int8) repr() string {
	return ""
}
func (i int32) repr() string {
	return ""
}
func (i int64) repr() string {
	return ""
}
func (i uint8) repr() string {
	return ""
}
func (i uint16) repr() string {
	return ""
}
func (i uint32) repr() string {
	return ""
}
func (i uint64) repr() string {
	return ""
}
func (i usize) repr() string {
	return ""
}

func (i voidptr) repr() string {
	return ""
}
func (i byteptr) repr() string {
	return ""
}
func (i charptr) repr() string {
	return ""
}
