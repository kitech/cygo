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
func memcpy3(dst voidptr, src voidptr, n int) voidptr {
	rv := C.memcpy(dst, src, n)
	return rv
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
func hehe_exped(a int, b string) int {
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

// func panic()   {}
func panicln()         {}
func fatal()           {}
func fatalln()         {}
func throw(err error)  {}
func report(err error) {}

// 非侵入式异常机制 - 即，不改变主逻辑的缩进结构
// 一种处理 err 的写法，是否可行
// 错误处理不掺杂在主代码逻辑
// 不会影响主逻辑的缩进结构
// 能够隐式向上传递错误
// 不受提前return的影响，即使在catch之前提前return， catch语句依旧会处理
// catch语句块可以写了函数的任意位置
// catch 可以拆分，编译器合并的时候是否会出现有岐义的可能，另外如果要区分则需要拆分catch
// 如果分支判断用case的话，break/continue的语义是否有原来的 case 中的语义冲突
// 没有处理的默认语义，是继续执行，还是立即返回呢？应该是继续执行，还是强调要处理错误，只是简化处理方式
// 接上，但是好像还是出错返回的多，所以默认出错没有相应的处理的话，立即中止并返回？
// 默认 return 的情况，还应该自动加入栈信息
// 由于可能在 catch中会使用当前作用域的变量，所以编译器还要把它拆分到各个不同的分支，或者把变量闭包
//   这个丢失上下文的情况比较麻烦，再像NIM一样，添加跳转过来的行号吧
// 如果当前的函数不返回error，则错误无法继续传播，默认应该为继续执行，因为这应该是必须处理的错误
// 是否要考虑fallthrough的问题，目前觉得不需要
// catch 块能不能用在非顶级块中呢？
// 如果在catch语句块中再有catch咋办？
// 有多个error 类型返回值是咋办？
// 看起来会让编译器有些复杂
// 一个问题是，错误处理距离出错函数的调用有一个函数（当前函数）的距离
func unierrchk() {
	//
	/*
		dosmth1
		dosmth2
		if ok {
			dosmth3
		}
		if notok {
			dosmth4
		}
		return

		//
		// Errorx,Errory, Errorz 可以是前面 dosmthx任意调用抛出的，且无顺序影响
		catch err {
		case err.Error() == "sometxt123":
		case err.Error() == "sometxt456":
		case Errorx:
		case Errory:
		case Errorz:
		default:
		}
	*/
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

// TODO need compiler
func errbreak()
func errcontinue()

func errreturn(err error, args ...interface{})

func errpanic(err error, args ...interface{}) {
	if err == nil {
		return
	}
	var errmsg string = err.Error()
	println("err", errmsg)
	memcpy3(0x1, 0x1, 0x1)
}
func errfatal(err error, args ...interface{}) {
	if err == nil {
		return
	}
	var errmsg string = err.Error()
	println("err", errmsg)
	C.exit(-1)
}
func errprint(err error, args ...interface{}) {
	if err != nil {
		var errmsg string = err.Error()
		argc := args.len()
		switch argc {
		case 0:
			println("err", errmsg, args...)
		case 1:
			println("err", errmsg, args...)
		case 2:
			println("err", errmsg, args...)
		case 3:
			println("err", errmsg, args...)
		case 4:
			println("err", errmsg, args...)
		case 5:
			println("err", errmsg, args...)
		default:
			println("err", errmsg, args...) // TODO compiler
		}
	}
}

func errdo(err error, errfn func(e error)) {
	if err == nil {
		return
	}
	if errfn == nil {
		return
	}
	errfn(err)
}

// TODO need compiler
func nilbreak(obj voidptr)
func nilcontinue(obj voidptr)

func nilreturn(obj voidptr, args ...interface{})

func nilpanic(obj voidptr, args ...interface{}) {
	if obj != nil {
		return
	}
	println("nil", obj, args...)
	memcpy3(0x1, 0x1, 0x1)
}
func nilfatal(obj voidptr, args ...interface{}) {
	if obj != nil {
		return
	}
	println("nil", obj, args...)
	C.exit(-1)
}
func nilprint(obj voidptr, args ...interface{}) {
	if obj != nil {
		return
	}
	println("nil", obj, args...)

}
func nildo(obj voidptr, nilfn func()) {
	if obj != nil {
		return
	}
	if nilfn == nil {
		return
	}
	nilfn()
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
