package xdl

/*
#include <dlfcn.h>
*/
import "C"

const (
	RTLD_LAZY         = C.RTLD_LAZY
	RTLD_NOW          = C.RTLD_NOW
	RTLD_BINDING_MASK = C.RTLD_BINDING_MASK
	RTLD_NOLOAD       = C.RTLD_NOLOAD
	RTLD_DEEPBIND     = C.RTLD_DEEPBIND
	RTLD_NODELETE     = C.RTLD_NODELETE

	RTLD_GLOBAL = C.RTLD_GLOBAL
	RTLD_LOCAL  = C.RTLD_LOCAL
)

func open(filename string) voidptr {
	return C.dlopen(filename.ptr, 0)
}
func close(handle voidptr) {
	C.dlclose(handle)
	return
}
func sym(handle voidptr, name string) voidptr {
	return C.dlsym(handle, name.ptr)
}

func error() string {
	p := C.dlerror()
	return C.GoString(p)
}
