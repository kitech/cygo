package builtin

/*
 */
import "C"

func (ch byte) isspace() bool {
	if ch == ' ' || ch == '\n' || ch == '\t' {
		return true
	}
	return false
}
func (ch rune) isspace() bool {
	if ch == ' ' || ch == '\n' || ch == '\t' {
		return true
	}
	return false
}
func (ch byte) isdigit() bool {
	if ch >= '0' && ch <= '9' {
		return true
	}
	return false
}
func (ch byte) isnumber() bool {
	if (ch >= '0' && ch <= '9') || ch == '.' {
		return true
	}
	return false
}
func (ch byte) isalpha() bool {
	if ch >= 'a' && ch <= 'z' {
		return true
	}
	if ch >= 'A' && ch <= 'Z' {
		return true
	}
	return false
}

///
// TODO
func (i rune) repr() string {
	// s := string(rune)
	// return s
	mem := malloc3(32)
	C.sprintf(mem, "%d".ptr, i)
	return gostring(mem)
}
func (i byte) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%c".ptr, i)
	return gostring(mem)
}
func (i int) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%d".ptr, i)
	return gostring(mem)
}

func (i float64) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%g".ptr, i)
	return gostring(mem)
}
func (i float32) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%g".ptr, i)
	return gostring(mem)
}

func (i int16) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%d".ptr, i)
	return gostring(mem)
}
func (i int8) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%d".ptr, i)
	return gostring(mem)
}
func (i int32) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%d".ptr, i)
	return gostring(mem)
}
func (i int64) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%ld".ptr, i)
	return gostring(mem)
}
func (i uint8) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%d".ptr, i)
	return gostring(mem)
}
func (i uint16) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%d".ptr, i)
	return gostring(mem)
}
func (i uint32) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%d".ptr, i)
	return gostring(mem)
}
func (i uint64) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%lu".ptr, i)
	return gostring(mem)
}
func (i usize) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%lu".ptr, i)
	return gostring(mem)
}

func (i voidptr) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%p".ptr, i)
	return gostring(mem)
}
func (i byteptr) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%p".ptr, i)
	return gostring(mem)
}
func (i charptr) repr() string {
	mem := malloc3(32)
	C.sprintf(mem, "%p".ptr, i)
	return gostring(mem)
}

func (v int) times(itfn func(it int)) {
	for i := 0; i < v; i++ {
		itfn(i)
	}
}
