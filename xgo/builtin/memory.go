package builtin

/*
#cgo CFLAGS: -DGC_THREADS
#cgo LDFLAGS: -lgc -lpthread

#include <gc/gc.h>

extern void* crn_gc_malloc(size_t);
extern void* crn_gc_realloc(void*, size_t);
extern void crn_gc_free(void*);

extern void crn_set_finalizer(void* ptr, void* fn);

extern void GC_allow_register_threads();
*/
import "C"

/////////////////
//export crn_gc_malloc
func crn_gc_malloc(n usize) voidptr {
	x := C.GC_malloc(n)
	return x
}

//export crn_gc_realloc
func crn_gc_realloc(ptr voidptr, n usize) voidptr {
	x := C.GC_realloc(ptr, n)
	return x
}

//export crn_gc_free
func crn_gc_free(ptr voidptr) {
	C.GC_free(ptr)
	return
}

//export crn_set_finalizer
func crn_set_finalizer(ptr voidptr, fn voidptr) {
	//C.GC_register_finalizer()
	return
}

func memory_init() {
	C.GC_set_all_interior_pointers(1)
	gcok := C.GC_is_init_called()
	C.GC_set_finalize_on_demand(0)
	C.GC_set_free_space_divisor(3) // default 3
	C.GC_set_dont_precollect(1)
	C.GC_set_dont_expand(0)
	C.GC_allow_register_threads()

	C.GC_init()

	gcok2 := C.GC_is_init_called()
	println("done", gcok, gcok2)
}

func memory_deinit() {
	C.GC_deinit()
}

/////////////////

//export cxmalloc
func bdwgc_malloc(n usize) voidptr {
	ptr := C.crn_gc_malloc(n)
	return ptr
}

func mallocgc(n usize) voidptr {
	ptr := C.crn_gc_malloc(n)
	return ptr
}
func reallocgc(ptr voidptr, n usize) voidptr {
	ptr2 := C.crn_gc_realloc(ptr, n)
	return ptr2
}
func freegc(ptr voidptr) {
	C.crn_gc_free(ptr)
}

func mallocuc(n usize) voidptr {
	ptr := C.GC_malloc_uncollectable(n)
	return ptr
}

// raw c
func mallocrc(n usize) voidptr {
	ptr := C.malloc(n)
	return ptr
}
func reallocrc(ptr voidptr, n usize) voidptr {
	ptr2 := C.realloc(ptr, n)
	return ptr2
}
func freerc(ptr voidptr) {
	C.free(ptr)
}

//export cxrealloc
func bdwgc_realloc(ptr voidptr, size usize) voidptr {
	ptr2 := C.crn_gc_realloc(ptr, size)
	return ptr2
}

//export cxfree
func bdwgc_free(ptr voidptr) {
	C.crn_gc_free(ptr)
}

//export cxcalloc
func bdwgc_calloc(blocks usize, size usize) voidptr {
	ptr := C.crn_gc_malloc(blocks * size)
	return ptr
}

//export cxstrdup
func bdwgc_strdup(str byteptr) byteptr {
	ds := bdwgc_malloc(C.strlen(str) + 1)
	C.strcpy(ds, str)
	return ds
}

//export cxstrndup
func bdwgc_strndup(str byteptr, n int) byteptr {
	ds := bdwgc_malloc(n + 1)
	C.strncpy(ds, str, n)
	return ds
}

func bdwgc_memdup(ptr voidptr, sz int) voidptr {
	dp := bdwgc_malloc(sz)
	memcpy3(dp, ptr, sz)
	return dp
}

//export cxrt_set_finalizer
func set_finalizer(obj voidptr, fnptr voidptr) {
	C.crn_set_finalizer(obj, fnptr)
}
