package strings

/*
 */
// import "C"

func HasPrefix(s string, pfx string) bool {
	xlen := len(pfx)
	slen := len(s)
	if xlen > slen {
		return false
	}

	ts := s[0:xlen]
	if ts == pfx {
		return true
	}
	return false
}
func HasSuffix(s string, sfx string) bool {
	xlen := len(sfx)
	slen := len(s)
	if xlen > slen {
		return false
	}

	ts := s[slen-xlen:]
	if ts == sfx {
		return true
	}
	return false
}
func Contains(s string, substr string) bool {
	pos := Index(s, substr)
	return pos >= 0
}
func Index(s string, substr string) int {
	xlen := len(substr)
	slen := len(s)
	if xlen > slen {
		return -1
	}
	for i := 0; i < slen-xlen+1; i++ {
		ts := s[i : i+xlen]
		if ts == substr {
			return i
		}
	}
	return -1
}
func LastIndex(s string, substr string) int {
	xlen := len(substr)
	slen := len(s)
	if xlen > slen {
		return -1
	}
	for i := slen - xlen; i >= 0; i-- {
		ts := s[i : i+xlen]
		if ts == substr {
			return i
		}
	}
	return -1
}

// func cxstring_dup(s string) string

func Title(s string) string {
	if len(s) == 0 {
		return s
	}
	c := s[0]
	if c >= 'a' && c <= 'z' {
		c = c - 32
	}
	rs := string(c)
	rs = rs + s[1:]
	return rs
}
func Untitle(s string) string {
	if len(s) == 0 {
		return s
	}
	c := s[0]
	if c >= 'A' && c <= 'Z' {
		c = c + 32
	}
	rs := string(c)
	rs = rs + s[1:]
	return rs
}
func Upper(s string) string {
	var rs string
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c = c - 32
		}
		rs = rs + string(c)
	}
	return rs
}
func Lower(s string) string {
	var rs string
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + 32
		}
		rs = rs + string(c)
	}
	return rs
}

func Trim(s string, cutset string) string {
	l := len(s)
	hpos := 0
	tpos := l

	for i := 0; i < l; i++ {
		c := s[i]
		found := false
		for j := 0; j < len(cutset); j++ {
			if c == cutset[j] {
				found = true
				break
			}
		}
		if found {
			hpos = i + 1
		} else {
			break
		}
	}
	for i := l - 1; i >= 0; i-- {
		c := s[i]
		found := false
		for j := 0; j < len(cutset); j++ {
			if c == cutset[j] {
				found = true
				break
			}
		}
		if found {
			tpos = i + 1
		} else {
			break
		}
	}
	return s[hpos:tpos]
}
func TrimSpace(s string) string {
	return Trim(s, " \t\r\n")
}
