package dl

/*
   #cgo CFLAGS: -D_GNU_SOURCE
   #cgo LDFLAGS: -ldl

   #include <dlfcn.h>

*/
import "C"

const (
	NOW = C.RTLD_NOW
	LAZY = C.RTLD_LAZY

	GLOBAL = C.RTLD_GLOBAL
	LOCAL = C.RTLD_LOCAL

	DEFAULT = C.RTLD_DEFAULT
	NEXT = C.RTLD_NEXT
)

struct Lib {
	h voidptr
}

func Self() *Lib {
	h := C.dlopen(nil, NOW)
	assert (h != nil)
	o := &Lib{}
	o.h = h
	//SetFinalizer
	return o
}

func Open(filename string, flags int) *Lib {
	println(filename,flags)
	h := C.dlopen(filename.ptr, flags)
	if h == nil {
		emsg := Error()
		println(h, emsg)
		return nil
	}
	o := &Lib{}
	o.h = h
	//SetFinalizer
	// println(fnsig2ptr(finclose), finclose)
	//println(fnsig2ptr(o.Close))
	return o
}

func (this*Lib) Close()  {
	finclose(this)
}
func finclose(this *Lib) {
	h := this.h
	this.h = nil
	if h == nil {
		return
	}
	C.dlclose(h)
}

func (this*Lib) Sym(sym string) voidptr {
	p := C.dlsym(this.h, sym.ptr)
	return p
}

func Error() string {
	p := C.dlerror()
	return refcstr(p)
}
func Haserr() bool {
	p := C.dlerror()
	return p != nil
}

struct Info {
	Fname string
	Fbase voidptr
	Sname string
	Saddr voidptr
}

func Addr(addr voidptr, info *Info) int {
	info2 := &C.Dl_info{}
	rv := C.dladdr(addr, info2)
	info.Fname = gostring_clone(info2.dli_fname)
	info.Sname = gostring_clone(info2.dli_sname)
	info.Fbase = info2.dli_fbase
	info.Saddr = info2.dli_saddr
	return rv
}

