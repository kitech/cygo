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
)

struct Loader {
	h voidptr
}

func Self() *Loader {
	h := C.dlopen(nil, NOW)
	assert (h != nil)
	o := &Loader{}
	o.h = h
	//SetFinalizer
	return o
}

func Open(filename string, flags int) *Loader {
	h := C.dlopen(filename.ptr, flags)
	if h == nil {
		return nil
	}
	o := &Loader{}
	o.h = h
	//SetFinalizer
	// println(fnsig2ptr(finclose), finclose)
	//println(fnsig2ptr(o.Close))
	return o
}

func (this*Loader) Close()  {
	finclose(this)
}
func finclose(this *Loader) {
	h := this.h
	this.h = nil
	C.dlclose(h)
}

func (this*Loader) Sym(sym string) voidptr {
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

