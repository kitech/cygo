package xstrings

/*
 */
import "C"

func Split(s string, sep string) []string {
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
func Join(a []string, sep string) string {
	s := ""
	for i := 0; i < len(a); i++ {
		s += a[i]
		if i < len(a)-1 {
			s += sep
		}
	}
	return s
}

func Index(s string, sep string) int {
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

func Left(s string, sep string) string {
	pos := Index(s, sep)
	if pos < 0 {
		println(111)
		return ""
	}
	return s[0:pos]
}
func Right(s string, sep string) string {
	pos := Index(s, sep)
	if pos < 0 {
		return ""
	}
	seplen := len(sep)
	return s[pos+seplen:]
}
func Prefixed(s string, sub string) bool {
	pos := Index(s, sub)
	return pos == 0
}
func Suffixed(s string, sub string) bool {
	pos := Index(s, sub)
	return pos+len(sub) == len(s)
}
func Repeat(s string, count int) string {
	ns := ""
	for i := 0; i < count; i++ {
		ns += s
	}
	return ns
}
func Replace(s string, old string, new string, n int) string {
	pos := 0
	ns := ""
	for cnt := 0; pos < len(s) && (n <= 0 || (n > 0 && cnt < n)); cnt++ {
		s1 := s[pos:]
		idx := Index(s[pos:], old)
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
func ReplaceAll(s string, old string, new string) string {
	return Replace(s, old, new, -1)
}

func ParseHex(s string) u64 {
	ns := s
	if Prefixed(s, "0x") {
		ns = s[2:]
	}
	rv := C.strtoll(ns.ptr, nil, 16)
	return rv
}
