package builtin

/*
#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <string.h>
*/
import "C"

///
type mirstring struct {
	ptr byteptr
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

func mirstring_new() string {
	var s string
	return s
}

func mirstring_new_cstr_ref(sptr byteptr) string {
	var s string
	s.ptr = sptr
	s.len = strlen3(sptr)

	return s
}

func mirstring_new_cstr(sptr byteptr) string {
	var s string
	s.ptr = strdup3(sptr)
	s.len = strlen3(sptr)
	return s
}

func mirstring_new_cstr2(sptr byteptr, len int) string {
	var s string
	s.ptr = strndup3(sptr, len)
	s.len = len
	return s
}
func mirstring_new_char(ch byte) string {
	var s string
	s.ptr = malloc3(8)
	s.ptr[0] = ch
	s.len = 1
	return s
}

// TODO
func mirstring_new_rune(ch rune) string {
	var s string
	s.ptr = malloc3(8)
	s.len = 3

	var p byteptr
	p = (byteptr)(&ch)
	s.ptr[0] = p[0]
	s.ptr[1] = p[1]
	s.ptr[2] = p[2]
	s.ptr[3] = p[3]
	return s
}

func (s string) Ptr() byteptr {
	return s.ptr
}
func (s string) Len() int {
	return s.len
}
func (s string) cstr() byteptr {
	return s.ptr
}
func (s string) empty() bool {
	return s.len == 0
}

func (s string) split(sep string) []string {
	res := []string{}
	pos := 0
	slen := s.len
	seplen := sep.len
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
func (s string) splitline() []string { return s.split("\n") }
func (s string) splitpath() []string { return s.split("/") }

func (s string) trimsp() string {
	slen := s.len
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
	slen := s.len
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
	slen := s.len
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

func (s string) trim_goimpl(cutset string) string {
	return s
}
func (s string) ltrim_goimpl(cutset string) string {
	return s
}
func (s string) rtrim_goimpl(cutset string) string {
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
	slen := s.len
	seplen := sep.len
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
	seplen := sep.len
	return s[pos+seplen:]
}
func (s string) leftn(n int) string {
	return s
}
func (s string) rightn(n int) string {
	return s
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
	slen := s.len
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
	slen := s.len
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
	slen := s.len
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

// sep with space or \t
func (s string) fields() []string {
	return nil
}

// TODO
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
	return f32(rv)
}
func (s string) tof64() f64 {
	rv := C.atof(s.ptr)
	return rv
}
func (s string) tobool() bool {
	return s == "true"
}

func (s string) isdigit() bool {
	slen := s.len
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
	slen := s.len
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
	slen := s.len
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
	slen := s.len
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
	slen := s.len
	for i := 0; i < slen; i++ {
		ch := s[i]
		if ch < 128 {
		} else {
			return false
		}
	}
	return true
}
