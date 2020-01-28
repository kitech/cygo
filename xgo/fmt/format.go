package fmt

func Sprintf(format string, args ...interface{}) string {
	slen := len(format)
	prepercnt := false
	for i := 0; i < slen; i++ {
		ch := format[i]
		if prepercnt {
			if ch == 'd' {
				println("int", args[i])
			} else if ch == 's' {
				println("str", args[i])
			} else if ch == 'f' {

			} else if ch == 'p' {

			} else {

			}
		}
		prepercnt = ch == '%'
	}
	return "aaa"
}
