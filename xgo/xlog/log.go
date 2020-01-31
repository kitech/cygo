package xlog

/*
#include <stdio.h>
#include <stdarg.h>

char* cxstring_unrefpp(cxstring** spp) {
    printf("%s\n", (*spp)->ptr);
    return (*spp)->ptr;
}
*/
import "C"

// import "xgo/xstrings"

const skip_depth = 4
const pkgsep = "__"
const mthsep = "_"
const newline = "\n"

var gopaths []string

func init() {
	eptr := C.getenv("GOPATH".ptr)
	estr := C.GoString(eptr)
	// gopaths = xstrings.Split(estr, ":")
	var estr2 string = estr
	gopaths = estr2.split(":")

	assert(pkgsep.len() > 0)
}

func dummy(args ...interface{}) {

}

func Println(args ...interface{}) {
	printx2(true, args...)
}
func Infoln(args ...interface{}) {
	printx2(true, args...)
}
func Warnln(args ...interface{}) {
	printx2(true, args...)
}
func Errorln(args ...interface{}) {
	printx2(true, args...)
}

func Fatalln(args ...interface{}) {
	printx2(true, args...)
}

func Printf(format string, args ...interface{}) {

}

func Sprintf(format string, args ...interface{}) string {
	return ""
}

///
func trim_gopath(s string) string {
	const clen = 5 // "/src/" length
	for i := 0; i < gopaths.len(); i++ {
		sub := gopaths[i]
		if s.prefixed(sub) {
			// if xstrings.Prefixed(s, sub) {
			return s[sub.len()+clen:]
		}
	}
	return s
}

func demangle_funcname(s string) string {
	// s2 := xstrings.Replace(s, pkgsep, ".", 1)
	s2 := s.replace(pkgsep, ".", 1)
	return s2
}

type metatype struct {
	size       uintptr // type size
	ptrdata    uintptr // size of memory prefix holding all pointers
	hash       uint32  // hash of type; avoids computation in hash tables
	tflag      uint8   // tflag   // extra type information flags
	align      uint8   // alignment of variable with this type
	fieldalign uint8   // alignment of struct field with this type
	kind       uint8   // enumeration for C
	alg        voidptr // *typeAlg // algorithm table
	gcdata     *byte   // garbage collection data
	str        byteptr // int32   // nameOff // string form
	ptrToThis  voidptr // int32   // typeOff // type for pointer to this type, may be zero
}
type eface struct {
	_type *metatype
	data  *voidptr
}

const (
	Invalid = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Ptr
	Slice
	// String
	Struct
	UnsafePointer
)
const (
	String  = 17
	Voidptr = 26
	Byteptr = 27
)

func dlsym1(sym string) voidptr {
	ptr := C.dlsym(nil, sym.ptr)
	return ptr
}

// func symasfn(fnptr voidptr) func() {	return fnptr}
// TODO support float/double
func unicall1(fnptr voidptr, args []*voidptr) voidptr {
	var tmpfn func() voidptr
	tmpfn = fnptr
	// argc := len(args)
	argc := args.len()

	var ret voidptr
	switch argc {
	case 0:
		ret = tmpfn()
	case 1:
		ret = tmpfn(*(args[0]))
	case 2:
		ret = tmpfn(*(args[0]), *(args[1]))
	case 3:
		ret = tmpfn(*(args[0]), *(args[1]), *(args[2]))
	case 4:
		ret = tmpfn(*(args[0]), *(args[1]), *(args[2]), *(args[3]))
	case 5:
		ret = tmpfn(*(args[0]), *(args[1]), *(args[2]), *(args[3]), *(args[4]))
	case 6:
		ret = tmpfn(*(args[0]), *(args[1]), *(args[2]), *(args[3]), *(args[4]),
			*(args[5]))
	case 7:
		ret = tmpfn(*(args[0]), *(args[1]), *(args[2]), *(args[3]), *(args[4]),
			*(args[5]), *(args[6]))
	case 8:
		ret = tmpfn(*(args[0]), *(args[1]), *(args[2]), *(args[3]), *(args[4]),
			*(args[5]), *(args[6]), *(args[7]))
	case 9:
		ret = tmpfn(*(args[0]), *(args[1]), *(args[2]), *(args[3]), *(args[4]),
			*(args[5]), *(args[6]), *(args[7]), *(args[8]))
	case 10:
		ret = tmpfn(*(args[0]), *(args[1]), *(args[2]), *(args[3]), *(args[4]),
			*(args[5]), *(args[6]), *(args[7]), *(args[8]), *(args[9]))
	}
	return ret
}

// return file:line
func getprintcaller(depth int, trimpfx bool) string {
	callers := Callers()
	assert(callers.len() > skip_depth)
	caller := callers[depth]
	// println(caller.File, ":", caller.Lineno, caller.Funcname, a0)
	trfile := trim_gopath(caller.File)
	linestr := caller.Lineno.repr()
	funcname := demangle_funcname(caller.Funcname)
	return trfile + ":" + linestr + " " + funcname
}

// with caller
func printx2(newline bool, args ...interface{}) int {
	fileline := getprintcaller(skip_depth+2, true)
	argc := args.len()
	if argc == 0 {
		if newline {
			C.printf("%.*s\n", fileline.len, fileline.ptr)
			return 0
		}
		return 0
	}

	varvals := []*variant{}
	for i := 0; i < argc; i++ {
		argx := args[i]
		var efc *eface = argx
		mty := efc._type
		var kind int = mty.kind
		var size int = mty.size
		varval := fmtstrbykind(kind, efc.data)
		// println(args.len(), i, kind, size, varval.repr)
		varvals = append(varvals, varval)
	}
	// println(varvals.len(), 111)

	res := ""
	for i := 0; i < varvals.len(); i++ {
		res += (varvals[i]).repr + " "
	}

	if newline {
		// res += newline // TODO compiler
		res += "\n"
	}
	rv := C.printf("%.*s %.*s".ptr, fileline.len, fileline.ptr, res.len, res.ptr)
	return rv
}

// without caller
func printx1(newline bool, args ...interface{}) int {
	// argc := len(args)
	argc := args.len()
	if argc == 0 {
		return 0
	}

	println(argc)
	varvals := []*variant{}
	for i := 0; i < argc; i++ {
		argx := args[i]
		var efc *eface = argx
		mty := efc._type
		var kind int = mty.kind
		var size int = mty.size
		println(args.len(), kind, size)
		varval := fmtstrbykind(kind, efc.data)
		varvals = append(varvals, varval)
	}
	println(varvals.len(), 111)
	fmtstr := ""
	for i := 0; i < varvals.len(); i++ {
		varval := varvals[i]
		fmtstr += varval.fmtstr + " "
	}
	if newline {
		fmtstr += "\n"
	}
	argptrs := []*voidptr{}
	addr1 := cxstring_ptraddr(fmtstr)
	argptrs = append(argptrs, addr1)
	for i := 0; i < varvals.len(); i++ {
		varval := varvals[i]
		argptrs = append(argptrs, varval.valpp)
	}

	println(fmtstr)
	printf_fnptr := dlsym1("printf")
	println(printf_fnptr)

	// unicall1(printf_fnptr, argptrs)
	res := ""
	for i := 0; i < varvals.len(); i++ {
		res += (varvals[i]).repr + " "
	}
	if newline {
		C.printf("%.*s\n".ptr, res.len, res.ptr)
	} else {
		C.printf("%.*s".ptr, res.len, res.ptr)
	}

	return 0
}

type variant struct {
	fmtstr string
	valpp  *voidptr
	repr   string
}

func fmtstrbykind(kind int, dato *voidptr) *variant {
	varval := &variant{}
	var fmtstr string
	valpp := dato

	mem := malloc3(64)
	switch kind {
	case Bool:
		fmtstr = "%d"
		ok := (int)(*dato) == 1
		fmtstr2 := "%s"
		if ok {
			C.sprintf(mem, fmtstr2.ptr, "true".ptr)
		} else {
			C.sprintf(mem, fmtstr2.ptr, "false".ptr)
		}
	case Int, Int16, Int32, Uint, Uint16, Uint32:
		fmtstr = "%d"
		C.sprintf(mem, fmtstr.ptr, (int)(*dato))
	case Int64:
		fmtstr = "%ld"
		C.sprintf(mem, fmtstr.ptr, (int64)(*dato))
	case Uint64:
		fmtstr = "%lu"
		C.sprintf(mem, fmtstr.ptr, (uint64)(*dato))
	case Uintptr:
		fmtstr = "%lu"
		C.sprintf(mem, fmtstr.ptr, (uintptr)(*dato))
	case Float32, Float64:
		fmtstr = "%g"
		C.sprintf(mem, fmtstr.ptr, *((*float64)(dato)))
	case Voidptr, UnsafePointer:
		fmtstr = "%p"
		C.sprintf(mem, fmtstr.ptr, *(dato))
	case String:
		fmtstr = "%s"
		// strp = C.cxstring_unrefpp(dato)
		// strp = cxstring_unrefpp(dato)
		// strp = cxstring_unref(dato)
		valpp = cxstring_unrefpp2(dato)
		mem = *valpp
	default:
		fmtstr = "unkmt%d"
		C.sprintf(mem, "unkmt-%d-%d".ptr, kind, *(dato))
	}
	varval.fmtstr = fmtstr
	varval.valpp = valpp
	varval.repr = string(mem)
	return varval
}

// good, works
func cxstring_ptraddr(cxstr string) voidptr {
	return &cxstr.ptr
}

// return cxstring.ptr
func cxstring_unrefpp(cxstrpp **usize) voidptr {
	var cxstrp string = *cxstrpp
	return cxstrp.ptr
}

// return &cxstring.ptr
func cxstring_unrefpp2(cxstrpp **usize) *voidptr {
	var cxstrp string = *cxstrpp
	return &cxstrp.ptr
}

// return cxstring
func cxstring_unref(cxstrpp **usize) string {
	var cxstrp string = *cxstrpp
	return cxstrp
}

func printx(a0 interface{}) {
	var efc *eface = a0
	mty := efc._type
	dato := efc.data
	var valpp *voidptr = dato
	// var strp voidptr

	// fmtstr := ""
	C.printf("eface mtyobj %p kind %d, dato %p\n".ptr, mty, mty.kind, dato)
	var kind int = mty.kind
	varval := fmtstrbykind(kind, dato)
	/*
		switch kind {
		case Int, Int16, Int32, Uint, Uint16, Uint32:
			fmtstr = "%d"
		case Int64:
			fmtstr = "%ld"
		case Uint64:
			fmtstr = "%uld"
		case Float32, Float64:
			fmtstr = "%g"
		case Voidptr, UnsafePointer:
			fmtstr = "%p"
		case String:
			fmtstr = "%s"
			strp = C.cxstring_unrefpp(dato)
			strp = cxstring_unrefpp(dato)
			strp = cxstring_unref(dato)
		default:
			fmtstr = "un%d"
		}
	*/
	// fmtstr += " by eface\n"
	fmtstr := varval.fmtstr + " by eface\n"

	C.printf(fmtstr.ptr, *varval.valpp)
	// if strp != nil {
	//	C.printf(fmtstr.ptr, strp)
	// } else {
	//	C.printf(fmtstr.ptr, *valpp)
	// }
}
func printint(a0 int) {
	callers := Callers()
	assert(callers.len() > skip_depth)
	caller := callers[skip_depth]
	// println(caller.File, ":", caller.Lineno, caller.Funcname, a0)
	trfile := trim_gopath(caller.File)
	funcname := demangle_funcname(caller.Funcname)
	C.printf("%s:%d %s %d\n".ptr, trfile.ptr, caller.Lineno, funcname.ptr, a0)
}
func printstr(a0 string) {
	callers := Callers()
	assert(callers.len() > skip_depth)
	caller := callers[skip_depth]
	// println(caller.File, ":", caller.Lineno, caller.Funcname, a0)
	trfile := trim_gopath(caller.File)
	funcname := demangle_funcname(caller.Funcname)
	C.printf("%s:%d %s %.*s\n".ptr,
		trfile.ptr, caller.Lineno, funcname.ptr, a0.len, a0.ptr)
}
func printptr(a0 voidptr) {
	callers := Callers()
	assert(callers.len() > skip_depth)
	caller := callers[skip_depth]
	// println(caller.File, ":", caller.Lineno, caller.Funcname, a0)
	trfile := trim_gopath(caller.File)
	funcname := demangle_funcname(caller.Funcname)
	C.printf("%s:%d %s %p\n".ptr, trfile.ptr, caller.Lineno, funcname.ptr, a0)
}
func printflt(a0 f64) {
	callers := Callers()
	assert(callers.len() > skip_depth)
	caller := callers[skip_depth]
	// println(caller.File, ":", caller.Lineno, caller.Funcname, a0)
	trfile := trim_gopath(caller.File)
	funcname := demangle_funcname(caller.Funcname)
	C.printf("%s:%d %s %g\n".ptr, trfile.ptr, caller.Lineno, funcname.ptr, a0)
}

func Keep() {}
