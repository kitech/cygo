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
func strlen3(ptr voidptr) int {
	rv := C.strlen(ptr)
	return rv
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
func memmove3(dst voidptr, src voidptr, n int) voidptr {
	rv := C.memmove(dst, src, n)
	return rv
}
func memset3(ptr voidptr, c int, n int) voidptr {
	rv := C.memset(ptr, c, n)
	return rv
}
func memcmp3(p1 voidptr, p2 voidptr, n int) int {
	rv := C.memcmp(p1, p2, n)
	return rv
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

// func panic()           {}
func fatal()           {}
func fatalln()         {}
func throw(err error)  {}
func report(err error) {}

//export panic
func panic_goimpl(v interface{}) {
	println("panic_goimpl")
	abort()
}

func fatal2()           {}
func fatalln2()         {}
func throw2(err error)  {}
func report2(err error) {}

func raise(sig int) int { return C.raise(sig) }
func abort()            { C.abort() }

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
	abort()
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
		argc := args.len
		switch argc {
		case 0:
			// println("err", errmsg, args...)
		case 1:
			// println("err", errmsg, args...)
		case 2:
			// println("err", errmsg, args...)
		case 3:
			// println("err", errmsg, args...)
		case 4:
			// println("err", errmsg, args...)
		case 5:
			// println("err", errmsg, args...)
		default:
			// println("err", errmsg, args...) // TODO compiler
		}
		println("err", errmsg)
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
	// println("nil", obj, args...)
	println("nil", obj)
	abort()
}
func nilfatal(obj voidptr, args ...interface{}) {
	if obj != nil {
		return
	}
	// println("nil", obj, args...)
	println("nil", obj)
	C.exit(-1)
}
func nilprint(obj voidptr, args ...interface{}) {
	if obj != nil {
		return
	}
	// println("nil", obj, args...)
	println("nil", obj)

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

type mirerror struct {
	obj   voidptr // error's this object
	Error func(obj voidptr) string
}

//export error_Errorddd
func error_Errorddd(err error) string {
	return ""
}
