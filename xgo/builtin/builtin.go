package builtin

// don't use other packages, only C is supported

/*
#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <string.h>
#include <cxrtbase.h>
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
func strdup3(ptr voidptr) voidptr {
	if ptr == nil {
		return nil
	}
	return C.cxstrdup(ptr)
}
func strndup3(ptr voidptr, n int) voidptr {
	if ptr == nil {
		return nil
	}
	return C.cxstrndup(ptr, n)
}
func memcpy3(dst voidptr, src voidptr, n int) {
	C.memcpy(dst, src, n)
}
func memdup3(src voidptr, n int) voidptr {
	if src == nil {
		return nil
	}
	dst := malloc3(n + 1)
	C.memcpy(dst, src, n)
	return dst
}

//export hehe_exped
func hehe(a int, b string) int {
	return 0
}

//[nomangle]
func assert()
func sizeof() int
func alignof() int
func offsetof() int

//export unsafe__Sizeof111
func unsafe__Sizeof111(a int) int {
	return 0
}

//export unsafe__Alignof111
func unsafe__Alignof111(a int) int {
	return 0
}

//export unsafe__Offsetof111
func unsafe__Offsetof111(a int) int {
	return 0
}

func cerrclr() {
	C.errno = 0
}
func cerrno() int {
	return C.errno
}
func cerrmsg() string {
	emsg := C.strerror(C.errno)
	return gostring(emsg)
}
func cerrmsgof(no int) string {
	emsg := C.strerror(no)
	return gostring(emsg)
}
func prtcerr(pfx string) {
	pfx = pfx + cerrmsg()
	println(pfx)
}

///
type mirstring struct {
	ptr voidptr
	len int
}

func gostring(ptr byteptr) string {
	if ptr == nil {
		return ""
	}
	return string(ptr)
}
func gostring_clone(ptr byteptr) string {
	if ptr == nil {
		return ""
	}
	ptr2 := strdup3(ptr)
	return string(ptr2)
}
func gostringn(ptr byteptr, n int) string {
	if ptr == nil {
		return ""
	}
	s := string(ptr)
	s.len = n
	return s
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
	slen := s.len()
	lspos := 0
	rspos := slen - 1

	for i := 0; i < slen; i++ {
		ch := s[i]
		if ch.isspace() {
			continue
		} else {
			lspos = i
			break
		}
	}
	for i := slen - 1; i >= 0; i-- {
		ch := s[i]
		if ch.isspace() {
			continue
		} else {
			rspos = i
			break
		}
	}

	ns := s[lspos : rspos+1]
	return ns
}
func (s string) ltrimsp() string {
	slen := s.len()
	lspos := 0
	rspos := slen - 1

	for i := 0; i < slen; i++ {
		ch := s[i]
		if ch.isspace() {
			continue
		} else {
			lspos = i
			break
		}
	}

	ns := s[lspos:rspos]
	return ns
}
func (s string) rtrimsp() string {
	slen := s.len()
	lspos := 0
	rspos := slen - 1

	for i := slen - 1; i >= 0; i-- {
		ch := s[i]
		if ch.isspace() {
			continue
		} else {
			rspos = i
			break
		}
	}

	ns := s[lspos : rspos+1]
	return ns
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

//export cxarray2_join
func cxarray2_join(arr []string, sep string) string {
	s := ""
	for i := 0; i < len(arr); i++ {
		s += arr[i]
		if i < len(arr)-1 {
			s += sep
		}
	}
	return s
}

func (s string) index(sep string) int {
	res := -1
	slen := len(s)
	seplen := len(sep)
	for i := 0; i < slen; i++ {
		if i+seplen > slen {
			break
		}
		// s1 := s[i : i+seplen]
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
	ns := s[0:pos]
	return ns
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
	slen := s.len()
	ns := ""
	for i := 0; i < slen; i++ {
		ch := s[i]
		if ch >= 'a' && ch <= 'z' {
			ch2 := ch - 32
			ns += string(ch2)
		} else {
			ns += string(ch)
		}
	}
	return ns
}
func (s string) tolower() string {
	slen := s.len()
	ns := ""
	for i := 0; i < slen; i++ {
		ch := s[i]
		if ch >= 'A' && ch <= 'Z' {
			ch2 := ch + 32
			ns += string(ch2)
		} else {
			ns += string(ch)
		}
	}
	return ns
}
func (s string) totitle() string {
	slen := s.len()
	ns := ""
	for i := 0; i < slen; i++ {
		ch := s[i]
		if i == 0 && ch >= 'a' && ch <= 'z' {
			ch2 := ch - 32
			ns += string(ch2)
		} else {
			ns += string(ch)
		}
	}
	return ns
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
	rv := C.atoi(s.ptr)
	return rv
}
func (s string) tof32() f32 {
	rv := C.atof(s.ptr)
	return rv
}
func (s string) tof64() f64 {
	rv := C.atof(s.ptr)
	return rv
}
func (s string) tobool() bool {
	return s == "true"
}

func (s string) isdigit() bool {
	slen := s.len()
	for i := 0; i < slen; i++ {
		ch := s[i]
		if ch >= '0' && ch <= '9' {
		} else {
			return false
		}
	}
	return true
}
func (s string) isnumber() bool {
	slen := s.len()
	for i := 0; i < slen; i++ {
		ch := s[i]
		if (ch >= '0' && ch <= '9') || ch == '.' {
		} else {
			return false
		}
	}
	return true
}
func (s string) isprintable() bool {
	slen := s.len()
	for i := 0; i < slen; i++ {
		ch := s[i]
		if ch >= '0' && ch <= '9' {
		} else if ch >= 'A' && ch <= 'Z' {
		} else if ch >= 'a' && ch <= 'z' {
		} else {
			return false
		}
	}
	return true
}
func (s string) ishex() bool {
	slen := s.len()
	for i := 0; i < slen; i++ {
		ch := s[i]
		if ch >= '0' && ch <= '9' {
		} else if ch >= 'A' && ch <= 'F' {
		} else if ch >= 'a' && ch <= 'f' {
		} else {
			return false
		}
	}
	return true
}
func (s string) isascii() bool {
	slen := s.len()
	for i := 0; i < slen; i++ {
		ch := s[i]
		if ch < 128 {
		} else {
			return false
		}
	}
	return true
}
